// Copyright 2024 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this FileInNonEmptyManagedFoldersTest  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package managed_folders

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	testDirNameForNonEmptyManagedFolder                   = "NonEmptyManagedFoldersTest"
	ViewPermission                                        = "objectViewer"
	ManagedFolder1                                        = "managedFolder1"
	ManagedFolder2                                        = "managedFolder2"
	SimulatedFolderNonEmptyManagedFoldersTest             = "simulatedFolderNonEmptyManagedFoldersTes"
	FileInNonEmptyManagedFoldersTest                      = "testFileInNonEmptyManagedFoldersTest"
	IAMRoleForViewPermission                              = "roles/storage.objectViewer"
	NumberOfObjectsInDirForNonEmptyManagedFoldersListTest = 4
	AdminPermission                                       = "objectAdmin"
	IAMRoleForAdminPermission                             = "roles/storage.objectAdmin"
)

type IAMPolicy struct {
	Bindings []struct {
		Role    string   `json:"role"`
		Members []string `json:"members"`
	} `json:"bindings"`
}

func providePermissionToManagedFolder(bucket, managedFolderPath, serviceAccount, iamRole string, t *testing.T) {
	policy := IAMPolicy{
		Bindings: []struct {
			Role    string   `json:"role"`
			Members []string `json:"members"`
		}{
			{
				Role: iamRole,
				Members: []string{
					"serviceAccount:" + serviceAccount,
				},
			},
		},
	}

	// Marshal the data into JSON format
	// Indent for readability
	jsonData, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		t.Fatalf(fmt.Sprintf("Error in marshal the data into JSON format: %v", err))
	}

	localIAMPolicyFilePath := path.Join(os.Getenv("HOME"), "iam_policy.json")
	// Write the JSON to a FileInNonEmptyManagedFoldersTest
	err = os.WriteFile(localIAMPolicyFilePath, jsonData, setup.FilePermission_0600)
	if err != nil {
		t.Fatalf(fmt.Sprintf("Error in writing iam policy in json FileInNonEmptyManagedFoldersTest : %v", err))
	}

	gcloudProvidePermissionCmd := fmt.Sprintf("alpha storage managed-folders set-iam-policy gs://%s/%s %s", bucket, managedFolderPath, localIAMPolicyFilePath)
	_, err = operations.ExecuteGcloudCommandf(gcloudProvidePermissionCmd)
	if err != nil {
		t.Fatalf(fmt.Sprintf("Error in providing permission to managed folder: %v", err))
	}
}

func revokePermissionToManagedFolder(bucket, managedFolderPath, serviceAccount, iamRole string, t *testing.T) {
	gcloudRevokePermissionCmd := fmt.Sprintf("alpha storage managed-folders remove-iam-policy-binding  gs://%s/%s --member=%s --role=%s", bucket, managedFolderPath, serviceAccount, iamRole)

	_, err := operations.ExecuteGcloudCommandf(gcloudRevokePermissionCmd)
	if err != nil && !strings.Contains(err.Error(), "Policy binding with the specified principal, role, and condition not found!") {
		t.Fatalf(fmt.Sprintf("Error in providing permission to managed folder: %v", err))
	}
}

func createDirectoryStructureForNonEmptyManagedFolders(t *testing.T) {
	// testBucket/NonEmptyManagedFoldersTest/managedFolder1
	// testBucket/NonEmptyManagedFoldersTest/managedFolder1/testFile
	// testBucket/NonEmptyManagedFoldersTest/managedFolder2
	// testBucket/NonEmptyManagedFoldersTest/managedFolder2/testFile
	// testBucket/NonEmptyManagedFoldersTest/SimulatedFolderNonEmptyManagedFoldersTest
	// testBucket/NonEmptyManagedFoldersTest/SimulatedFolderNonEmptyManagedFoldersTest/testFile
	// testBucket/NonEmptyManagedFoldersTest/testFile
	bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(testDirNameForNonEmptyManagedFolder)
	operations.CreateManagedFoldersInBucket(path.Join(testDir, ManagedFolder1), bucket, t)
	f := operations.CreateFile(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), setup.FilePermission_0600, t)
	defer operations.CloseFile(f)
	operations.CopyFileInBucket(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(testDir, ManagedFolder1), bucket, t)
	operations.CreateManagedFoldersInBucket(path.Join(testDir, ManagedFolder2), bucket, t)
	operations.CopyFileInBucket(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(testDir, ManagedFolder2), bucket, t)
	operations.CopyFileInBucket(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(testDir, SimulatedFolderNonEmptyManagedFoldersTest), bucket, t)
	operations.CopyFileInBucket(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), testDir, bucket, t)
}

