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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
)

func TestSeqReadThenRandomThenSeqRead(t *testing.T) {
	filePathInLocalDisk, filePathInMntDir := operations.CreateFileAndCopyToMntDir(t, 50*OneMB, DirForReadAlgoTests)

	// Current read algorithm:
	// https://github.com/GoogleCloudPlatform/gcsfuse/blob/v2.5.1/internal/gcsx/random_reader.go#L275
	// First 2 reads are considered sequential.
	offset := int64(40 * OneMB)
	chunkSize := int64(OneMB)
	operations.ReadAndCompare(t, filePathInMntDir, filePathInLocalDisk, offset, chunkSize)
	offset = int64(35 * OneMB)
	operations.ReadAndCompare(t, filePathInMntDir, filePathInLocalDisk, offset, chunkSize)

	// Perform a couple of random reads.
	offset = int64(30 * OneMB)
	operations.ReadAndCompare(t, filePathInMntDir, filePathInLocalDisk, offset, chunkSize)
	offset = int64(25 * OneMB)
	operations.ReadAndCompare(t, filePathInMntDir, filePathInLocalDisk, offset, chunkSize)
	offset = int64(20 * OneMB)
	operations.ReadAndCompare(t, filePathInMntDir, filePathInLocalDisk, offset, chunkSize)

	// Here we are reading a chunkSize of 40MB which gets converted to sequential because of our
	// current read algorithm.
	offset = int64(10 * OneMB)
	operations.ReadAndCompare(t, filePathInMntDir, filePathInLocalDisk, offset, 40*OneMB)
}
