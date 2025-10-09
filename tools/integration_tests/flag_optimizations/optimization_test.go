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
	"slices"
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

type optimizationTests struct {
	suite.Suite
	flags []string
}

func (s *optimizationTests) SetupTest() {
	setupForMountedDirectoryTests()
	mountGCSFuseAndSetupTestDir(s.flags, testEnv.ctx, testEnv.storageClient)
}

func (s *optimizationTests) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(testEnv.rootDir)
}

type noOptimizationTests struct {
	optimizationTests
}

type highEndMachineOptimizationTests struct {
	optimizationTests
}

type aimlProfileTests struct {
	optimizationTests
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

func (t *optimizationTests) testImplicitDirsNotEnabled() {
	implicitDirPath := filepath.Join(testDirName, "implicitDir", setup.GenerateRandomString(5))
	mountedImplicitDirPath := filepath.Join(setup.MntDir(), implicitDirPath)
	client.CreateImplicitDir(testEnv.ctx, testEnv.storageClient, implicitDirPath, t.T())
	defer client.MustDeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, implicitDirPath)

	_, err := os.Stat(mountedImplicitDirPath)

	require.Error(t.T(), err, "Found unexpected implicit directory %q", mountedImplicitDirPath)
}

func (t *optimizationTests) testRenameDirLimitNotSet() {
	srcDirPath := filepath.Join(testDirName, "srcDirContainingFiles", setup.GenerateRandomString(5))
	mountedSrcDirPath := filepath.Join(setup.MntDir(), srcDirPath)
	dstDirPath := filepath.Join(testDirName, "dstDirContainingFiles", setup.GenerateRandomString(5))
	mountedDstDirPath := filepath.Join(setup.MntDir(), dstDirPath)
	require.NoError(t.T(), client.CreateGcsDir(testEnv.ctx, testEnv.storageClient, srcDirPath, setup.TestBucket(), ""))
	client.CreateNFilesInDir(testEnv.ctx, testEnv.storageClient, 1, "file", 1024, srcDirPath, t.T())
	defer client.MustDeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, srcDirPath)

	err := os.Rename(mountedSrcDirPath, mountedDstDirPath)

	require.Error(t.T(), err, "Unexpectedly succeeded in renaming directory %q to %q", mountedSrcDirPath, mountedDstDirPath)
}

func (t *optimizationTests) testImplicitDirsEnabled() {
	implicitDirPath := filepath.Join(testDirName, "implicitDir", setup.GenerateRandomString(5))
	mountedImplicitDirPath := filepath.Join(setup.MntDir(), implicitDirPath)
	client.CreateImplicitDir(testEnv.ctx, testEnv.storageClient, implicitDirPath, t.T())
	defer client.MustDeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, implicitDirPath)

	fi, err := os.Stat(mountedImplicitDirPath)

	require.NoError(t.T(), err, "Got error statting %q: %v", mountedImplicitDirPath, err)
	require.NotNil(t.T(), fi, "Expected directory %q", mountedImplicitDirPath)
	assert.True(t.T(), fi.IsDir(), "Expected %q to be a directory, but got not-dir", mountedImplicitDirPath)
}

func (t *optimizationTests) testRenameDirLimitSet() {
	srcDirPath := filepath.Join(testDirName, "srcDirContainingFiles"+setup.GenerateRandomString(5))
	mountedSrcDirPath := filepath.Join(setup.MntDir(), srcDirPath)
	dstDirPath := filepath.Join(testDirName, "dstDirContainingFiles"+setup.GenerateRandomString(5))
	mountedDstDirPath := filepath.Join(setup.MntDir(), dstDirPath)
	require.NoError(t.T(), client.CreateGcsDir(testEnv.ctx, testEnv.storageClient, srcDirPath, setup.TestBucket(), ""))
	client.CreateNFilesInDir(testEnv.ctx, testEnv.storageClient, 1, "file", 1024, srcDirPath, t.T())
	defer func() {
		client.MustDeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, srcDirPath)
		client.MustDeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, dstDirPath)
	}()

	err := os.Rename(mountedSrcDirPath, mountedDstDirPath)

	require.NoError(t.T(), err, "Failed to rename directory %q to %q: %v", mountedSrcDirPath, mountedDstDirPath, err)
}

