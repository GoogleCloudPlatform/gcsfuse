// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package list_large_dir

import (
	"context"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type listLargeDir struct {
	suite.Suite
}

func (t *listLargeDir) TearDownSuite() {
	err := DeleteAllObjectsWithPrefix(ctx, storageClient, t.T().Name())
	assert.NoError(t.T(), err)
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

// validateDirectory checks if the directory listing matches expectations.
func validateDirectory(t *testing.T, objs []os.DirEntry, expectExplicitDirs, expectImplicitDirs bool) {
	t.Helper()

	var (
		numberOfFiles        int
		numberOfExplicitDirs int
		numberOfImplicitDirs int
	)

	for _, obj := range objs {
		if !obj.IsDir() {
			numberOfFiles++
			checkIfObjNameIsCorrect(t, obj.Name(), prefixFileInDirectoryWithTwelveThousandFiles, numberOfFilesInDirectoryWithTwelveThousandFiles)
		} else if strings.Contains(obj.Name(), prefixExplicitDirInLargeDirListTest) {
			numberOfExplicitDirs++
			checkIfObjNameIsCorrect(t, obj.Name(), prefixExplicitDirInLargeDirListTest, numberOfExplicitDirsInDirectoryWithTwelveThousandFiles)
		} else if strings.Contains(obj.Name(), prefixImplicitDirInLargeDirListTest) {
			numberOfImplicitDirs++
			checkIfObjNameIsCorrect(t, obj.Name(), prefixImplicitDirInLargeDirListTest, numberOfImplicitDirsInDirectoryWithTwelveThousandFiles)
		}
	}

	if numberOfFiles != numberOfFilesInDirectoryWithTwelveThousandFiles {
		t.Errorf("Incorrect number of files: got %d, want %d", numberOfFiles, numberOfFilesInDirectoryWithTwelveThousandFiles)
	}

	if expectExplicitDirs && numberOfExplicitDirs != numberOfExplicitDirsInDirectoryWithTwelveThousandFiles {
		t.Errorf("Incorrect number of explicit directories: got %d, want %d", numberOfExplicitDirs, numberOfExplicitDirsInDirectoryWithTwelveThousandFiles)
	}

	if expectImplicitDirs && numberOfImplicitDirs != numberOfImplicitDirsInDirectoryWithTwelveThousandFiles {
		t.Errorf("Incorrect number of implicit directories: got %d, want %d", numberOfImplicitDirs, numberOfImplicitDirsInDirectoryWithTwelveThousandFiles)
	}
}

// checkIfObjNameIsCorrect validates the object name against a prefix and expected range.
func checkIfObjNameIsCorrect(t *testing.T, objName string, prefix string, maxNumber int) {
	t.Helper()

	objNumberStr := strings.TrimPrefix(objName, prefix)
	objNumber, err := strconv.Atoi(objNumberStr)
	if err != nil {
		t.Errorf("Error extracting object number from %q: %v", objName, err)
	}
	if objNumber < 1 || objNumber > maxNumber {
		t.Errorf("Invalid object number in %q: %d (should be between 1 and %d)", objName, objNumber, maxNumber)
	}
}

func splitBucketNameAndDirPath(t *testing.T, bucketNameWithDirPath string) (bucketName, dirPathInBucket string) {
	t.Helper()

	var found bool
	bucketName, dirPathInBucket, found = strings.Cut(bucketNameWithDirPath, "/")
	if !found {
		t.Errorf("Unexpected bucketNameWithDirPath: %q. Expected form: <bucket>/<object-name>", bucketNameWithDirPath)
	}
	return
}

// This function is equivalent to testdata/upload_files_to_bucket.sh to replace gcloud with storage-client
// This is needed for ZB which is not supported by gcloud storage cp command yet.
func testdataUploadFilesToBucket(ctx context.Context, t *testing.T, storageClient *storage.Client, bucketNameWithDirPath, dirWith12KFiles, filesPrefix string) {
	t.Helper()

	bucketName, dirPathInBucket := splitBucketNameAndDirPath(t, bucketNameWithDirPath)

	dirWith12KFilesFullPathPrefix := filepath.Join(dirWith12KFiles, filesPrefix)
	matches, err := filepath.Glob(dirWith12KFilesFullPathPrefix + "*")
	if err != nil {
		t.Fatalf("Failed to get files of pattern %s*: %v", dirWith12KFilesFullPathPrefix, err)
	}

	type copyRequest struct {
		srcLocalFilePath string
		dstGCSObjectPath string
	}
	channel := make(chan copyRequest)

	// Copy request producer.
	go func() {
		for _, match := range matches {
			_, fileName := filepath.Split(match)
			if len(fileName) > 0 {
				req := copyRequest{srcLocalFilePath: match, dstGCSObjectPath: filepath.Join(dirPathInBucket, fileName)}
				channel <- req
			}
		}
		// close the channel to let the go-routines know that there is no more object to be copied.
		close(channel)
	}()

	// Copy request consumers.
	numCopyGoroutines := 16
	var wg sync.WaitGroup
	for copyGoroutine := 0; copyGoroutine < numCopyGoroutines; copyGoroutine++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				copyRequest, ok := <-channel
				if !ok {
					break
				}
				client.CopyFileInBucket(ctx, storageClient, copyRequest.srcLocalFilePath, copyRequest.dstGCSObjectPath, bucketName)
			}
		}()
	}
	wg.Wait()
}

