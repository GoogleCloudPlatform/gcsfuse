package write_large_files

import (
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"golang.org/x/sync/errgroup"
)

func TestWriteToSameFileConcurrently(t *testing.T) {
	seqWriteDir := path.Join(setup.MntDir(), DirForSeqWrite)
	setup.SetupTestDirectory(DirForSeqWrite)
	mountedFilePath := path.Join(seqWriteDir, FiveHundredMBFile)
	localFilePath := path.Join(TmpDir, FiveHundredMBFileForSeqWriteInLocalSystem)
	localFile := operations.CreateFile(localFilePath, setup.FilePermission_0600, t)

	// Clean up.
	defer operations.RemoveDir(seqWriteDir)
	defer operations.ReadFile(localFilePath)

	var eG errgroup.Group
	concurrentWriterCount := 5
	chunkSize := 50 * OneMiB / concurrentWriterCount

	// We will have x numbers of concurrent threads trying to write from the same file.
	// Every thread will start at offset = thread_index * (fileSize/thread_count).
	for i := 0; i < concurrentWriterCount; i++ {
		offset := i * chunkSize

		eG.Go(func() error {
			return writeToFileSequentially(localFile, mountedFilePath, offset, offset+chunkSize, t)
		})
	}

	// Wait on threads to end.
	err := eG.Wait()
	if err != nil {
		t.Errorf("writing failed")
	}

	// Close the local file since the below method will open the file again.
	operations.CloseFile(localFile)

	identical, err := operations.AreFilesIdentical(mountedFilePath, localFilePath)
	if !identical {
		t.Fatalf("Comparision failed: %v", err)
	}
}

func writeToFileSequentially(localFile *os.File, mountedFilePath string, startOffset int, endOffset int, t *testing.T) (err error) {
	mountedFile, err := os.OpenFile(mountedFilePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, setup.FilePermission_0600)
	filesToWrite := []*os.File{localFile, mountedFile}
	if err != nil {
		t.Fatalf("Error in opening file: %v", err)
	}

	// Closing file at the end.
	defer operations.CloseFile(mountedFile)

	var chunkSize = 5 * OneMiB
	for startOffset < endOffset {
		if (endOffset - startOffset) < chunkSize {
			chunkSize = endOffset - startOffset
		}

		err := operations.WriteChunkOfRandomBytesToFiles(filesToWrite, chunkSize, int64(startOffset))
		if err != nil {
			t.Fatalf("Error in writing chunk: %v", err)
		}

		startOffset = startOffset + chunkSize
	}
	return
}
