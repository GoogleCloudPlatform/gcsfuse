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

package integration_test

import (
	"log"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/util"
)

// A directory containing outputs created by build_gcsfuse, set up and deleted
// in TestMain.
var gBuildDir string

// On Linux, the path to fusermount, whose directory must be in gcsfuse's PATH
// variable in order to successfully mount. Set by TestMain.
var gFusermountPath string

func TestMain(m *testing.M) {
	// Parse flags from the setup.
	setup.ParseSetUpFlags()

	var err error

	// Find fusermount if we're running on Linux.
	if runtime.GOOS == "linux" {
		gFusermountPath, err = exec.LookPath("fusermount")
		if err != nil {
			log.Fatalf("LookPath(fusermount): %p", err)
		}
	}

	// Set up a directory into which we will build.
	gBuildDir, err = os.MkdirTemp("", "gcsfuse_integration_tests")
	if err != nil {
		log.Fatalf("TempDir: %p", err)
		return
	}

	// Build into that directory.
	if *setup.TestPackagePath == "" {
		err = util.BuildGcsfuse(gBuildDir)
		if err != nil {
			log.Fatalf("buildGcsfuse: %p", err)
			return
		}
	} else {
		err = setup.SetUpTestPackage(gBuildDir)
		if err != nil {
			log.Fatalf("SetUpTestPackage():%p", err)
			return
		}
	}

	// Run tests.
	code := m.Run()

	// Clean up and exit.
	os.RemoveAll(gBuildDir)
	os.Exit(code)
}
