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

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func createDirectoryStructureForTest(t *testing.T) {
	// Directory structure
	// testBucket/directoryForListTest                                                                            -- Dir
	// testBucket/directoryForListTest/fileInDirectoryForListTest1		                                      -- File
	// testBucket/directoryForListTest/firstSubDirectoryForListTest                                               -- Dir
	// testBucket/directoryForListTest/firstSubDirectoryForListTest/fileInFirstSubDirectoryForListTest1           -- File
	// testBucket/directoryForListTest/secondSubDirectoryForListTest                                              -- Dir
	// testBucket/directoryForListTest/secondSubDirectoryForListTest/fileInSecondSubDirectoryForListTest1         -- File
	// testBucket/directoryForListTest/secondSubDirectoryForListTest/fileInSecondSubDirectoryForListTest2         -- File
	// testBucket/directoryForListTest/emptySubDirInDirectoryForListTest                                          -- Dir

	// testBucket/directoryForListTest
	// testBucket/directoryForListTest/fileInFirstSubDirectoryForListTest1
	dirPath := path.Join(setup.MntDir(), DirectoryForListTest)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInDirectoryForListTest, dirPath, PrefixFileInDirectoryForListTest, t)

	// testBucket/directoryForListTest/firstSubDirectoryForListTest
	// testBucket/directoryForListTest/firstSubDirectoryForListTest/fileInFirstSubDirectoryForListTest1
	subDirPath := path.Join(dirPath, FirstSubDirectoryForListTest)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInFirstSubDirectoryForListTest, subDirPath, PrefixFileInFirstSubDirectoryForListTest, t)

	// testBucket/directoryForListTest/secondSubDirectoryForListTest
	// testBucket/directoryForListTest/secondSubDirectoryForListTest/fileInSecondSubDirectoryForListTest1
	// testBucket/directoryForListTest/secondSubDirectoryForListTest/fileInSecondSubDirectoryForListTest2
	subDirPath = path.Join(dirPath, SecondSubDirectoryForListTest)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInSecondSubDirectoryForListTest, subDirPath, PrefixFileInSecondSubDirectoryForListTest, t)

	// testBucket/directoryForListTest/emptySubDirInDirectoryForListTest
	subDirPath = path.Join(dirPath, EmptySubDirInDirectoryForListTest)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInEmptySubDirInDirectoryForListTest, subDirPath, "", t)
}

func TestListDirectoryRecursively(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

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

			// testBucket/directoryForListTest/fileInDirectoryForListTest1     -- File
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

			// testBucket/directoryForListTest/firstSubDirectoryForListTest/fileInFirstSubDirectoryForListTest1     -- File
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

			// testBucket/directoryForListTest/secondSubDirectoryForListTest/fileInSecondSubDirectoryForListTest1    -- File
			if objs[0].Name() != FirstFileInSecondSubDirectoryForListTest || objs[0].IsDir() == true {
				t.Errorf("Listed incorrect object")
			}

			// testBucket/directoryForListTest/secondSubDirectoryForListTest/fileInSecondSubDirectoryForListTest2   -- File
			if objs[1].Name() != SecondFileInSecondSubDirectoryForListTest || objs[1].IsDir() == true {
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
}
