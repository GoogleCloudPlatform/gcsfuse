// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Integration tests for sparse file functionality
package sparse_file

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	testDirName      = "sparse_file_test"
	largeFileSize    = 100 * 1024 * 1024 // 100MB
	chunkSize        = 20 * 1024 * 1024  // 20MB
	smallReadSize    = 1 * 1024 * 1024   // 1MB
)

type sparseFileTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	cacheDirPath  string
	suite.Suite
}

func (s *sparseFileTest) SetupSuite() {
	s.ctx = context.Background()
	var err error
	s.storageClient, err = client.CreateStorageClient(s.ctx)
	require.NoError(s.T(), err)

	s.cacheDirPath = path.Join(os.Getenv("HOME"), "cache-dir-sparse-file")
	s.flags = []string{
		"--implicit-dirs",
		"--file-cache-max-size-mb=-1",
		fmt.Sprintf("--cache-dir=%s", s.cacheDirPath),
		"--file-cache-enable-sparse-file",
		fmt.Sprintf("--file-cache-download-chunk-size-mb=%d", chunkSize/(1024*1024)),
	}

	setup.MountGCSFuseWithGivenMountFunc(s.flags, &s.ctx, s.storageClient, setup.MountGCSFuseForTesting)
}

func (s *sparseFileTest) SetupTest() {
	operations.RemoveDir(s.cacheDirPath)
	s.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *sparseFileTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *sparseFileTest) TearDownSuite() {
	setup.UnmountGCSFuse(setup.TestDir())
}

func TestSparseFile(t *testing.T) {
	suite.Run(t, new(sparseFileTest))
}

// Test that random reads on a large file only download the necessary chunks
func (s *sparseFileTest) TestRandomReadsDownloadOnlyRequiredChunks() {
	// Create a large file in GCS
	objectName := "large_sparse_file.txt"
	objectPath := path.Join(s.testDirPath, objectName)

	// Create object content
	content := make([]byte, largeFileSize)
	for i := range content {
		content[i] = byte(i % 256)
	}

	// Upload to GCS
	err := client.WriteToObject(s.ctx, s.storageClient, objectPath, content)
	require.NoError(s.T(), err)

	mountedFilePath := path.Join(setup.MntDir(), testDirName, objectName)

	// Read a small chunk at offset 50MB (should download chunk [40MB, 60MB))
	file, err := os.Open(mountedFilePath)
	require.NoError(s.T(), err)
	defer file.Close()

	offset := int64(50 * 1024 * 1024)
	_, err = file.Seek(offset, 0)
	require.NoError(s.T(), err)

	readBuf := make([]byte, smallReadSize)
	n, err := file.Read(readBuf)
	require.NoError(s.T(), err)
	require.Equal(s.T(), smallReadSize, n)

	// Verify content correctness
	expectedContent := content[offset : offset+int64(smallReadSize)]
	require.Equal(s.T(), expectedContent, readBuf, "Read content should match GCS object content")

	// Read another small chunk at offset 10MB (should download chunk [0MB, 20MB))
	offset2 := int64(10 * 1024 * 1024)
	_, err = file.Seek(offset2, 0)
	require.NoError(s.T(), err)

	n, err = file.Read(readBuf)
	require.NoError(s.T(), err)
	require.Equal(s.T(), smallReadSize, n)

	expectedContent2 := content[offset2 : offset2+int64(smallReadSize)]
	require.Equal(s.T(), expectedContent2, readBuf, "Second read content should match")
}

// Test that subsequent reads to the same chunk are cache hits
func (s *sparseFileTest) TestSubsequentReadsAreCacheHits() {
	objectName := "sparse_cache_hit.txt"
	objectPath := path.Join(s.testDirPath, objectName)

	content := make([]byte, largeFileSize)
	for i := range content {
		content[i] = byte(i % 256)
	}

	err := client.WriteToObject(s.ctx, s.storageClient, objectPath, content)
	require.NoError(s.T(), err)

	mountedFilePath := path.Join(setup.MntDir(), testDirName, objectName)

	// First read at offset 25MB
	file, err := os.Open(mountedFilePath)
	require.NoError(s.T(), err)
	defer file.Close()

	offset := int64(25 * 1024 * 1024)
	_, err = file.Seek(offset, 0)
	require.NoError(s.T(), err)

	readBuf := make([]byte, smallReadSize)
	n, err := file.Read(readBuf)
	require.NoError(s.T(), err)
	require.Equal(s.T(), smallReadSize, n)

	// Close and reopen to test cache persistence
	file.Close()

	file2, err := os.Open(mountedFilePath)
	require.NoError(s.T(), err)
	defer file2.Close()

	// Second read at the same offset should be a cache hit
	_, err = file2.Seek(offset, 0)
	require.NoError(s.T(), err)

	readBuf2 := make([]byte, smallReadSize)
	n, err = file2.Read(readBuf2)
	require.NoError(s.T(), err)
	require.Equal(s.T(), smallReadSize, n)

	// Content should be identical
	require.Equal(s.T(), readBuf, readBuf2, "Cached read should return same content")
}

// Test sequential reads work correctly with sparse files
func (s *sparseFileTest) TestSequentialReadsWithSparseFiles() {
	objectName := "sparse_sequential.txt"
	objectPath := path.Join(s.testDirPath, objectName)

	fileSize := 50 * 1024 * 1024 // 50MB
	content := make([]byte, fileSize)
	for i := range content {
		content[i] = byte(i % 256)
	}

	err := client.WriteToObject(s.ctx, s.storageClient, objectPath, content)
	require.NoError(s.T(), err)

	mountedFilePath := path.Join(setup.MntDir(), testDirName, objectName)

	// Read the entire file sequentially
	file, err := os.Open(mountedFilePath)
	require.NoError(s.T(), err)
	defer file.Close()

	readContent := make([]byte, fileSize)
	n, err := file.Read(readContent)
	require.NoError(s.T(), err)
	require.Equal(s.T(), fileSize, n)

	require.Equal(s.T(), content, readContent, "Sequential read should return correct content")
}

// Test reading at chunk boundaries
func (s *sparseFileTest) TestReadsAtChunkBoundaries() {
	objectName := "sparse_boundary.txt"
	objectPath := path.Join(s.testDirPath, objectName)

	content := make([]byte, largeFileSize)
	for i := range content {
		content[i] = byte(i % 256)
	}

	err := client.WriteToObject(s.ctx, s.storageClient, objectPath, content)
	require.NoError(s.T(), err)

	mountedFilePath := path.Join(setup.MntDir(), testDirName, objectName)
	file, err := os.Open(mountedFilePath)
	require.NoError(s.T(), err)
	defer file.Close()

	// Read exactly at the chunk boundary (20MB)
	offset := int64(chunkSize)
	_, err = file.Seek(offset, 0)
	require.NoError(s.T(), err)

	readBuf := make([]byte, smallReadSize)
	n, err := file.Read(readBuf)
	require.NoError(s.T(), err)
	require.Equal(s.T(), smallReadSize, n)

	expectedContent := content[offset : offset+int64(smallReadSize)]
	require.Equal(s.T(), expectedContent, readBuf, "Read at chunk boundary should work correctly")
}
