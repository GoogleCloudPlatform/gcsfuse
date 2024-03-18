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
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	TestDirForEmptyManagedFoldersTest = "EmptyManagedFoldersTest"
	NumberOfObjectsInDirForListTest   = 4
	EmptyManagedFolder1               = "emptyManagedFolder1"
	EmptyManagedFolder2               = "emptyManagedFolder2"
	SimulatedFolder                   = "simulatedFolder"
	File                              = "testFile"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type enableEmptyManagedFoldersTrue struct {
}

func (s *enableEmptyManagedFoldersTrue) Setup(t *testing.T) {
	setup.SetupTestDirectory(TestDirForEmptyManagedFoldersTest)
}

func (s *enableEmptyManagedFoldersTrue) Teardown(t *testing.T) {
	// Clean up test directory.
	bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(TestDirForEmptyManagedFoldersTest)
	operations.DeleteManagedFoldersInBucket(path.Join(testDir, EmptyManagedFolder1), setup.TestBucket(), t)
	operations.DeleteManagedFoldersInBucket(path.Join(testDir, EmptyManagedFolder2), setup.TestBucket(), t)
	setup.CleanupDirectoryOnGCS(path.Join(bucket, testDir))
}

////////////////////////////////////////////////////////////////////////
// Helper functions
////////////////////////////////////////////////////////////////////////

func createDirectoryStructureForEmptyManagedFoldersTest(t *testing.T) {
	// testBucket/EmptyManagedFoldersTest/managedFolder1
	// testBucket/EmptyManagedFoldersTest/managedFolder2
	// testBucket/EmptyManagedFoldersTest/simulatedFolder
	// testBucket/EmptyManagedFoldersTest/testFile
	bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(TestDirForEmptyManagedFoldersTest)
	operations.CreateManagedFoldersInBucket(path.Join(testDir, EmptyManagedFolder1), bucket, t)
	operations.CreateManagedFoldersInBucket(path.Join(testDir, EmptyManagedFolder2), bucket, t)
	operations.CreateDirectory(path.Join(setup.MntDir(), TestDirForEmptyManagedFoldersTest, SimulatedFolder), t)
	f := operations.CreateFile(path.Join(setup.MntDir(), TestDirForEmptyManagedFoldersTest, File), setup.FilePermission_0600, t)
	operations.CloseFile(f)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *enableEmptyManagedFoldersTrue) TestListDirectoryForEmptyManagedFolders(t *testing.T) {
	// Create directory structure for testing.
	createDirectoryStructureForEmptyManagedFoldersTest(t)

	// Recursively walk into directory and test.
	err := filepath.WalkDir(path.Join(setup.MntDir(), TestDirForEmptyManagedFoldersTest), func(path string, dir fs.DirEntry, err error) error {
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
		if dir.Name() == TestDirForEmptyManagedFoldersTest {
			// numberOfObjects - 4
			if len(objs) != NumberOfObjectsInDirForListTest {
				t.Errorf("Incorrect number of objects in the directory %s expected %d: got %d: ", dir.Name(), NumberOfObjectsInDirForListTest, len(objs))
			}

			// testBucket/managedFolderTest/emptyManagedFolder1   -- ManagedFolder1
			if objs[0].Name() != EmptyManagedFolder1 || objs[0].IsDir() != true {
				t.Errorf("Listed incorrect object expected %s: got %s: ", EmptyManagedFolder1, objs[0].Name())
			}

			// testBucket/managedFolderTest/emptyManagedFolder2     -- ManagedFolder2
			if objs[1].Name() != EmptyManagedFolder2 || objs[1].IsDir() != true {
				t.Errorf("Listed incorrect object expected %s: got %s: ", EmptyManagedFolder2, objs[1].Name())
			}

			// testBucket/managedFolderTest/simulatedFolder   -- SimulatedFolder
			if objs[2].Name() != SimulatedFolder || objs[2].IsDir() != true {
				t.Errorf("Listed incorrect object expected %s: got %s: ", SimulatedFolder, objs[2].Name())
			}

			// testBucket/managedFolderTest/testFile  -- File
			if objs[3].Name() != File || objs[3].IsDir() != false {
				t.Errorf("Listed incorrect object expected %s: got %s: ", File, objs[3].Name())
			}
			return nil
		}
		// Check if subDirectory is empty.
		if dir.Name() == EmptyManagedFolder1 || dir.Name() == EmptyManagedFolder2 || dir.Name() == SimulatedFolder {
			// numberOfObjects - 0
			if len(objs) != 0 {
				t.Errorf("Incorrect number of objects in the directory %s expected %d: got %d: ", dir.Name(), 0, len(objs))
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

	return mountConfig
}

// //////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
// //////////////////////////////////////////////////////////////////////
func TestEnableEmptyManagedFoldersTrue(t *testing.T) {
	ts := &enableEmptyManagedFoldersTrue{}

	// Run tests for mountedDirectory only if --mountedDirectory  and --testBucket flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	if setup.OnlyDirMounted() != "" {
		// Mount managed folder as only-dir mount
		operations.CreateManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
		// Delete managed folder resource after testing.
		defer operations.DeleteManagedFoldersInBucket(onlyDirMounted, setup.TestBucket(), t)
	}

	configFile := setup.YAMLConfigFile(
		getMountConfigForEmptyManagedFolders(),
		"config.yaml")
	flags := []string{"--implicit-dirs", "--config-file=" + configFile}

	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	defer func() {
		setup.SetMntDir(rootDir)
		setup.UnMountBucket()
	}()
	setup.SetMntDir(mountDir)

	// Run tests.
	log.Printf("Running tests with flags: %s", flags)
	test_setup.RunTests(t, ts)
}
