// Copyright 2023 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Provide integration tests for multi-writers writing to multiple GCSFuse mounts.
package multimounts_test

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/multiple_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	MountName1     = "mnt1"
	MountName2     = "mnt2"
	CommonFileName = "data.txt"
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// Disabling stat cache because that can go stale and interfere with tests.
	// It is anyway advised to users to disable stat cache in multiwriter scenarios.
	flags := [][]string{{"--implicit-dirs=false", "--stat-cache-capacity=0"}, {"--implicit-dirs=true", "--stat-cache-capacity=0"}}

	if setup.TestBucket() == "" && setup.MountedDirectory() != "" {
		log.Print("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
		os.Exit(1)
	}

	// Requires two GCSFuse mount inside MountedDirectory named as MountName1 &
	// MountName 2.
	// If --mountedDirectory flag is passed the program exits from inside this
	// method.
	setup.RunTestsForMountedDirectoryFlag(m)

	setup.SetUpTestDirForTestBucketFlag()

	mountPaths := []string{filepath.Join(setup.MntDir(), MountName1), filepath.Join(setup.MntDir(), MountName2)}
	successCode := multiple_mounting.RunTests(flags, mountPaths, m)

	setup.RemoveBinFileCopiedForTesting()

	os.Exit(successCode)
}
