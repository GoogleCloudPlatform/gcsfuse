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

package implicit_dir_test

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

func TestListImplicitObjectsFromBucket(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForImplicitDirTests)
	// Directory Structure
	// testBucket/dirForImplicitDirTests/implicitDirectory                                                  -- Dir
	// testBucket/dirForImplicitDirTests/implicitDirectory/fileInImplicitDir1                               -- File
	// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory                             -- Dir
	// testBucket/dirForImplicitDirTests/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File
	// testBucket/dirForImplicitDirTests/explicitDirectory                                                  -- Dir
	// testBucket/dirForImplicitDirTests/explicitFile                                                       -- File
	// testBucket/dirForImplicitDirTests/explicitDirectory/fileInExplicitDir1                               -- File
	// testBucket/dirForImplicitDirTests/explicitDirectory/fileInExplicitDir2                               -- File
	// testBucket/dirForImplicitDirTests/..                                                                 -- Dir
	// testBucket/dirForImplicitDirTests/../fileInUnsupportedImplicitDir1                                   -- File

	implicit_and_explicit_dir_setup.CreateImplicitDirectoryStructure(DirForImplicitDirTests)
	implicit_and_explicit_dir_setup.CreateExplicitDirectoryStructure(DirForImplicitDirTests, t)
	implicit_and_explicit_dir_setup.CreateUnsupportedImplicitDirectoryStructure(DirForImplicitDirTests)

	err := filepath.WalkDir(testDir, func(path string, dir fs.DirEntry, err error) error {
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
		if path == testDir {
			// numberOfObjects - 3
			if len(objs) != implicit_and_explicit_dir_setup.NumberOfTotalObjects {
				t.Fatalf("Incorrect number of objects in the bucket: actual=%v, expected=%v.", len(objs), implicit_and_explicit_dir_setup.NumberOfTotalObjects)
			}

			// testBucket/dirForImplicitDirTests/explicitDir     -- Dir
			if objs[0].Name() != implicit_and_explicit_dir_setup.ExplicitDirectory || objs[0].IsDir() != true {
				t.Errorf("Listed incorrect object")
			}
			// testBucket/dirForImplicitDirTests/explicitFile    -- File
			if objs[1].Name() != implicit_and_explicit_dir_setup.ExplicitFile || objs[1].IsDir() != false {
				t.Errorf("Listed incorrect object")
			}

			// testBucket/dirForImplicitDirTests/implicitDir     -- Dir
			if objs[2].Name() != implicit_and_explicit_dir_setup.ImplicitDirectory || objs[2].IsDir() != true {
				t.Errorf("Listed incorrect object")
			}
		}

		// Check if explictDir directory has correct data.
		if dir.IsDir() && dir.Name() == implicit_and_explicit_dir_setup.ExplicitDirectory {
			// numberOfObjects - 2
			if len(objs) != implicit_and_explicit_dir_setup.NumberOfFilesInExplicitDirectory {
				t.Fatalf("Incorrect number of objects in the explicitDirectory: actual=%v,expected:%v", len(objs), implicit_and_explicit_dir_setup.NumberOfFilesInExplicitDirectory)
			}

			// testBucket/dirForImplicitDirTests/explicitDir/fileInExplicitDir1   -- File
			if objs[0].Name() != implicit_and_explicit_dir_setup.FirstFileInExplicitDirectory || objs[0].IsDir() != false {
				t.Errorf("Listed incorrect object")
			}

			// testBucket/dirForImplicitDirTests/explicitDir/fileInExplicitDir2    -- File
			if objs[1].Name() != implicit_and_explicit_dir_setup.SecondFileInExplicitDirectory || objs[1].IsDir() != false {
				t.Errorf("Listed incorrect object")
			}
			return nil
		}

		// Check if implicitDir directory has correct data.
		if dir.IsDir() && dir.Name() == implicit_and_explicit_dir_setup.ImplicitDirectory {
			// numberOfObjects - 2
			if len(objs) != implicit_and_explicit_dir_setup.NumberOfFilesInImplicitDirectory {
				t.Fatalf("Incorrect number of objects in the implicitDirectory: actual=%v,expected=%v", len(objs), implicit_and_explicit_dir_setup.NumberOfFilesInImplicitDirectory)
			}

			// testBucket/dirForImplicitDirTests/implicitDir/fileInImplicitDir1  -- File
			if objs[0].Name() != implicit_and_explicit_dir_setup.FileInImplicitDirectory || objs[0].IsDir() != false {
				t.Errorf("Listed incorrect object")
			}
			// testBucket/dirForImplicitDirTests/implicitDir/implicitSubDirectory  -- Dir
			if objs[1].Name() != implicit_and_explicit_dir_setup.ImplicitSubDirectory || objs[1].IsDir() != true {
				t.Errorf("Listed incorrect object")
			}
			return nil
		}

		// Check if implicit sub directory has correct data.
		if dir.IsDir() && dir.Name() == implicit_and_explicit_dir_setup.ImplicitSubDirectory {
			// numberOfObjects - 1
			if len(objs) != implicit_and_explicit_dir_setup.NumberOfFilesInImplicitSubDirectory {
				t.Errorf("Incorrect number of objects in the implicitSubDirectoryt.")
			}

			// testBucket/dirForImplicitDirTests/implicitDir/implicitSubDir/fileInImplicitDir2   -- File
			if objs[0].Name() != implicit_and_explicit_dir_setup.FileInImplicitSubDirectory || objs[0].IsDir() != false {
				t.Errorf("Listed incorrect object")
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
