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
	"errors"
	"log"
	"os"
)

// A 'name=value' mount option. If '=value' is not present, only the name will
// be filled in.
type Option struct {
	Name  string
	Value string
}

// Attempt to parse the terrible undocumented format that mount(8) gives us.
// Return the 'device' (aka 'special' on OS X), the mount point, and a list of
// mount options encountered.
func parseArgs() (device string, mountPoint string, opts []Option, err error) {
	// Example invocation on OS X:
	//
	//     mount -t porp -o key_file=/some\ file.json bucket ~/tmp/mp
	//
	// becomes the following arguments:
	//
	//     Arg 0: "/path/to/mount_gcsfuse"
	//     Arg 1: "-o"
	//     Arg 2: "key_file=/some file.json"
	//     Arg 3: "bucket"
	//     Arg 4: "/Users/jacobsa/tmp/mp"
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

	err = errors.New("TODO: parseArgs")
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
		return
	}

	// Print what we gleaned.
	log.Printf("Device: %q", device)
	log.Printf("Mount point: %q", mountPoint)
	for _, opt := range opts {
		log.Printf("Option %q: %q", opt.Name, opt.Value)
	}

	os.Exit(1)
}
