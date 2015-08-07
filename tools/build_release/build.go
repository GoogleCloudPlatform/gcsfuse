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

// Build release binaries according to the supplied settings, returning the
// path to a directory containing exactly the output binaries.
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

	// TODO(jacobsa): Overwrite version.go.

	// Build the binaries.
	binaries := []string{
		"github.com/googlecloudplatform/gcsfuse",
		"github.com/googlecloudplatform/gcsfuse/gcsfuse_mount_helper",
	}

	for _, bin := range binaries {
		log.Printf("Building %s", bin)

		cmd = exec.Command(
			"go",
			"build",
			"-o",
			path.Join(dir, path.Base(bin)),
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

	return
}
