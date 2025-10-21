// Copyright 2023 Google LLC
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
	"math/rand"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
)

func TestReadLargeFileRandomly(t *testing.T) {
	// Create and copy the local file in mountedDirectory.
	fileInLocalDisk, fileInMntDir := operations.CreateFileAndCopyToMntDir(t, FiveHundredMB, DirForReadLargeFilesTests)

	for range NumberOfRandomReadCalls {
		offset := rand.Int63n(MaxReadableByteFromFile - MinReadableByteFromFile)
		// Randomly read the data from file in mountedDirectory.
		operations.ReadAndCompare(t, fileInMntDir, fileInLocalDisk, offset, ChunkSize)
	}

	// Removing file after testing.
	operations.RemoveFile(fileInLocalDisk)
}
