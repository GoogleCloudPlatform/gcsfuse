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

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (t *optimizationTests) testImplicitDirsNotEnabled() {
	implicitDirPath := filepath.Join(testDirName, "implicitDir"+setup.GenerateRandomString(5))
	mountedImplicitDirPath := filepath.Join(setup.MntDir(), implicitDirPath)
	client.CreateImplicitDir(testEnv.ctx, testEnv.storageClient, implicitDirPath, t.T())
	defer client.MustDeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, implicitDirPath)

	_, err := os.Stat(mountedImplicitDirPath)

	require.Error(t.T(), err, "Found unexpected implicit directory %q", mountedImplicitDirPath)
}

func (t *optimizationTests) testRenameDirLimitNotSet() {
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

	require.Error(t.T(), err, "Unexpectedly succeeded in renaming directory %q to %q", mountedSrcDirPath, mountedDstDirPath)
}

func (t *optimizationTests) testImplicitDirsEnabled() {
	implicitDirPath := filepath.Join(testDirName, "implicitDir"+setup.GenerateRandomString(5))
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

////////////////////////////////////////////////////////////////////////
// Test Functions (Runs before all tests)
////////////////////////////////////////////////////////////////////////

func TestImplicitDirsNotEnabled(t *testing.T) {
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			ts := &optimizationTests{}
			ts.SetT(t)
			ts.flags = flags
			ts.SetupTest()
			ts.testImplicitDirsNotEnabled()
			ts.TearDownTest()
		})
	}
}

func TestRenameDirLimitNotSet(t *testing.T) {
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			ts := &optimizationTests{}
			ts.SetT(t)
			ts.flags = flags
			ts.SetupTest()
			ts.testRenameDirLimitNotSet()
			ts.TearDownTest()
		})
	}
}

func TestImplicitDirsEnabled(t *testing.T) {
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			ts := &optimizationTests{}
			ts.SetT(t)
			ts.flags = flags
			ts.SetupTest()
			ts.testImplicitDirsEnabled()
			ts.TearDownTest()
		})
	}
}

func TestRenameDirLimitSet(t *testing.T) {
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			ts := &optimizationTests{}
			ts.SetT(t)
			ts.flags = flags
			ts.SetupTest()
			ts.testRenameDirLimitSet()
			ts.TearDownTest()
		})
	}
}
