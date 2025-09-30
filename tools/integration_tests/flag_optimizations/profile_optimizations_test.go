// Copyright 2025 Google LLC
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

package flag_optimizations

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type profileTests struct {
	suite.Suite
	flags []string
}

func (s *profileTests) SetupSuite() {
	if setup.MountedDirectory() != "" {
		setupForMountedDirectoryTests()
		return
	}
	// Mount GCSFuse.
	setup.MountGCSFuseWithGivenMountFunc(s.flags, testEnv.mountFunc)
}

func (s *profileTests) SetupTest() {
	setupForMountedDirectoryTests()
	mountGCSFuseAndSetupTestDir(s.flags, testEnv.ctx, testEnv.storageClient)
}

func (s *profileTests) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(testEnv.rootDir)
}

type aimlProfileTests struct {
	profileTests
}

type aimlTrainingProfileTests struct {
	aimlProfileTests
}

type aimlServingProfileTests struct {
	aimlProfileTests
}

type aimlCheckpointingProfileTests struct {
	aimlProfileTests
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (t *aimlProfileTests) TestImplicitDirsEnabled() {
	implicitDirPath := filepath.Join(testEnv.testDirPath, "implicitDir", setup.GenerateRandomString(5))
	client.CreateImplicitDir(testEnv.ctx, testEnv.storageClient, implicitDirPath, t.T())
	defer client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, implicitDirPath)

	mountedImplicitDirPath := filepath.Join(setup.MntDir(), implicitDirPath)
	assert.True(t.T(), operations.ValidateNoFileOrDirError(t.T(), mountedImplicitDirPath), "Expected directory %q but not found", mountedImplicitDirPath)
}

func (t *aimlTrainingProfileTests) TestUnnamedTrainingProfileTest() {
}

func (t *aimlServingProfileTests) TestUnnamedServingProfileTest() {
}

func (t *aimlCheckpointingProfileTests) TestUnnamedCheckpointingProfileTest() {
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestProfile(t *testing.T) {
	flagSet := func(profile string) [][]string {
		return [][]string{{"--profile", profile}}
	}
	tcName := func(flags []string) string {
		return strings.ReplaceAll(strings.Join(flags, ","), "--", "")
	}

	profiles := []string{"aiml-training", "aiml-serving", "aiml-checkpointing"}

	for _, profile := range profiles {
		t.Run(profile, func(t *testing.T) {
			var ts suite.TestingSuite
			var pTests *profileTests

			switch profile {
			case "aiml-training":
				s := &aimlTrainingProfileTests{}
				ts = s
				pTests = &s.profileTests
			case "aiml-serving":
				s := &aimlServingProfileTests{}
				ts = s
				pTests = &s.profileTests
			case "aiml-checkpointing":
				s := &aimlCheckpointingProfileTests{}
				ts = s
				pTests = &s.profileTests
			}

			if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
				// Run tests for mounted directory if the flag is set.
				suite.Run(t, ts)
				return
			}

			flagsSet := flagSet(profile)
			for _, flags := range flagsSet {
				pTests.flags = flags
				t.Run(tcName(flags), func(t *testing.T) {
					suite.Run(t, ts)
				})
			}
		})
	}
}
