// Copyright 2023 Google Inc. All Rights Reserved.
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

package list_large_dir_test

import (
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func checkIfObjNameIsCorrect(objName string, prefix string, maxNumber int, t *testing.T) {
	// Extracting object number.
	objNumberStr := strings.ReplaceAll(objName, prefix, "")
	objNumber, err := strconv.Atoi(objNumberStr)
	if err != nil {
		t.Errorf("Error in extracting file number.")
	}
	if objNumber < 1 || objNumber > maxNumber {
		t.Errorf("Correct object does not exist.")
	}
}

func createTwelveThousandFilesAndUploadOnTestBucket(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// Creating twelve thousand files in DirectoryWithTwelveThousandFiles directory to upload them on a bucket for testing.
	localDirPath := path.Join(os.Getenv("HOME"), DirectoryWithTwelveThousandFiles)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInDirectoryWithTwelveThousandFiles, localDirPath, PrefixFileInDirectoryWithTwelveThousandFiles, t)

	// Uploading twelve thousand files to directoryWithTwelveThousandFiles in testBucket.
	dirPath := path.Join(setup.TestBucket(), DirectoryWithTwelveThousandFiles)
	setup.RunScriptForTestData("testdata/upload_files_to_bucket.sh", dirPath, DirectoryWithTwelveThousandFiles, PrefixFileInDirectoryWithTwelveThousandFiles)
}

// Create a hundred explicit directories.
func createHundredExplicitDir(dirPath string, t *testing.T) {
	// Create hundred explicit directories.
	for i := 1; i <= NumberOfExplicitDirsInDirectoryWithTwelveThousandFiles; i++ {
		subDirPath := path.Join(dirPath, PrefixExplicitDirInLargeDirListTest+strconv.Itoa(i))
		operations.CreateDirectoryWithNFiles(0, subDirPath, "", t)
	}
}

// Test with a bucket with twelve thousand files.
func TestListDirectoryWithTwelveThousandFiles(t *testing.T) {
	createTwelveThousandFilesAndUploadOnTestBucket(t)

	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)

	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	// number of objs - 12000
	if len(objs) != NumberOfFilesInDirectoryWithTwelveThousandFiles {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", len(objs))
	}

	// Checking if all the object is File type.
	for i := 0; i < len(objs); i++ {
		if objs[i].IsDir() {
			t.Errorf("Listed object is incorrect.")
		}
	}

	for i := 0; i < len(objs); i++ {
		checkIfObjNameIsCorrect(objs[i].Name(), PrefixFileInDirectoryWithTwelveThousandFiles, NumberOfFilesInDirectoryWithTwelveThousandFiles, t)
	}

	// Clear the bucket after testing.
	setup.RunScriptForTestData("testdata/delete_objects.sh", setup.TestBucket())
}

// Test with a bucket with twelve thousand files and hundred explicit directories.
func TestListDirectoryWithTwelveThousandFilesAndHundredExplicitDir(t *testing.T) {
	createTwelveThousandFilesAndUploadOnTestBucket(t)

	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)

	// Create hundred explicit directories.
	createHundredExplicitDir(dirPath, t)

	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	var numberOfFiles = 0
	var numberOfDirs = 0

	// Checking if correct objects present in bucket.
	for i := 0; i < len(objs); i++ {
		if !objs[i].IsDir() {
			numberOfFiles++
			// Checking if Prefix1 to Prefix12000 present in the bucket
			checkIfObjNameIsCorrect(objs[i].Name(), PrefixFileInDirectoryWithTwelveThousandFiles, NumberOfFilesInDirectoryWithTwelveThousandFiles, t)
		}

		if objs[i].IsDir() {
			numberOfDirs++
			// Checking if Prefix1 to Prefix100 present in the bucket
			checkIfObjNameIsCorrect(objs[i].Name(), PrefixExplicitDirInLargeDirListTest, NumberOfExplicitDirsInDirectoryWithTwelveThousandFiles, t)
		}
	}

	// number of explicit dirs = 100
	if numberOfDirs != NumberOfExplicitDirsInDirectoryWithTwelveThousandFiles {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 100", numberOfDirs)
	}
	// number of files = 12000
	if numberOfFiles != NumberOfFilesInDirectoryWithTwelveThousandFiles {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", numberOfFiles)
	}

	// Clear the bucket after testing.
	setup.RunScriptForTestData("testdata/delete_objects.sh", setup.TestBucket())
}

// Test with a bucket with twelve thousand files, hundred explicit directories, and hundred implicit directories.
func TestListDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir(t *testing.T) {
	createTwelveThousandFilesAndUploadOnTestBucket(t)

	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)

	// Create hundred explicit directories.
	createHundredExplicitDir(dirPath, t)

	subDirPath := path.Join(setup.TestBucket(), DirectoryWithTwelveThousandFiles)
	setup.RunScriptForTestData("testdata/create_implicit_dir.sh", subDirPath, PrefixImplicitDirInLargeDirListTest, strconv.Itoa(NumberOfImplicitDirsInDirectoryWithTwelveThousandFiles))

	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	var numberOfFiles = 0
	var numberOfDirs = 0

	// Checking if correct objects present in bucket.
	for i := 0; i < len(objs); i++ {
		if !objs[i].IsDir() {
			numberOfFiles++

			// Checking if Prefix1 to Prefix12000 present in the bucket
			checkIfObjNameIsCorrect(objs[i].Name(), PrefixFileInDirectoryWithTwelveThousandFiles, NumberOfFilesInDirectoryWithTwelveThousandFiles, t)
		}

		if objs[i].IsDir() {
			numberOfDirs++

			if strings.Contains(objs[i].Name(), PrefixExplicitDirInLargeDirListTest) {
				// Checking if explicitDir1 to explicitDir100 present in the bucket.
				checkIfObjNameIsCorrect(objs[i].Name(), PrefixExplicitDirInLargeDirListTest, NumberOfExplicitDirsInDirectoryWithTwelveThousandFiles, t)
			} else {
				// Checking if implicitDir1 to implicitDir100 present in the bucket.
				checkIfObjNameIsCorrect(objs[i].Name(), PrefixImplicitDirInLargeDirListTest, NumberOfImplicitDirsInDirectoryWithTwelveThousandFiles, t)
			}
		}
	}

	// number of dirs = 200(Number of implicit + Number of explicit directories)
	if numberOfDirs != NumberOfImplicitDirsInDirectoryWithTwelveThousandFiles+NumberOfExplicitDirsInDirectoryWithTwelveThousandFiles {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 200", numberOfDirs)
	}
	// number of files = 12000
	if numberOfFiles != NumberOfFilesInDirectoryWithTwelveThousandFiles {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", numberOfFiles)
	}

	// Clear the bucket after testing.
	setup.RunScriptForTestData("testdata/delete_objects.sh", setup.TestBucket())
}
