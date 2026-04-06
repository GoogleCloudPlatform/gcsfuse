// Copyright 2025 Google LLC
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

package release_version

import (
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.ReleaseVersion) == 0 {
		log.Println("No configuration found for release_version tests in config. Using flags instead.")
		cfg.ReleaseVersion = make([]test_suite.TestConfig, 1)
		cfg.ReleaseVersion[0].TestBucket = setup.TestBucket()
		cfg.ReleaseVersion[0].GKEMountedDirectory = setup.MountedDirectory()
	}

	// 2. Not running mounted directory tests.
	if cfg.ReleaseVersion[0].GKEMountedDirectory != "" {
		log.Print("These tests will not run for mountedDirectory flag.")
		os.Exit(0)
	}

	// 3. The release_version test doesn't mount anything, but it needs the gcsfuse binary.
	setup.SetUpTestDirForTestBucket(&cfg.ReleaseVersion[0])

	// 4. Run tests.
	successCode := m.Run()
	os.Exit(successCode)
}
