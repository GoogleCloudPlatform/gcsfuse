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
	"path"
	"path/filepath"
	"testing"

	"cloud.google.com/go/storage"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"golang.org/x/net/context"
)

const (
	testDirName = "ImplicitDirTest"
)

var (
	testDirPath   string
	storageClient *storage.Client
	ctx           context.Context
)

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func TestNewFileUnderImplicitDirectoryShouldNotGetSyncedToGCSTillClose(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	CreateImplicitDir(ctx, storageClient, testDirName, t)
	fileName := path.Join(ImplicitDirName, FileName1)

	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t)
	operations.WriteWithoutClose(fh, FileContents, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, fileName, t)

	// Validate.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, fileName, FileContents, t)
}

func TestReadDirForImplicitDirWithLocalFile(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	CreateImplicitDir(ctx, storageClient, testDirName, t)
	fileName1 := path.Join(ImplicitDirName, FileName1)
	fileName2 := path.Join(ImplicitDirName, FileName2)
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName1, t)
	_, fh2 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName2, t)

	// Attempt to list implicit directory.
	entries := operations.ReadDirectory(path.Join(testDirPath, ImplicitDirName), t)

	// Verify entries received successfully.
	operations.VerifyCountOfDirectoryEntries(3, len(entries), t)
	operations.VerifyFileEntry(entries[0], FileName1, 0, t)
	operations.VerifyFileEntry(entries[1], FileName2, 0, t)
	operations.VerifyFileEntry(entries[2], ImplicitFileName1, GCSFileSize, t)
	// Close the local files.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh1, testDirName, fileName1, "", t)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh2, testDirName, fileName2, "", t)
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
	// Create local file in mnt/ dir.
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	// Create explicit dir with 1 local file.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t)
	_, fh2 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName2, t)
	// Create implicit dir with 1 local file1 and 1 synced file.
	CreateImplicitDir(ctx, storageClient, testDirName, t)
	_, fh3 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName3, t)

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
				operations.VerifyCountOfDirectoryEntries(3, len(objs), t)
				operations.VerifyDirectoryEntry(objs[0], ExplicitDirName, t)
				operations.VerifyFileEntry(objs[1], FileName1, 0, t)
				operations.VerifyDirectoryEntry(objs[2], ImplicitDirName, t)
			}

			// Check if mntDir/explicitFoo/ has correct objects.
			if walkPath == path.Join(testDirPath, ExplicitDirName) {
				// numberOfObjects = 1
				operations.VerifyCountOfDirectoryEntries(1, len(objs), t)
				operations.VerifyFileEntry(objs[0], ExplicitFileName1, 0, t)
			}

			// Check if mntDir/implicitFoo/ has correct objects.
			if walkPath == path.Join(testDirPath, ImplicitDirName) {
				// numberOfObjects = 2
				operations.VerifyCountOfDirectoryEntries(2, len(objs), t)
				operations.VerifyFileEntry(objs[0], FileName2, 0, t)
				operations.VerifyFileEntry(objs[1], ImplicitFileName1, GCSFileSize, t)
			}
			return nil
		})

	// Validate and close the files.
	if err != nil {
		t.Errorf("filepath.WalkDir() err: %v", err)
	}
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh1, testDirName, FileName1, "", t)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh2, testDirName, fileName2, "", t)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh3, testDirName, fileName3, "", t)
}
