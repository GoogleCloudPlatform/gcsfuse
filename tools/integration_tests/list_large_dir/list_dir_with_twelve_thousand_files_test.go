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

// Test with a bucket with twelve thousand files.
func TestDirectoryWithTwelveThousandFiles(t *testing.T) {
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
			t.Errorf("Listes object is incorrect.")
		}
	}

	for i := 0; i < len(objs); i++ {
		checkIfObjNameIsCorrect(objs[i].Name(), PrefixFileInDirectoryWithTwelveThousandFiles, NumberOfFilesInDirectoryWithTwelveThousandFiles, t)
	}
}

// Test with a bucket with twelve thousand files and hundred explicit directories.
func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)

	// Create hundred explicit directories.
	for i := 1; i <= 100; i++ {
		subDirPath := path.Join(dirPath, PrefixExplicitDirInLargeDirListTest+strconv.Itoa(i))
		// Create 100 Explicit directory.
		operations.CreateDirectoryWithNFiles(0, subDirPath, "", t)
	}

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
}

// Test with a bucket with twelve thousand files, hundred explicit directories, and hundred implicit directories.
func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)
	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	// Create hundred explicit directories.
	for i := 1; i <= 100; i++ {
		subDirPath := path.Join(dirPath, PrefixExplicitDirInLargeDirListTest+strconv.Itoa(i))
		// Check if directory exist in previous test.
		_, err := os.Stat(subDirPath)
		if err == nil {
			operations.CreateDirectoryWithNFiles(0, subDirPath, "", t)
		}
	}

	subDirPath := path.Join(setup.TestBucket(), DirectoryWithTwelveThousandFiles)
	setup.RunScriptForTestData("testdata/create_implicit_dir.sh", subDirPath)

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
}
