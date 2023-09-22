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

package client

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
)

const (
	FileName1          = "foo1"
	FileName2          = "foo2"
	ExplicitDirName    = "explicit"
	ExplicitFileName1  = "explicitFile1"
	ImplicitDirName    = "implicit"
	ImplicitFileName1  = "implicitFile1"
	FileContents       = "testString"
	SizeOfFileContents = 10
	GCSFileContent     = "GCSteststring"
	GCSFileSize        = 13
	FilePerms          = 0644
	ReadSize           = 1024
	SizeTruncate       = 5
	NewFileName        = "newName"
	NewDirName         = "newDirName"
)

func CreateImplicitDir(ctx context.Context, storageClient *storage.Client,
	testDirName string, t *testing.T) {
	err := CreateObjectOnGCS(
		ctx,
		storageClient,
		path.Join(testDirName, ImplicitDirName, ImplicitFileName1),
		GCSFileContent)
	if err != nil {
		t.Errorf("Error while creating implicit directory, err: %v", err)
	}
}

func ValidateObjectNotFoundErrOnGCS(ctx context.Context, storageClient *storage.Client,
	testDirName string, fileName string, t *testing.T) {
	_, err := ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, fileName), ReadSize)
	if err == nil || !strings.Contains(err.Error(), "storage: object doesn't exist") {
		t.Fatalf("Incorrect error returned from GCS for file %s: %v", fileName, err)
	}
}

func ValidateObjectContentsFromGCS(ctx context.Context, storageClient *storage.Client,
	testDirName string, fileName string, expectedContent string, t *testing.T) {
	gotContent, err := ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, fileName), ReadSize)
	if err != nil {
		t.Fatalf("Error while reading synced local file from GCS, Err: %v", err)
	}

	if expectedContent != gotContent {
		t.Fatalf("GCS file %s content mismatch. Got: %s, Expected: %s ", fileName, gotContent, expectedContent)
	}
}

func CloseFileAndValidateContentFromGCS(ctx context.Context, storageClient *storage.Client,
	fh *os.File, testDirName, fileName, content string, t *testing.T) {
	operations.CloseFileShouldNotThrowError(fh, t)
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, fileName, content, t)
}

func CreateLocalFileInTestDir(ctx context.Context, storageClient *storage.Client,
	testDirPath, fileName string, t *testing.T) (string, *os.File) {
	filePath := path.Join(testDirPath, fileName)
	fh := operations.CreateFile(filePath, FilePerms, t)
	testDirName := GetDirName(testDirPath)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, fileName, t)
	return filePath, fh
}

func GetDirName(testDirPath string) string {
	dirName := testDirPath[strings.LastIndex(testDirPath, "/")+1:]
	return dirName
}

func CreateObjectInGCSTestDir(ctx context.Context, storageClient *storage.Client,
	testDirName, fileName, content string, t *testing.T) {
	objectName := path.Join(testDirName, fileName)
	err := CreateObjectOnGCS(ctx, storageClient, objectName, content)
	if err != nil {
		t.Fatalf("Create Object %s on GCS: %v.", objectName, err)
	}
}
