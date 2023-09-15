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

package helpers

import (
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"golang.org/x/net/context"
)

const (
	FileName1                = "foo1"
	FileName2                = "foo2"
	FileName3                = "foo3"
	ExplicitDirName          = "explicit"
	ExplicitFileName1        = "explicitFile1"
	FilePerms                = 0644
	FileContents             = "teststring"
	GCSFileContent           = "gcsContent"
	NewFileName              = "newName"
	NewDirName               = "newDirName"
	SizeOfFileContents       = 10
	SizeOfGCSContent         = 10
	SizeTruncate             = 5
	LocalFileTestDirInBucket = "LocalFileTest"
	ReadSize                 = 1024
)

var StorageClient *storage.Client
var Ctx context.Context

func ValidateObjectNotFoundErrOnGCS(fileName string, t *testing.T) {
	_, err := client.ReadObjectFromGCS(
		StorageClient,
		path.Join(LocalFileTestDirInBucket, fileName),
		ReadSize,
		Ctx)
	if err == nil || !strings.Contains(err.Error(), "storage: object doesn't exist") {
		t.Fatalf("Incorrect error returned from GCS for file %s: %v", fileName, err)
	}
}

func ValidateObjectContents(fileName string, expectedContent string, t *testing.T) {
	gotContent, err := client.ReadObjectFromGCS(
		StorageClient,
		path.Join(LocalFileTestDirInBucket, fileName),
		ReadSize,
		Ctx)
	if err != nil {
		t.Fatalf("Error while reading synced local file from GCS, Err: %v", err)
	}

	if expectedContent != gotContent {
		t.Fatalf("GCS file %s content mismatch. Got: %s, Expected: %s ", fileName, gotContent, expectedContent)
	}
}

func CloseFileAndValidateObjectContents(f *os.File, fileName string, contents string, t *testing.T) {
	operations.CloseFile(f)
	ValidateObjectContents(fileName, contents, t)
}

func WritingToLocalFileShouldNotWriteToGCS(fh *os.File, fileName string, t *testing.T) {
	operations.WriteWithoutClose(fh, FileContents, t)
	ValidateObjectNotFoundErrOnGCS(fileName, t)
}

func NewFileShouldGetSyncedToGCSAtClose(testDirPath, fileName string, t *testing.T) {
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(testDirPath, fileName, t)

	// Writing contents to local file shouldn't create file on GCS.
	WritingToLocalFileShouldNotWriteToGCS(fh, fileName, t)

	// Close the file and validate if the file is created on GCS.
	CloseFileAndValidateObjectContents(fh, fileName, FileContents, t)
}

func ValidateNoFileOrDirError(path string, t *testing.T) {
	_, err := os.Stat(path)
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("os.Stat(%s). Expected: %s, Got: %v", path,
			"no such file or directory", err)
	}
}

func VerifyRenameOperationNotSupported(err error, t *testing.T) {
	if err == nil || !strings.Contains(err.Error(), "operation not supported") {
		t.Fatalf("os.Rename(), expected err: %s, got err: %v",
			"operation not supported", err)
	}
}

func CreateObjectInGCSTestDir(fileName, content string, t *testing.T) {
	objectName := path.Join(LocalFileTestDirInBucket, fileName)
	err := client.CreateObjectOnGCS(StorageClient, objectName, content, Ctx)
	if err != nil {
		t.Fatalf("Create Object %s on GCS: %v.", objectName, err)
	}
}

func CreateLocalFileInTestDir(testDirPath, fileName string, t *testing.T) (string, *os.File) {
	filePath := path.Join(testDirPath, fileName)
	fh := operations.CreateFile(filePath, FilePerms, t)
	ValidateObjectNotFoundErrOnGCS(fileName, t)
	return filePath, fh
}
