// Copyright 2021 Google Inc. All Rights Reserved.
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

package util

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
)

// Build bin/gcsfuse, sbin/mount_gcsfuse, etc. into the supplied directory.
func BuildGcsfuse(dstDir string) (err error) {
	// Ensure we have a copy of build_gcsfuse sitting around.
	var toolPath string
	{
		var toolDir string
		toolDir, err = os.MkdirTemp("", "gcsfuse_integration_tests")
		if err != nil {
			err = fmt.Errorf("TempDir: %w", err)
			return
		}

		defer os.RemoveAll(toolDir)

		toolPath = path.Join(toolDir, "build_gcsfuse")
		log.Printf("Building build_gcsfuse at %s", toolPath)

		err = buildBuildGcsfuse(toolPath)
		if err != nil {
			err = fmt.Errorf("buildBuildGcsfuse: %w", err)
			return
		}
	}

	// Figure out where we can find the source code for gcsfuse.
	var srcDir string
	{
		var pkg *build.Package
		pkg, err = build.Import(
			"github.com/googlecloudplatform/gcsfuse",
			"",
			build.FindOnly)

		if err != nil {
			err = fmt.Errorf("build.Import: %w", err)
			return
		}

		srcDir = pkg.Dir
	}

	// Use build_gcsfuse to perform a build.
	log.Printf("Building gcsfuse into %s", dstDir)

	{
		cmd := exec.Command(
			toolPath,
			srcDir,
			dstDir,
			"fake_version",
		)

		var output []byte
		output, err = cmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("build_gcsfuse: %w\nOutput:\n%s", err, output)
			return
		}
	}

	return
}

// Build the build_gcsfuse tool, writing the binary to the supplied path.
func buildBuildGcsfuse(dst string) (err error) {
	// Figure out where we can find the source for build_gcsfuse.
	var srcDir string
	{
		var pkg *build.Package
		pkg, err = build.Import(
			"github.com/googlecloudplatform/gcsfuse/tools/build_gcsfuse",
			"",
			build.FindOnly)

		if err != nil {
			err = fmt.Errorf("build.Import: %w", err)
			return
		}

		srcDir = pkg.Dir
	}

	// Create a directory to become GOPATH for our build below.
	gopath, err := os.MkdirTemp("", "build_gcsfuse_gopath")
	if err != nil {
		err = fmt.Errorf("TempDir: %w", err)
		return
	}
	defer os.RemoveAll(gopath)

	// Create a directory to become GOCACHE for our build below.
	var gocache string
	gocache, err = os.MkdirTemp("", "build_gcsfuse_gocache")
	if err != nil {
		err = fmt.Errorf("TempDir: %w", err)
		return
	}
	defer os.RemoveAll(gocache)

	// Build within that directory with no GOPATH -- it should have no external
	// dependencies besides the standard library.
	{
		cmd := exec.Command(
			"go", "build",
			"-o", dst,
		)

		cmd.Dir = srcDir
		cmd.Env = []string{
			fmt.Sprintf("GOROOT=%s", runtime.GOROOT()),
			fmt.Sprintf("GOPATH=%s", gopath),
			fmt.Sprintf("GOCACHE=%s", gocache),
		}

		var output []byte
		output, err = cmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("go build build_gcsfuse: %w\nOutput:\n%s", err, output)
			return
		}
	}

	return
}
