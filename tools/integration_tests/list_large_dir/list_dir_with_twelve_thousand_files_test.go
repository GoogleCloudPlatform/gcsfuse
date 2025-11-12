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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"golang.org/x/sync/errgroup"

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

		if strings.Contains(obj.Name(), prefixExplicitDirInLargeDirListTest) {
			numberOfExplicitDirs++
			checkIfObjNameIsCorrect(t, obj.Name(), prefixExplicitDirInLargeDirListTest, numberOfExplicitDirsInDirectoryWithTwelveThousandFiles)
		} else if strings.Contains(obj.Name(), prefixImplicitDirInLargeDirListTest) {
			numberOfImplicitDirs++
			checkIfObjNameIsCorrect(t, obj.Name(), prefixImplicitDirInLargeDirListTest, numberOfImplicitDirsInDirectoryWithTwelveThousandFiles)
		} else {
			numberOfFiles++
			checkIfObjNameIsCorrect(t, obj.Name(), prefixFileInDirectoryWithTwelveThousandFiles, numberOfFilesInDirectoryWithTwelveThousandFiles)
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

// testdataUploadFilesToBucket uploads matching files from a local directory to a specified path in a GCS bucket.
func testdataUploadFilesToBucket(ctx context.Context, t *testing.T, storageClient *storage.Client, bucketNameWithDirPath, dirWith12KFiles, filesPrefix string) {
	t.Helper()
	bucketName, dirPathInBucket := operations.SplitBucketNameAndDirPath(t, bucketNameWithDirPath)
	err := client.BatchUploadFilesWithoutIntermediateDelays(ctx, storageClient, bucketName, dirPathInBucket, dirWith12KFiles, filesPrefix)
	assert.NoError(t, err)
}

// createFilesAndUpload generates files and uploads them to the specified directory.
func createFilesAndUpload(t *testing.T, dirPath string) {
	t.Helper()

	localDirPath := path.Join(os.Getenv("HOME"), directoryWithTwelveThousandFiles)
	operations.CreateDirectoryWithNFiles(numberOfFilesInDirectoryWithTwelveThousandFiles, localDirPath, prefixFileInDirectoryWithTwelveThousandFiles, t)
	defer os.RemoveAll(localDirPath)

	testdataUploadFilesToBucket(ctx, t, storageClient, dirPath, localDirPath, prefixFileInDirectoryWithTwelveThousandFiles)
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
	for range 5 {
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

// testdataCreateImplicitDir creates implicit directories by uploading files with nested paths.
func testdataCreateImplicitDir(t *testing.T, ctx context.Context, storageClient *storage.Client, bucketNameWithDirPath string) {
	t.Helper()

	bucketName, dirPathInBucket := operations.SplitBucketNameAndDirPath(t, bucketNameWithDirPath)

	testFile, err := operations.CreateLocalTempFile("", false)
	if err != nil {
		t.Fatalf("Failed to create local file for creating copies ...")
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU()/2) // Concurrency limiter

	for suffix := 1; suffix <= numberOfImplicitDirsInDirectoryWithTwelveThousandFiles; suffix++ {
		objectPath := path.Join(dirPathInBucket, fmt.Sprintf("%s%d", prefixImplicitDirInLargeDirListTest, suffix), testFile)

		wg.Add(1)
		go func(destinationPath string) {
			defer wg.Done()
			sem <- struct{}{}        // acquire semaphore
			defer func() { <-sem }() // release semaphore

			client.CopyFileInBucketWithPreconditions(ctx, storageClient, testFile, destinationPath, bucketName, &storage.Conditions{DoesNotExist: true})
		}(objectPath)
	}

	wg.Wait()
}

// testdataCreateExplicitDir creates explicit directories (trailing slash objects) in the bucket.
func testdataCreateExplicitDir(t *testing.T, ctx context.Context, storageClient *storage.Client, bucketNameWithDirPath string) {
	t.Helper()

	bucketName, dirPathInBucket := operations.SplitBucketNameAndDirPath(t, bucketNameWithDirPath)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU() / 2) // Concurrency limiter

	for dirIndex := 1; dirIndex <= numberOfExplicitDirsInDirectoryWithTwelveThousandFiles; dirIndex++ {
		capturedIndex := dirIndex
		g.Go(func() error {
			dirName := fmt.Sprintf("%s%d", prefixExplicitDirInLargeDirListTest, capturedIndex)
			return client.CreateGcsDir(ctx, storageClient, dirName, bucketName, dirPathInBucket)
		})
	}

	if err := g.Wait(); err != nil {
		t.Fatalf("Failed to create explicit dirs: %v", err)
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
		testdataCreateExplicitDir(t, ctx, storageClient, testDirPathOnBucket)
	}

	if withImplicitDirs {
		testdataCreateImplicitDir(t, ctx, storageClient, testDirPathOnBucket)
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
	if setup.IsZonalBucketRun() {
		t.T().Skipf("Redundant test for ZB as implicit-dir is a non-HNS concept, hence not applicable here. ")
	}
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
