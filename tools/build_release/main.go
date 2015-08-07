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
var fOutputDir = flag.String("output_dir", "", "Where to write outputs.")

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func getSettings() (version, commit, osys, arch, outputDir string, err error) {
	if *fVersion == "" {
		err = errors.New("You must set --version.")
		return
	}
	version = *fVersion

	if *fCommit == "" {
		err = errors.New("You must set --commit.")
		return
	}
	commit = *fCommit

	if *fOS == "" {
		err = errors.New("You must set --os.")
		return
	}
	osys = *fOS

	if *fArch == "" {
		err = errors.New("You must set --arch.")
		return
	}
	arch = *fArch

	// Output dir
	outputDir = *fOutputDir
	if outputDir == "" {
		outputDir, err = os.Getwd()
		if err != nil {
			err = fmt.Errorf("Getwd: %v", err)
			return
		}
	}

	return
}

////////////////////////////////////////////////////////////////////////
// main logic
////////////////////////////////////////////////////////////////////////

func run() (err error) {
	// Ensure that all of the tools we need are present.
	err = checkForTools()
	if err != nil {
		return
	}

	// Read flags.
	version, commit, osys, arch, outputDir, err := getSettings()
	if err != nil {
		return
	}

	// Build release binaries.
	binDir, err := buildBinaries(version, commit, osys, arch)
	if err != nil {
		err = fmt.Errorf("buildBinaries: %v", err)
		return
	}

	defer os.RemoveAll(binDir)

	// Write out a tarball.
	err = packageTarball(binDir, version, osys, arch, outputDir)
	if err != nil {
		err = fmt.Errorf("packageTarball: %v", err)
		return
	}

	// Write out a .deb file if we're building for Linux.
	if osys == "linux" {
		err = packageDeb(binDir, version, osys, arch, *fOutputDir)
		if err != nil {
			err = fmt.Errorf("packageDeb: %v", err)
			return
		}
	}

	return
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	flag.Parse()

	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
