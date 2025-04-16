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

package metadata_cache

import (
	"context"
	"log"
	"os"
	"path"
	"syscall"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type positiveMetadataCache struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	suite.Suite
}

func (t *positiveMetadataCache) SetupTest() {
	t.testDirPath = client.SetupTestDirectory(t.ctx, t.storageClient, testDirName)
}

func (t *positiveMetadataCache) TeardownTest() {}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

//func (t *positiveMetadataCache) TestRenameOfStaleFile() {
//	// Create a file via GCSFuse so that it is cached.
//	fileName := path.Base(t.T().Name()) + setup.GenerateRandomString(5)
//	err := os.WriteFile(path.Join(t.testDirPath, fileName), []byte("taco"), 0500)
//	require.NoError(t.T(), err)
//	// Overwrite the object on GCS.
//	err = client.OverwriteObjectOnGCS(t.ctx, t.storageClient, path.Join(testDirName, fileName), "burrito")
//	require.NoError(t.T(), err)
//	// Because we are caching, the file should still appear to be the local
//	// version.
//	fi, err := os.Stat(path.Join(t.testDirPath, fileName))
//	require.NoError(t.T(), err)
//	assert.EqualValues(t.T(), len("taco"), fi.Size())
//
//	// Attempt to rename the file should throw ENOENT error.
//	err = os.Rename(path.Join(t.testDirPath, fileName), path.Join(t.testDirPath, "new"+fileName))
//	require.Error(t.T(), err)
//	assert.ErrorContains(t.T(), err, syscall.ENOENT.Error())
//
//	// After the ESTALE, we should see the new version.
//	fi, err = os.Stat(path.Join(t.testDirPath, fileName))
//	require.NoError(t.T(), err)
//	assert.EqualValues(t.T(), len("burrito"), fi.Size())
//	// Rename should work as expected.
//	err = os.Rename(path.Join(t.testDirPath, fileName), path.Join(t.testDirPath, "new"+fileName))
//	require.NoError(t.T(), err)
//}

func (t *positiveMetadataCache) TestRenameOfStaleDirectory() {
	// Create files & directories via GCSFuse so that they are cached.
	testSubDirectory := "testSubDirectory"
	operations.CreateDirectoryWithNFiles(3, path.Join(t.testDirPath, testSubDirectory), path.Base(t.T().Name()), t.T())
	// Remove the testSubDirectory on GCS.
	err := client.DeleteAllObjectsWithPrefix(t.ctx, t.storageClient, path.Join(testDirName, testSubDirectory))
	require.NoError(t.T(), err)
	// Because we are caching, the folder should still exist.
	_, err = os.Stat(path.Join(t.testDirPath, testSubDirectory))
	require.NoError(t.T(), err)

	// Attempt to rename the directory should throw ENOENT error.
	err = os.Rename(path.Join(t.testDirPath, testSubDirectory), path.Join(t.testDirPath, "new"+testSubDirectory))
	require.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, syscall.ENOENT.Error())

	// After the ESTALE, we should not see the directory.
	_, err = os.Stat(path.Join(t.testDirPath, testSubDirectory))
	require.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, syscall.ENOENT.Error())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestPositiveCacheTTLTest(t *testing.T) {
	ts := &positiveMetadataCache{ctx: context.Background()}
	// Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&ts.ctx, &ts.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			t.Errorf("closeStorageClient failed: %v", err)
		}
	}()

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--metadata-cache-ttl-secs=600", "--rename-dir-limit=3"},
		//{"--metadata-cache-ttl-secs=600", "--rename-dir-limit=3", "--client-protocol=grpc"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		setup.MountGCSFuseWithGivenMountFunc(ts.flags, mountFunc)
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
		setup.SaveGCSFuseLogFileInCaseOfFailure(t)
		setup.UnmountGCSFuseAndDeleteLogFile(setup.MntDir())
	}
}
