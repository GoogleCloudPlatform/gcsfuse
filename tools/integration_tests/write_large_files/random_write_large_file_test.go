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
	"crypto/rand"
	"log"
	rand2 "math/rand"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const NumberOfRandomWriteCalls = 20
const DirForRandomWrite = "dirForRandomWrite"
const MinWritableByteFromFile = 0
const MaxWritableByteFromFile = 500 * OneMB
const FileDownloadedFromBucket = "fileDownloadedFromBucket"
const FiveHundredMBFileForRandomWriteInLocalSystem = "fiveHundredMBFileForRandomWriteInLocalSystem"

func TestWriteLargeFileRandomly(t *testing.T) {
	randomWriteDir := path.Join(setup.MntDir(), DirForRandomWrite)
	filePath := path.Join(randomWriteDir, FiveHundredMBFile)

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Open file for write at start: %v", err)
		return
	}

	for i := 0; i < NumberOfRandomWriteCalls; i++ {
		offset := rand2.Int63n(MaxWritableByteFromFile - MinWritableByteFromFile)

		// Generate chunk with random data.
		chunk := make([]byte, ChunkSize)
		_, err := rand.Read(chunk)
		if err != nil {
			log.Fatalf("error while generating random string: %s", err)
		}

		// Write data in the file.
		n, err := f.WriteAt(chunk, offset)
		if err != nil {
			t.Errorf("Error in writing randomly in file:%v", err)
		}
		if n != ChunkSize {
			t.Errorf("Incorrect number of bytes written in the file.")
		}

		err = f.Sync()
		if err != nil {
			t.Errorf("Error in syncing file:%v", err)
		}

		// Download the file from a bucket in which we write the content and compare with
		// the file content we wrote in mntDir.
		filePathInGcsBucket := path.Join(setup.TestBucket(), DirForRandomWrite, FiveHundredMBFile)
		localFilePath := path.Join(os.Getenv("HOME"), FiveHundredMBFileForRandomWriteInLocalSystem)
		err = compareFileFromGCSBucketAndMntDir(filePathInGcsBucket, filePath, localFilePath)
		if err != nil {
			t.Errorf("File content did not match:%v", err)
		}
	}

	// Clean up.
	operations.RemoveDir(randomWriteDir)
}
