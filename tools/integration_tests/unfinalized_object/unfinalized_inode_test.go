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

package unfinalized_object

import (
	"context"
	"log"
	"os"
	"path"
	"syscall"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type unfinalizedObjectInodeTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	fileName      string
	suite.Suite
}

func (t *unfinalizedObjectInodeTest) SetupTest() {
	t.testDirPath = client.SetupTestDirectory(t.ctx, t.storageClient, testDirName)
	t.fileName = path.Base(t.T().Name()) + setup.GenerateRandomString(5)
}

func (t *unfinalizedObjectInodeTest) TearDownSuite() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (t *unfinalizedObjectInodeTest) SetupSuite() {
	// Mount with StatCacheTTL=0 to ensure LookupInode is called on every stat.
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, t.flags, static_mounting.MountGcsfuseWithStaticMountingWithConfigFile)
	if testEnv.cfg.GKEMountedDirectory == "" {
		setup.SetMntDir(testEnv.cfg.GCSFuseMountedDirectory)
	}
	testEnv.testDirPath = client.SetupTestDirectory(t.ctx, t.storageClient, testDirName)
}

func (t *unfinalizedObjectInodeTest) TestInodeIDPreservedOnRemoteAppend() {
	if !setup.IsZonalBucketRun() {
		t.T().Skip("This test is only for Zonal buckets.")
	}

	filePath := path.Join(t.testDirPath, t.fileName)
	initialContent := setup.GenerateRandomString(initialSize)
	appendContent := setup.GenerateRandomString(appendSize)

	// 1. Create an unfinalized object.
	_ = client.CreateUnfinalizedObject(t.ctx, t.T(), t.storageClient, path.Join(testDirName, t.fileName), initialContent)

	// 2. Stat the file to get initial Inode ID and Size.
	fi, err := os.Stat(filePath)
	require.NoError(t.T(), err)
	initialStat, ok := fi.Sys().(*syscall.Stat_t)
	require.True(t.T(), ok)
	initialInodeID := initialStat.Ino
	assert.Equal(t.T(), int64(initialSize), fi.Size())

	// 3. Remotely append content to the object.
	obj, err := t.storageClient.Bucket(setup.TestBucket()).Object(path.Join(testDirName, t.fileName)).Attrs(t.ctx)
	require.NoError(t.T(), err)

	writer, err := client.TakeoverWriter(t.ctx, t.storageClient, path.Join(testDirName, t.fileName), obj.Generation)
	require.NoError(t.T(), err)
	_, err = writer.Write([]byte(appendContent))
	require.NoError(t.T(), err)
	err = writer.Close()
	require.NoError(t.T(), err)

	// Validate that the content was appended to the unfinalized object without changing the object generation.
	finalObject, err := t.storageClient.Bucket(setup.TestBucket()).Object(path.Join(testDirName, t.fileName)).Attrs(t.ctx)
	require.NoError(t.T(), err)
	require.Equal(t.T(), obj.Generation, finalObject.Generation)

	// 4. Stat the file again.
	// Since we are using StatCacheTTL=0, this should trigger LookupInode.
	fiNew, err := os.Stat(filePath)
	require.NoError(t.T(), err)
	newStat, ok := fiNew.Sys().(*syscall.Stat_t)
	require.True(t.T(), ok)

	// 5. Assert Inode ID is preserved and Size is updated.
	assert.Equal(t.T(), initialInodeID, newStat.Ino, "Inode ID should be preserved")
	assert.Equal(t.T(), int64(initialSize+appendSize), fiNew.Size(), "Size should be updated")
}

func (t *unfinalizedObjectInodeTest) TestInodeIDChangedOnRemoteOverwrite() {
	if !setup.IsZonalBucketRun() {
		t.T().Skip("This test is only for Zonal buckets.")
	}

	filePath := path.Join(t.testDirPath, t.fileName)
	initialContent := setup.GenerateRandomString(initialSize)
	newContent := setup.GenerateRandomString(initialSize)

	// 1. Create an unfinalized object.
	_ = client.CreateUnfinalizedObject(t.ctx, t.T(), t.storageClient, path.Join(testDirName, t.fileName), initialContent)

	// 2. Stat the file to get initial Inode ID.
	fi, err := os.Stat(filePath)
	require.NoError(t.T(), err)
	initialStat, ok := fi.Sys().(*syscall.Stat_t)
	require.True(t.T(), ok)
	initialInodeID := initialStat.Ino

	// 3. Remotely overwrite the object (this changes generation).
	// We use CreateUnfinalizedObject again which overwrites by default and creates a new generation.
	_ = client.CreateUnfinalizedObject(t.ctx, t.T(), t.storageClient, path.Join(testDirName, t.fileName), newContent)

	// 4. Stat the file again.
	fiNew, err := os.Stat(filePath)
	require.NoError(t.T(), err)
	newStat, ok := fiNew.Sys().(*syscall.Stat_t)
	require.True(t.T(), ok)

	// 5. Assert Inode ID is DIFFERENT.
	assert.NotEqual(t.T(), initialInodeID, newStat.Ino, "Inode ID should change when generation changes")
	assert.Equal(t.T(), int64(initialSize), fiNew.Size())
}

func TestUnfinalizedObjectInodeTest(t *testing.T) {
	ts := &unfinalizedObjectInodeTest{ctx: context.Background(), storageClient: testEnv.storageClient}

	// Run tests for mounted directory if the flag is set.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	// Run tests for GCE environment otherwise.
	// We specifically want to test with StatCacheTTL=0.
	// We will filter or modify flags to ensure this.
	// Actually, we can just add it to the flags.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		// Append --stat-cache-ttl=0s to ensure we don't hit the cache.
		ts.flags = append(flags, "--stat-cache-ttl=0s")
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
