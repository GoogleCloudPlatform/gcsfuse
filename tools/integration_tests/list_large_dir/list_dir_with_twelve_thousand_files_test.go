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
	"slices"
	"strconv"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func storeFileAndDirNameInArray(objs []os.DirEntry) (dirName []string, fileName []string) {
	for _, obj := range objs {
		if obj.IsDir() {
			dirName = append(dirName, obj.Name())
		} else {
			fileName = append(fileName, obj.Name())
		}
	}
	return
}
func TestDirectoryWithTwelveThousandFiles(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)

	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	// Checking number of objects in the bucket.
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
		// Checking if Prefix1 to Prefix12000 present in the bucket
		index := slices.Index(fileName, PrefixFileInDirectoryWithTwelveThousandFiles+strconv.Itoa(i+1))
		if index < 0 {
			t.Errorf("Correct object does not exist.")
		}
	}
}

func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFilesAndHundredExplicitDir)
	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	// Storing file and dir name.
	dirName, fileName := storeFileAndDirNameInArray(objs)

	if len(dirName) != NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDir {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 100", len(dirName))
	}
	if len(fileName) != NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDir {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", len(fileName))
	}

	// Checking if correct objects present in bucket.
	for i := 0; i < len(fileName); i++ {
		// Checking if Prefix1 to Prefix12000 present in the bucket
		index := slices.Index(fileName, PrefixFileInDirectoryWithTwelveThousandFilesAndHundredExplicitDir+strconv.Itoa(i+1))
		if index < 0 {
			t.Errorf("Correct object does not exist.")
		}
	}

	for i := 0; i < len(dirName); i++ {
		// Checking if Prefix1 to Prefix100 present in the bucket
		index := slices.Index(dirName, ExplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDir+strconv.Itoa(i+1))
		if index < 0 {
			t.Errorf("Correct object does not exist.")
		}
	}

}

func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir)
	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	// Storing file and dir name.
	dirName, fileName := storeFileAndDirNameInArray(objs)

	if len(dirName) != NumberOfImplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir+NumberOfExplicitDirsInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 100", len(dirName))
	}
	if len(fileName) != NumberOfFilesInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", len(fileName))
	}

	// Checking if correct objects present in bucket.
	for i := 0; i < len(fileName); i++ {
		// Checking if Prefix1 to Prefix12000 present in the bucket
		index := slices.Index(fileName, PrefixFileInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir+strconv.Itoa(i+1))
		if index < 0 {
			t.Errorf("Correct object does not exist.")
		}
	}

	for i := 0; i < (len(dirName) / 2); i++ {
		// Checking if explicitDir1 to explicitDir100 present in the bucket.
		index := slices.Index(dirName, ExplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir+strconv.Itoa(i+1))
		if index < 0 {
			t.Errorf("Correct object does not exist.")
		}

		// Checking if implicitDir1 to implicitDir100 present in the bucket.
		index = slices.Index(dirName, ImplicitDirInDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir+strconv.Itoa(i+1))
		if index < 0 {
			t.Errorf("Correct object does not exist.")
		}
	}
}
