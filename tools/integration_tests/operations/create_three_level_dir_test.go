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

	os.RemoveAll(setup.MntDir())

	dirPath := path.Join(setup.MntDir(), DirOneInCreateThreeLevelDirTest)

	operations.CreateDirectoryWithNFiles(0, dirPath, "", t)

	subDirPath := path.Join(dirPath, DirTwoInCreateThreeLevelDirTest)

	operations.CreateDirectoryWithNFiles(0, subDirPath, "", t)

	subDirPath2 := path.Join(subDirPath, DirThreeInCreateThreeLevelDirTest)

	operations.CreateDirectoryWithNFiles(0, subDirPath2, "", t)

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
				t.Errorf("Incorrect number of objects in the directoryForListTest.")
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
				t.Errorf("Incorrect number of objects in the firstSubDirectoryForListTest.")
			}

			// testBucket/dirOneInCreateThreeLevelDirTest/dirTwoInCreateThreeLevelDirTest/dirThreeInCreateThreeLevelDirTest    -- Dir
			if objs[0].Name() != DirThreeInCreateThreeLevelDirTest || objs[0].IsDir() != true {
				t.Errorf("Directory is not created.")
			}
			return nil
		}

		// Check if dirThreeInCreateThreeLevelDirTest directory has correct data.
		if dir.IsDir() && dir.Name() == DirThreeInCreateThreeLevelDirTest {
			// numberOfObjects - 0
			if len(objs) != NumberOfObjectsInDirThreeInCreateThreeLevelDirTest {
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
