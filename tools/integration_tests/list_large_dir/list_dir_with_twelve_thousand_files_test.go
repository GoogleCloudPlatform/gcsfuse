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
	"sort"
	"strconv"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func differentiateFileAndDirName(objs []os.DirEntry) (dirName []string, fileName []string) {
	for _, obj := range objs {
		if obj.IsDir() {
			dirName = append(dirName, obj.Name())
		} else {
			fileName = append(fileName, obj.Name())
		}
	}
	return
}

func findIfCorrectObjectExistInList(objs []string, obj string, t *testing.T) {
	index := sort.SearchStrings(objs, obj)
	if objs[index] != obj {
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

	// Checking if correct file present in bucket.
	var fileName []string
	for i := 0; i < len(objs); i++ {
		fileName = append(fileName, objs[i].Name())
	}

	for i := 0; i < len(objs); i++ {
		findIfCorrectObjectExistInList(fileName, PrefixFileInDirectoryWithTwelveThousandFiles+strconv.Itoa(i+1), t)
	}
}

// Test with a bucket with twelve thousand files and hundred explicit directories.
func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFilesAndHundredExplicitDir)
	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	// Differentiate file and dir name.
	dirName, fileName := differentiateFileAndDirName(objs)

	// number of explicit dirs = 100
	if len(dirName) != NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDir {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 100", len(dirName))
	}
	// number of files = 12000
	if len(fileName) != NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDir {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", len(fileName))
	}

	// Checking if correct objects present in bucket.
	for i := 0; i < len(fileName); i++ {
		// Checking if Prefix1 to Prefix12000 present in the bucket
		findIfCorrectObjectExistInList(fileName, PrefixFileInDirectoryWithTwelveThousandFilesAndHundredExplicitDir+strconv.Itoa(i+1), t)
	}

	for i := 0; i < len(dirName); i++ {
		// Checking if Prefix1 to Prefix100 present in the bucket
		findIfCorrectObjectExistInList(dirName, ExplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDir+strconv.Itoa(i+1), t)
	}
}

// Test with a bucket with twelve thousand files, hundred explicit directories, and hundred implicit directories.
func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir)
	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	// Differentiate file and dir name.
	dirName, fileName := differentiateFileAndDirName(objs)

	// number of dirs = 200(Number of implicit + Number of explicit directories)
	if len(dirName) != NumberOfImplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir+NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 200", len(dirName))
	}
	// number of files = 12000
	if len(fileName) != NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", len(fileName))
	}

	// Checking if correct objects present in bucket.
	for i := 0; i < len(fileName); i++ {
		// Checking if Prefix1 to Prefix12000 present in the bucket
		findIfCorrectObjectExistInList(fileName, PrefixFileInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir+strconv.Itoa(i+1), t)
	}

	for i := 0; i < (len(dirName) / 2); i++ {
		// Checking if explicitDir1 to explicitDir100 present in the bucket.
		findIfCorrectObjectExistInList(dirName, ExplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir+strconv.Itoa(i+1), t)
		// Checking if implicitDir1 to implicitDir100 present in the bucket.
		findIfCorrectObjectExistInList(dirName, ImplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir+strconv.Itoa(i+1), t)
	}
}
