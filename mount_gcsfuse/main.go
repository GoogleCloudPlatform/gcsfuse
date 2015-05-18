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

// mount_gcsfuse is a small helper for using gcsfuse with mount(8).
//
// mount_gcsfuse can be invoked using a command-line of the form expected for
// mount helpers. It calls the gcsfuse binary, which must be in $PATH, and
// waits for it to complete. The device is passed as --bucket, and other known
// options are converted to appropriate flags.
//
// mount_gcsfuse does not daemonize, and therefore must be used with a wrapper
// that performs daemonization if it is to be used directly with mount(8).
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// A 'name=value' mount option. If '=value' is not present, only the name will
// be filled in.
type Option struct {
	Name  string
	Value string
}

// Parse a single comma-separated list of mount options.
func parseOpts(s string) (opts []Option, err error) {
	// NOTE(jacobsa): The man pages don't define how escaping works, and as far
	// as I can tell there is no way to properly escape or quote a comma in the
	// options list for an fstab entry. So put our fingers in our ears and hope
	// that nobody needs a comma.
	for _, p := range strings.Split(s, ",") {
		var opt Option

		// Split on the first equals sign.
		if equalsIndex := strings.IndexByte(p, '='); equalsIndex != -1 {
			opt.Name = p[:equalsIndex]
			opt.Value = p[equalsIndex+1:]
		} else {
			opt.Name = p
		}

		opts = append(opts, opt)
	}

	return
}

// Attempt to parse the terrible undocumented format that mount(8) gives us.
// Return the 'device' (aka 'special' on OS X), the mount point, and a list of
// mount options encountered.
func parseArgs() (device string, mountPoint string, opts []Option, err error) {
	// Example invocation on OS X:
	//
	//     mount -t porp -o key_file=/some\ file.json -o ro,blah bucket ~/tmp/mp
	//
	// becomes the following arguments:
	//
	//     Arg 0: "/path/to/mount_gcsfuse"
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
	//     Arg 0: "/path/to/mount_gcsfuse"
	//     Arg 1: "bucket"
	//     Arg 2: "/path/to/mp"
	//     Arg 3: "-o"
	//     Arg 4: "rw,noexec,nosuid,nodev,user,key_file=/some file.json"
	//

	// Linux and OS X differ on the position of the options. So scan all
	// arguments (aside from the name of the binary), and:
	//
	//  *  Treat the first argument not following "-o" as the device name.
	//  *  Treat the second argument not following "-o" as the mount point.
	//  *  Treat the third argument not following "-o" as an error.
	//  *  Treat all arguments following "-o" as comma-separated options lists.
	//
	rawArgs := 0
	for i, arg := range os.Args {
		// Skip the binary name.
		if i == 0 {
			continue
		}

		// Skip "-o"; we will look back on the next iteration.
		if arg == "-o" {
			continue
		}

		// If the previous argument was "-o", this is a list of options.
		if os.Args[i-1] == "-o" {
			var tmp []Option
			tmp, err = parseOpts(arg)
			if err != nil {
				err = fmt.Errorf("parseOpts(%q): %v", arg, err)
				return
			}

			opts = append(opts, tmp...)
			continue
		}

		// Otherwise this is a non-option argument.
		switch rawArgs {
		case 0:
			device = arg

		case 1:
			mountPoint = arg

		default:
			err = fmt.Errorf(
				"Too many non-option arguments. The straw that broke the "+
					"camel's back: %q",
				arg)
			return
		}

		rawArgs++
	}

	// Did we see all of the raw arguments we expected?
	if rawArgs != 2 {
		err = fmt.Errorf("Expected 2 non-option arguments; got %d", rawArgs)
	}

	return
}

// Turn mount-style options into gcsfuse arguments. Skip known detritus that
// the mount command gives us.
//
// The result of this function should be appended to exec.Command.Args.
func makeGcsfuseArgs(
	device string,
	mountPoint string,
	opts []Option) (args []string, err error) {
	// Deal with options.
	for _, opt := range opts {
		switch opt.Name {
		case "key_file":
			args = append(args, "--key_file="+opt.Value)

		case "ro":
			args = append(args, "--read_only")

		// Ignore arguments for default and unsupported behavior automatically
		// added by mount(8) on Linux.
		case "rw":
		case "noexec":
		case "nosuid":
		case "nodev":
		case "user":

		default:
			err = fmt.Errorf(
				"Unrecognized mount option: %q (value %q)",
				opt.Name,
				opt.Value)

			return
		}
	}

	// Set up the bucket.
	args = append(args, "--bucket="+device)

	// Include the mount point last.
	args = append(args, mountPoint)

	return
}

func main() {
	// Print out each argument.
	//
	// TODO(jacobsa): Get rid of some or all of the debug logging.
	for i, arg := range os.Args {
		log.Printf("Arg %d: %q", i, arg)
	}

	// Attempt to parse arguments.
	device, mountPoint, opts, err := parseArgs()
	if err != nil {
		log.Fatalf("parseArgs: %v", err)
	}

	// Print what we gleaned.
	log.Printf("Device: %q", device)
	log.Printf("Mount point: %q", mountPoint)
	for _, opt := range opts {
		log.Printf("Option %q: %q", opt.Name, opt.Value)
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
