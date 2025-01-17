package read_gcs_algo

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

type testCase struct {
	name      string // Name of the test case
	offset    int64
	chunkSize int64
}

func TestReadSequentialWithDifferentBlockSizes(t *testing.T) {
	fileSize := 10 * OneMB
	filePathInLocalDisk, filePathInMntDir := setup.CreateFileAndCopyToMntDir(t, fileSize, DirForReadAlgoTests)

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
