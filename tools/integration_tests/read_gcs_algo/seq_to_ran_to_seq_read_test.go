package read_gcs_algo

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

func TestSeqReadThenRandomThenSeqRead(t *testing.T) {
	filePathInLocalDisk, filePathInMntDir := setup.CreateFileAndCopyToMntDir(t, 50*OneMB, DirForReadAlgoTests)

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
