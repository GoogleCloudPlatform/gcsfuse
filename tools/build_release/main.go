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

// Perform a hermetic build of gcsfuse at a particular version, producing
// release binaries and packages.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
)

var fVersion = flag.String("version", "", "Version number of the release.")
var fCommit = flag.String("commit", "", "Commit at which to build.")
var fOS = flag.String("os", "", "OS for which to build, e.g. linux or darwin.")
var fArch = flag.String("arch", "amd64", "Architecture for which to build.")

func run() (err error) {
	err = errors.New("TODO")
	return
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	flag.Parse()

	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
