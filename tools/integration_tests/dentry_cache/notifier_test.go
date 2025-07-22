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

	"cloud.google.com/go/storage"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type notifierTest struct {
	flags []string
}

func (s *notifierTest) Setup(t *testing.T) {
	mountGCSFuseAndSetupTestDir(s.flags, testDirName)
}

func (s *notifierTest) Teardown(t *testing.T) {
	if setup.MountedDirectory() == "" { // Only unmount if not using a pre-mounted directory
		setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
		setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	}
}

func (s *notifierTest) TestWriteFileWithDentryCacheEnabled(t *testing.T) {
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

	// First Write File attempt.
	err = operations.WriteFile(filePath, "ShouldNotWrite")

	// First Write File attempt should fail because file has been clobbered.
	assert.NotNil(t, err)
	// Second Write File attempt.
	err = operations.WriteFile(filePath, "ShouldWrite")
	// The notifier is triggered after the first write failure, invalidating the kernel cache entry.
	// Therefore, the second write succeeds even before the metadata cache TTL expires.
	assert.Nil(t, err)
}

func (s *notifierTest) TestReadFileWithDentryCacheEnabled(t *testing.T) {
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

	// First Read File attempt.
	_, err = operations.ReadFile(filePath)

	// First Read File attempt should fail because file has been clobbered.
	assert.NotNil(t, err)
	// Second Read File attempt.
	_, err = operations.ReadFile(filePath)
	// The notifier is triggered after the first read failure, invalidating the kernel cache entry.
	// Therefore, the second read succeeds even before the metadata cache TTL expires.
	assert.Nil(t, err)
}

func TestNotifierTest(t *testing.T) {
	ts := &notifierTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	// Setup flags and run tests.
	ts.flags = []string{"--implicit-dirs", "--experimental-enable-dentry-cache", "--metadata-cache-ttl-secs=1000"}
	log.Printf("Running tests with flags: %s", ts.flags)
	test_setup.RunTests(t, ts)
}
