// Copyright 2024 Google LLC
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

package read_cache

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"
	"syscall"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Expected is a helper struct that stores list of attributes to be validated from logs.
type Expected struct {
	StartTimeStampSeconds int64
	EndTimeStampSeconds   int64
	BucketName            string
	ObjectName            string
	content               string
}

func readFileAndGetExpectedOutcome(testDirPath, fileName string, readFullFile bool, offset int64, t *testing.T) *Expected {
	expected := &Expected{
		StartTimeStampSeconds: time.Now().Unix(),
		BucketName:            setup.TestBucket(),
		ObjectName:            path.Join(testDirName, fileName),
	}
	if setup.DynamicBucketMounted() != "" {
		expected.BucketName = setup.DynamicBucketMounted()
	}

	var content []byte
	var err error

	if readFullFile {
		file, err := os.OpenFile(path.Join(testDirPath, fileName), os.O_RDONLY|syscall.O_DIRECT, setup.FilePermission_0600)
		require.NoError(t, err)
		content, err = operations.ReadFileSequentially(file, chunkSizeToRead)
		if err != nil {
			t.Errorf("Failed to read file sequentially: %v", err)
		}
	} else {
		content, err = operations.ReadChunkFromFile(path.Join(testDirPath, fileName), chunkSizeToRead, offset, os.O_RDONLY|syscall.O_DIRECT)
		if err != nil {
			t.Errorf("Failed to read random file chunk: %v", err)
		}
	}
	expected.EndTimeStampSeconds = time.Now().Unix()
	expected.content = string(content)

	return expected
}

func validate(expected *Expected, logEntry *read_logs.StructuredReadLogEntry, isSeq, cacheHit bool, chunkCount int, t *testing.T) {
	t.Helper()
	if logEntry.StartTimeSeconds < expected.StartTimeStampSeconds {
		t.Errorf("start time in logs %d less than actual start time %d.", logEntry.StartTimeSeconds, expected.StartTimeStampSeconds)
	}
	if logEntry.BucketName != expected.BucketName {
		t.Errorf("Bucket names don't match! Expected: %s, Got from logs: %s",
			expected.BucketName, logEntry.BucketName)
	}
	if logEntry.ObjectName != expected.ObjectName {
		t.Errorf("Object names don't match! Expected: %s, Got from logs: %s",
			expected.ObjectName, logEntry.ObjectName)
	}
	if len(logEntry.Chunks) != chunkCount {
		t.Errorf("chunks read don't match! Expected: %d, Got from logs: %d",
			chunkCount, len(logEntry.Chunks))
	}
	// If no chunks are found in logs, we can't validate further.
	if len(logEntry.Chunks) == 0 {
		return
	}
	if logEntry.Chunks[len(logEntry.Chunks)-1].StartTimeSeconds > expected.EndTimeStampSeconds {
		t.Errorf("end time in logs more than actual end time.")
	}
	if cacheHit != logEntry.Chunks[0].CacheHit {
		t.Errorf("Expected Cache Hit: %t, Got from logs: %t", cacheHit, logEntry.Chunks[0].CacheHit)
	}
	if isSeq != logEntry.Chunks[0].IsSequential {
		t.Errorf("Expected Is Sequential: %t, Got from logs: %t", isSeq, logEntry.Chunks[0].IsSequential)
	}
}

func getCachedFilePath(fileName string) string {
	bucketName := testEnv.cfg.TestBucket
	if setup.DynamicBucketMounted() != "" {
		bucketName = setup.DynamicBucketMounted()
	}
	return path.Join(testEnv.cacheDirPath, cacheSubDirectoryName, bucketName, testDirName, fileName)
}

func validateFileSizeInCacheDirectory(fileName string, filesize int64, t *testing.T) {
	maxRetries := 25
	retryDelay := 500 * time.Millisecond
	expectedPathOfCachedFile := getCachedFilePath(fileName)
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Validate that the file is present in cache location.
		var fileInfo *fs.FileInfo
		fileInfo, err = operations.StatFile(expectedPathOfCachedFile)
		// Validate file size in cache directory matches actual file size.
		if err == nil && fileInfo != nil {
			if filesize != (*fileInfo).Size() {
				err = fmt.Errorf("incorrect cached file size. Expected: %d, Got %d", filesize, (*fileInfo).Size())
				t.Logf("Incorrect cached file size, retrying %d...", attempt)
			} else {
				break
			}
		}
		time.Sleep(retryDelay)
	}
	require.Nil(t, err)
}

