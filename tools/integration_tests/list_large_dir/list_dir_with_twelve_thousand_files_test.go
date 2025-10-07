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
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
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
	start := time.Now()
	log.Printf("(anushkadhn) Test: %s : Starting to delete all files ....\n", t.T().Name())
	err := DeleteAllObjectsWithPrefix(ctx, storageClient, t.T().Name())
	assert.NoError(t.T(), err)
	log.Printf("(anushkadhn) Test: %s : Time taken to delete all objects : %v\n", t.T().Name(), time.Since(start))
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

	dirWith12KFilesFullPathPrefix := filepath.Join(dirWith12KFiles, filesPrefix)
	log.Printf("(anushkadhn) Test: %s : Walk through directory structure begin (filepath.Glob)....\n", t.Name())
	matches, err := filepath.Glob(dirWith12KFilesFullPathPrefix + "*")
	log.Printf("(anushkadhn) Test: %s : Walk through directory structure done (filepath.Glob)....\n", t.Name())
	if err != nil {
		t.Fatalf("Failed to get files of pattern %s*: %v", dirWith12KFilesFullPathPrefix, err)
	}

	type copyRequest struct {
		srcLocalFilePath string
		dstGCSObjectPath string
	}
	channel := make(chan copyRequest, len(matches))

	// Copy request producer.
	go func() {
		for _, match := range matches {
			_, fileName := filepath.Split(match)
			if len(fileName) > 0 {
				req := copyRequest{srcLocalFilePath: match, dstGCSObjectPath: filepath.Join(dirPathInBucket, fileName)}
				channel <- req
			}
		}
		// Close the channel to let the go-routines know that there is no more object to be copied.
		close(channel)
	}()

	// Copy request consumers.
	numCopyGoroutines := runtime.NumCPU() / 2
	log.Printf("(anushkadhn) Test: %s : Num of copy go routines : %v\n", t.Name(), numCopyGoroutines)
	var wg sync.WaitGroup
	for range numCopyGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				copyRequest, ok := <-channel
				if !ok {
					break
				}
				log.Printf("(anushkadhn) Test: %s : Starting copy file %s\n", copyRequest.srcLocalFilePath, t.Name())
				client.CopyFileInBucketWithPreconditions(ctx, storageClient, copyRequest.srcLocalFilePath, copyRequest.dstGCSObjectPath, bucketName, &storage.Conditions{DoesNotExist: true})
				log.Printf("(anushkadhn) Test: %s : Done with copying file %s\n", copyRequest.srcLocalFilePath, t.Name())
			}
		}()
	}
	wg.Wait()
	log.Printf("(anushkadhn) Test: %s : Done uploading all files ...\n", t.Name())
}

