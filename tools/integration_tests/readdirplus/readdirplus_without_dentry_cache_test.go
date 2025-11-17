// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package readdirplus

import (
	"context"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/jacobsa/fuse/fusetesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

type ReaddirplusWithoutDentryCacheTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *ReaddirplusWithoutDentryCacheTest) SetupTest() {
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, path.Join(testDirName, s.T().Name()))
}

func (s *ReaddirplusWithoutDentryCacheTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *ReaddirplusWithoutDentryCacheTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *ReaddirplusWithoutDentryCacheTest) SetupSuite() {
	setup.SetUpLogFilePath(s.baseTestName, GKETempDir, OldGKElogFilePath, testEnv.cfg)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *ReaddirplusWithoutDentryCacheTest) TestReaddirplusWithoutDentryCache() {
	// Create directory structure:
	// testBucket/dirForReaddirplusTest/target_dir/
	// testBucket/dirForReaddirplusTest/target_dir/file
	// testBucket/dirForReaddirplusTest/target_dir/emptySubDirectory/
	// testBucket/dirForReaddirplusTest/target_dir/subDirectory/
	// testBucket/dirForReaddirplusTest/target_dir/subDirectory/file1
	targetDir := path.Join(testEnv.testDirPath, targetDirName)
	operations.CreateDirectory(targetDir, s.T())
	// Create a file in the target directory.
	f1 := operations.CreateFile(path.Join(targetDir, "file"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f1)
	// Create an empty subdirectory
	operations.CreateDirectory(path.Join(targetDir, "emptySubDirectory"), s.T())
	// Create a subdirectory with file
	operations.CreateDirectoryWithNFiles(1, path.Join(targetDir, "subDirectory"), "file", s.T())

	// Call Readdirplus to list the directory.
	startTime := time.Now()
	entries, err := fusetesting.ReadDirPlusPicky(targetDir)
	endTime := time.Now()

	require.NoError(s.T(), err, "ReadDirPlusPicky failed")
	// Verify the entries.
	expectedEntries := []struct {
		name  string
		isDir bool
		mode  os.FileMode
	}{
		{name: "emptySubDirectory", isDir: true, mode: os.ModeDir | 0755},
		{name: "file", isDir: false, mode: 0644},
		{name: "subDirectory", isDir: true, mode: os.ModeDir | 0755},
	}
	assert.Equal(s.T(), len(expectedEntries), len(entries), "Number of entries mismatch")
	for i, expected := range expectedEntries {
		entry := entries[i]
		assert.Equal(s.T(), expected.name, entry.Name(), "Name mismatch for entry %d", i)
		assert.Equal(s.T(), expected.isDir, entry.IsDir(), "IsDir mismatch for entry %s", entry.Name())
		assert.Equal(s.T(), expected.mode, entry.Mode(), "Mode mismatch for entry %s", entry.Name())
	}

	// Validate logs to check that ReadDirPlus was called and ReadDir was not.
	// Dentry cache is not enabled, so LookUpInode should be called for
	// parent directory as well as for all the entries.
	validateLogsForReaddirplus(s.T(), testEnv.cfg.LogFile, false, startTime, endTime)
}

func TestReaddirplusWithoutDentryCacheTest(t *testing.T) {
	ts := &ReaddirplusWithoutDentryCacheTest{ctx: context.Background(), storageClient: testEnv.storageClient, baseTestName: t.Name()}

	// Run tests for mounted directory if the flag is set. This assumes that run flag is properly passed by GKE team as per the config.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}
	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
