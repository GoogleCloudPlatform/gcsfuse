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

package requester_pays_bucket

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type requesterPaysBucketTests struct {
	suite.Suite
	flags []string
}

func (s *requesterPaysBucketTests) SetupTest() {
	setupForMountedDirectoryTests()
	mountGCSFuseAndSetupTestDir(s.flags, testEnv.ctx, testEnv.storageClient)
}

func (s *requesterPaysBucketTests) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(testEnv.rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (t *requesterPaysBucketTests) TestDirOperations() {
	var err error
	dirName := "dir" + setup.GenerateRandomString(5)
	mountedDirPath := filepath.Join(testEnv.testDirPath, dirName)

	_, err = os.Stat(mountedDirPath)

	assert.Error(t.T(), err)

	err = os.Mkdir(mountedDirPath, setup.FilePermission_0600)

	assert.NoError(t.T(), err)

	_, err = os.Stat(mountedDirPath)

	assert.NoError(t.T(), err)

	renamedDirName := "dir" + setup.GenerateRandomString(5)
	mountedRenamedDirPath := filepath.Join(testEnv.testDirPath, renamedDirName)
	err = os.Rename(mountedDirPath, mountedRenamedDirPath)

	assert.NoError(t.T(), err)

	err = os.RemoveAll(mountedRenamedDirPath)

	assert.NoError(t.T(), err)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestOperations(t *testing.T) {
	// Helper functions to create flags, test case names etc.
	tcNameFromFlags := func(flags []string) string {
		if len(flags) > 0 {
			return strings.ReplaceAll(strings.Join(flags, ","), "--", "")
		} else {
			return "noflags"
		}
	}

	// Define test cases to be run.
	testCases := []struct {
		name  string
		flags []string
	}{
		{name: "WithoutBillingProject"},
		{name: "WithBillingProject", flags: []string{"--billing-project=gcs-fuse-test"}},
	}

	// Run test cases.
	ts := &requesterPaysBucketTests{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts.flags = tc.flags
			t.Run(tcNameFromFlags(ts.flags), func(t *testing.T) {
				suite.Run(t, ts)
			})
		})
	}
}
