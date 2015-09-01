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
// complete. The device and mount point are passed on as positional arguments,
// and other known options are converted to appropriate flags.
//
// This binary does not daemonize, and therefore must be used with a wrapper
// that performs daemonization if it is to be used directly with mount(8).
package main

// Example invocation on OS X:
//
//     mount -t porp -o foo=bar\ baz -o ro,blah bucket ~/tmp/mp
//
// becomes the following arguments:
//
//     Arg 0: "/sbin/mount_gcsfuse "
//     Arg 1: "-o"
//     Arg 2: "foo=bar baz"
//     Arg 3: "-o"
//     Arg 4: "ro"
//     Arg 5: "-o"
//     Arg 6: "blah"
//     Arg 7: "bucket"
//     Arg 8: "/path/to/mp"
//
// On Linux, the fstab entry
//
//     bucket /path/to/mp porp user,foo=bar\040baz
//
// becomes
//
//     Arg 0: "/sbin/mount.gcsfuse"
//     Arg 1: "bucket"
//     Arg 2: "/path/to/mp"
//     Arg 3: "-o"
//     Arg 4: "rw,noexec,nosuid,nodev,user,foo=bar baz"
//

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/googlecloudplatform/gcsfuse/internal/mount"
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
		case "fuse_debug":
			args = append(args, "--fuse.debug")

		case "gcs_debug":
			args = append(args, "--gcs.debug")

		case "uid":
			args = append(args, "--uid="+value)

		case "gid":
			args = append(args, "--gid="+value)

		case "file_mode":
			args = append(args, "--file-mode="+value)

		case "dir_mode":
			args = append(args, "--dir-mode="+value)

		// Don't pass through options that are relevant to mount(8) but not to
		// gcsfuse, and that fusermount chokes on with "Invalid argument" on Linux.
		case "user", "nouser", "auto", "noauto":

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

	// Set the bucket and mount point.
	args = append(args, device, mountPoint)

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
			mount.ParseOptions(opts, s)

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
	args := os.Args

	// If invoked with a single "--help" argument, print a usage message and exit
	// successfully.
	if len(args) == 2 && args[1] == "--help" {
		fmt.Fprintf(
			os.Stderr,
			"Usage: %s [-o options] bucket_name mount_point\n",
			os.Args[0])

		os.Exit(0)
	}

	// Print out each argument.
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