func validateFileInCacheDirectory(fileName string, filesize int64, ctx context.Context, storageClient *storage.Client, t *testing.T) {
	validateFileSizeInCacheDirectory(fileName, filesize, t)

	gcsCRC, err := client.GetCRCFromGCS(path.Join(testDirName, fileName), ctx, storageClient)
	require.NoError(t, err)
	maxRetries := 20
	retryDelay := 500 * time.Millisecond
	cachedFilePath := getCachedFilePath(fileName)

	// Validate CRC of cached file matches GCS CRC.
	for attempt := 1; attempt <= maxRetries; attempt++ {
		var cachedFileCRC uint32
		cachedFileCRC, err = operations.CalculateFileCRC32(cachedFilePath)
		require.NoError(t, err)
		if gcsCRC != cachedFileCRC {
			err = fmt.Errorf("CRC32 mismatch. Expected %d, Got %d", gcsCRC, cachedFileCRC)
		} else {
			break
		}
		time.Sleep(retryDelay)
	}
	require.NoError(t, err)
}

func validateFileIsNotCached(fileName string, t *testing.T) {
	// Validate that the file is not present in cache location.
	expectedPathOfCachedFile := getCachedFilePath(fileName)
	_, err := operations.StatFile(expectedPathOfCachedFile)
	if err == nil {
		t.Errorf("File %s found in cache directory", expectedPathOfCachedFile)
	}
}

func validateFileIsCached(fileName string, t *testing.T) {
	// Validate that the file is present in cache location.
	expectedPathOfCachedFile := getCachedFilePath(fileName)
	_, err := operations.StatFile(expectedPathOfCachedFile)
	if err != nil {
		t.Errorf("File %s not found in cache directory", expectedPathOfCachedFile)
	}
}

func remountGCSFuse(flags []string) {
	setup.SetMntDir(rootDir)
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)

	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, flags, mountFunc)
	setup.SetMntDir(mountDir)
}

func readFileAndValidateCacheWithGCS(ctx context.Context, storageClient *storage.Client, filename string, fileSize int64, checkCacheSize bool, t *testing.T) (expectedOutcome *Expected) {
	// Read file via gcsfuse mount.
	expectedOutcome = readFileAndGetExpectedOutcome(testEnv.testDirPath, filename, true, zeroOffset, t)
	// Validate CRC32 of content read via gcsfuse with CRC32 value on gcs.
	gotCRC32Value, err := operations.CalculateCRC32(strings.NewReader(expectedOutcome.content))
	if err != nil {
		t.Errorf("CalculateCRC32 Failed: %v", err)
	}
	gcsCRC, err := client.GetCRCFromGCS(path.Join(testDirName, filename), ctx, storageClient)
	if err != nil || gcsCRC != gotCRC32Value {
		t.Errorf("Content served CRC mismatch: %v", err)
	}
	// Validate cached content with gcs.
	validateFileInCacheDirectory(filename, fileSize, ctx, storageClient, t)
	if checkCacheSize {
		// Validate cache size within limit.
		validateCacheSizeWithinLimit(cacheCapacityInMB, t)
	}

	return expectedOutcome
}

func readChunkAndValidateObjectContentsFromGCS(ctx context.Context, storageClient *storage.Client,
	filename string, offset int64, t *testing.T) (expectedOutcome *Expected) {
	// Read file via gcsfuse mount.
	expectedOutcome = readFileAndGetExpectedOutcome(testEnv.testDirPath, filename, false, offset, t)
	// Validate content read via gcsfuse with gcs.
	client.ValidateObjectChunkFromGCS(ctx, storageClient, testDirName, filename, offset, chunkSizeToRead,
		expectedOutcome.content, t)

	return expectedOutcome
}

func readFileAndValidateFileIsNotCached(ctx context.Context, storageClient *storage.Client, fileName string, readFullFile bool, offset int64, t *testing.T) (expectedOutcome *Expected) {
	// Read file via gcsfuse mount.
	expectedOutcome = readFileAndGetExpectedOutcome(testEnv.testDirPath, fileName, readFullFile, offset, t)
	// Validate that the file is not cached.
	validateFileIsNotCached(fileName, t)
	// validate the content read matches the content on GCS.
	if readFullFile {
		client.ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, fileName, expectedOutcome.content, t)
	} else {
		client.ValidateObjectChunkFromGCS(ctx, storageClient, testDirName, fileName, offset, chunkSizeToRead, expectedOutcome.content, t)
	}
	return expectedOutcome
}

