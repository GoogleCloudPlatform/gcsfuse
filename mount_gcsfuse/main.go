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
// mount helpers. It simply calls the gcsfuse binary, which must be in $PATH,
// and waits for it to complete.
//
// mount_gcsfuse does not daemonize, and therefore must be used with a wrapper
// that performs daemonization if it is to be used directly with mount(8).
package main

import (
	"log"
	"os"
)

func main() {
	// Print out each argument.
	for i, arg := range os.Args {
		log.Printf("Arg %d: %q", i, arg)
	}

	os.Exit(1)
}
