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

// Provides integration tests for write large files sequentially and randomly.

package write_large_files

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
	"golang.org/x/sync/errgroup"
)

func TestWriteToSameFileConcurrently(t *testing.T) {
	// Setup Test directory and files to write to.
	seqWriteDir := setup.SetupTestDirectory(DirForSeqWrite)
	mountedDirFilePath := path.Join(seqWriteDir, setup.GenerateRandomString(5))
	localFilePath := path.Join(TmpDir, setup.GenerateRandomString(5))
	t.Cleanup(func() { operations.RemoveFile(localFilePath) })
	// We will have x numbers of concurrent writers trying to write to the same file.
	// Every thread will start at offset = writer_index * (fileSize/thread_count).
	var eG errgroup.Group
	concurrentWriterCount := 5
	chunkSize := 50 * OneMiB / concurrentWriterCount

	for i := range concurrentWriterCount {
		offset := i * chunkSize
		eG.Go(func() error {
			writeToFileSequentially(t, []string{localFilePath, mountedDirFilePath}, offset, offset+chunkSize)
			return nil
		})
	}

	// Wait on threads to end.
	err := eG.Wait()
	require.NoError(t, err)
	identical, err := operations.AreFilesIdentical(mountedDirFilePath, localFilePath)
	require.NoError(t, err)
	assert.True(t, identical)
}

func writeToFileSequentially(t *testing.T, filePaths []string, startOffset int, endOffset int) {
	t.Helper()
	filesToWrite := operations.OpenFiles(t, filePaths)
	defer operations.CloseFiles(t, filesToWrite)

	var chunkSize = OneMiB
	for startOffset < endOffset {
		chunkSize = min(chunkSize, endOffset-startOffset)

		err := operations.WriteChunkOfRandomBytesToFiles(filesToWrite, chunkSize, int64(startOffset))
		require.NoError(t, err)

		startOffset = startOffset + chunkSize
	}
	if setup.IsZonalBucketRun() {
		operations.SyncFiles(filesToWrite, t)
	}
}
