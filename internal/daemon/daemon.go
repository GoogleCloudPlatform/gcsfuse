// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Helper code for daemonizing gcsfuse, synchronizing on successful mount.
//
// The details of this package are subject to change.
package daemon

import (
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"syscall"
)

// The name of an environment variable used to communicate a file descriptor
// set up by Mount to the gcsfuse subprocess. Gob encoding is used to
// communicate back to Mount.
const envVar = "MOUNT_STATUS_FD"

// A message containing logging output for the process of mounting the file
// system.
type logMsg struct {
	Msg []byte
}

// A message indicating the outcome of the process of mounting the file system.
// The receiver ignores further messages.
type outcomeMsg struct {
	Successful bool

	// Meaningful only if !Successful.
	ErrorMsg string
}

func init() {
	gob.Register(logMsg{})
	gob.Register(outcomeMsg{})
}

// The file provded to this process via the environment variable, or nil if
// none.
var gFile *os.File

// A gob encoder that writes into gFile, or nil.
var gGobEncoder *gob.Encoder

func init() {
	// Is the environment variable set?
	fdStr, ok := os.LookupEnv(envVar)
	if !ok {
		return
	}

	// Parse the file descriptor.
	fd, err := strconv.ParseUint(fdStr, 10, 32)
	if err != nil {
		log.Fatalf("Couldn't parse %s value %q: %v", envVar, fdStr, err)
	}

	// Set up the file and the encoder that wraps it.
	gFile = os.NewFile(uintptr(fd), envVar)
	gGobEncoder = gob.NewEncoder(gFile)
}

// Send the supplied message as an interface{}, matching the decoder.
func sendMsg(msg interface{}) (err error) {
	err = gGobEncoder.Encode(&msg)
	return
}

// For use by gcsfuse: signal that mounting was successful (allowing the caller
// of the process to return in success) or that there was a failure to mount
// the file system (allowing the caller of the process to display an
// appropriate error message).
//
// Do nothing if the process wasn't invoked with Mount.
func SignalOutcome(outcome error) (err error) {
	// Is there anything to do?
	if gGobEncoder == nil {
		return
	}

	// Write out the outcome.
	msg := &outcomeMsg{
		Successful: outcome == nil,
	}

	if !msg.Successful {
		msg.ErrorMsg = outcome.Error()
	}

	err = sendMsg(msg)

	return
}

// An io.Writer that sends logMsg messages over gGobEncoder.
type logMsgWriter struct {
}

func (w *logMsgWriter) Write(p []byte) (n int, err error) {
	msg := &logMsg{
		Msg: p,
	}

	err = sendMsg(msg)
	if err != nil {
		return
	}

	n = len(p)
	return
}

// For use by gcsfuse: return a writer that should be used for logging status
// messages while in the process of mounting.
//
// The returned writer must not be written to after calling SignalOutcome.
func StatusWriter() (w io.Writer) {
	if gGobEncoder == nil {
		w = os.Stderr
		return
	}

	w = &logMsgWriter{}
	return
}

// Invoke gcsfuse with the supplied arguments, waiting until it successfully
// mounts or reports that is has failed. Write status updates while mounting
// into the supplied writer (which may be nil for silence). Return nil only if
// it mounts successfully.
func Mount(
	gcsfusePath string,
	fusermountPath string,
	args []string,
	status io.Writer) (err error) {
	if status == nil {
		status = ioutil.Discard
	}

	// Set up the pipe that we will hand to the gcsfuse subprocess.
	pipeR, pipeW, err := os.Pipe()
	if err != nil {
		err = fmt.Errorf("Pipe: %v", err)
		return
	}

	// Attempt to start gcsfuse. If we encounter an error in so doing, write it
	// to the channel.
	startGcsfuseErr := make(chan error, 1)
	go func() {
		defer pipeW.Close()
		err := startGcsfuse(gcsfusePath, fusermountPath, args, pipeW)
		if err != nil {
			startGcsfuseErr <- err
		}
	}()

	// Read communication from gcsfuse from the pipe, writing nil into the
	// channel only if the mount succeeds.
	readFromGcsfuseOutcome := make(chan error, 1)
	go func() {
		defer pipeR.Close()
		readFromGcsfuseOutcome <- readFromGcsfuse(pipeR, status)
	}()

	// Wait for a result from one of the above.
	select {
	case err = <-startGcsfuseErr:
		err = fmt.Errorf("startGcsfuse: %v", err)
		return

	case err = <-readFromGcsfuseOutcome:
		if err == nil {
			// All is good.
			return
		}

		err = fmt.Errorf("readFromGcsfuse: %v", err)
		return
	}
}

// Start gcsfuse, handing it the supplied pipe for communication. Do not wait
// for it to return.
func startGcsfuse(
	gcsfusePath string,
	fusermountPath string,
	args []string,
	pipeW *os.File) (err error) {
	cmd := exec.Command(gcsfusePath)
	cmd.Args = append(cmd.Args, args...)
	cmd.ExtraFiles = []*os.File{pipeW}

	// Change working directories so that we don't prevent unmounting of the
	// volume of our current working directory.
	cmd.Dir = "/"

	// Call setsid after forking in order to avoid being killed when the user
	// logs out.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	// Send along the write end of the pipe, and let gcsfuse find fusermount.
	cmd.Env = []string{
		fmt.Sprintf("%s=3", envVar),
		fmt.Sprintf("PATH=%s", path.Dir(fusermountPath)),
	}

	// Start. Clean up in the background, ignoring errors.
	err = cmd.Start()
	go cmd.Wait()

	return
}

// Process communication from a gcsfuse subprocess. Write log messages to the
// supplied writer (which must be non-nil). Return nil only if the output of
// mounting is success.
func readFromGcsfuse(
	r io.Reader,
	status io.Writer) (err error) {
	decoder := gob.NewDecoder(r)

	for {
		// Read a message.
		var msg interface{}
		err = decoder.Decode(&msg)
		if err != nil {
			err = fmt.Errorf("Decode: %v", err)
			return
		}

		// Handle the message.
		switch msg := msg.(type) {
		case logMsg:
			_, err = status.Write(msg.Msg)
			if err != nil {
				err = fmt.Errorf("status.Write: %v", err)
				return
			}

		case outcomeMsg:
			if msg.Successful {
				return
			}

			err = fmt.Errorf("gcsfuse: %s", msg.ErrorMsg)
			return

		default:
			err = fmt.Errorf("Unhandled message type: %T", msg)
			return
		}
	}
}
