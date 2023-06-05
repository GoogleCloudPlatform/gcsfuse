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
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

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

func createDirectoryStructureForTest(t *testing.T) {
	// Directory structure
	// testBucket/directoryForListTest                                                                      -- Dir
	// testBucket/directoryForListTest/fileInDirectoryForListTest		                                        -- File
	// testBucket/directoryForListTest/firstSubDirectoryForListTest                                         -- Dir
	// testBucket/directoryForListTest/firstSubDirectoryForListTest/fileInFirstSubDirectoryForListTest      -- File
	// testBucket/directoryForListTest/secondSubDirectoryForListTest                                        -- Dir
	// testBucket/directoryForListTest/secondSubDirectoryForListTest/fileInSecondSubDirectoryForListTest    -- File
	// testBucket/directoryForListTest/emptySubDirInDirectoryForListTest                                    -- Dir

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
}

func TestListDirectoryRecursively(t *testing.T) {
	// Clean the bucket for list testing.
	os.RemoveAll(setup.MntDir())

	// Create directory structure for testing.
	createDirectoryStructureForTest(t)

	// Recursively walk into directory and test.
	err := filepath.WalkDir(setup.MntDir(), func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		// The object type is not directory.
		if !dir.IsDir() {
			return nil
		}

		objs, err := os.ReadDir(path)
		if err != nil {
			log.Fatal(err)
		}

		// Check if mntDir has correct objects.
		if path == setup.MntDir() {
			// numberOfObjects - 1
			if len(objs) != NumberOfObjectsInBucketDirectoryListTest {
				t.Errorf("Incorrect number of objects in the bucket.")
			}

			// testBucket/directoryForListTest   -- Dir
			if objs[0].Name() != DirectoryForListTest || objs[0].IsDir() != true {
				t.Errorf("Listed incorrect object")
			}
		}

		// Check if directoryForListTest directory has correct data.
		if dir.IsDir() && dir.Name() == DirectoryForListTest {
			// numberOfObjects - 4
			if len(objs) != NumberOfObjectsInDirectoryForListTest {
				t.Errorf("Incorrect number of objects in the directoryForListTest.")
			}

			// testBucket/directoryForListTest/emptySubDirectoryForListTest   -- Dir
			if objs[0].Name() != EmptySubDirInDirectoryForListTest || objs[0].IsDir() != true {
				t.Errorf("Listed incorrect object")
			}

			// testBucket/directoryForListTest/fileInDirectoryForListTest     -- File
			if objs[1].Name() != FileInDirectoryForListTest || objs[1].IsDir() != false {
				t.Errorf("Listed incorrect object")
			}

			// testBucket/directoryForListTest/firstSubDirectoryForListTest   -- Dir
			if objs[2].Name() != FirstSubDirectoryForListTest || objs[2].IsDir() != true {
				t.Errorf("Listed incorrect object")
			}

			// testBucket/directoryForListTest/secondSubDirectoryForListTest  -- Dir
			if objs[3].Name() != SecondSubDirectoryForListTest || objs[3].IsDir() != true {
				t.Errorf("Listed incorrect object")
			}

			return nil
		}

		// Check if firstSubDirectoryForListTest directory has correct data.
		if dir.IsDir() && dir.Name() == FirstSubDirectoryForListTest {
			// numberOfObjects - 1
			if len(objs) != NumberOfObjectsInFirstSubDirectoryForListTest {
				t.Errorf("Incorrect number of objects in the firstSubDirectoryForListTest.")
			}

			// testBucket/directoryForListTest/firstSubDirectoryForListTest/fileInFirstSubDirectoryForListTest     -- File
			if objs[0].Name() != FileInFirstSubDirectoryForListTest || objs[0].IsDir() == true {
				t.Errorf("Listed incorrect object")
			}

			return nil
		}

		// Check if secondSubDirectoryForListTest directory has correct data.
		if dir.IsDir() && dir.Name() == SecondSubDirectoryForListTest {
			// numberOfObjects - 1
			if len(objs) != NumberOfObjectsInSecondSubDirectoryForListTest {
				t.Errorf("Incorrect number of objects in the secondSubDirectoryForListTest.")
			}

			// testBucket/directoryForListTest/secondSubDirectoryForListTest/fileInSecondSubDirectoryForListTest   -- File
			if objs[0].Name() != FileInSecondSubDirectoryForListTest || objs[0].IsDir() == true {
				t.Errorf("Listed incorrect object")
			}

			return nil
		}

		// Check if emptySubDirInDirectoryForListTest directory has correct data.
		if dir.IsDir() && dir.Name() == EmptySubDirInDirectoryForListTest {
			// numberOfObjects - 0
			if len(objs) != NumberOfObjectsInEmptySubDirInDirectoryForListTest {
				t.Errorf("Incorrect number of objects in the emptySubDirInDirectoryForListTest.")
			}

			return nil
		}

		return nil
	})
	if err != nil {
		t.Errorf("error walking the path : %v\n", err)
		return
	}

	os.RemoveAll(setup.MntDir())
}
