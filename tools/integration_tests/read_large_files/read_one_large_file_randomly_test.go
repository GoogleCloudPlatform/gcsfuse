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

package read_large_files

import (
	"bytes"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestReadLargeFileRandomly(t *testing.T) {
	fileInLocalDisk := path.Join(os.Getenv("HOME"), FiveHundredMBFile)
	file := path.Join(setup.MntDir(), FiveHundredMBFile)

	// Create File in local disk with given size and copy it in mountedDirectory.
	createFileInLocalDiskWithGivenSizeAndCopyInMntDir(fileInLocalDisk, file, t)

	for i := 0; i < NumberOfRandomReadCalls; i++ {
		offset := rand.Int63n(MaxReadbleByteFromFile - MinReadbleByteFromFile)
		// Randomly read the data from file in mountedDirectory.
		content, err := operations.ReadChunkFromFile(file, chunkSize, offset)
		if err != nil {
			t.Errorf("Error in reading file: %v", err)
		}

		// Read actual content from file located in local disk.
		actualContent, err := operations.ReadChunkFromFile(fileInLocalDisk, chunkSize, offset)
		if err != nil {
			t.Errorf("Error in reading file: %v", err)
		}

		// Compare actual content and expect content.
		if bytes.Equal(actualContent, content) == false {
			t.Errorf("Error in reading file randomly.")
		}
	}

	// Removing file after testing.
	operations.RemoveFile(fileInLocalDisk)
}
