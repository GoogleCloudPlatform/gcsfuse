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

// A helper that allows using gcsfuse with mount(8).
//
// Can be invoked using a command-line of the form expected for mount helpers.
// Calls the gcsfuse binary, which it finds from one of a list of expected
// locations, and waits for it to complete. The device and mount point are
// passed on as positional arguments, and other known options are converted to
// appropriate flags.
//
// This binary returns with exit code zero only after gcsfuse has reported that
// it has successfuly mounted the file system. Further output from gcsfuse is
// suppressed.
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
	"os"
	"os/exec"
	"path"
	"strings"

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
		// Don't pass through options that are relevant to mount(8) but not to
		// gcsfuse, and that fusermount chokes on with "Invalid argument" on Linux.
		case "user", "nouser", "auto", "noauto", "_netdev", "no_netdev":

		// Special case: support mount-like formatting for gcsfuse bool flags.
		case "implicit_dirs":
			args = append(
				args,
				"--"+strings.Replace(name, "_", "-", -1),
			)

		// Special case: support mount-like formatting for gcsfuse string flags.
		case "key_file":
			args = append(
				args,
				"--"+strings.Replace(name, "_", "-", -1),
				value,
			)

		// Special case: support mount-like formatting for gcsfuse string flags.
		case "dir_mode", "file_mode":
			args = append(
				args,
				"--"+strings.Replace(name, "_", "-", -1),
				value,
			)

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

		// Work around for systemd .mount file integration with gcsfuse
		case s == "-n":
			continue

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

	if positionalCount != 2 {
		err = fmt.Errorf("Expected two positional arguments; got %d.", positionalCount)
		return
	}

	return
}

func run(args []string) (err error) {
	// If invoked with a single "--help" argument, print a usage message and exit
	// successfully.
	if len(args) == 2 && args[1] == "--help" {
		fmt.Fprintf(
			os.Stderr,
			"Usage: %s [-o options] bucket_name mount_point\n",
			args[0])

		return
	}

	// Find the path to gcsfuse.
	gcsfusePath, err := findGcsfuse()
	if err != nil {
		err = fmt.Errorf("findGcsfuse: %v", err)
		return
	}

	// Find the path to fusermount.
	fusermountPath, err := findFusermount()
	if err != nil {
		err = fmt.Errorf("findFusermount: %v", err)
		return
	}

	// Attempt to parse arguments.
	device, mountPoint, opts, err := parseArgs(args)
	if err != nil {
		err = fmt.Errorf("parseArgs: %v", err)
		return
	}

	// Choose gcsfuse args.
	gcsfuseArgs, err := makeGcsfuseArgs(device, mountPoint, opts)
	if err != nil {
		err = fmt.Errorf("makeGcsfuseArgs: %v", err)
		return
	}

	fmt.Fprintf(
		os.Stderr,
		"Calling gcsfuse with arguments: %s\n",
		strings.Join(gcsfuseArgs, " "))

	// Run gcsfuse.
	cmd := exec.Command(gcsfusePath, gcsfuseArgs...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s", path.Dir(fusermountPath)))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("running gcsfuse: %v", err)
		return
	}

	return
}

func main() {
	err := run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
