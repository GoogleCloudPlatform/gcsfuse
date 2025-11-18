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
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
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
	fileName      string
	suite.Suite
}

func (t *unfinalizedObjectOperations) SetupTest() {
	t.testDirPath = client.SetupTestDirectory(t.ctx, t.storageClient, testDirName)
	t.fileName = path.Base(t.T().Name()) + setup.GenerateRandomString(5)
}

func (s *unfinalizedObjectOperations) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *unfinalizedObjectOperations) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, mountFunc)
	if setup.MountedDirectory() == "" {
		setup.SetMntDir(testEnv.cfg.GCSFuseMountedDirectory)
	}
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (t *unfinalizedObjectOperations) TestUnfinalizedObjectCreatedOutsideOfMountReportsNonZeroSize() {
	size := operations.MiB
	_ = client.CreateUnfinalizedObject(t.ctx, t.T(), t.storageClient, path.Join(testDirName, t.fileName), setup.GenerateRandomString(size))

	statRes, err := operations.StatFile(path.Join(t.testDirPath, t.fileName))

	require.NoError(t.T(), err)
	assert.Equal(t.T(), t.fileName, (*statRes).Name())
	assert.EqualValues(t.T(), size, (*statRes).Size())
}

func (t *unfinalizedObjectOperations) TestUnfinalizedObjectCreatedFromSameMountReportsCorrectSize() {
	size := operations.MiB
	// Create un-finalized object via same mount.
	fh := operations.CreateFile(path.Join(t.testDirPath, t.fileName), setup.FilePermission_0600, t.T())
	operations.WriteWithoutClose(fh, setup.GenerateRandomString(size), t.T())
	operations.SyncFile(fh, t.T())

	statRes, err := operations.StatFile(path.Join(t.testDirPath, t.fileName))

	require.NoError(t.T(), err)
	assert.Equal(t.T(), t.fileName, (*statRes).Name())
	assert.EqualValues(t.T(), size, (*statRes).Size())
	// Write more data to the object and finalize.
	operations.WriteWithoutClose(fh, setup.GenerateRandomString(size), t.T())
	err = fh.Close()
	require.NoError(t.T(), err)
	// After object is finalized, correct size should be reported.
	statRes, err = operations.StatFile(path.Join(t.testDirPath, t.fileName))
	require.NoError(t.T(), err)
	assert.EqualValues(t.T(), 2*size, (*statRes).Size())
}

func (t *unfinalizedObjectOperations) TestOverWritingUnfinalizedObjectsReturnsESTALE() {
	// TODO(b/411333280): Enable the test once flush on unfinalized Object is fixed.
	t.T().Skip("Skipping the test due to b/411333280")
	size := operations.MiB
	_ = client.CreateUnfinalizedObject(t.ctx, t.T(), t.storageClient, path.Join(testDirName, t.fileName), setup.GenerateRandomString(size))
	fh := operations.OpenFile(path.Join(t.testDirPath, t.fileName), t.T())

	// Overwrite unfinalized object.
	operations.WriteWithoutClose(fh, setup.GenerateRandomString(int(size)), t.T())
	err := fh.Close()

	operations.ValidateESTALEError(t.T(), err)
}

func (t *unfinalizedObjectOperations) TestUnfinalizedObjectCanBeRenamedIfCreatedFromSameMount() {
	size := operations.MiB
	content := setup.GenerateRandomString(size)
	newFileName := "new" + t.fileName
	// Create un-finalized object via same mount.
	fh := operations.CreateFile(path.Join(t.testDirPath, t.fileName), setup.FilePermission_0600, t.T())
	operations.WriteWithoutClose(fh, content, t.T())
	operations.SyncFile(fh, t.T())

	err := operations.RenameFile(path.Join(t.testDirPath, t.fileName), path.Join(t.testDirPath, newFileName))

	require.NoError(t.T(), err)
	client.ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, testDirName, t.fileName, t.T())
	client.ValidateObjectContentsFromGCS(t.ctx, t.storageClient, testDirName, newFileName, content, t.T())
	// validate writing to the renamed file via stale file handle returns ESTALE error.
	_, err = fh.Write([]byte(content))
	operations.ValidateESTALEError(t.T(), err)
}

func (t *unfinalizedObjectOperations) TestUnfinalizedObjectCanBeRenamedIfCreatedFromDifferentMount() {
	size := operations.MiB
	_ = client.CreateUnfinalizedObject(t.ctx, t.T(), t.storageClient, path.Join(testDirName, t.fileName), setup.GenerateRandomString(size))

	// Overwrite unfinalized object.
	err := operations.RenameFile(path.Join(t.testDirPath, t.fileName), path.Join(t.testDirPath, "New"+t.fileName))

	require.NoError(t.T(), err)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestUnfinalizedObjectOperationTest(t *testing.T) {
	ts := &unfinalizedObjectOperations{ctx: context.Background(), storageClient: testEnv.storageClient}

	// Run tests for mounted directory if the flag is set.
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
