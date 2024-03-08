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
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	FileName1          = "foo1"
	FileName2          = "foo2"
	FileName3          = "foo3"
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
	_, err := ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, fileName))
	if err == nil || !strings.Contains(err.Error(), "storage: object doesn't exist") {
		t.Fatalf("Incorrect error returned from GCS for file %s: %v", fileName, err)
	}
}

func ValidateObjectContentsFromGCS(ctx context.Context, storageClient *storage.Client,
	testDirName string, fileName string, expectedContent string, t *testing.T) {
	gotContent, err := ReadObjectFromGCS(ctx, storageClient, path.Join(testDirName, fileName))
	if err != nil {
		t.Fatalf("Error while reading file from GCS, Err: %v", err)
	}

	if expectedContent != gotContent {
		t.Fatalf("GCS file %s content mismatch. Got file size: %d, Expected file size: %d ", fileName, len(gotContent), len(expectedContent))
	}
}

func ValidateObjectChunkFromGCS(ctx context.Context, storageClient *storage.Client,
	testDirName string, fileName string, offset, size int64, expectedContent string,
	t *testing.T) {
	gotContent, err := ReadChunkFromGCS(ctx, storageClient,
		path.Join(testDirName, fileName), offset, size)
	if err != nil {
		t.Fatalf("Error while reading file from GCS, Err: %v", err)
	}

	if expectedContent != gotContent {
		t.Fatalf("GCS file %s content mismatch. Got file size: %d, Expected "+
			"file size: %d ", fileName, len(gotContent), len(expectedContent))
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

func SetupFileInTestDirectory(ctx context.Context, storageClient *storage.Client,
	testDirName, testFileName string, size int64, t *testing.T) {
	randomData, err := operations.GenerateRandomData(size)
	randomDataString := string(randomData)
	if err != nil {
		t.Errorf("operations.GenerateRandomData: %v", err)
	}
	// Setup file with content in test directory.
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName, testFileName, randomDataString, t)
}

func SetupTestDirectory(ctx context.Context, storageClient *storage.Client, testDirName string) string {
	testDirPath := path.Join(setup.MntDir(), testDirName)
	err := DeleteAllObjectsWithPrefix(ctx, storageClient, path.Join(setup.OnlyDirMounted(), testDirName))
	if err != nil {
		log.Printf("Failed to clean up test directory: %v", err)
	}
	err = CreateObjectOnGCS(ctx, storageClient, testDirName+"/", "")
	if err != nil {
		log.Printf("Failed to create test directory: %v", err)
	}
	return testDirPath
}

func CreateNFilesInDir(ctx context.Context, storageClient *storage.Client, numFiles int, fileName string, fileSize int64, dirName string, t *testing.T) (fileNames []string) {
	for i := 0; i < numFiles; i++ {
		testFileName := fileName + setup.GenerateRandomString(4)
		fileNames = append(fileNames, testFileName)
		SetupFileInTestDirectory(ctx, storageClient, dirName, testFileName, fileSize, t)
	}
	return fileNames
}
