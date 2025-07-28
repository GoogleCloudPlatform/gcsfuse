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

package dentry_cache

import (
	"log"
	"os"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type statWithDentryCacheEnabledTest struct {
	flags []string
}

func (s *statWithDentryCacheEnabledTest) Setup(t *testing.T) {
	mountGCSFuseAndSetupTestDir(s.flags, testDirName)
}

func (s *statWithDentryCacheEnabledTest) Teardown(t *testing.T) {
	if setup.MountedDirectory() == "" { // Only unmount if not using a pre-mounted directory
		setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
		setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	}
}

func (s *statWithDentryCacheEnabledTest) TestStatWithDentryCacheEnabled(t *testing.T) {
	// Create a file with initial content directly in GCS.
	filePath := path.Join(setup.MntDir(), testDirName, testFileName)
	client.SetupFileInTestDirectory(ctx, storageClient, testDirName, testFileName, initialContentSize, t)
	// Stat file to cache the entry
	_, err := os.Stat(filePath)
	require.Nil(t, err)
	// Modify the object on GCS.
	objectName := path.Join(testDirName, testFileName)
	smallContent, err := operations.GenerateRandomData(updatedContentSize)
	require.Nil(t, err)
	require.Nil(t, client.WriteToObject(ctx, storageClient, objectName, string(smallContent), storage.Conditions{}))

	// Stat again, it should give old cached attributes.
	fileInfo, err := os.Stat(filePath)

	assert.Nil(t, err)
	assert.Equal(t, int64(initialContentSize), fileInfo.Size())
	// Wait until entry expires in cache.
	time.Sleep(1100 * time.Millisecond)
	// Stat again, it should give updated attributes.
	fileInfo, err = os.Stat(filePath)
	assert.Nil(t, err)
	assert.Equal(t, int64(updatedContentSize), fileInfo.Size())
}

func (s *statWithDentryCacheEnabledTest) TestStatWhenFileIsDeletedDirectlyFromGCS(t *testing.T) {
	// Create a file with initial content directly in GCS.
	filePath := path.Join(setup.MntDir(), testDirName, testFileName)
	client.SetupFileInTestDirectory(ctx, storageClient, testDirName, testFileName, initialContentSize, t)
	// Stat file to cache the entry
	_, err := os.Stat(filePath)
	require.Nil(t, err)
	// Delete the object directly from GCS.
	objectName := path.Join(testDirName, testFileName)
	require.Nil(t, client.DeleteObjectOnGCS(ctx, storageClient, objectName))

	// Stat again, it should give old cached attributes rather than giving not found error.
	fileInfo, err := os.Stat(filePath)

	assert.Nil(t, err)
	assert.Equal(t, int64(initialContentSize), fileInfo.Size())
	// Wait until entry expires in cache.
	time.Sleep(1100 * time.Millisecond)
	// Stat again, it should give error as file does not exist.
	_, err = os.Stat(filePath)
	assert.NotNil(t, err)
}

func TestStatWithDentryCacheEnabledTest(t *testing.T) {
	ts := &statWithDentryCacheEnabledTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	// Setup flags and run tests.
	ts.flags = []string{"--implicit-dirs", "--experimental-enable-dentry-cache", "--metadata-cache-ttl-secs=1"}
	log.Printf("Running tests with flags: %s", ts.flags)
	test_setup.RunTests(t, ts)
}
