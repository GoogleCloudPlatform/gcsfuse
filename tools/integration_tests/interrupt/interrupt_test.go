// Copyright 2024 Google Inc. All Rights Reserved.
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

package interrupt

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	testDirName = "InterruptTest"
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	setup.RunTestsForMountedDirectoryFlag(m)

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Set up flags to run tests on.
	mountConfig := config.MountConfig{
		FileSystemConfig: config.FileSystemConfig{IgnoreInterrupts: true},
		LogConfig: config.LogConfig{
			Severity:        config.TRACE,
			LogRotateConfig: config.DefaultLogRotateConfig(),
		},
	}
	flags := [][]string{{"--implicit-dirs=true", "--ignore-interrupts"},
		{"--config-file=" + setup.YAMLConfigFile(mountConfig, "config.yaml")}}

	successCode := static_mounting.RunTests(flags, m)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
