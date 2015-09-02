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
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
)

// Build at the supplied commit (or branch or tag), embedding the given version
// name and returning a path to a directory containing exactly the
// root-relative file system structure we desire.
func build(
	commit string,
	version string,
	osys string) (dir string, err error) {
	log.Printf("Building version %s from %s.", version, commit)

	// Create a directory to hold our outputs. Kill it if we later return in
	// error.
	dir, err = ioutil.TempDir("", "package_gcsfuse_build")
	if err != nil {
		err = fmt.Errorf("TempDir: %v", err)
		return
	}

	defer func() {
		if err != nil {
			os.RemoveAll(dir)
		}
	}()

	// Set up the destination for a call to build_gcsfuse, which writes files
	// like
	//
	//     bin/gcsfuse
	//     sbin/mount.gcsfuse
	//
	// On Linux and OS X we want these to go into different places.
	var buildDir string
	switch osys {
	case "linux":
		buildDir = path.Join(dir, "usr")

	case "darwin":
		buildDir = path.Join(dir, "usr/local")

	default:
		err = fmt.Errorf("Unhandled OS: %q", osys)
		return
	}

	err = os.MkdirAll(buildDir, 0755)
	if err != nil {
		err = fmt.Errorf("MkdirAll: %v", err)
		return
	}

	// Create another directory into which we will clone the git repo bloe.
	gitDir, err := ioutil.TempDir("", "package_gcsfuse_git")
	if err != nil {
		err = fmt.Errorf("TempDir: %v", err)
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
			err = fmt.Errorf("Cloning: %v\nOutput:\n%s", err, output)
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
			"GOPATH=/does/not/exist",
		}

		var output []byte
		output, err = cmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("Building build_gcsfuse: %v\nOutput:\n%s", err, output)
			return
		}
	}

	// Run build_gcsfuse.
	{
		log.Printf("Running build_gcsfuse...")

		cmd := exec.Command(
			buildTool,
			gitDir,
			buildDir,
			version)

		var output []byte
		output, err = cmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("go run build_gcsfuse: %v\nOutput:\n%s", err, output)
			return
		}
	}

	// Add symlink(s) from /sbin to /usr/sbin or /usr/local/sbin, as the case may
	// be.
	{
		symlinks := map[string]string{}
		switch osys {
		case "linux":
			symlinks["sbin/mount.fuse.gcsfuse"] = "/usr/sbin/mount.fuse.gcsfuse"
			symlinks["sbin/mount.gcsfuse"] = "/usr/sbin/mount.gcsfuse"

		case "darwin":
			symlinks["sbin/mount_gcsfuse"] = "/usr/sbin/mount_gcsfuse"

		default:
			err = fmt.Errorf("Unhandled OS: %q", osys)
			return
		}

		for relativeSrc, target := range symlinks {
			src := path.Join(dir, relativeSrc)

			err = os.MkdirAll(path.Dir(src), 0755)
			if err != nil {
				err = fmt.Errorf("MkdirAll: %v", err)
				return
			}

			err = os.Symlink(target, src)
			if err != nil {
				err = fmt.Errorf("Symlink: %v", err)
				return
			}
		}
	}

	return
}
