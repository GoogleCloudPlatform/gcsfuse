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
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"golang.org/x/sync/errgroup"
)

func TestWriteToSameFileConcurrently(t *testing.T) {
	seqWriteDir := setup.SetupTestDirectory(DirForSeqWrite)
	mountedFilePath := path.Join(seqWriteDir, "50mb"+setup.GenerateRandomString(5)+".txt")
	localFilePath := path.Join(TmpDir, "50mbLocal"+setup.GenerateRandomString(5)+".txt")
	localFile := operations.CreateFile(localFilePath, setup.FilePermission_0600, t)

	// Clean up.
	defer operations.RemoveDir(seqWriteDir)
	defer operations.RemoveFile(localFilePath)

	var eG errgroup.Group
	concurrentWriterCount := 5
	chunkSize := 50 * OneMiB / concurrentWriterCount

	// We will have x numbers of concurrent threads trying to write from the same file.
	// Every thread will start at offset = thread_index * (fileSize/thread_count).
	for i := 0; i < concurrentWriterCount; i++ {
		offset := i * chunkSize

		eG.Go(func() error {
			return writeToFileSequentially(localFilePath, mountedFilePath, offset, offset+chunkSize, t)
		})
	}

	// Wait on threads to end.
	err := eG.Wait()
	if err != nil {
		t.Errorf("writing failed")
	}

	// Close the local file since the below method will open the file again.
	operations.CloseFile(localFile)

	identical, err := operations.AreFilesIdentical(mountedFilePath, localFilePath)
	if !identical {
		t.Fatalf("Comparision failed: %v", err)
	}
}

func writeToFileSequentially(localFilePath string, mountedFilePath string, startOffset int, endOffset int, t *testing.T) (err error) {
	mountedFile, err := os.OpenFile(mountedFilePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, setup.FilePermission_0600)
	if err != nil {
		t.Fatalf("Error in opening file: %v", err)
	}

	localFile, err := os.OpenFile(localFilePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, setup.FilePermission_0600)
	if err != nil {
		t.Fatalf("Error in opening file: %v", err)
	}

	filesToWrite := []*os.File{localFile, mountedFile}

	// Closing file at the end.
	defer operations.CloseFile(mountedFile)
	defer operations.CloseFile(localFile)

	var chunkSize = 5 * OneMiB
	for startOffset < endOffset {
		if (endOffset - startOffset) < chunkSize {
			chunkSize = endOffset - startOffset
		}

		err := operations.WriteChunkOfRandomBytesToFiles(filesToWrite, chunkSize, int64(startOffset))
		if err != nil {
			t.Fatalf("Error in writing chunk: %v", err)
		}

		startOffset = startOffset + chunkSize
	}
	return
}