func modifyFile(ctx context.Context, storageClient *storage.Client, testFileName string, t *testing.T) {
	objectName := path.Join(testDirName, testFileName)
	smallContent, err := operations.GenerateRandomData(smallContentSize)
	if err != nil {
		t.Errorf("Could not generate random data to modify file: %v", err)
	}
	err = client.WriteToObject(ctx, storageClient, objectName, string(smallContent), storage.Conditions{})
	if err != nil {
		t.Errorf("Could not modify object %s: %v", objectName, err)
	}
}

func validateCacheSizeWithinLimit(cacheCapacity int64, t *testing.T) {
	cacheSize, err := operations.DirSizeMiB(testEnv.cacheDirPath)
	if err != nil {
		t.Errorf("Error in getting cache size: %v", cacheSize)
	}
	if cacheSize > cacheCapacity {
		t.Errorf("CacheSize %d is more than cache capacity %d ", cacheSize, cacheCapacity)
	}
}

func setupFileInTestDir(ctx context.Context, storageClient *storage.Client, fileSize int64, t *testing.T) (fileName string) {
	testFileName := testFileName + setup.GenerateRandomString(testFileNameSuffixLength)
	client.SetupFileInTestDirectory(ctx, storageClient, testDirName, testFileName, fileSize, t)

	return testFileName
}

func runTestsOnlyForDynamicMount(t *testing.T) {
	if !strings.Contains(setup.MntDir(), setup.TestBucket()) {
		log.Println("This test will run only for dynamic mounting...")
		t.SkipNow()
	}
}

func validateAllocatedFileSize(t *testing.T, fileName string, expectedSize int64) {
	cachedFilePath := getCachedFilePath(fileName)
	fi, err := os.Stat(cachedFilePath)
	require.NoError(t, err)
	stat := fi.Sys().(*syscall.Stat_t)
	allocatedSize := stat.Blocks * 512

	// Allow small overhead (1KB) for metadata stored for this file.
	overhead := int64(1024)
	assert.LessOrEqual(t, allocatedSize, expectedSize+overhead, "Allocated size should be close to expected size")
	assert.GreaterOrEqual(t, allocatedSize, expectedSize, "Allocated size should be at least expected size")
}

func validateDownloads(t *testing.T, downloads []read_logs.ChunkDownloadLogEntry, expectedRanges []data.ObjectRange, fileName string) {
	require.Len(t, downloads, len(expectedRanges), "Expected exactly %d downloads", len(expectedRanges))

	// Count expected
	expectedCounts := make(map[data.ObjectRange]int)
	for _, r := range expectedRanges {
		expectedCounts[r]++
	}

	// Count actual
	actualCounts := make(map[data.ObjectRange]int)
	for _, d := range downloads {
		r := data.ObjectRange{Start: d.StartOffset, End: d.EndOffset}
		actualCounts[r]++
		if fileName != "" {
			validateContent(t, fileName, d.StartOffset, d.EndOffset)
		}
	}
	assert.Equal(t, expectedCounts, actualCounts, "Download ranges mismatch")
}

func validateContent(t *testing.T, fileName string, start, end int64) {
	// Read from cache
	cacheFilePath := getCachedFilePath(fileName)
	file, err := os.Open(cacheFilePath)
	require.NoError(t, err)
	defer file.Close()
	size := end - start
	cacheContent := make([]byte, size)
	_, err = file.ReadAt(cacheContent, start)
	require.NoError(t, err)

	// Read from GCS
	objectName := path.Join(testDirName, fileName)
	bucketName, objectName := setup.GetBucketAndObjectBasedOnTypeOfMount(objectName)
	rc, err := testEnv.storageClient.Bucket(bucketName).Object(objectName).NewRangeReader(testEnv.ctx, start, size)
	require.NoError(t, err)
	defer rc.Close()

	gcsContent, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, gcsContent, cacheContent, "Content mismatch for range [%d, %d)", start, end)
}
