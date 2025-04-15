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

type unfinalizedObjectOperations struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	suite.Suite
}

func (t *unfinalizedObjectOperations) SetupTest() {
	t.testDirPath = client.SetupTestDirectory(t.ctx, t.storageClient, testDirName)
}

func (t *unfinalizedObjectOperations) TeardownTest() {}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (t *unfinalizedObjectOperations) TestUnfinalizedObjectCreatedOutsideOfMountReports0Size() {
	fileName := path.Base(t.T().Name()) + setup.GenerateRandomString(5)
	var size int64 = operations.MiB
	writer := client.CreateUnfinalizedObject(t.ctx, t.T(), t.storageClient, path.Join(testDirName, fileName), size)

	statRes, err := operations.StatFile(path.Join(t.testDirPath, fileName))

	require.NoError(t.T(), err)
	assert.Equal(t.T(), fileName, (*statRes).Name())
	assert.EqualValues(t.T(), 0, (*statRes).Size())
	// After object is finalized, correct size should be reported.
	err = writer.Close()
	require.NoError(t.T(), err)
	statRes, err = operations.StatFile(path.Join(t.testDirPath, fileName))
	require.NoError(t.T(), err)
	assert.EqualValues(t.T(), size, (*statRes).Size())
}

func (t *unfinalizedObjectOperations) TestUnfinalizedObjectCreatedFromSameMountReportsCorrectSize() {
	fileName := path.Base(t.T().Name()) + setup.GenerateRandomString(5)
	size := operations.MiB
	// Create un-finalized object via same mount.
	fh := operations.CreateFile(path.Join(t.testDirPath, fileName), setup.FilePermission_0600, t.T())
	operations.WriteWithoutClose(fh, setup.GenerateRandomString(size), t.T())
	operations.SyncFile(fh, t.T())

	statRes, err := operations.StatFile(path.Join(t.testDirPath, fileName))

	require.NoError(t.T(), err)
	assert.Equal(t.T(), fileName, (*statRes).Name())
	assert.EqualValues(t.T(), size, (*statRes).Size())
	// Write more data to the object and finalize.
	operations.WriteWithoutClose(fh, setup.GenerateRandomString(size), t.T())
	err = fh.Close()
	require.NoError(t.T(), err)
	// After object is finalized, correct size should be reported.
	statRes, err = operations.StatFile(path.Join(t.testDirPath, fileName))
	require.NoError(t.T(), err)
	assert.EqualValues(t.T(), 2*size, (*statRes).Size())
}

func (t *unfinalizedObjectOperations) TestOverWritingUnfinalizedObjectsReturnsESTALE() {
	fileName := path.Base(t.T().Name()) + setup.GenerateRandomString(5)
	var size int64 = operations.MiB
	_ = client.CreateUnfinalizedObject(t.ctx, t.T(), t.storageClient, path.Join(testDirName, fileName), size)
	fh := operations.OpenFile(path.Join(t.testDirPath, fileName), t.T())

	// Overwrite unfinalized object.
	operations.WriteWithoutClose(fh, setup.GenerateRandomString(int(size)), t.T())
	err := fh.Close()

	operations.ValidateESTALEError(t.T(), err)
}

func (t *unfinalizedObjectOperations) TestUnfinalizedObjectCanBeRenamedIfCreatedFromSameMount() {
	fileName := path.Base(t.T().Name()) + setup.GenerateRandomString(5)
	size := operations.MiB
	content := setup.GenerateRandomString(size)
	newFileName := "new" + fileName
	// Create un-finalized object via same mount.
	fh := operations.CreateFile(path.Join(t.testDirPath, fileName), setup.FilePermission_0600, t.T())
	operations.WriteWithoutClose(fh, content, t.T())
	operations.SyncFile(fh, t.T())

	err := operations.RenameFile(path.Join(t.testDirPath, fileName), path.Join(t.testDirPath, newFileName))

	require.NoError(t.T(), err)
	client.ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, testDirName, fileName, t.T())
	client.ValidateObjectContentsFromGCS(t.ctx, t.storageClient, testDirName, newFileName, content, t.T())
	// validate writing to the renamed file via stale file handle returns ESTALE error.
	_, err = fh.Write([]byte(content))
	operations.ValidateESTALEError(t.T(), err)
}

func (t *unfinalizedObjectOperations) TestUnfinalizedObjectCantBeRenamedIfCreatedFromDifferentMount() {
	fileName := path.Base(t.T().Name()) + setup.GenerateRandomString(5)
	var size int64 = operations.MiB
	_ = client.CreateUnfinalizedObject(t.ctx, t.T(), t.storageClient, path.Join(testDirName, fileName), size)

	// Overwrite unfinalized object.
	err := operations.RenameFile(path.Join(t.testDirPath, fileName), path.Join(t.testDirPath, "New"+fileName))

	require.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, syscall.EIO.Error())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestUnfinalizedObjectOperationTest(t *testing.T) {
	ts := &unfinalizedObjectOperations{ctx: context.Background()}
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
		{"--enable-streaming-writes=true", "--metadata-cache-ttl-secs=0"},
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
