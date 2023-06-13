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

func checkIfObjNameIsCorrect(objName string, prefix string, t *testing.T) (objNumber int) {
	// Extracting file number.
	objNumberStr := strings.ReplaceAll(objName, prefix, "")
	objNumber, err := strconv.Atoi(objNumberStr)
	if err != nil {
		t.Errorf("Error in extracting file number.")
	}
	return
}

func throwErrorForIncorrectFileNumber(fileNumber int, t *testing.T) {
	if fileNumber < 1 || fileNumber > 12000 {
		t.Errorf("Correct object does not exist.")
	}
}
func throwErrorForIncorrectDirNumber(dirNumber int, t *testing.T) {
	if dirNumber < 1 || dirNumber > 100 {
		t.Errorf("Correct object does not exist.")
	}
}

// Test with a bucket with twelve thousand files.
func TestDirectoryWithTwelveThousandFiles(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInDirectoryWithTwelveThousandFiles, dirPath, PrefixFileInDirectoryWithTwelveThousandFiles, t)

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
		fileNumber := checkIfObjNameIsCorrect(objs[i].Name(), PrefixFileInDirectoryWithTwelveThousandFiles, t)
		throwErrorForIncorrectFileNumber(fileNumber, t)
	}
}

// Test with a bucket with twelve thousand files and hundred explicit directories.
func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFilesAndHundredExplicitDir)
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
			fileNumber := checkIfObjNameIsCorrect(objs[i].Name(), PrefixFileInDirectoryWithTwelveThousandFilesAndHundredExplicitDir, t)
			throwErrorForIncorrectFileNumber(fileNumber, t)
		}

		if objs[i].IsDir() {
			numberOfDirs++
			// Checking if Prefix1 to Prefix100 present in the bucket
			dirNumber := checkIfObjNameIsCorrect(objs[i].Name(), ExplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDir, t)
			throwErrorForIncorrectDirNumber(dirNumber, t)
		}
	}

	// number of explicit dirs = 100
	if numberOfDirs != NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDir {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 100", numberOfDirs)
	}
	// number of files = 12000
	if numberOfFiles != NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDir {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", numberOfFiles)
	}
}

// Test with a bucket with twelve thousand files, hundred explicit directories, and hundred implicit directories.
func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir)
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
			fileNumber := checkIfObjNameIsCorrect(objs[i].Name(), PrefixFileInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir, t)
			throwErrorForIncorrectFileNumber(fileNumber, t)
		}

		if objs[i].IsDir() {
			numberOfDirs++

			// Checking if explicitDir1 to explicitDir100 present in the bucket.
			if strings.Contains(objs[i].Name(), ExplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir) {
				dirNumber := checkIfObjNameIsCorrect(objs[i].Name(), ExplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir, t)
				throwErrorForIncorrectDirNumber(dirNumber, t)
			} else {
				// Checking if implicitDir1 to implicitDir100 present in the bucket.
				dirNumber := checkIfObjNameIsCorrect(objs[i].Name(), ImplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir, t)
				throwErrorForIncorrectDirNumber(dirNumber, t)
			}
		}
	}

	// number of dirs = 200(Number of implicit + Number of explicit directories)
	if numberOfDirs != NumberOfImplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir+NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 200", numberOfDirs)
	}
	// number of files = 12000
	if numberOfFiles != NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", numberOfFiles)
	}
}
