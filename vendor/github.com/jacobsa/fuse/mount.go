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

package fuse

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Server is an interface for any type that knows how to serve ops read from a
// connection.
type Server interface {
	// Read and serve ops from the supplied connection until EOF. Do not return
	// until all operations have been responded to. Must not be called more than
	// once.
	ServeOps(*Connection)
}

// Mount attempts to mount a file system on the given directory, using the
// supplied Server to serve connection requests. It blocks until the file
// system is successfully mounted.
func Mount(
	dir string,
	server Server,
	config *MountConfig) (*MountedFileSystem, error) {
	// Sanity check: make sure the mount point exists and is a directory. This
	// saves us from some confusing errors later on OS X.
	if err := checkMountPoint(dir); err != nil {
		return nil, err
	}

	// Initialize the struct.
	mfs := &MountedFileSystem{
		dir:                 dir,
		joinStatusAvailable: make(chan struct{}),
	}

	// Begin the mounting process, which will continue in the background.
	if config.DebugLogger != nil {
		config.DebugLogger.Println("Beginning the mounting kickoff process")
	}
	ready := make(chan error, 1)
	dev, err := mount(dir, config, ready)
	if err != nil {
		return nil, fmt.Errorf("mount: %v", err)
	}
	if config.DebugLogger != nil {
		config.DebugLogger.Println("Completed the mounting kickoff process")
	}

	// Choose a parent context for ops.
	cfgCopy := *config
	if cfgCopy.OpContext == nil {
		cfgCopy.OpContext = context.Background()
	}

	if config.DebugLogger != nil {
		config.DebugLogger.Println("Creating a connection object")
	}
	// Create a Connection object wrapping the device.
	connection, err := newConnection(
		cfgCopy,
		config.DebugLogger,
		config.ErrorLogger,
		dev)
	if err != nil {
		return nil, fmt.Errorf("newConnection: %v", err)
	}
	if config.DebugLogger != nil {
		config.DebugLogger.Println("Successfully created the connection")
	}

	// Serve the connection in the background. When done, set the join status.
	go func() {
		server.ServeOps(connection)
		mfs.joinStatus = connection.close()
		close(mfs.joinStatusAvailable)
	}()

	if config.DebugLogger != nil {
		config.DebugLogger.Println("Waiting for mounting process to complete")
	}

	// Wait for the mount process to complete.
	if err := <-ready; err != nil {
		return nil, fmt.Errorf("mount (background): %v", err)
	}

	return mfs, nil
}

func checkMountPoint(dir string) error {
	if strings.HasPrefix(dir, "/dev/fd") {
		return nil
	}

	fi, err := os.Stat(dir)
	switch {
	case os.IsNotExist(err):
		return err

	case err != nil:
		return fmt.Errorf("Statting mount point: %v", err)

	case !fi.IsDir():
		return fmt.Errorf("Mount point %s is not a directory", dir)
	}

	return nil
}

func fusermount(binary string, argv []string, additionalEnv []string, wait bool, debugLogger *log.Logger) (*os.File, error) {
	if debugLogger != nil {
		debugLogger.Println("Creating a socket pair")
	}
	// Create a socket pair.
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("Socketpair: %v", err)
	}

	if debugLogger != nil {
		debugLogger.Println("Creating files to wrap the sockets")
	}
	// Wrap the sockets into os.File objects that we will pass off to fusermount.
	writeFile := os.NewFile(uintptr(fds[0]), "fusermount-child-writes")
	defer writeFile.Close()

	readFile := os.NewFile(uintptr(fds[1]), "fusermount-parent-reads")
	defer readFile.Close()

	if debugLogger != nil {
		debugLogger.Println("Starting fusermount/os mount")
	}
	// Start fusermount/mount_macfuse/mount_osxfuse.
	cmd := exec.Command(binary, argv...)
	cmd.Env = append(os.Environ(), "_FUSE_COMMFD=3")
	cmd.Env = append(cmd.Env, additionalEnv...)
	cmd.ExtraFiles = []*os.File{writeFile}
	cmd.Stderr = os.Stderr

	// Run the command.
	if wait {
		err = cmd.Run()
	} else {
		err = cmd.Start()
	}
	if err != nil {
		return nil, fmt.Errorf("running %v: %v", binary, err)
	}

	if debugLogger != nil {
		debugLogger.Println("Wrapping socket pair in a connection")
	}
	// Wrap the socket file in a connection.
	c, err := net.FileConn(readFile)
	if err != nil {
		return nil, fmt.Errorf("FileConn: %v", err)
	}
	defer c.Close()

	if debugLogger != nil {
		debugLogger.Println("Checking that we have a unix domain socket")
	}
	// We expect to have a Unix domain socket.
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return nil, fmt.Errorf("Expected UnixConn, got %T", c)
	}

	if debugLogger != nil {
		debugLogger.Println("Read a message from socket")
	}
	// Read a message.
	buf := make([]byte, 32) // expect 1 byte
	oob := make([]byte, 32) // expect 24 bytes
	_, oobn, _, _, err := uc.ReadMsgUnix(buf, oob)
	if err != nil {
		return nil, fmt.Errorf("ReadMsgUnix: %v", err)
	}

	// Parse the message.
	scms, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return nil, fmt.Errorf("ParseSocketControlMessage: %v", err)
	}

	// We expect one message.
	if len(scms) != 1 {
		return nil, fmt.Errorf("expected 1 SocketControlMessage; got scms = %#v", scms)
	}

	scm := scms[0]

	if debugLogger != nil {
		debugLogger.Println("Successfully read the socket message.")
	}

	// Pull out the FD returned by fusermount
	gotFds, err := syscall.ParseUnixRights(&scm)
	if err != nil {
		return nil, fmt.Errorf("syscall.ParseUnixRights: %v", err)
	}

	if len(gotFds) != 1 {
		return nil, fmt.Errorf("wanted 1 fd; got %#v", gotFds)
	}

	if debugLogger != nil {
		debugLogger.Println("Converting FD into os.File")
	}
	// Turn the FD into an os.File.
	return os.NewFile(uintptr(gotFds[0]), "/dev/fuse"), nil
}
