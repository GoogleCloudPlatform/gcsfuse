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
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
)

// Build release binaries according to the supplied settings, returning the
// path to a directory containing exactly root-relative file system structure
// we desire.
func buildBinaries(
	version string,
	commit string,
	osys string,
	arch string) (dir string, err error) {
	log.Printf("Building %s at %s for %s (%s).", version, commit, osys, arch)
	// Create a directory to hold our outputs. Kill it if we later return in
	// error.
	dir, err = ioutil.TempDir("", "build_release_binaries")
	if err != nil {
		err = fmt.Errorf("TempDir: %v", err)
		return
	}

	defer func() {
		if err != nil {
			os.RemoveAll(dir)
		}
	}()

	// Create the target structure.
	binDir, helperPaths, err := makeTarballDirs(osys, dir)
	if err != nil {
		err = fmt.Errorf("makeTarballDirs: %v", err)
		return
	}

	// Create another directory to become GOPATH for our build below.
	gopath, err := ioutil.TempDir("", "build_release_gopath")
	if err != nil {
		err = fmt.Errorf("TempDir: %v", err)
		return
	}

	defer os.RemoveAll(gopath)

	// Create a directory to store the source code.
	gitDir := path.Join(gopath, "src/github.com/googlecloudplatform/gcsfuse")
	err = os.MkdirAll(gitDir, 0700)
	if err != nil {
		err = fmt.Errorf("MkdirAll: %v", err)
		return
	}

	// Clone the source code into that directory.
	log.Printf("Cloning into %s", gitDir)

	cmd := exec.Command(
		"git",
		"clone",
		"https://github.com/GoogleCloudPlatform/gcsfuse.git",
		gitDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("Cloning: %v\nOutput:\n%s", err, output)
		return
	}

	// Check out the appropriate commit.
	cmd = exec.Command("git", "checkout", commit)
	cmd.Dir = gitDir

	output, err = cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("git checkout: %v\nOutput:\n%s", err, output)
		return
	}

	// Overwrite version.go with a file containing the appropriate constant.
	versionContents := fmt.Sprintf(
		"package main\nconst gcsfuseVersion = \"%s (commit %s)\"",
		version,
		commit)

	err = ioutil.WriteFile(
		path.Join(gitDir, "version.go"),
		[]byte(versionContents),
		0400)

	if err != nil {
		err = fmt.Errorf("WriteFile: %v", err)
		return
	}

	// Build the binaries.
	binaries := []string{
		"github.com/googlecloudplatform/gcsfuse",
		"github.com/googlecloudplatform/gcsfuse/tools/mount_gcsfuse",
	}

	for _, bin := range binaries {
		log.Printf("Building %s", bin)

		cmd = exec.Command(
			"go",
			"build",
			"-o",
			path.Join(dir, binDir, path.Base(bin)),
			bin)

		cmd.Env = []string{
			"GO15VENDOREXPERIMENT=1",
			fmt.Sprintf("GOPATH=%s", gopath),
			fmt.Sprintf("GOOS=%s", osys),
			fmt.Sprintf("GOARCH=%s", arch),
		}

		output, err = cmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("Building %s: %v\nOutput:\n%s", bin, err, output)
			return
		}
	}

	// Copy the mount(8) helper script into place.
	err = copyFile(
		path.Join(dir, helperPaths[0]),
		path.Join(
			gopath,
			fmt.Sprintf(
				"src/github.com/googlecloudplatform/gcsfuse/tools/mount_gcsfuse/%s.sh",
				osys)),
		0755)

	if err != nil {
		err = fmt.Errorf("Copying mount(8) helper script: %v", err)
		return
	}

	// Symlink any further mount(8) helpers to the first one.
	for i := range helperPaths {
		if i == 0 {
			continue
		}

		err = os.Symlink(helperPaths[0], path.Join(dir, helperPaths[i]))
		if err != nil {
			err = fmt.Errorf("Symlink: %v", err)
			return
		}
	}

	return
}

// Create the appropriate hierarchy for the tarball, returning the relative
// path of the directory to which the usual binaries should be written and the
// relative paths to which the mount(8) external helpers should be written.
func makeTarballDirs(
	osys string,
	baseDir string) (binDir string, helperPaths []string, err error) {
	// Fill out the return values.
	switch osys {
	case "darwin":
		binDir = "usr/local/bin"
		helperPaths = append(helperPaths, "sbin/mount_gcsfuse")

	case "linux":
		binDir = "usr/bin"
		helperPaths = append(helperPaths, "sbin/mount.gcsfuse")

		// Also support `mount -t fuse.gcsfuse`. If there's no explicit helper for
		// this type, /sbin/mount.fuse will call the gcsfuse executable directly,
		// but it doesn't support the right argument format and doesn't daemonize.
		// So we install an explicit helper.
		helperPaths = append(helperPaths, "sbin/mount.fuse.gcsfuse")

	default:
		err = fmt.Errorf("Don't know what directories to use for %s", osys)
		return
	}

	// Create the appropriate directories.
	dirs := []string{
		binDir,
	}

	for _, hp := range helperPaths {
		dirs = append(dirs, path.Dir(hp))
	}

	for _, d := range dirs {
		err = os.MkdirAll(path.Join(baseDir, d), 0755)
		if err != nil {
			err = fmt.Errorf("MkdirAll: %v", err)
			return
		}
	}

	return
}

func copyFile(dst string, src string, perm os.FileMode) (err error) {
	// Open the source.
	s, err := os.Open(src)
	if err != nil {
		return
	}

	defer s.Close()

	// Open the destination.
	d, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return
	}

	defer d.Close()

	// Copy contents.
	_, err = io.Copy(d, s)
	if err != nil {
		err = fmt.Errorf("Copy: %v", err)
		return
	}

	// Finish up.
	err = d.Close()
	if err != nil {
		return
	}

	return
}
