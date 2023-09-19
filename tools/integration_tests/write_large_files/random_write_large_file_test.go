// Copyright 2023 Google Inc. All Rights Reserved.
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

package write_large_files

import (
	rand2 "math/rand"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	NumberOfRandomWriteCalls                     = 20
	DirForRandomWrite                            = "dirForRandomWrite"
	MaxWritableByteFromFile                      = 500 * OneMiB
	FiveHundredMBFileForRandomWriteInLocalSystem = "fiveHundredMBFileForRandomWriteInLocalSystem"
)

func TestWriteLargeFileRandomly(t *testing.T) {
	randomWriteDir := path.Join(setup.MntDir(), DirForRandomWrite)
	err := os.Mkdir(randomWriteDir, setup.FilePermission_0600)
	if err != nil {
		t.Fatalf("Error in creating directory: %v", err)
	}
	filePath := path.Join(randomWriteDir, FiveHundredMBFile)

	// Clean up.
	defer operations.RemoveDir(randomWriteDir)

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|syscall.O_DIRECT, WritePermission_0200)
	if err != nil {
		t.Fatalf("Open file for write at start: %v", err)
	}

	defer operations.CloseFile(f)

	for i := 0; i < NumberOfRandomWriteCalls; i++ {
		offset := rand2.Int63n(MaxWritableByteFromFile)

		// Generate chunk with random data.
		chunkSize := ChunkSize
		if offset+ChunkSize > MaxWritableByteFromFile {
			chunkSize = int(MaxWritableByteFromFile - offset)
		}

		err := operations.WriteChunkOfRandomBytesToFile(f, chunkSize, offset)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		filePathInGcsBucket := path.Join(setup.TestBucket(), DirForRandomWrite, FiveHundredMBFile)
		localFilePath := path.Join(TmpDir, FiveHundredMBFileForRandomWriteInLocalSystem)
		err = compareFileFromGCSBucketAndMntDir(filePathInGcsBucket, filePath, localFilePath)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
	}
}