func cleanup(bucket, testDir, serviceAccount, iam_role string, t *testing.T) {
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, iam_role, t)
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, iam_role, t)
	operations.DeleteManagedFoldersInBucket(path.Join(testDir, ManagedFolder1), setup.TestBucket(), t)
	operations.DeleteManagedFoldersInBucket(path.Join(testDir, ManagedFolder2), setup.TestBucket(), t)
	setup.CleanupDirectoryOnGCS(path.Join(bucket, testDir))
}

func listNonEmptyManagedFolders(t *testing.T) {
	// Recursively walk into directory and test.
	err := filepath.WalkDir(path.Join(setup.MntDir(), testDirNameForNonEmptyManagedFolder), func(path string, dir fs.DirEntry, err error) error {
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
		if dir.Name() == testDirNameForNonEmptyManagedFolder {
			// numberOfObjects - 4
			if len(objs) != NumberOfObjectsInDirForNonEmptyManagedFoldersListTest {
				t.Errorf("Incorrect number of objects in the directory %s expected %d: got %d: ", dir.Name(), NumberOfObjectsInDirForNonEmptyManagedFoldersListTest, len(objs))
			}

			// testBucket/NonEmptyManagedFoldersTest/managedFolder1  -- ManagedFolder1
			if objs[0].Name() != ManagedFolder1 || !objs[0].IsDir() {
				t.Errorf("Listed incorrect object expected %s: got %s: ", ManagedFolder1, objs[0].Name())
			}

			// testBucket/NonEmptyManagedFoldersTest/managedFolder2     -- ManagedFolder2
			if objs[1].Name() != ManagedFolder2 || !objs[1].IsDir() {
				t.Errorf("Listed incorrect object expected %s: got %s: ", ManagedFolder2, objs[1].Name())
			}

			// testBucket/NonEmptyManagedFoldersTest/SimulatedFolderNonEmptyManagedFoldersTest   -- SimulatedFolderNonEmptyManagedFoldersTest
			if objs[2].Name() != SimulatedFolderNonEmptyManagedFoldersTest || !objs[2].IsDir() {
				t.Errorf("Listed incorrect object expected %s: got %s: ", SimulatedFolderNonEmptyManagedFoldersTest, objs[2].Name())
			}

			// testBucket/NonEmptyManagedFoldersTest/testFile  -- FileInNonEmptyManagedFoldersTest
			if objs[3].Name() != FileInNonEmptyManagedFoldersTest || objs[3].IsDir() {
				t.Errorf("Listed incorrect object expected %s: got %s: ", FileInNonEmptyManagedFoldersTest, objs[3].Name())
			}
			return nil
		}
		// Check if subDirectory is empty.
		if dir.Name() == ManagedFolder1 {
			// numberOfObjects - 1
			if len(objs) != 1 {
				t.Errorf("Incorrect number of objects in the directory %s expected %d: got %d: ", dir.Name(), 1, len(objs))
			}
			// testBucket/NonEmptyManagedFoldersTest/managedFolder1/testFile  -- FileInNonEmptyManagedFoldersTest
			if objs[0].Name() != FileInNonEmptyManagedFoldersTest || objs[0].IsDir() {
				t.Errorf("Listed incorrect object expected %s: got %s: ", FileInNonEmptyManagedFoldersTest, objs[3].Name())
			}
		}
		// Ensure subDirectory is not empty.
		if dir.Name() == ManagedFolder2 {
			// numberOfObjects - 1
			if len(objs) != 1 {
				t.Errorf("Incorrect number of objects in the directory %s expected %d: got %d: ", dir.Name(), 1, len(objs))
			}
			// testBucket/NonEmptyManagedFoldersTest/managedFolder2/testFile  -- FileInNonEmptyManagedFoldersTest
			if objs[0].Name() != FileInNonEmptyManagedFoldersTest || objs[0].IsDir() {
				t.Errorf("Listed incorrect object expected %s: got %s: ", FileInNonEmptyManagedFoldersTest, objs[3].Name())
			}
		}
		// Check if subDirectory is empty.
		if dir.Name() == SimulatedFolderNonEmptyManagedFoldersTest {
			// numberOfObjects - 1
			if len(objs) != 1 {
				t.Errorf("Incorrect number of objects in the directory %s expected %d: got %d: ", dir.Name(), 1, len(objs))
			}

			// testBucket/NonEmptyManagedFoldersTest/SimulatedFolderNonEmptyManagedFoldersTest/testFile  -- FileInNonEmptyManagedFoldersTest
			if objs[0].Name() != FileInNonEmptyManagedFoldersTest || objs[0].IsDir() {
				t.Errorf("Listed incorrect object expected %s: got %s: ", FileInNonEmptyManagedFoldersTest, objs[3].Name())
			}
		}
		return nil
	})
	if err != nil {
		t.Errorf("error walking the path : %v\n", err)
		return
	}
}
