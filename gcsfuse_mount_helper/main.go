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

// A small helper for using gcsfuse with mount(8).
//
// Can be invoked using a command-line of the form expected for mount helpers.
// Calls the gcsfuse binary, which must be in $PATH, and waits for it to
// complete. The device is passed as --bucket, and other known options are
// converted to appropriate flags.
//
// This binary does not daemonize, and therefore must be used with a wrapper
// that performs daemonization if it is to be used directly with mount(8).
package main

// Example invocation on OS X:
//
//     mount -t porp -o key_file=/some\ file.json -o ro,blah bucket ~/tmp/mp
//
// becomes the following arguments:
//
//     Arg 0: "/path/to/gcsfuse_mount_helper"
//     Arg 1: "-o"
//     Arg 2: "key_file=/some file.json"
//     Arg 3: "-o"
//     Arg 4: "ro"
//     Arg 5: "-o"
//     Arg 6: "blah"
//     Arg 7: "bucket"
//     Arg 8: "/path/to/mp"
//
// On Linux, the fstab entry
//
//     bucket /path/to/mp porp user,key_file=/some\040file.json
//
// becomes
//
//     Arg 0: "/path/to/gcsfuse_mount_helper"
//     Arg 1: "bucket"
//     Arg 2: "/path/to/mp"
//     Arg 3: "-o"
//     Arg 4: "rw,noexec,nosuid,nodev,user,key_file=/some file.json"
//

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/googlecloudplatform/gcsfuse/mount"
)

// Turn mount-style options into gcsfuse arguments. Skip known detritus that
// the mount command gives us.
//
// The result of this function should be appended to exec.Command.Args.
func makeGcsfuseArgs(
	device string,
	mountPoint string,
	opts map[string]string) (args []string, err error) {
	// Deal with options.
	for name, value := range opts {
		switch name {
		case "key_file":
			args = append(args, "--key_file="+value)

		case "fuse_debug":
			args = append(args, "--fuse.debug")

		case "gcs_debug":
			args = append(args, "--gcs.debug")

		case "uid":
			args = append(args, "--uid="+value)

		case "gid":
			args = append(args, "--gid="+value)

		case "file_mode":
			args = append(args, "--file_mode="+value)

		case "dir_mode":
			args = append(args, "--dir_mode="+value)

		// On Linux, option 'user' is necessary for mount(8) to let a non-root user
		// mount a file system. It is passed through to us, but we don't want to
		// pass it on to gcsfuse because fusermount chokes on it with
		//
		//     fusermount: mount failed: Invalid argument
		//
		case "user":

		// Pass through everything else.
		default:
			var formatted string
			if value == "" {
				formatted = name
			} else {
				formatted = fmt.Sprintf("%s=%s", name, value)
			}

			args = append(args, "-o", formatted)
		}
	}

	// Set the bucket.
	args = append(args, "--bucket="+device)

	// Set the mount point.
	args = append(args, "--mount_point="+mountPoint)

	return
}

// Parse the supplied command-line arguments from a mount(8) invocation on OS X
// or Linux.
func parseArgs(
	args []string) (
	device string,
	mountPoint string,
	opts map[string]string,
	err error) {
	opts = make(map[string]string)

	// Process each argument in turn.
	positionalCount := 0
	for i, s := range args {
		switch {
		// Skip the program name.
		case i == 0:
			continue

		// "-o" is illegal only when at the end. We handle its argument in the case
		// below.
		case s == "-o":
			if i == len(args)-1 {
				err = fmt.Errorf("Unexpected -o at end of args.")
				return
			}

		// Is this an options string following a "-o"?
		case i > 0 && args[i-1] == "-o":
			err = mount.ParseOptions(opts, s)
			if err != nil {
				err = fmt.Errorf("ParseOptions(%q): %v", s, err)
				return
			}

		// Is this the device?
		case positionalCount == 0:
			device = s
			positionalCount++

		// Is this the mount point?
		case positionalCount == 1:
			mountPoint = s
			positionalCount++

		default:
			err = fmt.Errorf("Unexpected arg %d: %q", i, s)
			return
		}
	}

	return
}

func main() {
	// Print out each argument.
	args := os.Args
	for i, arg := range args {
		log.Printf("Arg %d: %q", i, arg)
	}

	// Attempt to parse arguments.
	device, mountPoint, opts, err := parseArgs(args)
	if err != nil {
		log.Fatalf("parseArgs: %v", err)
	}

	// Print what we gleaned.
	log.Printf("Device: %q", device)
	log.Printf("Mount point: %q", mountPoint)
	for name, value := range opts {
		log.Printf("Option %q: %q", name, value)
	}

	// Choose gcsfuse args.
	gcsfuseArgs, err := makeGcsfuseArgs(device, mountPoint, opts)
	if err != nil {
		log.Fatalf("makeGcsfuseArgs: %v", err)
	}

	for _, a := range gcsfuseArgs {
		log.Printf("gcsfuse arg: %q", a)
	}

	// Run gcsfuse and wait for it to complete.
	cmd := exec.Command("gcsfuse", gcsfuseArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Fatalf("gcsfuse failed or failed to run: %v", err)
	}

	log.Println("gcsfuse completed successfully.")
}