// createFilesAndUpload generates files and uploads them to the specified directory.
func createFilesAndUpload(t *testing.T, dirPath string) {
	t.Helper()

	localDirPath := path.Join(os.Getenv("HOME"), directoryWithTwelveThousandFiles)
	operations.CreateDirectoryWithNFiles(numberOfFilesInDirectoryWithTwelveThousandFiles, localDirPath, prefixFileInDirectoryWithTwelveThousandFiles, t)
	defer os.RemoveAll(localDirPath)

	if setup.IsZonalBucketRun() {
		testdataUploadFilesToBucket(ctx, t, storageClient, dirPath, localDirPath, prefixFileInDirectoryWithTwelveThousandFiles)
	} else {
		setup.RunScriptForTestData("testdata/upload_files_to_bucket.sh", dirPath, localDirPath, prefixFileInDirectoryWithTwelveThousandFiles)
	}
}

// createExplicitDirs creates empty explicit directories in the specified directory.
func createExplicitDirs(t *testing.T, dirPath string) {
	t.Helper()

	for i := 1; i <= numberOfExplicitDirsInDirectoryWithTwelveThousandFiles; i++ {
		subDirPath := path.Join(dirPath, fmt.Sprintf("%s%d", prefixExplicitDirInLargeDirListTest, i))
		operations.CreateDirectoryWithNFiles(0, subDirPath, "", t)
	}
}

// listDirTime measures the time taken to list a directory with and without cache.
func listDirTime(t *testing.T, dirPath string, expectExplicitDirs bool, expectImplicitDirs bool) (time.Duration, time.Duration) {
	t.Helper()

	startTime := time.Now()
	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Fatalf("Error in listing directory: %v", err)
	}
	endTime := time.Now()

	validateDirectory(t, objs, expectExplicitDirs, expectImplicitDirs)
	firstListTime := endTime.Sub(startTime)

	minSecondListTime := time.Duration(math.MaxInt64)
	for i := 0; i < 5; i++ {
		startTime = time.Now()
		objs, err = os.ReadDir(dirPath)
		if err != nil {
			t.Fatalf("Error in listing directory: %v", err)
		}
		endTime = time.Now()
		validateDirectory(t, objs, expectExplicitDirs, expectImplicitDirs)
		secondListTime := endTime.Sub(startTime)
		if secondListTime < minSecondListTime {
			minSecondListTime = secondListTime
		}
	}
	return firstListTime, minSecondListTime
}

// This function is equivalent to testdata/create_implicit_dir.sh to replace gcloud with storage-client
// This is needed for ZB which is not supported by gcloud storage cp command yet.
func testdataCreateImplicitDir(ctx context.Context, t *testing.T, storageClient *storage.Client, bucketNameWithDirPath, prefixImplicitDirInLargeDirListTest string, numberOfImplicitDirsInDirectory int) {
	t.Helper()

	bucketName, dirPathInBucket := splitBucketNameAndDirPath(t, bucketNameWithDirPath)

	testFile, err := operations.CreateLocalTempFile("", false)
	if err != nil {
		t.Fatalf("Failed to create local file for creating copies ...")
	}
	for suffix := 1; suffix <= numberOfImplicitDirsInDirectory; suffix++ {
		client.CopyFileInBucket(ctx, storageClient, testFile, path.Join(dirPathInBucket, fmt.Sprintf("%s%d", prefixImplicitDirInLargeDirListTest, suffix), testFile), bucketName)
	}
}

// prepareTestDirectory sets up a test directory with files and required explicit and implicit directories.
func prepareTestDirectory(t *testing.T, withExplicitDirs bool, withImplicitDirs bool) string {
	t.Helper()

	testDirPathOnBucket := path.Join(setup.TestBucket(), t.Name())
	testDirPath := path.Join(setup.MntDir(), t.Name())

	err := os.MkdirAll(testDirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	createFilesAndUpload(t, testDirPathOnBucket)

	if withExplicitDirs {
		createExplicitDirs(t, testDirPath)
	}

	if withImplicitDirs {
		if setup.IsZonalBucketRun() {
			testdataCreateImplicitDir(ctx, t, storageClient, testDirPathOnBucket, prefixImplicitDirInLargeDirListTest, numberOfImplicitDirsInDirectoryWithTwelveThousandFiles)
		} else {
			setup.RunScriptForTestData("testdata/create_implicit_dir.sh", testDirPathOnBucket, prefixImplicitDirInLargeDirListTest, strconv.Itoa(numberOfImplicitDirsInDirectoryWithTwelveThousandFiles))
		}
	}

	return testDirPath
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *listLargeDir) TestListDirectoryWithTwelveThousandFiles() {
	dirPath := prepareTestDirectory(t.T(), false, false)

	firstListTime, secondListTime := listDirTime(t.T(), dirPath, false, false)

	assert.Less(t.T(), secondListTime, firstListTime)
	assert.Less(t.T(), 2*secondListTime, firstListTime)
}

func (t *listLargeDir) TestListDirectoryWithTwelveThousandFilesAndHundredExplicitDir() {
	dirPath := prepareTestDirectory(t.T(), true, false)

	firstListTime, secondListTime := listDirTime(t.T(), dirPath, true, false)

	assert.Less(t.T(), secondListTime, firstListTime)
	assert.Less(t.T(), 2*secondListTime, firstListTime)
}

func (t *listLargeDir) TestListDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir() {
	dirPath := prepareTestDirectory(t.T(), true, true)

	firstListTime, secondListTime := listDirTime(t.T(), dirPath, true, true)

	assert.Less(t.T(), secondListTime, firstListTime)
	assert.Less(t.T(), 2*secondListTime, firstListTime)
}

////////////////////////////////////////////////////////////////////////
// Test Suite Function
////////////////////////////////////////////////////////////////////////

func TestListLargeDir(t *testing.T) {
	suite.Run(t, new(listLargeDir))
}
