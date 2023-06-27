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

// Provides integration tests for create three level directories.
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

func TestCreateThreeLevelDirectories(t *testing.T) {
	// Directory structure
	// testBucket/dirOneInCreateThreeLevelDirTest                                                                       -- Dir
	// testBucket/dirOneInCreateThreeLevelDirTest/dirTwoInCreateThreeLevelDirTest                                       -- Dir
	// testBucket/dirOneInCreateThreeLevelDirTest/dirTwoInCreateThreeLevelDirTest/dirThreeInCreateThreeLevelDirTest     -- Dir
	// testBucket/dirOneInCreateThreeLevelDirTest/dirTwoInCreateThreeLevelDirTest/dirThreeInCreateThreeLevelDirTest/fileInDirThreeInCreateThreeLevelDirTest     -- File

	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	dirPath := path.Join(setup.MntDir(), DirOneInCreateThreeLevelDirTest)

	operations.CreateDirectoryWithNFiles(0, dirPath, "", t)

	subDirPath := path.Join(dirPath, DirTwoInCreateThreeLevelDirTest)

	operations.CreateDirectoryWithNFiles(0, subDirPath, "", t)

	subDirPath2 := path.Join(subDirPath, DirThreeInCreateThreeLevelDirTest)

	operations.CreateDirectoryWithNFiles(1, subDirPath2, PrefixFileInDirThreeInCreateThreeLevelDirTest, t)
	filePath := path.Join(subDirPath2, FileInDirThreeInCreateThreeLevelDirTest)
	err := operations.WriteFileInAppendMode(filePath, ContentInFileInDirThreeInCreateThreeLevelDirTest)
	if err != nil {
		t.Errorf("Write file error: %v", err)
	}

	// Recursively walk into directory and test.
	err = filepath.WalkDir(setup.MntDir(), func(dirPath string, dir fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dirPath, err)
			return err
		}

		// The object type is not directory.
		if !dir.IsDir() {
			return nil
		}

		objs, err := os.ReadDir(dirPath)
		if err != nil {
			log.Fatal(err)
		}

		// Check if mntDir has correct objects.
		if dirPath == setup.MntDir() {
			// numberOfObjects - 1
			if len(objs) != NumberOfObjectsInBucketDirectoryCreateTest {
				t.Errorf("Incorrect number of objects in the bucket.")
			}

			// testBucket/dirOneInCreateThreeLevelDirTest   -- Dir
			if objs[0].Name() != DirOneInCreateThreeLevelDirTest || objs[0].IsDir() != true {
				t.Errorf("Directory is not created.")
			}
		}

		// Check if dirOneInCreateThreeLevelDirTest directory has correct data.
		if dir.IsDir() && dir.Name() == DirOneInCreateThreeLevelDirTest {
			// numberOfObjects - 1
			if len(objs) != NumberOfObjectsInDirOneInCreateThreeLevelDirTest {
				t.Errorf("Incorrect number of objects in the dirOneInCreateThreeLevelDirTest.")
			}

			// testBucket/dirOneInCreateThreeLevelDirTest/dirTwoInCreateThreeLevelDirTest    -- Dir
			if objs[0].Name() != DirTwoInCreateThreeLevelDirTest || objs[0].IsDir() != true {
				t.Errorf("Directory is not created.")
			}
			return nil
		}

		// Check if dirTwoInCreateThreeLevelDirTest directory has correct data.
		if dir.IsDir() && dir.Name() == DirTwoInCreateThreeLevelDirTest {
			// numberOfObjects - 1
			if len(objs) != NumberOfObjectsInDirTwoInCreateThreeLevelDirTest {
				t.Errorf("Incorrect number of objects in the dirTwoInCreateThreeLevelDirTest.")
			}

			// testBucket/dirOneInCreateThreeLevelDirTest/dirTwoInCreateThreeLevelDirTest/dirThreeInCreateThreeLevelDirTest    -- Dir
			if objs[0].Name() != DirThreeInCreateThreeLevelDirTest || objs[0].IsDir() != true {
				t.Errorf("Directory is not created.")
			}
			return nil
		}

		// Check if dirThreeInCreateThreeLevelDirTest directory has correct data.
		if dir.IsDir() && dir.Name() == DirThreeInCreateThreeLevelDirTest {
			// numberOfObjects - 1
			if len(objs) != NumberOfObjectsInDirThreeInCreateThreeLevelDirTest {
				t.Errorf("Incorrect number of objects in the dirThreeInCreateThreeLevelDirTest.")
			}

			// testBucket/dirOneInCreateThreeLevelDirTest/dirTwoInCreateThreeLevelDirTest/dirThreeInCreateThreeLevelDirTest/fileInDirThreeInCreateThreeLevelDirTest     -- File
			if objs[0].Name() != FileInDirThreeInCreateThreeLevelDirTest || objs[0].IsDir() != false {
				t.Errorf("Incorrect object exist in the dirThreeInCreateThreeLevelDirTest directory.")
			}

			// Check if the content of the file is correct.
			filePath := path.Join(dirPath, objs[0].Name())
			content, err := operations.ReadFile(filePath)
			if err != nil {
				t.Errorf("Error in reading file:%v", err)
			}

			if got, want := string(content), ContentInFileInDirThreeInCreateThreeLevelDirTest; got != want {
				t.Errorf("File content %q not match %q", got, want)
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
