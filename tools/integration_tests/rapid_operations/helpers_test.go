// Copyright 2026 Google LLC
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

package rapid_operations

import (
	"math/rand/v2"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
)

func (t *BaseSuite) createGCSFile(useAppendableAPI bool, fileSize int64) (filePath string, content []byte) {
	t.T().Helper()

	t.fileName = fileNamePrefix + setup.GenerateRandomString(5)
	filePath = path.Join(t.primaryMount.testDirPath, t.fileName)
	content, err := operations.GenerateRandomData(fileSize)
	require.NoError(t.T(), err)

	// Create file directly in GCS using Go SDK with the chosen API.
	err = client.CreateObjectWithAPI(testEnv.ctx, testEnv.storageClient, path.Join(testDirName, t.fileName), content, useAppendableAPI)
	require.NoError(t.T(), err)
	return filePath, content
}

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
	file, err := os.OpenFile(filePath, os.O_RDONLY, setup.FilePermission_0600)
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
