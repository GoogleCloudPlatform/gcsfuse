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

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup/clean_mount_dir"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

func TestListImplicitObjectsFromBucket(t *testing.T) {
	// Directory Structure
	// testBucket/implicitDirectory                                                  -- Dir
	// testBucket/implicitDirectory/fileInImplicitDir1                               -- File
	// testBucket/implicitDirectory/implicitSubDirectory                             -- Dir
	// testBucket/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File
	// testBucket/explicitDirectory                                                  -- Dir
	// testBucket/explicitFile                                                       -- File
	// testBucket/explicitDirectory/fileInExplicitDir1                               -- File
	// testBucket/explicitDirectory/fileInExplicitDir2                               -- File

	implicit_and_explicit_dir_setup.CreateImplicitDirectory()
	implicit_and_explicit_dir_setup.CreateExplicitDirectory(t)

	// Delete objects from mountedDirectory after testing.
	defer clean_mount_dir.CleanMntDir(setup.MntDir())

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
			// numberOfObjects - 3
			if len(objs) != implicit_and_explicit_dir_setup.NumberOfTotalObjects {
				t.Errorf("Incorrect number of objects in the bucket.")
			}

			// testBucket/explicitDir     -- Dir
			if objs[0].Name() != implicit_and_explicit_dir_setup.ExplicitDirectory || objs[0].IsDir() != true {
				t.Errorf("Listed incorrect object")
			}
			// testBucket/explicitFile    -- File
			if objs[1].Name() != implicit_and_explicit_dir_setup.ExplicitFile || objs[1].IsDir() != false {
				t.Errorf("Listed incorrect object")
			}

			// testBucket/implicitDir     -- Dir
			if objs[2].Name() != implicit_and_explicit_dir_setup.ImplicitDirectory || objs[2].IsDir() != true {
				t.Errorf("Listed incorrect object")
			}
		}

		// Check if explictDir directory has correct data.
		if dir.IsDir() && dir.Name() == implicit_and_explicit_dir_setup.ExplicitDirectory {
			// numberOfObjects - 2
			if len(objs) != implicit_and_explicit_dir_setup.NumberOfFilesInExplicitDirectory {
				t.Errorf("Incorrect number of objects in the explicitDirectory.")
			}

			// testBucket/explicitDir/fileInExplicitDir1   -- File
			if objs[0].Name() != implicit_and_explicit_dir_setup.FirstFileInExplicitDirectory || objs[0].IsDir() != false {
				t.Errorf("Listed incorrect object")
			}

			// testBucket/explicitDir/fileInExplicitDir2    -- File
			if objs[1].Name() != implicit_and_explicit_dir_setup.SecondFileInExplicitDirectory || objs[1].IsDir() != false {
				t.Errorf("Listed incorrect object")
			}
			return nil
		}

		// Check if implicitDir directory has correct data.
		if dir.IsDir() && dir.Name() == implicit_and_explicit_dir_setup.ImplicitDirectory {
			// numberOfObjects - 2
			if len(objs) != implicit_and_explicit_dir_setup.NumberOfFilesInImplicitDirectory {
				t.Errorf("Incorrect number of objects in the implicitDirectory.")
			}

			// testBucket/implicitDir/fileInImplicitDir1  -- File
			if objs[0].Name() != implicit_and_explicit_dir_setup.FileInImplicitDirectory || objs[0].IsDir() != false {
				t.Errorf("Listed incorrect object")
			}
			// testBucket/implicitDir/implicitSubDirectory  -- Dir
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

			// testBucket/implicitDir/implicitSubDir/fileInImplicitDir2   -- File
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
