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
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/googlecloudplatform/gcsfuse/mount"
)

var fOptions = make(map[string]string)

func init() {
	flag.Var(
		mount.OptionValue(fOptions),
		"o",
		"Mount options. May be repeated.")
}

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
	for i, s := range args {
		switch {
		// Is this an options string following a "-o"?
		case i > 0 && args[i-1] == "-o":
			err = mount.ParseOptions(opts, s)
			if err != nil {
				err = fmt.Errorf("ParseOptions(%q): %v", s, err)
				return
			}

		default:
			err = fmt.Errorf("Unexpected arg: %q", s)
			return
		}
	}
}

func main() {
	flag.Parse()

	// Print out each argument.
	args := flag.Args()
	for i, arg := range args {
		log.Printf("Arg %d: %q", i, arg)
	}

	// Attempt to parse arguments.
	device, mountPoint, err := parseArgs(args)
	if err != nil {
		log.Fatalf("parseArgs: %v", err)
	}

	// Print what we gleaned.
	log.Printf("Device: %q", device)
	log.Printf("Mount point: %q", mountPoint)
	for name, value := range fOptions {
		log.Printf("Option %q: %q", name, value)
	}

	// Choose gcsfuse args.
	gcsfuseArgs, err := makeGcsfuseArgs(device, mountPoint, fOptions)
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
