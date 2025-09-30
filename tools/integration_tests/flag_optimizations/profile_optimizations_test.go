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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type noProfileTests struct {
	profileTests
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

func (t *noProfileTests) TestNoImplicitDirsEnabled() {
	if setup.ResolveIsHierarchicalBucket(testEnv.ctx, setup.TestBucket(), testEnv.storageClient) {
		t.T().Skipf("test not applicable for HNS buckets")
	}
	implicitDirPath := filepath.Join(testDirName, "implicitDir", setup.GenerateRandomString(5))
	mountedImplicitDirPath := filepath.Join(setup.MntDir(), implicitDirPath)
	client.CreateImplicitDir(testEnv.ctx, testEnv.storageClient, implicitDirPath, t.T())
	defer client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, implicitDirPath)

	_, err := os.Stat(mountedImplicitDirPath)

	require.Error(t.T(), err, "Found unexpected implicit directory %q", mountedImplicitDirPath)
}

func (t *noProfileTests) TestZeroRenameDirLimit() {
	if setup.ResolveIsHierarchicalBucket(testEnv.ctx, setup.TestBucket(), testEnv.storageClient) {
		t.T().Skipf("test not applicable for HNS buckets")
	}
	srcDirPath := filepath.Join(testDirName, "srcDirContainingFiles", setup.GenerateRandomString(5))
	mountedSrcDirPath := filepath.Join(setup.MntDir(), srcDirPath)
	dstDirPath := filepath.Join(testDirName, "dstDirContainingFiles", setup.GenerateRandomString(5))
	mountedDstDirPath := filepath.Join(setup.MntDir(), dstDirPath)
	client.CreateGcsDir(testEnv.ctx, testEnv.storageClient, srcDirPath, setup.TestBucket(), "")
	client.CreateNFilesInDir(testEnv.ctx, testEnv.storageClient, 1, "file", 1024, srcDirPath, t.T())
	defer client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, srcDirPath)

	err := os.Rename(mountedSrcDirPath, mountedDstDirPath)

	require.Error(t.T(), err, "Unexpectedly succeeded in renaming directory %q to %q", mountedSrcDirPath, mountedDstDirPath)
}

func (t *aimlProfileTests) TestImplicitDirsEnabled() {
	if setup.ResolveIsHierarchicalBucket(testEnv.ctx, setup.TestBucket(), testEnv.storageClient) {
		t.T().Skipf("test not applicable for HNS buckets")
	}
	implicitDirPath := filepath.Join(testDirName, "implicitDir", setup.GenerateRandomString(5))
	mountedImplicitDirPath := filepath.Join(setup.MntDir(), implicitDirPath)
	client.CreateImplicitDir(testEnv.ctx, testEnv.storageClient, implicitDirPath, t.T())
	defer client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, implicitDirPath)

	fi, err := os.Stat(mountedImplicitDirPath)

	require.NoError(t.T(), err, "Got error statting %q: %v", mountedImplicitDirPath, err)
	require.NotNil(t.T(), fi, "Expected directory %q", mountedImplicitDirPath)
	assert.True(t.T(), fi.IsDir(), "Expected %q to be a directory, but got not-dir", mountedImplicitDirPath)
}

func (t *aimlCheckpointingProfileTests) TestNonZeroRenameDirLimit() {
	if setup.ResolveIsHierarchicalBucket(testEnv.ctx, setup.TestBucket(), testEnv.storageClient) {
		t.T().Skipf("test not applicable for HNS buckets")
	}
	srcDirPath := filepath.Join(testDirName, "srcDirContainingFiles"+setup.GenerateRandomString(5))
	mountedSrcDirPath := filepath.Join(setup.MntDir(), srcDirPath)
	dstDirPath := filepath.Join(testDirName, "dstDirContainingFiles"+setup.GenerateRandomString(5))
	mountedDstDirPath := filepath.Join(setup.MntDir(), dstDirPath)
	client.CreateGcsDir(testEnv.ctx, testEnv.storageClient, srcDirPath, setup.TestBucket(), "")
	client.CreateNFilesInDir(testEnv.ctx, testEnv.storageClient, 1, "file", 1024, srcDirPath, t.T())
	defer client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, srcDirPath)

	err := os.Rename(mountedSrcDirPath, mountedDstDirPath)

	require.NoError(t.T(), err, "Failed to rename directory %q to %q: %v", mountedSrcDirPath, mountedDstDirPath, err)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestProfile(t *testing.T) {
	flagSet := func(profile string) [][]string {
		if profile != "" {
			return [][]string{{"--profile", profile}}
		} else {
			return [][]string{{}}
		}
	}
	tcNameFromProfile := func(profile string) string {
		if profile != "" {
			return profile
		} else {
			return "no-profile"
		}
	}
	tcNameFromFlags := func(flags []string) string {
		if len(flags) > 0 {
			return strings.ReplaceAll(strings.Join(flags, ","), "--", "")
		} else {
			return "noflags"
		}
	}

	profiles := []string{"aiml-training", "aiml-serving", "aiml-checkpointing", ""}

	for _, profile := range profiles {
		t.Run(tcNameFromProfile(profile), func(t *testing.T) {
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
			case "":
				s := &noProfileTests{}
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
				t.Run(tcNameFromFlags(flags), func(t *testing.T) {
					suite.Run(t, ts)
				})
			}
		})
	}
}
