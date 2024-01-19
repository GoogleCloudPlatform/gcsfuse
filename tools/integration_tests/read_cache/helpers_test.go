// Copyright 2024 Google Inc. All Rights Reserved.
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
	"log"
	"os"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

// Expected is a helper struct that stores list of attributes to be validated from logs.
type Expected struct {
	StartTimeStampSeconds int64
	EndTimeStampSeconds   int64
	BucketName            string
	ObjectName            string
	content               string
}

func readFileAndGetExpectedOutcome(testDirPath, fileName string, t *testing.T) *Expected {
	expected := &Expected{
		StartTimeStampSeconds: time.Now().Unix(),
		BucketName:            setup.TestBucket(),
		ObjectName:            path.Join(testDirName, fileName),
	}

	content, err := operations.ReadFileSequentially(path.Join(testDirPath, fileName), chunkSizeToRead)
	if err != nil {
		t.Errorf("Failed to read file in first iteration: %v", err)
	}
	expected.EndTimeStampSeconds = time.Now().Unix()
	expected.content = string(content)

	return expected
}

func validate(expected *Expected, logEntry *read_logs.StructuredReadLogEntry,
	isSeq, cacheHit bool, chunkCount int, t *testing.T) {
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
	return path.Join(cacheLocationPath, cacheSubDirectoryName, setup.TestBucket(), testDirName, fileName)
}

func validateFileSizeInCacheDirectory(fileName string, filesize int64, t *testing.T) {
	// Validate that the file is present in cache location.
	expectedPathOfCachedFile := getCachedFilePath(fileName)
	fileInfo, err := operations.StatFile(expectedPathOfCachedFile)
	if err != nil {
		t.Errorf("Failed to find cached file %s: %v", expectedPathOfCachedFile, err)
	}
	// Validate file size in cache directory matches actual file size.
	if (*fileInfo).Size() != filesize {
		t.Errorf("Incorrect cached file size. Expected %d, Got: %d", filesize, (*fileInfo).Size())
	}
}

func validateFileInCacheDirectory(fileName string, filesize int64, ctx context.Context, storageClient *storage.Client, t *testing.T) {
	validateFileSizeInCacheDirectory(fileName, filesize, t)
	// Validate content of file in cache directory matches GCS.
	expectedPathOfCachedFile := getCachedFilePath(fileName)
	content, err := operations.ReadFile(expectedPathOfCachedFile)
	if err != nil {
		t.Errorf("Failed to read cached file %s: %v", expectedPathOfCachedFile, err)
	}
	client.ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, fileName, string(content), t)
}

func validateFileIsNotCached(fileName string, t *testing.T) {
	// Validate that the file is not present in cache location.
	expectedPathOfCachedFile := getCachedFilePath(fileName)
	_, err := operations.StatFile(expectedPathOfCachedFile)
	if err == nil {
		t.Errorf("File %s found in cache directory", expectedPathOfCachedFile)
	}
}

func unmountGCSFuseAndDeleteLogFile() {
	if setup.MountedDirectory() == "" {
		// Unmount GCSFuse only when tests are not running on mounted directory.
		err := setup.UnMount()
		if err != nil {
			setup.LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
		}
		// delete log file created
		err = os.Remove(setup.LogFile())
		if err != nil {
			setup.LogAndExit(fmt.Sprintf("Error in deleting log file: %v", err))
		}
	}
}

func remountGCSFuseAndValidateCacheDeleted(flags []string, t *testing.T) {
	setup.SetMntDir(rootDir)
	unmountGCSFuseAndDeleteLogFile()

	validateCacheSizeWithinLimit(0, t)

	mountGCSFuse(flags)
	setup.SetMntDir(mountDir)
}

func mountGCSFuse(flags []string) {
	if setup.MountedDirectory() == "" {
		// Mount GCSFuse only when tests are not running on mounted directory.
		if err := mountFunc(flags); err != nil {
			setup.LogAndExit(fmt.Sprintf("Failed to mount GCSFuse: %v", err))
		}
	}
}

func createStorageClient(t *testing.T, ctx *context.Context, storageClient **storage.Client) func() {
	var err error
	var cancel context.CancelFunc
	*ctx, cancel = context.WithTimeout(*ctx, time.Minute*15)
	*storageClient, err = client.CreateStorageClient(*ctx)
	if err != nil {
		log.Fatalf("client.CreateStorageClient: %v", err)
	}
	// return func to close storage client and release resources.
	return func() {
		err := (*storageClient).Close()
		if err != nil {
			t.Log("Failed to close storage client")
		}
		defer cancel()
	}
}

func readFileAndValidateCacheWithGCS(ctx context.Context, storageClient *storage.Client,
	filename string, fileSize int64, t *testing.T) (expectedOutcome *Expected) {
	// Read file via gcsfuse mount.
	expectedOutcome = readFileAndGetExpectedOutcome(testDirPath, filename, t)
	// Validate cached content with gcs.
	validateFileInCacheDirectory(filename, fileSize, ctx, storageClient, t)
	// Validate cache size within limit.
	validateCacheSizeWithinLimit(cacheCapacityInMB, t)
	// Validate content read via gcsfuse with gcs.
	client.ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, filename,
		expectedOutcome.content, t)

	return expectedOutcome
}

func ReadFileAndValidateFileIsNotCached(ctx context.Context, storageClient *storage.Client, filename string, t *testing.T) (expectedOutcome *Expected) {
	// Read file via gcsfuse mount.
	expectedOutcome = readFileAndGetExpectedOutcome(testDirPath, filename, t)
	// Validate that the file is not cached.
	validateFileIsNotCached(filename, t)
	// validate the content read matches the content on GCS.
	client.ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, filename,
		expectedOutcome.content, t)

	return expectedOutcome
}

func modifyFile(ctx context.Context, storageClient *storage.Client, testFileName string, t *testing.T) {
	objectName := path.Join(testDirName, testFileName)
	err := client.WriteToObject(ctx, storageClient, objectName, smallContent, storage.Conditions{})
	if err != nil {
		t.Errorf("Could not modify object %s: %v", objectName, err)
	}
}

func validateCacheSizeWithinLimit(cacheCapacity int64, t *testing.T) {
	cacheSize, err := operations.DirSizeMiB(cacheLocationPath)
	if err != nil {
		t.Errorf("Error in getting cache size: %v", cacheSize)
	}
	if cacheSize > cacheCapacity {
		t.Errorf("CacheSize %d is more than cache capacity %d ", cacheSize, cacheCapacity)
	}
}

func setupFileInTestDir(ctx context.Context, storageClient *storage.Client, testDirName string, fileSize int64, t *testing.T) (fileName string) {
	testFileName := testFileName + setup.GenerateRandomString(testFileNameSuffixLength)
	client.SetupFileInTestDirectory(ctx, storageClient, testDirName, testFileName, fileSize, t)

	return testFileName
}
