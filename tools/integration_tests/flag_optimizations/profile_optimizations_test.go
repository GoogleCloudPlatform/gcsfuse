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
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type profileTests struct {
	suite.Suite
	flags []string
}

func (s *profileTests) SetupTest() {
	setupForMountedDirectoryTests()
	mountGCSFuseAndSetupTestDir(s.flags, testEnv.ctx, testEnv.storageClient)
}

func (s *profileTests) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(testEnv.rootDir)
}

type aimlTrainingProfileTests struct {
	profileTests
}

func (s *aimlTrainingProfileTests) SetupTest() {
	s.profileTests.SetupTest()
}

func (s *aimlTrainingProfileTests) TearDownTest() {
	s.profileTests.TearDownTest()
}

type aimlServingProfileTests struct {
	profileTests
}

func (s *aimlServingProfileTests) SetupTest() {
	s.profileTests.SetupTest()
}

func (s *aimlServingProfileTests) TearDownTest() {
	s.profileTests.TearDownTest()
}

type aimlCheckpointingProfileTests struct {
	profileTests
}

func (s *aimlCheckpointingProfileTests) SetupTest() {
	s.profileTests.SetupTest()
}

func (s *aimlCheckpointingProfileTests) TearDownTest() {
	s.profileTests.TearDownTest()
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (t *aimlTrainingProfileTests) TestUnnamedTest() {
	t.T().Log("Unnamed test for AIML training profile")
}

func (t *aimlServingProfileTests) TestUnnamedTest() {
	t.T().Log("Unnamed test for AIML serving profile")
}

func (t *aimlCheckpointingProfileTests) TestUnnamedTest() {
	t.T().Log("Unnamed test for AIML checkpointing profile")
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

	profile := "aiml-training"
	t.Run(profile, func(t *testing.T) {
		ts := &aimlTrainingProfileTests{}
		if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
			// Run tests for mounted directory if the flag is set.
			suite.Run(t, ts)
			return
		}

		flagsSet := flagSet(profile)
		for _, flags := range flagsSet {
			ts.flags = flags
			t.Run(tcName(flags), func(t *testing.T) {
				suite.Run(t, ts)
			})
		}
	})
	profile = "aiml-serving"
	t.Run(profile, func(t *testing.T) {
		ts := &aimlTrainingServingTests{}
		if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
			// Run tests for mounted directory if the flag is set.
			suite.Run(t, ts)
			return
		}

		flagsSet := flagSet(profile)
		for _, flags := range flagsSet {
			ts.flags = flags
			t.Run(tcName(flags), func(t *testing.T) {
				suite.Run(t, ts)
			})
		}
	})
	profile = "aiml-checkpointing"
	t.Run(profile, func(t *testing.T) {
		ts := &aimlTrainingCheckpointingTests{}
		if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
			// Run tests for mounted directory if the flag is set.
			suite.Run(t, ts)
			return
		}

		flagsSet := flagSet(profile)
		for _, flags := range flagsSet {
			ts.flags = flags
			t.Run(tcName(flags), func(t *testing.T) {
				suite.Run(t, ts)
			})
		}
	})
}