func (t *noOptimizationTests) TestImplicitDirsNotEnabled() {
	t.optimizationTests.testImplicitDirsNotEnabled()
}

func (t *noOptimizationTests) TestRenameDirLimitNotSet() {
	t.optimizationTests.testRenameDirLimitNotSet()
}

func (t *highEndMachineOptimizationTests) TestImplicitDirsEnabled() {
	t.optimizationTests.testImplicitDirsEnabled()
}

func (t *highEndMachineOptimizationTests) TestRenameDirLimitSet() {
	t.optimizationTests.testRenameDirLimitSet()
}

func (t *aimlProfileTests) TestImplicitDirsEnabled() {
	t.optimizationTests.testImplicitDirsEnabled()
}

func (t *aimlCheckpointingProfileTests) TestRenameDirLimitSet() {
	t.optimizationTests.testRenameDirLimitSet()
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestOptimization(t *testing.T) {
	// Currently all the tests in this suite are applicable only for non-HNS buckets,
	// so skipping for HNS buckets (and zonal by extension).
	// Remove this check when tests are added which work on HNS buckets.
	if setup.ResolveIsHierarchicalBucket(testEnv.ctx, setup.TestBucket(), testEnv.storageClient) {
		t.Skipf("test not applicable for HNS buckets")
	}

	// Helper functions to create flags, test case names etc.
	flags := func(profile string, machineType string) []string {
		flags := []string{}
		if profile != "" {
			flags = append(flags, "--profile="+profile)
		}
		if machineType != "" {
			flags = append(flags, "--machine-type="+machineType)
		}
		return flags
	}
	tcNameFromFlags := func(flags []string) string {
		if len(flags) > 0 {
			return strings.ReplaceAll(strings.Join(flags, ","), "--", "")
		} else {
			return "noflags"
		}
	}

	// Define test cases to be run.
	highEndMachineType := highEndMachines[0]
	testCases := []struct {
		profile     string
		machineType string
		name        string
	}{
		{profile: "aiml-training", name: "training_on_low_end_machine"},
		{profile: "aiml-serving", name: "serving_on_low_end_machine"},
		{profile: "aiml-checkpointing", name: "checkpointing_on_low_end_machine"},
		{name: "no_profile_on_low_end_machine"},
		{machineType: highEndMachineType, name: "no_profile_on_high_end_machine"},
		{machineType: highEndMachineType, profile: "aiml-checkpointing", name: "checkpointing_on_high_end_machine"},
		{machineType: highEndMachineType, profile: "aiml-serving", name: "serving_on_high_end_machine"},
		{machineType: highEndMachineType, profile: "aiml-training", name: "training_on_high_end_machine"},
	}

	// Run test cases.
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var ts suite.TestingSuite
			var pTests *optimizationTests

			switch tc.profile {
			case "aiml-training":
				s := &aimlTrainingProfileTests{}
				ts = s
				pTests = &s.optimizationTests
			case "aiml-serving":
				s := &aimlServingProfileTests{}
				ts = s
				pTests = &s.optimizationTests
			case "aiml-checkpointing":
				s := &aimlCheckpointingProfileTests{}
				ts = s
				pTests = &s.optimizationTests
			case "":
				// handled in fallback.
			default:
				t.Errorf("Unexpected profile: %v", tc.profile)
			}
			// fallback
			if ts == nil {
				// fallback to high-end machine-type if applicable.
				if slices.Contains(highEndMachines, tc.machineType) {
					s := &highEndMachineOptimizationTests{}
					ts = s
					pTests = &s.optimizationTests
				} else {
					s := &noOptimizationTests{}
					ts = s
					pTests = &s.optimizationTests
				}
			}

			pTests.flags = flags(tc.profile, tc.machineType)
			t.Run(tcNameFromFlags(pTests.flags), func(t *testing.T) {
				suite.Run(t, ts)
			})
		})
	}
}
