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

package rapid_appends

import (
	"os"
	"path"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	numAppends            = 3  // Number of appends to perform on test file.
	appendSize            = 10 // Size in bytes for each append.
	unfinalizedObjectSize = 10 // Size in bytes of initial unfinalized Object.
)

// //////////////////////////////////////////////////////////////////////
// Tests for the AppendsTestSuite
// //////////////////////////////////////////////////////////////////////

func (s *AppendsTestSuite) TestAppendSessionInvalidatedByAnotherClientUponTakeover() {
	if !s.cfg.isDualMount {
		s.T().Skip("This test requires a dual-mount configuration.")
	}

	const initialContent = "dummy content"
	const appendContent = "appended content"

	s.createUnfinalizedObject()
	defer s.deleteUnfinalizedObject()

	// Initiate an append session using the primary file handle.
	appendFileHandle := operations.OpenFileInMode(s.T(), path.Join(s.primaryMount.testDirPath, s.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	_, err := appendFileHandle.WriteString(initialContent)
	require.NoError(s.T(), err)

	// Open a new file handle from the secondary mount to the same file.
	newAppendFileHandle := operations.OpenFileInMode(s.T(), path.Join(s.secondaryMount.testDirPath, s.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	defer operations.CloseFileShouldNotThrowError(s.T(), newAppendFileHandle)

	// This append should succeed, confirming the takeover.
	_, err = newAppendFileHandle.WriteString(appendContent)
	assert.NoError(s.T(), err)

	// This should now fail, as its append session has been invalidated.
	_, _ = appendFileHandle.WriteString(appendContent)
	err = appendFileHandle.Sync()
	operations.ValidateESTALEError(s.T(), err)

	// Syncing from the new handle must succeed.
	err = newAppendFileHandle.Sync()
	assert.NoError(s.T(), err)

	// Close the stale handle and validate the final content.
	operations.CloseFileShouldThrowError(s.T(), appendFileHandle)
	expectedContent := s.fileContent + appendContent
	content, err := operations.ReadFile(path.Join(s.primaryMount.testDirPath, s.fileName))
	require.NoError(s.T(), err)
	assert.Equal(s.T(), expectedContent, string(content))
}

func (s *AppendsTestSuite) TestContentAppendedInNonAppendModeNotVisibleTillClose() {
	if s.cfg.isDualMount {
		s.T().Skip("This test is designed for a single-mount configuration.")
	}
	s.T().Skip("Skipping test until CreateObject() is supported for unfinalized objects (b/424253611).")

	s.createUnfinalizedObject()
	defer s.deleteUnfinalizedObject()

	initialContent := s.fileContent
	wh, err := os.OpenFile(path.Join(s.primaryMount.testDirPath, s.fileName), os.O_WRONLY|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(s.T(), err)

	data := setup.GenerateRandomString(contentSizeForBW * operations.OneMiB)
	n, err := wh.WriteAt([]byte(data), int64(len(initialContent)))
	require.NoError(s.T(), err)
	require.Equal(s.T(), len(data), n)

	// Read from GCS to validate that appended content is not yet visible.
	contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, s.fileName))
	require.NoError(s.T(), err)
	assert.Equal(s.T(), initialContent, string(contentBeforeClose))

	// Close the file handle to persist the data.
	err = wh.Close()
	require.NoError(s.T(), err)

	// Validate that the content is now visible in GCS.
	expectedContent := initialContent + data
	contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, s.fileName))
	require.NoError(s.T(), err)
	assert.Equal(s.T(), expectedContent, string(contentAfterClose))
}

func (s *AppendsTestSuite) TestAppendsToFinalizedObjectNotVisibleUntilClose() {
	if s.cfg.isDualMount {
		s.T().Skip("This test is designed for a single-mount configuration.")
	}

	const initialContent = "dummy content"

	s.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName, s.fileName, initialContent, s.T())

	data := setup.GenerateRandomString(contentSizeForBW * operations.OneMiB)
	filePath := path.Join(s.primaryMount.testDirPath, s.fileName)
	fh, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR|syscall.O_DIRECT, operations.FilePermission_0600)
	require.NoError(s.T(), err)
	_, err = fh.Write([]byte(data))
	require.NoError(s.T(), err)

	// Read from GCS to validate appended content is not yet visible.
	contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, s.fileName))
	require.NoError(s.T(), err)
	assert.Equal(s.T(), initialContent, string(contentBeforeClose))

	// Close the file handle and verify content is now visible.
	require.NoError(s.T(), fh.Close())
	expectedContent := initialContent + data
	contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, s.fileName))
	require.NoError(s.T(), err)
	assert.Equal(s.T(), expectedContent, string(contentAfterClose))
}

