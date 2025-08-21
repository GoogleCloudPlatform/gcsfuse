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
	"math/rand/v2"
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// declare a function type for read and verify
type readAndVerifyFunc func(t *testing.T, filePath string, expectedContent []byte)

func readSequentiallyAndVerify(t *testing.T, filePath string, expectedContent []byte) {
	readContent, err := operations.ReadFileSequentially(filePath, 1024*1024)

	// For sequential reads, we expect the content to be exactly as expected.
	require.NoErrorf(t, err, "failed to read file %q sequentially: %v", filePath, err)
	require.Equal(t, expectedContent, readContent)
}

func readRandomlyAndVerify(t *testing.T, filePath string, expectedContent []byte) {
	file, err := operations.OpenFileAsReadonly(filePath)
	require.NoErrorf(t, err, "failed to open file %q: %v", filePath, err)
	defer operations.CloseFileShouldNotThrowError(t, file)
	if len(expectedContent) == 0 {
		t.SkipNow()
	}
	fileInfo, err := file.Stat()
	require.NoError(t, err)
	fileSize := fileInfo.Size()
	require.GreaterOrEqualf(t, fileSize, int64(len(expectedContent)), "file %q is too small to read %d bytes", filePath, len(expectedContent))

	// Content to be read from [0, maxOffset) .
	maxOffset := len(expectedContent)
	// Limit number of reads if the content to read is too small.
	numReads := min(maxOffset, 10)
	for i := range numReads {
		// Ensure offset <= maxOffset-1 .
		offset := rand.IntN(maxOffset)
		// Ensure (offset+readSize) <= maxOffset and readSize >= 1.
		readSize := rand.IntN(maxOffset-offset) + 1
		buffer := make([]byte, readSize)

		n, err := file.ReadAt(buffer, int64(offset))

		require.NoErrorf(t, err, "Random-read failed at iter#%d to read file %q at [%d, %d): %v", i, filePath, offset, offset+readSize, err)
		require.Equalf(t, readSize, n, "failed to read %v bytes from %q at offset %v. Read bytes = %v.", readSize, filePath, offset, n)
		require.Equalf(t, expectedContent[offset:offset+n], buffer[:n], "content mismatch in random read at iter#%d at offset [%d, %d): expected %q, got %q", i, offset, offset+readSize, expectedContent[offset:offset+n], buffer[:n])
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

// runAppendAndReadTest contains the core test logic for the ReadsTestSuite.
func (t *ReadsTestSuite) runAppendAndReadTest(verifyFunc readAndVerifyFunc) {
	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	appendPath := path.Join(s.getAppendPath(), s.fileName)
	appendFileHandle := operations.OpenFileInMode(s.T(), appendPath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
	defer operations.CloseFileShouldNotThrowError(s.T(), appendFileHandle)

	readPath := path.Join(s.primaryMount.testDirPath, s.fileName)
	for i := 0; i < numAppends; i++ {
		sizeBeforeAppend := len(s.fileContent)
		s.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))
		sizeAfterAppend := len(s.fileContent)

		// If metadata cache is enabled, gcsfuse reads up to the cached file size.
		// The initial read (i=0) bypasses cache, seeing the latest file size.
		if !s.cfg.metadataCacheOnRead || !s.cfg.isDualMount || (i == 0) {
			time.Sleep(time.Minute)
			verifyFunc(s.T(), readPath, []byte(s.fileContent[:sizeAfterAppend]))
		} else {
			// Read only up to the cached file size (before append).
			tc.readAndVerify(t.T(), readPath, []byte(t.fileContent[:sizeBeforeAppend]))

			// Wait for metadata cache to expire to fetch the latest size for the next read.
			time.Sleep(time.Duration(metadataCacheTTLSecs) * time.Second)
			// Expect read up to the latest file size which is the size after the append.
			tc.readAndVerify(t.T(), readPath, []byte(t.fileContent[:sizeAfterAppend]))
		}
	}
}

func (s *ReadsTestSuite) TestSequentialRead() {
	s.runAppendAndReadTest(readSequentiallyAndVerify)
}

func (s *ReadsTestSuite) TestRandomRead() {
	s.runAppendAndReadTest(readRandomlyAndVerify)
}
