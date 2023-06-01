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

// Provides integration tests for list directory.
package operations_test

import (
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func getDirNameFromPath(dirPath string) (dirName string) {
	if strings.Contains(dirPath, EmptySubDirInDirectoryForListTest) {
		// testBucket/directoryForListTest/emptySubDirectoryForListTest
		dirName = EmptySubDirInDirectoryForListTest
	} else if strings.Contains(dirPath, SecondSubDirectoryForListTest) {
		// testBucket/directoryForListTest/secondSubDirectoryForListTest
		dirName = SecondSubDirectoryForListTest
	} else if strings.Contains(dirPath, FirstSubDirectoryForListTest) {
		// testBucket/directoryForListTest/firstSubDirectoryForListTest
		dirName = FirstSubDirectoryForListTest
	} else if strings.Contains(dirPath, DirectoryForListTest) {
		// testBucket/directoryForListTest
		dirName = DirectoryForListTest
	} else if strings.Contains(dirPath, setup.MntDir()) {
		// testBucket/
		dirName = setup.MntDir()
	}

	return dirName
}
func createDirectoryWithFile(dirPath string, filePath string, t *testing.T) {
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", dirPath, err)
		return
	}

	_, err = os.Create(filePath)
	if err != nil {
		t.Errorf("Error in creating file %v:", err)
	}
}

func checkIfListedCorrectDirectory(dirPath string, obj fs.DirEntry, t *testing.T) {
	dirName := getDirNameFromPath(dirPath)

	switch dirName {
	case setup.MntDir():
		{
			// testBucket/directoryForListTest    -- Dir
			if obj.Name() != DirectoryForListTest || obj.IsDir() != true {
				t.Errorf("Listed incorrect object.")
			}
		}
	case DirectoryForListTest:
		{
			// testBucket/directoryForListTest/fileInDirectoryForListTest     -- File
			// testBucket/directoryForListTest/firstSubDirectoryForListTest   -- Dir
			// testBucket/directoryForListTest/secondSubDirectoryForListTest  -- Dir
			// testBucket/directoryForListTest/emptySubDirectoryForListTest   -- Dir
			if (obj.Name() != FileInDirectoryForListTest && obj.IsDir() == true) && (obj.Name() != FirstSubDirectoryForListTest && obj.IsDir() != true) && (obj.Name() != SecondSubDirectoryForListTest && obj.IsDir() != true) && (obj.Name() != EmptySubDirInDirectoryForListTest && obj.IsDir() != true) {
				t.Errorf("Listed incorrect object")
			}
		}
	case FirstSubDirectoryForListTest:
		{
			// testBucket/directoryForListTest/firstSubDirectoryForListTest/fileInFirstSubDirectoryForListTest     -- File
			if obj.Name() != FileInFirstSubDirectoryForListTest && obj.IsDir() == true {
				t.Errorf("Listed incorrect object")
			}
		}
	case SecondSubDirectoryForListTest:
		{
			// testBucket/directoryForListTest/secondSubDirectoryForListTest/fileInSecondSubDirectoryForListTest   -- File
			if obj.Name() != FileInSecondSubDirectoryForListTest && obj.IsDir() == true {
				t.Errorf("Listed incorrect object")
			}
		}
	}
}

