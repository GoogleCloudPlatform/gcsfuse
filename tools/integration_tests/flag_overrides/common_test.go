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

package flag_overrides_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type mountTests struct {
	suite.Suite
	flags []string
}

func (s *mountTests) SetupTest() {
	setupForMountedDirectoryTests()
	mountGCSFuseAndSetupTestDir(s.flags, testEnv.ctx, testEnv.storageClient)
}

func (s *mountTests) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(testEnv.rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (t *mountTests) TestMount() {
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestCommon(t *testing.T) {
	ts := &mountTests{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	flagsSet := [][]string{}
	for _, machineType := range highEndMachines {
		flagsSet = append(flagsSet, []string{"--machine-type=" + machineType})
	}
	for _, profile := range supportedAIMLProfiles {
		flagsSet = append(flagsSet, []string{"--profile=" + profile})
	}

	for _, flags := range flagsSet {
		ts.flags = flags
		tcName := strings.ReplaceAll(strings.Join(flags, ","), "--", "")
		t.Run(tcName, func(t *testing.T) {
			suite.Run(t, ts)
		})
	}
}
