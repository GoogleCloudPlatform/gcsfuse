// Copyright 2024 Google Inc. All Rights Reserved.
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

// Provides integration tests for listing empty managed folders.
package managed_folders

import (
	"fmt"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	testDirName                     = "EmptyManagedFoldersTest"
	NumberOfObjectsInDirForListTest = 4
	EmptyManagedFolder1             = "emptyManagedFolder1"
	EmptyManagedFolder2             = "emptyManagedFolder2"
	SimulatedFolder                 = "simulatedFolder"
	File                            = "testFile"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type enableEmptyManagedFoldersTrue struct {
	flags []string
}

func (s *enableEmptyManagedFoldersTrue) Setup(t *testing.T) {
	setup.SetupTestDirectory(testDirName)
}

func (s *enableEmptyManagedFoldersTrue) Teardown(t *testing.T) {
}

func createDirectoryStructureForTest(t *testing.T) {
	bucket := setup.TestBucket()
	testDir := testDirName
	setup.SetBucketAndObjectBasedOnTypeOfMount(&bucket, &testDir)

	operations.CreateManagedFoldersInTestDir(EmptyManagedFolder1, bucket, testDir, t)
	operations.CreateManagedFoldersInTestDir(EmptyManagedFolder2, bucket, testDir, t)
	operations.CreateDirectory(path.Join(setup.MntDir(), testDirName, SimulatedFolder), t)
	f := operations.CreateFile(path.Join(setup.MntDir(), testDirName, File), setup.FilePermission_0600, t)
	operations.CloseFile(f)
}

func (s *enableEmptyManagedFoldersTrue) TTestListDirectoryForEmptyManagedFolders(t *testing.T) {
	// Create directory structure for testing.
	createDirectoryStructureForTest(t)

	// Recursively walk into directory and test.
	err := filepath.WalkDir(path.Join(setup.MntDir(), testDirName), func(path string, dir fs.DirEntry, err error) error {
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
		// Check if managedFolderTest directory has correct data.
		if dir.Name() == testDirName {
			fmt.Println(dir.Name())
			// numberOfObjects - 4
			if len(objs) != NumberOfObjectsInDirForListTest {
				t.Errorf("Incorrect number of objects in the directory %s expectected %d: got %d: ", dir.Name(), NumberOfObjectsInDirForListTest, len(objs))
			}

			// testBucket/managedFolderTest/emptyManagedFolder1   -- ManagedFolder1
			if objs[0].Name() != EmptyManagedFolder1 || objs[0].IsDir() != true {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", EmptyManagedFolder1, objs[0].Name())
			}

			// testBucket/managedFolderTest/emptyManagedFolder2     -- ManagedFolder2
			if objs[1].Name() != EmptyManagedFolder2 || objs[1].IsDir() != true {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", EmptyManagedFolder2, objs[1].Name())
			}

			// testBucket/managedFolderTest/simulatedFolder   -- SimulatedFolder
			if objs[2].Name() != SimulatedFolder || objs[2].IsDir() != true {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", SimulatedFolder, objs[2].Name())
			}

			// testBucket/managedFolderTest/testFile  -- File
			if objs[3].Name() != File || objs[3].IsDir() != false {
				t.Errorf("Listed incorrect object expectected %s: got %s: ", File, objs[3].Name())
			}
			return nil
		}
		// Check if subDirectory is empty.
		if dir.Name() == EmptyManagedFolder1 || dir.Name() == EmptyManagedFolder2 || dir.Name() == SimulatedFolder {
			// numberOfObjects - 0
			if len(objs) != 0 {
				t.Errorf("Incorrect number of objects in the directory %s expectected %d: got %d: ", dir.Name(), 0, len(objs))
			}
		}

		return nil
	})
	if err != nil {
		t.Errorf("error walking the path : %v\n", err)
		return
	}
}

func getMountConfigForEmptyManagedFolders() config.MountConfig {
	mountConfig := config.MountConfig{
		ListConfig: config.ListConfig{
			EnableEmptyManagedFolders: true,
		},
		LogConfig: config.LogConfig{
			Severity:        config.TRACE,
			LogRotateConfig: config.DefaultLogRotateConfig(),
		},
	}

	fmt.Println(mountConfig)
	return mountConfig
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestEnableEmptyManagedFoldersTrue(t *testing.T) {
	ts := &enableEmptyManagedFoldersTrue{}

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Run tests for mountedDirectory only if --mountedDirectory  and --testBucket flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	configFile := setup.YAMLConfigFile(
		getMountConfigForEmptyManagedFolders(),
		"config.yaml")

	flagSet := [][]string{{"--implicit-dirs", "--config-file=" + configFile}}

	// Run tests.
	for _, flags := range flagSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
