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
	"io"
	"log"
	"os"
	"path"
	"syscall"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
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

const (
	initialSize          = operations.MiB
	appendSize           = operations.MiB
	readFlags            = syscall.O_RDONLY
	readFlagsWithODirect = syscall.O_RDONLY | syscall.O_DIRECT
)

type unfinalizedObjectReads struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	fileName      string
	suite.Suite
}

func (t *unfinalizedObjectReads) SetupTest() {
	t.testDirPath = client.SetupTestDirectory(t.ctx, t.storageClient, testDirName)
	t.fileName = path.Base(t.T().Name()) + setup.GenerateRandomString(5)
}

func (s *unfinalizedObjectReads) TearDownSuite() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *unfinalizedObjectReads) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, mountFunc)
	if testEnv.cfg.GKEMountedDirectory == "" {
		setup.SetMntDir(testEnv.cfg.GCSFuseMountedDirectory)
	}
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

////////////////////////////////////////////////////////////////////////
// Helper Methods
////////////////////////////////////////////////////////////////////////

// setupAndAppend creates an unfinalized object of size `initialSize` and then attempts an
// initial reads via GCSFuse mount. Finally, it remotely appends to the unfinalized object
// by taking over at the same generation and returns the already open filehandle along with
// initialContent and appendContent to be further used by tests.
func (t *unfinalizedObjectReads) setupAndAppend(filePath string, initialSize, appendSize, openFlags int) (fh *os.File, initialContent, appendContent string) {
	initialContent = setup.GenerateRandomString(initialSize)
	appendContent = setup.GenerateRandomString(appendSize)

	// 1. Create an unfinalized object and open it.
	_ = client.CreateUnfinalizedObject(t.ctx, t.T(), t.storageClient, path.Join(testDirName, t.fileName), initialContent)
	fh = operations.OpenFileInMode(t.T(), filePath, openFlags)

	// 2. Read initial content to cache the state.
	buffer := make([]byte, initialSize)
	n, err := fh.Read(buffer)
	require.NoError(t.T(), err)
	require.Equal(t.T(), initialSize, n)
	assert.Equal(t.T(), initialContent, string(buffer))

	// 3. Remotely append content to the object.
	obj, err := t.storageClient.Bucket(setup.TestBucket()).Object(path.Join(testDirName, t.fileName)).Attrs(t.ctx)
	require.NoError(t.T(), err)

	writer, err := client.TakeoverWriter(t.ctx, t.storageClient, path.Join(testDirName, t.fileName), obj.Generation)
	require.NoError(t.T(), err)
	n, err = writer.Write([]byte(appendContent))
	require.NoError(t.T(), err)
	assert.Equal(t.T(), appendSize, n)
	err = writer.Close()
	require.NoError(t.T(), err)

	// Validate that the content was appended to the unfinalized object without changing the object generation.
	finalObject, err := t.storageClient.Bucket(setup.TestBucket()).Object(path.Join(testDirName, t.fileName)).Attrs(t.ctx)
	require.NoError(t.T(), err)
	require.Equal(t.T(), obj.Generation, finalObject.Generation)

	return fh, initialContent, appendContent
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (t *unfinalizedObjectReads) TestUnfinalizedObjectsCanBeRead() {
	var size int = operations.MiB
	writtenContent := setup.GenerateRandomString(size)
	// Create un-finalized object via same mount.
	fh := operations.CreateFile(path.Join(t.testDirPath, t.fileName), setup.FilePermission_0600, t.T())
	operations.WriteWithoutClose(fh, writtenContent, t.T())
	defer operations.CloseFileShouldNotThrowError(t.T(), fh)

	// Read un-finalized object.
	file, err := os.OpenFile(path.Join(t.testDirPath, t.fileName), os.O_RDONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	require.NoError(t.T(), err)
	readContent, err := operations.ReadFileSequentially(file, util.MiB)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), writtenContent, string(readContent))
}

func (t *unfinalizedObjectReads) TestReadRemotelyModifiedUnfinalizedObject() {
	if !setup.IsZonalBucketRun() {
		t.T().Skip("This test is only for Zonal buckets.")
	}

	testCases := []struct {
		name               string
		openFlags          int
		readOffset         int64
		bytesToRead        int
		expectedBytesRead  int
		expectedErr        error
		getExpectedContent func(initial, appended string) string
	}{
		{
			name:              "WithODirect_Read_Beyond_Cached_Object_Size",
			openFlags:         readFlagsWithODirect,
			readOffset:        initialSize,
			bytesToRead:       appendSize,
			expectedBytesRead: appendSize,
			expectedErr:       nil,
			getExpectedContent: func(_, appended string) string {
				return appended
			},
		},
		{
			name:              "WithODirect_Read_Partially_Within_Cached_Object_Size",
			openFlags:         readFlagsWithODirect,
			readOffset:        initialSize / 2,
			bytesToRead:       appendSize,
			expectedBytesRead: appendSize,
			expectedErr:       nil,
			getExpectedContent: func(initial, appended string) string {
				offset := initialSize / 2
				return initial[offset:] + appended[:offset]
			},
		},
		{
			name:              "WithODirect_Read_Beyond_Actual_Object_Size",
			openFlags:         readFlagsWithODirect,
			readOffset:        initialSize + appendSize,
			bytesToRead:       appendSize,
			expectedBytesRead: 0,
			expectedErr:       io.EOF,
		},
		{
			name:              "WithoutODirect_Read_Beyond_Cached_Size",
			openFlags:         readFlags,
			readOffset:        initialSize,
			bytesToRead:       1,
			expectedBytesRead: 0,
			expectedErr:       io.EOF,
		},
	}

	for _, tc := range testCases {
		t.T().Run(tc.name, func(T *testing.T) {
			// We need to use a new file for each subtest to ensure isolation.
			t.fileName = path.Base(T.Name()) + setup.GenerateRandomString(5)
			filePath := path.Join(t.testDirPath, t.fileName)

			fh, initialContent, appendContent := t.setupAndAppend(filePath, initialSize, appendSize, tc.openFlags)
			defer operations.CloseFileShouldNotThrowError(T, fh)

			readBuffer := make([]byte, tc.bytesToRead)
			bytesRead, err := fh.ReadAt(readBuffer, tc.readOffset)

			assert.Equal(T, tc.expectedBytesRead, bytesRead)
			if tc.expectedErr != nil {
				assert.Equal(T, tc.expectedErr, err)
			} else {
				require.NoError(T, err)
			}

			if tc.getExpectedContent != nil {
				expectedContent := tc.getExpectedContent(initialContent, appendContent)
				assert.Equal(T, expectedContent, string(readBuffer[:bytesRead]))
			}
		})
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestUnfinalizedObjectReadTest(t *testing.T) {
	ts := &unfinalizedObjectReads{ctx: context.Background(), storageClient: testEnv.storageClient}

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
