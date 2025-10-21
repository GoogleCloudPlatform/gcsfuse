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

package read_gcs_algo

import (
	"bytes"
	"io"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"golang.org/x/sync/errgroup"
)

func TestReadSameFileConcurrently(t *testing.T) {
	fileSize := 30 * OneMB
	filePathInLocalDisk, filePathInMntDir := operations.CreateFileAndCopyToMntDir(t, fileSize, DirForReadAlgoTests)

	var eG errgroup.Group
	concurrentReaderCount := 3

	// We will x numbers of concurrent threads trying to read from the same file.
	for range concurrentReaderCount {
		randomOffset := rand.Int64N(int64(fileSize))

		eG.Go(func() error {
			readAndCompare(t, filePathInMntDir, filePathInLocalDisk, randomOffset, 5*OneMB)
			return nil
		})
	}

	// Wait on threads to end. No error is returned by the read method. Hence,
	// nothing handling it.
	_ = eG.Wait()
}

func readAndCompare(t *testing.T, filePathInMntDir string, filePathInLocalDisk string, offset int64, chunkSize int64) {
	mountedFile, err := operations.OpenFileAsReadonly(filePathInMntDir)
	if err != nil {
		t.Fatalf("error in opening file from mounted directory :%d", err)
	}
	defer operations.CloseFileShouldNotThrowError(t, mountedFile)

	// Perform 5 reads on each file.
	numberOfReads := 5
	for range numberOfReads {
		mountContents := make([]byte, chunkSize)
		// Reading chunk size randomly from the file.
		_, err = mountedFile.ReadAt(mountContents, offset)
		if err == io.EOF {
			err = nil
		}
		if err != nil {
			t.Fatalf("error in read file from mounted directory :%d", err)
		}

		diskContents, err := operations.ReadChunkFromFile(filePathInLocalDisk, chunkSize, offset, os.O_RDONLY)
		if err != nil {
			t.Fatalf("error in read file from local directory :%d", err)
		}

		if !bytes.Equal(mountContents, diskContents) {
			t.Fatalf("data mismatch between mounted directory and local disk")
		}
	}
}
