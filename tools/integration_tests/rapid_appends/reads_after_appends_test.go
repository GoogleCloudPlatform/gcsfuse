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
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// declare a function type for read and verify
type readAndVerifyFunc func(t *testing.T, filePath string, expectedContent []byte)

func readSequentiallyAndVerify(t *testing.T, filePath string, expectedContent []byte) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, setup.FilePermission_0600)
	require.NoError(t, err)
	readContent, err := operations.ReadFileSequentially(file, 1024*1024)

	// For sequential reads, we expect the content to be exactly as expected.
	require.NoErrorf(t, err, "failed to read file %q sequentially: %v", filePath, err)
	require.Equal(t, expectedContent, readContent)
}

func readRandomlyAndVerify(t *testing.T, filePath string, expectedContent []byte) {
	file, err := operations.OpenFileAsReadonly(filePath)
	require.NoErrorf(t, err, "failed to open file %q: %v", filePath, err)
	defer operations.CloseFileShouldNotThrowError(t, file) // This line is already correct.
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
// Tests for the SingleMountReadsTestSuite
////////////////////////////////////////////////////////////////////////

// runAppendAndReadTest contains the core test logic for the SingleMountReadsTestSuite.
func (t *SingleMountReadsTestSuite) runAppendAndReadTest(verifyFunc readAndVerifyFunc) {
	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(t.primaryMount.testDirPath, t.fileName), fileOpenModeAppend|syscall.O_DIRECT)
	defer operations.CloseFileShouldNotThrowError(t.T(), appendFileHandle)

	readPath := path.Join(t.primaryMount.testDirPath, t.fileName)
	for i := range numAppends {
		// Wait for a minute for stat to return the correct file size, which is needed by appendToFile.
		if i > 0 {
			time.Sleep(operations.WaitDurationAfterFlushZB)
		}

		t.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))
		sizeAfterAppend := len(t.fileContent)

		// For same-mount appends/reads, file size is always current.
		verifyFunc(t.T(), readPath, []byte(t.fileContent[:sizeAfterAppend]))
	}
}

func (t *SingleMountReadsTestSuite) TestSequentialRead() {
	t.runAppendAndReadTest(readSequentiallyAndVerify)
}

func (t *SingleMountReadsTestSuite) TestRandomRead() {
	t.runAppendAndReadTest(readRandomlyAndVerify)
}

////////////////////////////////////////////////////////////////////////
// Tests for the DualMountReadsTestSuite
////////////////////////////////////////////////////////////////////////

// runAppendAndReadTest contains the core test logic for the DualMountReadsTestSuite.
func (t *DualMountReadsTestSuite) runAppendAndReadTest(verifyFunc readAndVerifyFunc) {
	t.createUnfinalizedObject()
	defer t.deleteUnfinalizedObject()

	appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(t.getAppendPath(), t.fileName), fileOpenModeAppend|syscall.O_DIRECT)
	defer operations.CloseFileShouldNotThrowError(t.T(), appendFileHandle)

	readPath := path.Join(t.primaryMount.testDirPath, t.fileName)
	for i := range numAppends {
		sizeBeforeAppend := len(t.fileContent)
		t.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))
		sizeAfterAppend := len(t.fileContent)

		// If metadata cache is enabled, gcsfuse reads up to the cached file size.
		// The initial read (i=0) bypasses cache, seeing the latest file size.
		if !t.cfg.metadataCacheEnabled || (i == 0) {
			verifyFunc(t.T(), readPath, []byte(t.fileContent[:sizeAfterAppend]))
		} else {
			// Read only up to the cached file size (before append).
			verifyFunc(t.T(), readPath, []byte(t.fileContent[:sizeBeforeAppend]))

			// Wait for metadata cache to expire to fetch the latest size for the next read.
			// Metadata update for appends in current iteration itself takes a minute, so the
			// cached size will expire in ttl-60 secs from now, so wait accordingly.
			time.Sleep(time.Duration(metadataCacheTTLSecs*time.Second - operations.WaitDurationAfterFlushZB))
			// Expect read up to the latest file size which is the size after the append.
			verifyFunc(t.T(), readPath, []byte(t.fileContent[:sizeAfterAppend]))
		}
	}
}

func (t *DualMountReadsTestSuite) TestSequentialRead() {
	t.runAppendAndReadTest(readSequentiallyAndVerify)
}

func (t *DualMountReadsTestSuite) TestRandomRead() {
	t.runAppendAndReadTest(readRandomlyAndVerify)
}

////////////////////////////////////////////////////////////////////////
// Test Runner
////////////////////////////////////////////////////////////////////////

// readTestConfigs defines the matrix of configurations for the ReadsTestSuite.
var readTestConfigs = []*testConfig{
	// Single-Mount Scenarios
	{
		name:                 "SingleMount_NoCache",
		isDualMount:          false,
		metadataCacheEnabled: false,
		fileCacheEnabled:     false,
	},
	{
		name:                 "SingleMount_MetadataCache",
		isDualMount:          false,
		metadataCacheEnabled: true,
		fileCacheEnabled:     false,
	},
	{
		name:                 "SingleMount_FileCache",
		isDualMount:          false,
		metadataCacheEnabled: false,
		fileCacheEnabled:     true,
	},
	{
		name:                 "SingleMount_MetadataAndFileCache",
		isDualMount:          false,
		metadataCacheEnabled: true,
		fileCacheEnabled:     true,
	},
	// Dual-Mount Scenarios
	{
		name:                 "DualMount_NoCache",
		isDualMount:          true,
		metadataCacheEnabled: false,
		fileCacheEnabled:     false,
	},
	{
		name:                 "DualMount_MetadataCache",
		isDualMount:          true,
		metadataCacheEnabled: true,
		fileCacheEnabled:     false,
	},
	{
		name:                 "DualMount_FileCache",
		isDualMount:          true,
		metadataCacheEnabled: false,
		fileCacheEnabled:     true,
	},
	{
		name:                 "DualMount_MetadataAndFileCache",
		isDualMount:          true,
		metadataCacheEnabled: true,
		fileCacheEnabled:     true,
	},
}

// TestReadsSuiteRunner executes all read-after-append tests against the readTestConfigs matrix.
func TestReadsSuiteRunner(t *testing.T) {
	for _, cfg := range readTestConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			if cfg.isDualMount {
				suite.Run(t, &DualMountReadsTestSuite{BaseSuite{cfg: cfg}})
			} else {
				suite.Run(t, &SingleMountReadsTestSuite{BaseSuite{cfg: cfg}})
			}
		})
	}
}
