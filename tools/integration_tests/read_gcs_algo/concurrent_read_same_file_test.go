package read_gcs_algo

import (
	"bytes"
	"io"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"golang.org/x/sync/errgroup"
)

func TestReadSameFileConcurrently(t *testing.T) {
	fileSize := 30 * OneMB
	filePathInLocalDisk, filePathInMntDir := setup.CreateFileAndCopyToMntDir(t, fileSize, DirForReadAlgoTests)

	var eG errgroup.Group
	concurrentReaderCount := 3

	// We will x numbers of concurrent threads trying to read from the same file.
	for i := 0; i < concurrentReaderCount; i++ {
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
	defer operations.CloseFile(mountedFile)

	// Perform 5 reads on each file.
	numberOfReads := 5
	for i := 0; i < numberOfReads; i++ {
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
