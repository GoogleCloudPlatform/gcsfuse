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
	"github.com/stretchr/testify/suite"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deleteOperationTest struct {
	flags []string
	suite.Suite
}

func (s *deleteOperationTest) SetupTest() {
	mountGCSFuseAndSetupTestDir(s.flags, testDirName)
}

func (s *deleteOperationTest) TearDownTest() {
	if setup.MountedDirectory() == "" { // Only unmount if not using a pre-mounted directory
		setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
		setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	}
}

func (s *deleteOperationTest) TestDeleteFileWhenFileIsClobbered() {
	// Create a file with initial content directly in GCS.
	filePath := path.Join(setup.MntDir(), testDirName, testFileName)
	client.SetupFileInTestDirectory(ctx, storageClient, testDirName, testFileName, initialContentSize, s.T())
	// Stat file to cache the entry
	_, err := os.Stat(filePath)
	require.Nil(s.T(), err)
	// Modify the object on GCS.
	objectName := path.Join(testDirName, testFileName)
	smallContent, err := operations.GenerateRandomData(updatedContentSize)
	require.Nil(s.T(), err)
	require.Nil(s.T(), client.WriteToObject(ctx, storageClient, objectName, string(smallContent), storage.Conditions{}))

	// Deleting the file should not give error
	err = os.Remove(filePath)

	assert.Nil(s.T(), err)
}

func TestDeleteOperationTest(t *testing.T) {
	ts := &deleteOperationTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	// Setup flags and run tests.
	ts.flags = []string{"--implicit-dirs", "--experimental-enable-dentry-cache", "--metadata-cache-ttl-secs=1000"}
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)
}