func checkIfListedDirectoryHasCorrectNumberOfObjects(dirPath string, numberOfObjects int, t *testing.T) {
	dirName := getDirNameFromPath(dirPath)

	switch dirName {
	case setup.MntDir():
		{
			// numberOfObjects - 1
			if numberOfObjects != NumberOfObjectsInBucketDirectoryListTest {
				t.Errorf("Incorrect number of objects in the bucket directory.")
			}
		}
	case DirectoryForListTest:
		{
			// numberOfObjects - 4
			if numberOfObjects != NumberOfObjectsInDirectoryForListTest {
				t.Errorf("Incorrect number of objects in the directoryForListTest.")
			}
		}
	case FirstSubDirectoryForListTest:
		{
			// numberOfObjects - 1
			if numberOfObjects != NumberOfObjectsInFirstSubDirectoryForListTest {
				t.Errorf("Incorrect number of objects in the fileInDirectoryForListTest.")
			}
		}
	case SecondSubDirectoryForListTest:
		{
			// numberOfObjects - 1
			if numberOfObjects != NumberOfObjectsInSecondSubDirectoryForListTest {
				t.Errorf("Incorrect number of objects in the secondSubDirectoryForListTest.")
			}
		}
	case EmptySubDirInDirectoryForListTest:
		{
			// numberOfObjects - 0
			if numberOfObjects != NumberOfObjectsInEmptySubDirInDirectoryForListTest {
				t.Errorf("Incorrect number of objects in the emptySubDirInDirectoryForListTest.")
			}
		}
	}
}

// List directory recursively
func listDirectory(path string, t *testing.T) {
	//Reading contents of the directory
	objs, err := os.ReadDir(path)

	if err != nil {
		log.Fatal(err)
	}

	checkIfListedDirectoryHasCorrectNumberOfObjects(path, len(objs), t)

	for _, obj := range objs {
		checkIfListedCorrectDirectory(path, obj, t)
		if obj.IsDir() {
			subDirectoryPath := filepath.Join(path, obj.Name()) // path of the subdirectory
			listDirectory(subDirectoryPath, t)                  // calling listFiles() again for subdirectory
		}
	}
}

func TestListDirectoryRecursively(t *testing.T) {
	// Clean the bucket for list testing.
	os.RemoveAll(setup.MntDir())

	// Directory structure
	// testBucket/directoryForListTest                                                                     -- Dir
	// testBucket/directoryForListTest/fileInDirectoryForListTest		                                       -- File
	// testBucket/directoryForListTest/firstSubDirectoryForListTest                                        -- Dir
	// testBucket/directoryForListTest/firstSubDirectoryForListTest/fileInFirstSubDirectoryForListTest     -- File
	// testBucket/directoryForListTest/secondSubDirectoryForListTest                                       -- Dir
	// testBucket/directoryForListTest/secondSubDirectoryForListTest/fileInSecondSubDirectoryForListTest   -- File
	// testBucket/directoryForListTest/emptySubDirInDirectoryForListTest                                   -- Dir

	// testBucket/directoryForListTest
	// testBucket/directoryForListTest/fileInDirectoryForListTest
	dirPath := path.Join(setup.MntDir(), DirectoryForListTest)
	filePath := path.Join(dirPath, FileInDirectoryForListTest)
	createDirectoryWithFile(dirPath, filePath, t)

	// testBucket/directoryForListTest/firstSubDirectoryForListTest
	// testBucket/directoryForListTest/firstSubDirectoryForListTest/fileInFirstSubDirectoryForListTest
	subDirPath := path.Join(dirPath, FirstSubDirectoryForListTest)
	subDirFilePath := path.Join(subDirPath, FileInFirstSubDirectoryForListTest)
	createDirectoryWithFile(subDirPath, subDirFilePath, t)

	// testBucket/directoryForListTest/secondSubDirectoryForListTest
	// testBucket/directoryForListTest/secondSubDirectoryForListTest/fileInSecondSubDirectoryForListTest
	subDirPath = path.Join(dirPath, SecondSubDirectoryForListTest)
	subDirFilePath = path.Join(subDirPath, FileInSecondSubDirectoryForListTest)
	createDirectoryWithFile(subDirPath, subDirFilePath, t)

	// testBucket/directoryForListTest/emptySubDirInDirectoryForListTest
	subDirPath = path.Join(dirPath, EmptySubDirInDirectoryForListTest)
	err := os.Mkdir(subDirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", subDirPath, err)
		return
	}

	// Test directory listing recursively.
	listDirectory(setup.MntDir(), t)

	// Clean the bucket after list testing.
	os.RemoveAll(setup.MntDir())
}