func (s *AppendsTestSuite) TestAppendsVisibleInRealTimeWithConcurrentRPlusHandle() {
	if s.cfg.isDualMount {
		s.T().Skip("This test is designed for a single-mount configuration.")
	}

	const initialContent = "dummy content"
	s.createUnfinalizedObject()
	defer s.deleteUnfinalizedObject()

	primaryPath := path.Join(s.primaryMount.testDirPath, s.fileName)
	appendFileHandle := operations.OpenFileInMode(s.T(), primaryPath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	defer appendFileHandle.Close()
	readHandle := operations.OpenFileInMode(s.T(), primaryPath, os.O_RDWR|syscall.O_DIRECT)
	defer readHandle.Close()

	// Write initial content with append handle to trigger buffered write workflow.
	_, err := appendFileHandle.Write([]byte(initialContent))
	require.NoError(s.T(), err)

	// Append additional content with the "r+" handle.
	data := setup.GenerateRandomString(contentSizeForBW * blockSize)
	appendOffset := int64(unfinalizedObjectSize + len(initialContent))
	_, err = readHandle.WriteAt([]byte(data), appendOffset)
	require.NoError(s.T(), err)

	// The first 1MiB block is guaranteed to be flushed. Verify its content.
	dataInBlockOffset := blockSize - len(initialContent)
	expectedContent := s.fileContent + initialContent + data[0:dataInBlockOffset]
	contentRead, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, s.fileName))
	require.NoError(s.T(), err)
	require.GreaterOrEqual(s.T(), len(contentRead), len(expectedContent))
	assert.Equal(s.T(), expectedContent, string(contentRead[0:len(expectedContent)]))
}

func (s *AppendsTestSuite) TestRandomWritesVisibleAfterCloseWithConcurrentRPlusHandle() {
	if s.cfg.isDualMount {
		s.T().Skip("This test is designed for a single-mount configuration.")
	}
	s.T().Skip("Skipping test until CreateObject() is supported for unfinalized objects (b/424253611).")

	const initialContent = "dummy content"
	s.createUnfinalizedObject()
	defer s.deleteUnfinalizedObject()

	primaryPath := path.Join(s.primaryMount.testDirPath, s.fileName)
	appendFileHandle := operations.OpenFileInMode(s.T(), primaryPath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	defer appendFileHandle.Close()
	readHandle := operations.OpenFileInMode(s.T(), primaryPath, os.O_RDWR|syscall.O_DIRECT)

	_, err := appendFileHandle.Write([]byte(initialContent))
	require.NoError(s.T(), err)

	// Random write at an incorrect offset.
	data := setup.GenerateRandomString(contentSizeForBW * blockSize)
	_, err = readHandle.WriteAt([]byte(data), int64(len(initialContent))+1)
	require.NoError(s.T(), err)

	// Validate content is not yet visible.
	contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, s.fileName))
	require.NoError(s.T(), err)
	assert.Equal(s.T(), initialContent, string(contentBeforeClose))

	// Close handle and validate final content (with null byte for the gap).
	readHandle.Close()
	expectedContent := s.fileContent + initialContent + "\x00" + data
	contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, s.fileName))
	require.NoError(s.T(), err)
	assert.Equal(s.T(), expectedContent, string(contentAfterClose))
}

func (s *AppendsTestSuite) TestFallbackHappensWhenNonAppendHandleDoesFirstWrite() {
	if s.cfg.isDualMount {
		s.T().Skip("This test is designed for a single-mount configuration.")
	}
	s.T().Skip("Skipping test until CreateObject() is supported for unfinalized objects (b/424253611).")

	s.createUnfinalizedObject()
	defer s.deleteUnfinalizedObject()

	primaryPath := path.Join(s.primaryMount.testDirPath, s.fileName)
	appendFileHandle := operations.OpenFileInMode(s.T(), primaryPath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	defer appendFileHandle.Close()
	readHandle := operations.OpenFileInMode(s.T(), primaryPath, os.O_RDWR|syscall.O_DIRECT)

	// Append content using the "r+" handle first.
	data := setup.GenerateRandomString(contentSizeForBW * blockSize)
	_, err := readHandle.WriteAt([]byte(data), int64(len(s.fileContent)))
	require.NoError(s.T(), err)

	// Validate content is not yet visible.
	contentBeforeClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, s.fileName))
	require.NoError(s.T(), err)
	assert.Equal(s.T(), s.fileContent, string(contentBeforeClose))

	// Close handle and validate final content.
	readHandle.Close()
	expectedContent := s.fileContent + data
	contentAfterClose, err := client.ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, s.fileName))
	require.NoError(s.T(), err)
	assert.Equal(s.T(), expectedContent, string(contentAfterClose))
}

func (s *AppendsTestSuite) TestKernelShouldSeeUpdatedSizeOnAppends() {
	if s.cfg.isDualMount {
		s.T().Skip("This test is designed for a single-mount configuration.")
	}
	const initialContent = "dummy content"

	testCases := []struct {
		name        string
		expireCache bool
	}{
		{"validStatCache", false},
		{"expiredStatCache", true},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.createUnfinalizedObject()
			defer s.deleteUnfinalizedObject()
			filePath := path.Join(s.primaryMount.testDirPath, s.fileName)

			// Append to the object and close the file handle.
			appendFileHandle := operations.OpenFileInMode(s.T(), filePath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
			_, err := appendFileHandle.Write([]byte(initialContent))
			require.NoError(s.T(), err)
			appendFileHandle.Close()

			if tc.expireCache {
				time.Sleep(time.Second)
			}

			// Stat the file to assert on the file size as viewed by the kernel.
			expectedFileSize := int64(unfinalizedObjectSize + len(initialContent))
			fileInfo, err := operations.StatFile(filePath)
			assert.NoError(s.T(), err)
			assert.Equal(s.T(), expectedFileSize, (*fileInfo).Size())
		})
	}
}
