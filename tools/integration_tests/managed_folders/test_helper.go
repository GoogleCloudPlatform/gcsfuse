// Copyright 2024 Google LLC
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
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	control "cloud.google.com/go/storage/control/apiv2"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	TestDirForManagedFolderTest                           = "TestDirForManagedFolderTest"
	ViewPermission                                        = "objectViewer"
	ManagedFolder1                                        = "managedFolder1"
	ManagedFolder2                                        = "managedFolder2"
	SimulatedFolderNonEmptyManagedFoldersTest             = "simulatedFolderNonEmptyManagedFoldersTes"
	IAMRoleForViewPermission                              = "roles/storage.objectViewer"
	NumberOfObjectsInDirForNonEmptyManagedFoldersListTest = 4
	AdminPermission                                       = "objectAdmin"
	IAMRoleForAdminPermission                             = "roles/storage.objectAdmin"
)

var FileInNonEmptyManagedFoldersTest = "testFileInNonEmptyManagedFoldersTest" + setup.GenerateRandomString(5)

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
		t.Fatalf("Error in marshal the data into JSON format: %v", err)
	}

	f, err := os.CreateTemp(os.TempDir(), "iam-policy-*.json")
	if err != nil {
		t.Fatalf("error in creating file for iam-policy : %v", err)
	}
	defer operations.RemoveFile(f.Name())
	// Write the JSON to a FileInNonEmptyManagedFoldersTest
	_, err = f.Write(jsonData)
	if err != nil {
		t.Fatalf("Error in writing iam policy in json FileInNonEmptyManagedFoldersTest : %v", err)
	}

	gcloudProvidePermissionCmd := fmt.Sprintf("alpha storage managed-folders set-iam-policy gs://%s/%s %s", bucket, managedFolderPath, f.Name())
	_, err = operations.ExecuteGcloudCommand(gcloudProvidePermissionCmd)
	if err != nil {
		t.Fatalf("Error in providing permission to managed folder: %v", err)
	}
}

func revokePermissionToManagedFolder(bucket, managedFolderPath, serviceAccount, iamRole string, t *testing.T) {
	gcloudRevokePermissionCmd := fmt.Sprintf("alpha storage managed-folders remove-iam-policy-binding  gs://%s/%s --member=%s --role=%s", bucket, managedFolderPath, serviceAccount, iamRole)

	_, err := operations.ExecuteGcloudCommand(gcloudRevokePermissionCmd)
	if err != nil && !strings.Contains(err.Error(), "Policy binding with the specified principal, role, and condition not found!") && !strings.Contains(err.Error(), "The specified managed folder does not exist.") {
		t.Fatalf("Error in removing permission to managed folder: %v", err)
	}
}

func createDirectoryStructureForNonEmptyManagedFolders(ctx context.Context, storageClient *storage.Client, controlClient *control.StorageControlClient, t *testing.T) {
	// testBucket/NonEmptyManagedFoldersTest/managedFolder1
	// testBucket/NonEmptyManagedFoldersTest/managedFolder1/testFile
	// testBucket/NonEmptyManagedFoldersTest/managedFolder2
	// testBucket/NonEmptyManagedFoldersTest/managedFolder2/testFile
	// testBucket/NonEmptyManagedFoldersTest/SimulatedFolderNonEmptyManagedFoldersTest
	// testBucket/NonEmptyManagedFoldersTest/SimulatedFolderNonEmptyManagedFoldersTest/testFile
	// testBucket/NonEmptyManagedFoldersTest/testFile
	bucket, testDir := setup.GetBucketAndObjectBasedOnTypeOfMount(TestDirForManagedFolderTest)
	err := client.DeleteAllObjectsWithPrefix(ctx, storageClient, testDir)
	if err != nil {
		log.Fatalf("Failed to clean up test directory: %v", err)
	}
	f := operations.CreateFile(path.Join("/tmp", FileInNonEmptyManagedFoldersTest), setup.FilePermission_0600, t)
	defer operations.CloseFileShouldNotThrowError(t, f)
	managedFolder1 := path.Join(testDir, ManagedFolder1)
	managedFolder2 := path.Join(testDir, ManagedFolder2)
	simulatedFolderNonEmptyManagedFoldersTest := path.Join(testDir, SimulatedFolderNonEmptyManagedFoldersTest)

	client.CreateManagedFoldersInBucket(ctx, controlClient, path.Join(testDir, ManagedFolder1), bucket)
	client.CopyFileInBucket(ctx, storageClient, path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(managedFolder1, FileInNonEmptyManagedFoldersTest), bucket)
	client.CreateManagedFoldersInBucket(ctx, controlClient, path.Join(testDir, ManagedFolder2), bucket)
	client.CopyFileInBucket(ctx, storageClient, path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(managedFolder2, FileInNonEmptyManagedFoldersTest), bucket)
	client.CopyFileInBucket(ctx, storageClient, path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(simulatedFolderNonEmptyManagedFoldersTest, FileInNonEmptyManagedFoldersTest), bucket)
	client.CopyFileInBucket(ctx, storageClient, path.Join("/tmp", FileInNonEmptyManagedFoldersTest), path.Join(testDir, FileInNonEmptyManagedFoldersTest), bucket)
}

func cleanup(ctx context.Context, storageClient *storage.Client, controlClient *control.StorageControlClient, bucket, testDir, serviceAccount, iam_role string, t *testing.T) {
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder1), serviceAccount, iam_role, t)
	revokePermissionToManagedFolder(bucket, path.Join(testDir, ManagedFolder2), serviceAccount, iam_role, t)
	client.DeleteManagedFoldersInBucket(ctx, controlClient, path.Join(testDir, ManagedFolder1), setup.TestBucket())
	client.DeleteManagedFoldersInBucket(ctx, controlClient, path.Join(testDir, ManagedFolder2), setup.TestBucket())
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(bucket, testDir))
}

func listNonEmptyManagedFolders(t *testing.T) {
	// Recursively walk into directory and test.
	err := filepath.WalkDir(path.Join(setup.MntDir(), TestDirForManagedFolderTest), func(path string, dir fs.DirEntry, err error) error {
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
		if dir.Name() == TestDirForManagedFolderTest {
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

func copyDirAndCheckErrForViewPermission(src, dest string, t *testing.T) {
	err := operations.CopyDir(src, dest)
	if err == nil {
		t.Errorf(" Managed folder unexpectedly got copied with view only permission.")
	}

	operations.CheckErrorForReadOnlyFileSystem(t, err)
}

func copyObjectAndCheckErrForViewPermission(src, dest string, t *testing.T) {
	err := operations.CopyObject(src, dest)
	if err == nil {
		t.Errorf("Objects in managed folder unexpectedly got copied with view only permission.")
	}

	operations.CheckErrorForReadOnlyFileSystem(t, err)
}

func moveAndCheckErrForViewPermission(src, dest string, t *testing.T) {
	err := operations.Move(src, dest)
	if err == nil {
		t.Errorf("Objects in managed folder unexpectedly got moved with view only permission.")
	}

	operations.CheckErrorForReadOnlyFileSystem(t, err)
}

func createFileForTest(filePath string, t *testing.T) {
	file, err := os.Create(filePath)
	defer operations.CloseFileShouldNotThrowError(t, file)
	if err != nil {
		t.Errorf("Error in creating local file, %v", err)
	}
}
