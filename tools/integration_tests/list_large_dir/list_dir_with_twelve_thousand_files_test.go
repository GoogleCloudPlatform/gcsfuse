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
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
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

func testdataUploadFilesToBucket(ctx context.Context, storageClient *storage.Client, bucketNameWithDirPath, dirWithTwelveThousandFiles, filesPrefix string, t *testing.T) {
	t.Helper()
	idx := strings.Index(bucketNameWithDirPath, "/")
	if idx <= 0 {
		t.Errorf("Unexpected bucketNameWithDirPath: %q. Expected form: <bucket>/<object-name>", bucketNameWithDirPath)
	}
	bucketName := bucketNameWithDirPath[:idx]
	dirPathInBucket := bucketNameWithDirPath[idx+1:]
	dirWithTwelveThousandFilesFullPathPrefix := filepath.Join(dirWithTwelveThousandFiles, filesPrefix)
	filepath.WalkDir(dirWithTwelveThousandFiles, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("Failed to walk at path=%q: %w", path, err)
		}
		if !d.IsDir() && strings.HasPrefix(path, dirWithTwelveThousandFilesFullPathPrefix) {
			client.CopyFileInBucket(ctx, storageClient, path, filepath.Join(dirPathInBucket, d.Name()), bucketName)
		}
		return nil
	})
	os.RemoveAll(dirWithTwelveThousandFiles)
}

// createFilesAndUpload generates files and uploads them to the specified directory.
func createFilesAndUpload(t *testing.T, dirPath string) {
	t.Helper()

	localDirPath := path.Join(os.Getenv("HOME"), directoryWithTwelveThousandFiles)
	operations.CreateDirectoryWithNFiles(numberOfFilesInDirectoryWithTwelveThousandFiles, localDirPath, prefixFileInDirectoryWithTwelveThousandFiles, t)

	if setup.IsZonalBucketRun() {
		testdataUploadFilesToBucket(ctx, storageClient, dirPath, localDirPath, prefixFileInDirectoryWithTwelveThousandFiles, t)
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

func testdataCreateImplicitDirUsingStorageClient(ctx context.Context, storageClient *storage.Client, bucketNameWithDirPath, prefixImplicitDirInLargeDirListTest string, numberOfImplicitDirsInDirectory int, t *testing.T) {
	t.Helper()
	idx := strings.Index(bucketNameWithDirPath, "/")
	if idx <= 0 {
		t.Errorf("Unexpected bucketNameWithDirPath: %q. Expected form: <bucket>/<object-name>", bucketNameWithDirPath)
	}
	bucketName := bucketNameWithDirPath[:idx]
	dirPathInBucket := bucketNameWithDirPath[idx+1:]
	testFile, err := operations.CreateLocalTempFile("", false)
	if err != nil {
		t.Fatalf("Failed to local file for creating copies ...")
	}
	for a := 1; a <= numberOfImplicitDirsInDirectory; a++ {
		client.CopyFileInBucket(ctx, storageClient, testFile, path.Join(dirPathInBucket, prefixImplicitDirInLargeDirListTest+fmt.Sprintf("%d", a), testFile), bucketName)
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
			testdataCreateImplicitDirUsingStorageClient(ctx, storageClient, testDirPathOnBucket, prefixImplicitDirInLargeDirListTest, numberOfImplicitDirsInDirectoryWithTwelveThousandFiles, t)
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
