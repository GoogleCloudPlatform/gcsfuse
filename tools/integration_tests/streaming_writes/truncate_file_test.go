package streaming_writes

import (
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (t *defaultMountCommonTest) TestTruncate() {
	truncateSize := 2 * 1024 * 1024
	fileName := "truncate"
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t.T())

	err := fh.Truncate(int64(truncateSize))

	assert.NoError(t.T(), err)
	data := make([]byte, truncateSize)
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, fileName, string(data[:]), t.T())
}

func (t *defaultMountCommonTest) TestWriteAfterTruncate() {
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
		t.Run(tc.name, func() {
			data := make([]byte, tc.fileSize)
			// Create a local file.
			_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, tc.name, t.T())
			// Perform truncate.
			err := fh.Truncate(int64(truncateSize))
			require.NoError(t.T(), err)

			// Triggers writes after truncate.
			newData := []byte("hi")
			_, err = fh.WriteAt(newData, tc.offset)

			require.NoError(t.T(), err)
			data[tc.offset] = newData[0]
			data[tc.offset+1] = newData[1]
			// Close the file and validate that the file is created on GCS.
			CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, tc.name, string(data[:]), t.T())
		})
	}

}

func (t *defaultMountCommonTest) TestWriteAndTruncate() {
	truncateSize := 20
	fileName := "writeAndTruncate"
	// Create a local file and write
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t.T())
	operations.WriteWithoutClose(fh, FileContents, t.T())

	err := fh.Truncate(int64(truncateSize))

	require.NoError(t.T(), err)
	data := make([]byte, 10)
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, fileName, FileContents+string(data[:]), t.T())
}

func (t *defaultMountCommonTest) TestWriteTruncateWrite() {
	truncateSize := 30
	fileName := "writeTruncateWrite"
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t.T())

	// Write
	operations.WriteWithoutClose(fh, FileContents, t.T())
	// Perform truncate
	err := fh.Truncate(int64(truncateSize))
	require.NoError(t.T(), err)
	// Write
	operations.WriteWithoutClose(fh, FileContents, t.T())

	data := make([]byte, 10)
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, fileName, FileContents+FileContents+string(data[:]), t.T())
}

func (t *defaultMountCommonTest) TestTruncateToLowerSizeAfterWrite() {
	fileName := "truncateToLowerSizeAfterWrite"
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t.T())

	// Write
	operations.WriteWithoutClose(fh, FileContents+FileContents, t.T())
	// Perform truncate
	err := fh.Truncate(int64(5))

	// Truncating to lower size after writes are not allowed.
	require.Error(t.T(), err)
}
