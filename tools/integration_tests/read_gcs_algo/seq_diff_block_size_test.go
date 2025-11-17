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

	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
)

type testCase struct {
	name      string // Name of the test case
	offset    int64
	chunkSize int64
}

func TestReadSequentialWithDifferentBlockSizes(t *testing.T) {
	fileSize := 10 * OneMB
	filePathInLocalDisk, filePathInMntDir := operations.CreateFileAndCopyToMntDir(t, fileSize, DirForReadAlgoTests)

	tests := []testCase{
		{
			name:      "0.5MB", // < 1MB
			offset:    0,
			chunkSize: OneMB / 2,
		},
		{
			name:      "1MB", // Equal to kernel max buffer size i.e, 1MB
			offset:    0,
			chunkSize: OneMB,
		},
		{
			name:      "1.5MB", // Not multiple of 1MB
			offset:    0,
			chunkSize: OneMB + (OneMB / 2),
		},
		{
			name:      "5MB", // Multiple of 1MB
			offset:    0,
			chunkSize: 5 * OneMB,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			operations.ReadAndCompare(t, filePathInMntDir, filePathInLocalDisk, tc.offset, tc.chunkSize)
		})
	}
}
