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
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

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

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func validateDirectory(t *testing.T, objs []os.DirEntry, expectExplicitDirs, expectImplicitDirs bool) {
	t.Helper()

	var numberOfFiles, numberOfExplicitDirs, numberOfImplicitDirs int

	for _, obj := range objs {
		if !obj.IsDir() {
			numberOfFiles++
			checkIfObjNameIsCorrect(t, obj.Name(), prefixFileInDirectoryWithTwelveThousandFiles, numberOfFilesInDirectoryWithTwelveThousandFiles)
		} else {
			if strings.Contains(obj.Name(), prefixExplicitDirInLargeDirListTest) {
				numberOfExplicitDirs++
				checkIfObjNameIsCorrect(t, obj.Name(), prefixExplicitDirInLargeDirListTest, numberOfExplicitDirsInDirectoryWithTwelveThousandFiles)
			} else {
				numberOfImplicitDirs++
				checkIfObjNameIsCorrect(t, obj.Name(), prefixImplicitDirInLargeDirListTest, numberOfImplicitDirsInDirectoryWithTwelveThousandFiles)
			}
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

func createFilesAndUpload(t *testing.T, dirPath string) {
	t.Helper()

	operations.CreateDirectoryWithNFiles(numberOfFilesInDirectoryWithTwelveThousandFiles, dirPath, prefixFileInDirectoryWithTwelveThousandFiles, t)

	dirPathOnBucket := path.Join(setup.TestBucket(), directoryForListLargeFileTests, t.Name())
	setup.RunScriptForTestData("testdata/upload_files_to_bucket.sh", dirPathOnBucket, t.Name(), prefixFileInDirectoryWithTwelveThousandFiles)
}

func createExplicitDirs(t *testing.T, dirPath string) {
	t.Helper()

	for i := 1; i <= numberOfExplicitDirsInDirectoryWithTwelveThousandFiles; i++ {
		subDirPath := path.Join(dirPath, fmt.Sprintf("%s%d", prefixExplicitDirInLargeDirListTest, i))
		operations.CreateDirectoryWithNFiles(0, subDirPath, "", t)
	}
}

func listDirTime(t *testing.T, dirPath string, validateDir func(*testing.T, []os.DirEntry, bool, bool)) (time.Duration, time.Duration) { // Modified signature
	t.Helper()

	startTime := time.Now()
	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Fatalf("Error listing directory: %v", err)
	}
	endTime := time.Now()

	// Determine if explicit and implicit dirs are expected based on the test name
	expectExplicitDirs := strings.Contains(t.Name(), "HundredExplicitDir")
	expectImplicitDirs := strings.Contains(t.Name(), "HundredImplicitDir")

	validateDir(t, objs, expectExplicitDirs, expectImplicitDirs) // Pass the flags here
	firstListTime := endTime.Sub(startTime)

	minSecondListTime := time.Duration(math.MaxInt64)
	for i := 0; i < 5; i++ {
		startTime = time.Now()
		objs, err = os.ReadDir(dirPath)
		if err != nil {
			t.Fatalf("Error listing directory: %v", err)
		}
		endTime = time.Now()
		validateDir(t, objs, expectExplicitDirs, expectImplicitDirs) // Pass the flags here
		secondListTime := endTime.Sub(startTime)
		if secondListTime < minSecondListTime {
			minSecondListTime = secondListTime
		}
	}
	return firstListTime, minSecondListTime
}

func shouldRecreateDirectory(t *testing.T, objs []os.DirEntry, err error, withExplicitDirs, withImplicitDirs bool) bool {
	t.Helper()

	if os.IsNotExist(err) { // Explicitly check if directory doesn't exist
		return true
	}

	if len(objs) == 0 { // Directory is empty
		return true
	}

	if countFiles(objs) != numberOfFilesInDirectoryWithTwelveThousandFiles {
		return true
	}

	if withExplicitDirs && countExplicitDirs(objs) != numberOfExplicitDirsInDirectoryWithTwelveThousandFiles {
		return true
	}

	if withImplicitDirs && countImplicitDirs(objs) != numberOfImplicitDirsInDirectoryWithTwelveThousandFiles {
		return true
	}

	return false // Directory structure is valid
}

func prepareTestDirectory(t *testing.T, withExplicitDirs bool, withImplicitDirs bool) string {
	t.Helper()

	testDirName := t.Name()
	testDirPathOnBucket := path.Join(setup.TestBucket(), directoryForListLargeFileTests, testDirName)
	testDirPath := path.Join(setup.MntDir(), directoryForListLargeFileTests, testDirName)

	objs, err := os.ReadDir(testDirPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Error reading directory: %v", err)
	}

	if shouldRecreateDirectory(t, objs, err, withExplicitDirs, withImplicitDirs) {
		t.Logf("Recreating test directory: %s", testDirPathOnBucket)

		// Create the directory if it doesn't exist.
		if os.IsNotExist(err) {
			err := os.MkdirAll(testDirPath, 0755)
			if err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
		}

		// Safeguard: Only remove if it exists and is not empty
		if len(objs) > 0 {
			err = os.RemoveAll(testDirPath)
			if err != nil {
				t.Fatalf("Failed to remove directory: %v", err)
			}
		}

		createFilesAndUpload(t, testDirPath)

		if withExplicitDirs {
			createExplicitDirs(t, testDirPath)
		}

		if withImplicitDirs {
			subDirPath := path.Join(testDirPathOnBucket, testDirName)
			setup.RunScriptForTestData("testdata/create_implicit_dir.sh", subDirPath, prefixImplicitDirInLargeDirListTest, strconv.Itoa(numberOfImplicitDirsInDirectoryWithTwelveThousandFiles))
		}
	} else {
		t.Logf("Using existing test directory: %s", testDirPathOnBucket)
	}

	return testDirPath
}

func countExplicitDirs(objs []os.DirEntry) int {
	count := 0
	for _, obj := range objs {
		if obj.IsDir() && strings.HasPrefix(obj.Name(), prefixExplicitDirInLargeDirListTest) {
			count++
		}
	}
	return count
}

func countImplicitDirs(objs []os.DirEntry) int {
	count := 0
	for _, obj := range objs {
		if obj.IsDir() && strings.HasPrefix(obj.Name(), prefixImplicitDirInLargeDirListTest) {
			count++
		}
	}
	return count
}

func countFiles(objs []os.DirEntry) int {
	count := 0
	for _, obj := range objs {
		if !obj.IsDir() {
			count++
		}
	}
	return count
}

// SetupSuite runs once before all tests.
func (t *listLargeDir) SetupSuite() {
	_, err := os.Stat(path.Join(setup.MntDir(), directoryForListLargeFileTests))
	if os.IsNotExist(err) {
		err = os.Mkdir(path.Join(setup.MntDir(), directoryForListLargeFileTests), 0755)
		if err != nil {
			t.T().Fatalf("Failed to create directory: %v", err)
		}
	}
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

// Test with a bucket with twelve thousand files.
func (t *listLargeDir) TestListDirectoryWithTwelveThousandFiles() {
	dirPath := prepareTestDirectory(t.T(), false, false)
	firstListTime, secondListTime := listDirTime(t.T(), dirPath, validateDirectory) // No need to wrap validateDirectory

	assert.Less(t.T(), secondListTime, firstListTime)
	assert.Less(t.T(), 2*secondListTime, firstListTime)
}

// Test with a bucket with twelve thousand files and hundred explicit directories.
func (t *listLargeDir) TestListDirectoryWithTwelveThousandFilesAndHundredExplicitDir() {
	dirPath := prepareTestDirectory(t.T(), true, false)
	firstListTime, secondListTime := listDirTime(t.T(), dirPath, validateDirectory) // No need to wrap validateDirectory

	assert.Less(t.T(), secondListTime, firstListTime)
	assert.Less(t.T(), 2*secondListTime, firstListTime)
}

// Test with a bucket with twelve thousand files, hundred explicit directories, and hundred implicit directories.
func (t *listLargeDir) TestListDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir() {
	dirPath := prepareTestDirectory(t.T(), true, true)
	firstListTime, secondListTime := listDirTime(t.T(), dirPath, validateDirectory) // No need to wrap validateDirectory

	assert.Less(t.T(), secondListTime, firstListTime)
	assert.Less(t.T(), 2*secondListTime, firstListTime)
}

////////////////////////////////////////////////////////////////////////
// Test Suite Function
////////////////////////////////////////////////////////////////////////

func TestListLargeDir(t *testing.T) {
	suite.Run(t, new(listLargeDir))
}
