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
	"errors"
	"io"
	"log"
	"os"
	"strconv"
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
	Succesful bool

	// Meaningful only if !Succesful.
	ErrorMsg string
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
		Succesful: outcome == nil,
	}

	if !msg.Succesful {
		msg.ErrorMsg = outcome.Error()
	}

	err = gGobEncoder.Encode(msg)
	return
}

// For use by gcsfuse: return a writer that should be used for logging status
// messages while in the process of mounting.
//
// The returned writer must not be written to after calling SignalOutcome.
func StatusWriter() (w io.Writer) {
	panic("TODO")
}

// Invoke gcsfuse with the supplied arguments, waiting until it successfully
// mounts or reports that is has failed. Write status updates while mounting
// into the supplied writer (which may be nil for silence). Return nil only if
// it mounts successfully.
func Mount(
	gcsfusePath string,
	args []string,
	status io.Writer) (err error) {
	err = errors.New("TODO")
	return
}
