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
// version name, producing .deb and .rpm files for Linux.
//
// Usage:
//
//	package_gcsfuse dst_dir version [commit]
//
// This will cause the gcsfuse git repo to be cloned to a temporary location
// and a build performed, embedding the given version name. The build will be
// performed at the given commit (or branch or tag), which defaults to
// `v<version>`.
//
// .deb and .rpm files will be written to dst_dir.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
)

func run(args []string) (err error) {
	osys := runtime.GOOS
	arch := runtime.GOARCH

	// Extract arguments.
	if len(args) < 2 || len(args) > 3 {
		err = fmt.Errorf("Usage: %s dst_dir version [commit]", os.Args[0])
		return
	}

	dstDir := args[0]
	version := args[1]

	commit := fmt.Sprintf("v%s", version)
	if len(args) >= 3 {
		commit = args[2]
	}

	log.Printf("Using settings:")
	log.Printf("  dstDir:  %s", dstDir)
	log.Printf("  commit:  %s", commit)
	log.Printf("  version: %s", version)

	// Ensure that all of the tools we need are present.
	err = checkForTools()
	if err != nil {
		return
	}

	// Assemble binaries, mount(8) helper scripts, etc.
	buildDir, err := build(commit, version, osys)
	if err != nil {
		err = fmt.Errorf("build: %w", err)
		return
	}

	defer os.RemoveAll(buildDir)

	// Write out .deb and .rpm files if we're building for Linux.
	if osys == "linux" {
		err = packageDeb(buildDir, version, osys, arch, dstDir)
		if err != nil {
			err = fmt.Errorf("packageDeb: %w", err)
			return
		}

		err = packageRpm(buildDir, version, osys, arch, dstDir)
		if err != nil {
			err = fmt.Errorf("packageDeb: %w", err)
			return
		}
	}

	return
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	flag.Parse()

	err := run(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
