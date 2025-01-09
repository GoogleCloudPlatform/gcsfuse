package streaming_writes

import (
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
)

func (tt *defaultMountTest) TestTruncate(t *testing.T) {
	truncateSize := 2 * 1024 * 1024
	fileName := "truncate"
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t)

	err := fh.Truncate(int64(truncateSize))

	if err != nil {
		t.Fatalf("Error in truncating: %v", err)
	}
	data := make([]byte, truncateSize)
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, fileName, string(data[:]), t)
}

func (tt *defaultMountTest) TestWriteAfterTruncate(t *testing.T) {
	truncateSize := 10

	testCases := []struct {
		name     string
		offset   int64
		fileSize int64
	}{
		{
			name:     "ZeroOffset",
			offset:   0,
			fileSize: 10,
		},
		{
			name:     "RandomOffset",
			offset:   5,
			fileSize: 10,
		},
		{
			name:     "Append",
			offset:   10,
			fileSize: 12,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, tc.fileSize)
			// Create a local file.
			_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, tc.name, t)
			// Perform truncate.
			err := fh.Truncate(int64(truncateSize))
			if err != nil {
				t.Fatalf("Error in truncating: %v", err)
			}

			// Triggers writes after truncate.
			newData := []byte("hi")
			_, err = fh.WriteAt(newData, tc.offset)
			if err != nil {
				t.Fatalf("Error in writing: %v", err)
			}

			data[tc.offset] = newData[0]
			data[tc.offset+1] = newData[1]
			// Close the file and validate that the file is created on GCS.
			CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, tc.name, string(data[:]), t)
		})
	}

}

func (tt *defaultMountTest) TestWriteAndTruncate(t *testing.T) {
	truncateSize := 20
	fileName := "writeAndTruncate"
	// Create a local file and write
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t)
	operations.WriteWithoutClose(fh, FileContents, t)

	err := fh.Truncate(int64(truncateSize))
	if err != nil {
		t.Fatalf("Error in truncating: %v", err)
	}

	data := make([]byte, 10)
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, fileName, FileContents+string(data[:]), t)
}

func (tt *defaultMountTest) TestWriteTruncateWrite(t *testing.T) {
	truncateSize := 30
	fileName := "writeTruncateWrite"
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t)

	// Write
	operations.WriteWithoutClose(fh, FileContents, t)
	// Perform truncate
	err := fh.Truncate(int64(truncateSize))
	if err != nil {
		t.Fatalf("Error in truncating: %v", err)
	}
	// Write
	operations.WriteWithoutClose(fh, FileContents, t)

	data := make([]byte, 10)
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, fileName, FileContents+FileContents+string(data[:]), t)
}

func (tt *defaultMountTest) TestTruncateToLowerSizeAfterWrite(t *testing.T) {
	fileName := "truncateToLowerSizeAfterWrite"
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t)

	// Write
	operations.WriteWithoutClose(fh, FileContents+FileContents, t)

	// Perform truncate
	err := fh.Truncate(int64(5))
	if err == nil {
		// Truncating to lower size after writes are not allowed.
		t.Fatalf("Truncate should fail, but it didnt: %v", err)
	}
}