// createFilesAndUpload generates files and uploads them to the specified directory.
func createFilesAndUpload(t *testing.T, dirPath string) {
	t.Helper()

	localDirPath := path.Join(os.Getenv("HOME"), directoryWithTwelveThousandFiles)
	start := time.Now()
	log.Printf("(anushkadhn) Test: %s : Starting to create local files....\n", t.Name())
	operations.CreateDirectoryWithNFiles(numberOfFilesInDirectoryWithTwelveThousandFiles, localDirPath, prefixFileInDirectoryWithTwelveThousandFiles, t)
	log.Printf("(anushkadhn) Test: %s : Time taken to create local dirs  : %v\n", t.Name(), time.Since(start))
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

	log.Printf("(anushkadhn) Test: %s : Round 1: starting directory validation....\n", t.Name())
	validateDirectory(t, objs, expectExplicitDirs, expectImplicitDirs)
	log.Printf("(anushkadhn) Test: %s : Round 1:Done with directory validation....\n", t.Name())
	firstListTime := endTime.Sub(startTime)

	minSecondListTime := time.Duration(math.MaxInt64)
	for i := 0; i < 5; i++ {
		startTime = time.Now()
		objs, err = os.ReadDir(dirPath)
		if err != nil {
			t.Fatalf("Error in listing directory: %v", err)
		}
		endTime = time.Now()
		log.Printf("(anushkadhn) Test: %s : Round 2: starting directory validation....\n", t.Name())
		validateDirectory(t, objs, expectExplicitDirs, expectImplicitDirs)
		log.Printf("(anushkadhn) Test: %s : Round 2: Done with directory validation....\n", t.Name())
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

	log.Printf("(anushkadhn) Test: %s : Creating local file ....\n", t.Name())
	testFile, err := operations.CreateLocalTempFile("", false)
	log.Printf("(anushkadhn) Test: %s : Created local file\n", t.Name())

	if err != nil {
		t.Fatalf("Failed to create local file for creating copies ...")
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU()/2) // Concurrency limiter
	log.Printf("(anushkadhn) Test: %s : Concurrency limit set to : %v", t.Name(), runtime.NumCPU()/2)

	for suffix := 1; suffix <= numberOfImplicitDirsInDirectoryWithTwelveThousandFiles; suffix++ {
		objectPath := path.Join(dirPathInBucket, fmt.Sprintf("%s%d", prefixImplicitDirInLargeDirListTest, suffix), testFile)

		wg.Add(1)
		go func(destinationPath string) {
			defer wg.Done()
			sem <- struct{}{}        // acquire semaphore
			defer func() { <-sem }() // release semaphore
			log.Printf("(anushkadhn) Test: %s : Copying file in bucket ....\n", t.Name())
			client.CopyFileInBucketWithPreconditions(ctx, storageClient, testFile, destinationPath, bucketName, &storage.Conditions{DoesNotExist: true})
			log.Printf("(anushkadhn) Test: %s : Copied file in bucket....\n", t.Name())
		}(objectPath)
	}

	wg.Wait()
	log.Printf("(anushkadhn) Test: %s : Done with creating all implicit dirs....\n", t.Name())
}

// testdataCreateExplicitDir creates explicit directories (trailing slash objects) in the bucket.
func testdataCreateExplicitDir(t *testing.T, ctx context.Context, storageClient *storage.Client, bucketNameWithDirPath string) {
	t.Helper()

	bucketName, dirPathInBucket := operations.SplitBucketNameAndDirPath(t, bucketNameWithDirPath)

	g, ctx := errgroup.WithContext(ctx)
	concurrencyLimit := runtime.NumCPU() / 2
	g.SetLimit(concurrencyLimit) // Concurrency limiter

	log.Printf("(anushkadhn) Test: %s : Concurrency limit set to : %v", t.Name(), concurrencyLimit)

	for dirIndex := 1; dirIndex <= numberOfExplicitDirsInDirectoryWithTwelveThousandFiles; dirIndex++ {
		capturedIndex := dirIndex
		g.Go(func() error {
			defer func() {
				log.Printf("(anushkadhn) Test: %s : Created GCS directory....\n", t.Name())
			}()
			dirName := fmt.Sprintf("%s%d", prefixExplicitDirInLargeDirListTest, capturedIndex)
			log.Printf("(anushkadhn) Test: %s : Started creating directory in gcs bucket....\n", t.Name())
			return client.CreateGcsDir(ctx, storageClient, dirName, bucketName, dirPathInBucket)
		})
	}

	if err := g.Wait(); err != nil {
		t.Fatalf("Failed to create explicit dirs: %v", err)
	}
	log.Printf("(anushkadhn) Test: %s : Done with creating all explicit dirs....\n", t.Name())
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

	start := time.Now()
	log.Printf("(anushkadhn) Test: %s : Starting to create files .... \n", t.Name())
	createFilesAndUpload(t, testDirPathOnBucket)
	log.Printf("(anushkadhn) Test: %s : Time taken to create files  : %v\n", t.Name(), time.Since(start))

	if withExplicitDirs {
		start := time.Now()
		log.Printf("(anushkadhn) Test: %s : Starting to create explicit dirs .... \n", t.Name())
		testdataCreateExplicitDir(t, ctx, storageClient, testDirPathOnBucket)
		log.Printf("(anushkadhn) Test: %s : Time taken to create explicit dirs  : %v\n", t.Name(), time.Since(start))
	}

	if withImplicitDirs {
		start := time.Now()
		log.Printf("(anushkadhn) Test: %s : Starting to create implicit dirs .... \n", t.Name())
		testdataCreateImplicitDir(t, ctx, storageClient, testDirPathOnBucket)
		log.Printf("(anushkadhn) Test: %s : Time taken to create implicit dirs  : %v\n", t.Name(), time.Since(start))
	}

	return testDirPath
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *listLargeDir) TestListDirectoryWithTwelveThousandFiles() {
	start := time.Now()
	log.Printf("(anushkadhn) Test: %s : Starting to prep test dir .... \n", t.T().Name())
	dirPath := prepareTestDirectory(t.T(), false, false)
	log.Printf("(anushkadhn) Test: %s : Time taken to prepare test directory  : %v\n", t.T().Name(), time.Since(start))

	start = time.Now()
	log.Printf("(anushkadhn) Test: %s : Starting to list test dir .... \n", t.T().Name())
	firstListTime, secondListTime := listDirTime(t.T(), dirPath, false, false)
	log.Printf("(anushkadhn) Test: %s : Time taken to list directory for : %v\n", t.T().Name(), time.Since(start))

	assert.Less(t.T(), secondListTime, firstListTime)
	assert.Less(t.T(), 2*secondListTime, firstListTime)
}

func (t *listLargeDir) TestListDirectoryWithTwelveThousandFilesAndHundredExplicitDir() {
	start := time.Now()
	log.Printf("(anushkadhn) Test: %s : Starting to prep test dir .... \n", t.T().Name())
	dirPath := prepareTestDirectory(t.T(), true, false)
	log.Printf("(anushkadhn) Test: %s : Time taken to prepare test directory  : %v\n", t.T().Name(), time.Since(start))

	start = time.Now()
	log.Printf("(anushkadhn) Test: %s : Starting to list test dir .... \n", t.T().Name())
	firstListTime, secondListTime := listDirTime(t.T(), dirPath, true, false)
	log.Printf("(anushkadhn) Test: %s : Time taken to list directory for : %v\n", t.T().Name(), time.Since(start))

	assert.Less(t.T(), secondListTime, firstListTime)
	assert.Less(t.T(), 2*secondListTime, firstListTime)
}

func (t *listLargeDir) TestListDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir() {
	start := time.Now()
	log.Printf("(anushkadhn) Test: %s : Starting to prep test dir .... \n", t.T().Name())
	dirPath := prepareTestDirectory(t.T(), true, true)
	log.Printf("(anushkadhn) Test: %s : Time taken to prepare test directory  : %v\n", t.T().Name(), time.Since(start))

	start = time.Now()
	log.Printf("(anushkadhn) Test: %s : Starting to list test dir .... \n", t.T().Name())
	firstListTime, secondListTime := listDirTime(t.T(), dirPath, true, true)
	log.Printf("(anushkadhn) Test: %s : Time taken to list directory for : %v\n", t.T().Name(), time.Since(start))

	assert.Less(t.T(), secondListTime, firstListTime)
	assert.Less(t.T(), 2*secondListTime, firstListTime)
}

////////////////////////////////////////////////////////////////////////
// Test Suite Function
////////////////////////////////////////////////////////////////////////

func TestListLargeDir(t *testing.T) {
	suite.Run(t, new(listLargeDir))
}
