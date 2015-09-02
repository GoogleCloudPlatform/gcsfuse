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

// Perform a hermetic build of gcsfuse at a particular git tag specifying the
// version name, producing a release tarball. On Linux, also produce release
// packages.
//
// Usage:
//
//     package_gcsfuse dst_dir version
//
// This will cause the gcsfuse git repo to be cloned to a temporary location
// and a build performed at the given version tag. A tarball will be written to
// dst_dir, along with .deb and .rpm files if building on Linux
package main

import (
	"errors"
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
)

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

	// Use the compiled code's OS and architecture, allowing the user to override
	// in the environment.
	osys = build.Default.GOOS
	arch = build.Default.GOARCH

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

	// Write out .deb and maybe .rpm files if we're building for Linux.
	if osys == "linux" {
		err = packageDeb(binDir, version, osys, arch, *fOutputDir)
		if err != nil {
			err = fmt.Errorf("packageDeb: %v", err)
			return
		}

		if *fRPM {
			err = packageRpm(binDir, version, osys, arch, *fOutputDir)
			if err != nil {
				err = fmt.Errorf("packageDeb: %v", err)
				return
			}
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
