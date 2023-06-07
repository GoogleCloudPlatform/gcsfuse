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

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
)

// Build at the supplied commit (or branch or tag), embedding the given version
// name and returning a path to a directory containing exactly the
// root-relative file system structure we desire.
func build(
	commit string,
	version string,
	osys string) (dir string, err error) {
	log.Printf("Building version %s from %s.", version, commit)

	// Create a directory to become GOCACHE below.
	var gocache string
	gocache, err = os.MkdirTemp("", "package_gcsfuse_gocache")
	if err != nil {
		err = fmt.Errorf("TempDir: %w", err)
		return
	}
	defer os.RemoveAll(gocache)

	// Create a directory to hold our outputs. Kill it if we later return in
	// error.
	dir, err = os.MkdirTemp("", "package_gcsfuse_build")
	if err != nil {
		err = fmt.Errorf("TempDir: %w", err)
		return
	}

	defer func() {
		if err != nil {
			os.RemoveAll(dir)
		}
	}()

	// Create another directory into which we will clone the git repo bloe.
	gitDir, err := os.MkdirTemp("", "package_gcsfuse_git")
	if err != nil {
		err = fmt.Errorf("TempDir: %w", err)
		return
	}

	defer os.RemoveAll(gitDir)

	// Clone the git repo, checking out the correct tag.
	{
		log.Printf("Cloning into %s", gitDir)

		cmd := exec.Command(
			"git",
			"clone",
			"-b", commit,
			"https://github.com/GoogleCloudPlatform/gcsfuse.git",
			gitDir)

		var output []byte
		output, err = cmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("Cloning: %w\nOutput:\n%s", err, output)
			return
		}
	}

	// Build build_gcsfuse.
	buildTool := path.Join(gitDir, "build_gcsfuse")
	{
		log.Printf("Building build_gcsfuse...")

		cmd := exec.Command(
			"go",
			"build",
			"-o", buildTool,
		)

		cmd.Dir = path.Join(gitDir, "tools/build_gcsfuse")
		cmd.Env = []string{
			"GO15VENDOREXPERIMENT=1",
			fmt.Sprintf("GOROOT=%s", runtime.GOROOT()),
			fmt.Sprintf("GOCACHE=%s", gocache),
			"GOPATH=/does/not/exist",
		}

		var output []byte
		output, err = cmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("Building build_gcsfuse: %w\nOutput:\n%s", err, output)
			return
		}
	}

	// Run build_gcsfuse.
	{
		log.Printf("Running build_gcsfuse...")

		cmd := exec.Command(
			buildTool,
			gitDir,
			dir,
			version)

		var output []byte
		output, err = cmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("go run build_gcsfuse: %w\nOutput:\n%s", err, output)
			return
		}
	}

	// build_gcsfuse writes files like:
	//
	//     bin/gcsfuse
	//     sbin/mount.gcsfuse
	//
	// Which is what we want for e.g. a homebrew cellar. But for a Linux package,
	// we want the first to live in /usr/bin.
	err = os.MkdirAll(path.Join(dir, "usr"), 0755)
	if err != nil {
		err = fmt.Errorf("MkdirAll: %w", err)
		return
	}

	err = os.Rename(path.Join(dir, "bin"), path.Join(dir, "usr/bin"))
	if err != nil {
		err = fmt.Errorf("Rename: %w", err)
		return
	}

	return
}
