// Copyright 2024 Google LLC
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

// This file contains append tests specific to zonal buckets.

package appends_zb

import (
	"context"
	"log"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////
type AppendsLookUpTest struct {
	suite.Suite
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
}

func (t *AppendsLookUpTest) SetupSuite() {
	if setup.MountedDirectory() != "" {
		mountDir = setup.MountedDirectory()
	}
	setup.MountGCSFuseWithGivenMountFunc(t.flags, mountFunc)
	setup.SetMntDir(mountDir)
}

func (t *AppendsLookUpTest) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
}

func (t *AppendsLookUpTest) SetupTest() {
	t.testDirPath = client.SetupTestDirectory(t.ctx, t.storageClient, testDirName)
}

func (t *AppendsLookUpTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(t.T())
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

//func (t *AppendsLookUpTest) TestUnfinalizedObjectsCreatedFromSameMountCanBeLookedUp() {
//	fileName := path.Base(t.T().Name()) + setup.GenerateRandomString(5)
//	filePath := path.Join(t.testDirPath, fileName)
//	fh1 := operations.CreateFile(filePath, setup.FilePermission_0600, t.T())
//	defer func(fh1 *os.File) {
//		err := fh1.Close()
//		require.NoError(t.T(), err)
//	}(fh1)
//	err := fh1.Sync()
//	require.NoError(t.T(), err)
//
//	statRes, err := operations.StatFile(filePath)
//
//	require.NoError(t.T(), err)
//	assert.Equal(t.T(), fileName, (*statRes).Name())
//	assert.Equal(t.T(), int64(0), (*statRes).Size())
//}

func (t *AppendsLookUpTest) TestUnfinalizedObjectsCreatedFromDifferentMountCanBeLookedUp() (string, *storage.Writer) {
	fileName := path.Base(t.T().Name()) + setup.GenerateRandomString(5)
	writer, err := client.AppendableWriter(t.ctx, t.storageClient, path.Join(testDirName, fileName), storage.Conditions{})
	require.NoError(t.T(), err)
	offset, err := writer.Flush()
	require.NoError(t.T(), err)
	assert.Equal(t.T(), int64(0), offset)
	offset1, err := writer.Write([]byte(setup.GenerateRandomString(util.MiB)))
	require.NoError(t.T(), err)
	assert.EqualValues(t.T(), int64(util.MiB), offset1)
	offset, err = writer.Flush()
	require.NoError(t.T(), err)
	assert.Equal(t.T(), int64(util.MiB), offset)

	statRes, err := operations.StatFile(path.Join(t.testDirPath, fileName))

	require.NoError(t.T(), err)
	assert.Equal(t.T(), fileName, (*statRes).Name())
	assert.Equal(t.T(), int64(util.MiB), (*statRes).Size())
	return fileName, writer
}

//func (t *AppendsLookUpTest) TestUnfinalizedObjectsSizeChangeIsReflected() {
//	fileName, writer := t.TestUnfinalizedObjectsCreatedFromDifferentMountCanBeLookedUp()
//	var dataLength int64 = util.MiB * 32.5
//	content, err := operations.GenerateRandomData(dataLength)
//	require.NoError(t.T(), err)
//	size, err := writer.Write(content)
//	require.NoError(t.T(), err)
//	assert.EqualValues(t.T(), dataLength, size)
//	flushOffset, err := writer.Flush()
//	require.NoError(t.T(), err)
//	require.Equal(t.T(), dataLength+util.MiB, flushOffset)
//
//	statRes, err := operations.StatFile(path.Join(t.testDirPath, fileName))
//
//	require.NoError(t.T(), err)
//	assert.Equal(t.T(), fileName, (*statRes).Name())
//	assert.Equal(t.T(), dataLength+util.MiB, (*statRes).Size())
//}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestAppendsZBTest(t *testing.T) {
	ts := &AppendsLookUpTest{ctx: context.Background()}
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
		{"--metadata-cache-ttl-secs=0", "--enable-streaming-writes", "--client-protocol=grpc"},
	}

	// Run tests.
	for _, ts.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
