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

// //////////////////////////////////////////////////////////////////////
// Tests for the SingleMountReadsTestSuite
// //////////////////////////////////////////////////////////////////////

// runSingleMountAppendAndReadTest contains the core test logic for single-mount read tests.
// It verifies that appends are eventually consistent and visible on the same mount.
func (t *SingleMountReadsTestSuite) runSingleMountAppendAndReadTest(verifyFunc readAndVerifyFunc) {
	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	filePath := path.Join(t.primaryMount.testDirPath, t.fileName)
	appendFileHandle := operations.OpenFileInMode(t.T(), filePath, FileOpenModeA|syscall.O_DIRECT)
	defer operations.CloseFileShouldNotThrowError(t.T(), appendFileHandle)

	for range numAppends {
		t.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))
		// Wait for a minute to ensure consistency.
		time.Sleep(time.Minute)
		verifyFunc(t.T(), filePath, []byte(t.fileContent))
	}
}

func (t *SingleMountReadsTestSuite) TestSequentialRead() {
	t.runSingleMountAppendAndReadTest(readSequentiallyAndVerify)
}

func (t *SingleMountReadsTestSuite) TestRandomRead() {
	t.runSingleMountAppendAndReadTest(readRandomlyAndVerify)
}

// //////////////////////////////////////////////////////////////////////
// Tests for the DualMountReadsTestSuite
// //////////////////////////////////////////////////////////////////////

// runDualMountAppendAndReadTest contains the core test logic for dual-mount read tests.
// It verifies the behavior of metadata caching by appending on a secondary mount
// and reading from a primary mount.
func (t *DualMountReadsTestSuite) runDualMountAppendAndReadTest(verifyFunc readAndVerifyFunc) {
	if !t.cfg.metadataCacheEnabled {
		t.T().Skip("This test is only meaningful when metadata caching is enabled.")
	}

	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	appendPath := path.Join(t.secondaryMount.testDirPath, t.fileName)
	appendFileHandle := operations.OpenFileInMode(t.T(), appendPath, FileOpenModeA|syscall.O_DIRECT)
	defer operations.CloseFileShouldNotThrowError(t.T(), appendFileHandle)

	readPath := path.Join(t.primaryMount.testDirPath, t.fileName)

	// Perform initial append and read to populate the cache.
	t.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))
	time.Sleep(time.Minute) // Wait for consistency before the first read.
	verifyFunc(t.T(), readPath, []byte(t.fileContent))

	// Subsequent appends will test cache invalidation.
	for i := 0; i < numAppends-1; i++ {
		sizeBeforeAppend := len(t.fileContent)
		t.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))

		// Read should see stale content due to the cache.
		verifyFunc(t.T(), readPath, []byte(t.fileContent[:sizeBeforeAppend]))

		// Wait for metadata cache TTL to expire.
		time.Sleep(time.Duration(metadataCacheTTLSecs) * time.Second)

		// After TTL expiration, read should see the updated content.
		verifyFunc(t.T(), readPath, []byte(t.fileContent))
	}
}

func (t *DualMountReadsTestSuite) TestSequentialRead() {
	t.runDualMountAppendAndReadTest(readSequentiallyAndVerify)
}

func (t *DualMountReadsTestSuite) TestRandomRead() {
	t.runDualMountAppendAndReadTest(readRandomlyAndVerify)
}
