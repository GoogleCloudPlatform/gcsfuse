package streaming_writes

import (
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

func TestWritesWithDifferentConfig(t *testing.T) {
	testCases := []struct {
		name     string
		flags    []string
		fileSize int64
	}{
		{
			name:     "BlockSizeGreaterThanFileSize",
			flags:    []string{"--enable-streaming-writes=true", "--write-block-size-mb=5", "--write-max-blocks-per-file=2"},
			fileSize: 2 * 1024 * 1024,
		},
		{
			name:     "BlockSizeLessThanFileSize",
			flags:    []string{"--enable-streaming-writes=true", "--write-block-size-mb=1", "--write-max-blocks-per-file=20"},
			fileSize: 5 * 1024 * 1024,
		},
		{
			name:     "NumberOfBlocksLessThanFileSize",
			flags:    []string{"--enable-streaming-writes=true", "--write-block-size-mb=1", "--write-max-blocks-per-file=2"},
			fileSize: 10 * 1024 * 1024,
		},
		{
			name:     "BlockSizeEqualToFileSize",
			flags:    []string{"--enable-streaming-writes=true", "--write-block-size-mb=5", "--write-max-blocks-per-file=2"},
			fileSize: 5 * 1024 * 1024,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setup.MountGCSFuseWithGivenMountFunc(tc.flags, mountFunc)
			defer setup.UnmountGCSFuse(rootDir)
			testDirPath = setup.SetupTestDirectory(testDirName)
			// Create a local file.
			_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
			data, err := operations.GenerateRandomData(tc.fileSize)
			if err != nil {
				t.Fatalf("Error in generating data: %v", err)
			}

			// Write data to file.
			operations.WriteAt(string(data[:]), 0, fh, t)

			// Close the file and validate that the file is created on GCS.
			CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, string(data[:]), t)
		})
	}
}
