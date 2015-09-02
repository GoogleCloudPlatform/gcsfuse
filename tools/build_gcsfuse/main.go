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

// A tool that builds gcsfuse binaries, assembles helper scripts, etc., and
// writes them out to a destination directory with the correct hierarchy.
//
// Usage:
//
//     build_gcsfuse src_dir dst_dir
//
// where src_dir is the root of the gcsfuse git repository (or a tarball
// thereof).
//
// For Linux, writes the following to dst_dir:
//
//     bin/gcsfuse
//     bin/mount_gcsfuse
//     sbin/mount.gcsfuse
//
// For OS X:
//
//     bin/gcsfuse
//     bin/mount_gcsfuse
//     sbin/mount_gcsfuse
//
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

// Build release binaries according to the supplied settings, setting up the
// the file system structure we desire (see package-level comments).
//
// version is the gcsfuse version being built (e.g. "0.11.1"), or the empty
// string if this is not for an official release. commit is a short git commit
// (e.g. "33cbefc"), or the empty string if unknown.
func buildBinaries(
	dstDir string,
	srcDir string,
	version string,
	commit string) (err error) {
	// Create the target structure.
	{
		dirs := []string{
			"bin",
			"sbin",
		}

		for _, d := range dirs {
			err = os.Mkdir(path.Join(dstDir, d), 0755)
			if err != nil {
				err = fmt.Errorf("Mkdir: %v", err)
				return
			}
		}
	}

	// Create a directory to become GOPATH for our build below.
	gopath, err := ioutil.TempDir("", "build_gcsfuse_gopath")
	if err != nil {
		err = fmt.Errorf("TempDir: %v", err)
		return
	}

	defer os.RemoveAll(gopath)

	// Make it appear as if the source directory is at the appropriate position
	// in $GOPATH.
	gcsfuseDir := path.Join(gopath, "src/github.com/googlecloudplatform/gcsfuse")
	err = os.MkdirAll(path.Dir(gcsfuseDir), 0700)
	if err != nil {
		err = fmt.Errorf("MkdirAll: %v", err)
		return
	}

	err = os.Symlink(srcDir, gcsfuseDir)
	if err != nil {
		err = fmt.Errorf("Symlink: %v", err)
		return
	}

	// Build the binaries.
	binaries := []string{
		"github.com/googlecloudplatform/gcsfuse",
		"github.com/googlecloudplatform/gcsfuse/tools/mount_gcsfuse",
	}

	for _, bin := range binaries {
		log.Printf("Building %s", bin)

		cmd := exec.Command(
			"go",
			"build",
			"-o",
			path.Join(dstDir, "bin", path.Base(bin)),
			bin)

		cmd.Env = []string{
			"GO15VENDOREXPERIMENT=1",
			fmt.Sprintf("GOPATH=%s", gopath),
		}

		if path.Base(bin) == "gcsfuse" {
			var ldflags []string
			if version != "" {
				ldflags = append(
					ldflags,
					fmt.Sprintf("-X main.gcsfuseVersion=%s", version))
			}

			if commit != "" {
				ldflags = append(
					ldflags,
					fmt.Sprintf("-X main.gcsfuseCommit=%s", commit))
			}

			if len(ldflags) > 0 {
				cmd.Args = append(cmd.Args, "-ldflags", strings.Join(ldflags, " "))
			}
		}

		var output []byte
		output, err = cmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("Building %s: %v\nOutput:\n%s", bin, err, output)
			return
		}
	}

	// Copy the mount(8) helper script into place.
	err = writeMountHelper(runtime.GOOS, gopath, path.Join(dstDir, "sbin"))
	if err != nil {
		err = fmt.Errorf("writeMountHelper: %v", err)
		return
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

// Copy the mount(8) helper(s) into place from $GOPATH.
func writeMountHelper(
	osys string,
	gopath string,
	helperDir string) (err error) {
	// Choose the filename.
	var filename string
	switch osys {
	case "darwin":
		filename = "mount_gcsfuse"

	case "linux":
		filename = "mount.gcsfuse"

	default:
		err = fmt.Errorf("Unsupported OS: %q", osys)
		return
	}

	// Copy the file into place.
	err = copyFile(
		path.Join(helperDir, filename),
		path.Join(
			gopath,
			fmt.Sprintf(
				"src/github.com/googlecloudplatform/gcsfuse/tools/mount_gcsfuse/%s.sh",
				osys)),
		0755)

	if err != nil {
		err = fmt.Errorf("copyFile: %v", err)
		return
	}

	// On Linux, also support `mount -t fuse.gcsfuse`. If there's no explicit
	// helper for this type, /sbin/mount.fuse will call the gcsfuse executable
	// directly, but it doesn't support the right argument format and doesn't
	// daemonize. So we install an explicit helper.
	if osys == "linux" {
		err = os.Symlink("mount.gcsfuse", path.Join(helperDir, "mount.fuse.gcsfuse"))
		if err != nil {
			err = fmt.Errorf("Symlink: %v", err)
			return
		}
	}

	return
}
