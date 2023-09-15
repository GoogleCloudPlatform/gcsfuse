// Copyright 2023 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package implicit_dir_test

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"golang.org/x/net/context"
)

const (
	FileName1         = "foo1"
	FileName2         = "foo2"
	ImplicitDirName   = "implicit"
	ImplicitFileName1 = "implicitFile1"
	ExplicitDirName   = "explicit"
	ExplicitFileName1 = "explicitFile1"
	FilePerms         = 0644
	FileContents      = "teststring"
	ReadSize          = 1024
	testDirName       = "ImplicitDirTest"
)

var (
	testDirPath   string
	storageClient *storage.Client
	ctx           context.Context
)

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func createImplicitDir(t *testing.T) {
	err := client.CreateObjectOnGCS(
		storageClient,
		path.Join(testDirName, ImplicitDirName, ImplicitFileName1),
		FileContents,
		ctx)
	if err != nil {
		t.Errorf("Error while creating implicit directory, err: %v", err)
	}
}

func validateObjectNotFoundErrOnGCS(fileName string, t *testing.T) {
	_, err := client.ReadObjectFromGCS(storageClient, path.Join(testDirName, fileName), ReadSize, ctx)
	if err == nil || !strings.Contains(err.Error(), "storage: object doesn't exist") {
		t.Fatalf("Incorrect error returned from GCS for file %s: %v", fileName, err)
	}
}

func validateObjectContents(fileName string, expectedContent string, t *testing.T) {
	gotContent, err := client.ReadObjectFromGCS(storageClient, path.Join(testDirName, fileName), ReadSize, ctx)
	if err != nil {
		t.Fatalf("Error while reading synced local file from GCS, Err: %v", err)
	}

	if expectedContent != gotContent {
		t.Fatalf("GCS file %s content mismatch. Got: %s, Expected: %s ", fileName, gotContent, expectedContent)
	}
}

func closeFileAndValidateContent(fh *os.File, fileName, content string, t *testing.T) {
	operations.CloseFile(fh)
	validateObjectContents(fileName, content, t)
}

func createLocalFile(filePath, fileName string, t *testing.T) (fh *os.File) {
	fh = operations.CreateFile(filePath, FilePerms, t)
	validateObjectNotFoundErrOnGCS(fileName, t)
	return
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func TestNewFileUnderImplicitDirectoryShouldNotGetSyncedToGCSTillClose(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	createImplicitDir(t)
	fileName := path.Join(ImplicitDirName, FileName1)
	filePath := path.Join(testDirPath, fileName)

	fh := createLocalFile(filePath, fileName, t)
	operations.WriteWithoutClose(fh, FileContents, t)
	validateObjectNotFoundErrOnGCS(fileName, t)

	// Validate.
	closeFileAndValidateContent(fh, fileName, FileContents, t)
}

func TestReadDirForImplicitDirWithLocalFile(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	createImplicitDir(t)
	fileName1 := path.Join(ImplicitDirName, FileName1)
	fileName2 := path.Join(ImplicitDirName, FileName2)
	filePath1 := path.Join(testDirPath, fileName1)
	filePath2 := path.Join(testDirPath, fileName2)
	fh1 := createLocalFile(filePath1, fileName1, t)
	fh2 := createLocalFile(filePath2, fileName2, t)

	// Attempt to list implicit directory.
	entries := operations.ReadDirectory(path.Join(testDirPath, ImplicitDirName), t)

	// Verify entries received successfully.
	operations.VerifyCountOfEntries(3, len(entries), t)
	operations.VerifyFileEntry(entries[0], FileName1, 0, t)
	operations.VerifyFileEntry(entries[1], FileName2, 0, t)
	operations.VerifyFileEntry(entries[2], ImplicitFileName1, 10, t)
	// Close the local files.
	closeFileAndValidateContent(fh1, fileName1, "", t)
	closeFileAndValidateContent(fh2, fileName2, "", t)
}

func TestRecursiveListingWithLocalFiles(t *testing.T) {
	// Structure
	// mntDir/
	// mntDir/foo1 										--- file
	// mntDir/explicit/		    				--- directory
	// mntDir/explicit/explicitFile1  --- file
	// mntDir/implicit/ 							--- directory
	// mntDir/implicit/foo2  					--- file
	// mntDir/implicit/implicitFile1	--- file

	testDirPath = setup.SetupTestDirectory(testDirName)

	fileName2 := path.Join(ExplicitDirName, ExplicitFileName1)
	fileName3 := path.Join(ImplicitDirName, FileName2)
	filePath1 := path.Join(testDirPath, FileName1)
	filePath2 := path.Join(testDirPath, fileName2)
	filePath3 := path.Join(testDirPath, fileName3)

	// Create local file in mnt/ dir.
	fh1 := createLocalFile(filePath1, FileName1, t)
	// Create explicit dir with 1 local file.
	operations.CreateExplicitDir(path.Join(testDirPath, ExplicitDirName), t)
	fh2 := createLocalFile(filePath2, fileName2, t)
	// Create implicit dir with 1 local file1 and 1 synced file.
	createImplicitDir(t)
	fh3 := createLocalFile(filePath3, fileName3, t)

	// Recursively list mntDir/ directory.
	err := filepath.WalkDir(testDirPath,
		func(walkPath string, dir fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// The object type is not directory.
			if !dir.IsDir() {
				return nil
			}

			objs := operations.ReadDirectory(walkPath, t)

			// Check if mntDir has correct objects.
			if walkPath == setup.MntDir() {
				// numberOfObjects = 3
				operations.VerifyCountOfEntries(3, len(objs), t)
				operations.VerifyDirectoryEntry(objs[0], ExplicitDirName, t)
				operations.VerifyFileEntry(objs[1], FileName1, 0, t)
				operations.VerifyDirectoryEntry(objs[2], ImplicitDirName, t)
			}

			// Check if mntDir/explicitFoo/ has correct objects.
			if walkPath == path.Join(testDirPath, ExplicitDirName) {
				// numberOfObjects = 1
				operations.VerifyCountOfEntries(1, len(objs), t)
				operations.VerifyFileEntry(objs[0], ExplicitFileName1, 0, t)
			}

			// Check if mntDir/implicitFoo/ has correct objects.
			if walkPath == path.Join(testDirPath, ImplicitDirName) {
				// numberOfObjects = 2
				operations.VerifyCountOfEntries(2, len(objs), t)
				operations.VerifyFileEntry(objs[0], FileName2, 0, t)
				operations.VerifyFileEntry(objs[1], ImplicitFileName1, 10, t)
			}
			return nil
		})

	// Validate and close the files.
	if err != nil {
		t.Errorf("filepath.WalkDir() err: %v", err)
	}
	closeFileAndValidateContent(fh1, FileName1, "", t)
	closeFileAndValidateContent(fh2, fileName2, "", t)
	closeFileAndValidateContent(fh3, fileName3, "", t)
}
