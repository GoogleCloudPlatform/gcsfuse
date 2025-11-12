// Copyright 2025 Google LLC
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

package local_file

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
)

const (
	onlyDirMounted       = "OnlyDirMountLocalFiles"
	testDirLocalFileTest = "LocalFileTest"
)

var (
	testDirName   string
	testDirPath   string
	storageClient *storage.Client
	ctx           context.Context
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func WritingToLocalFileShouldNotWriteToGCS(ctx context.Context, storageClient *storage.Client,
	fh *os.File, testDirName, fileName string, t *testing.T) {
	operations.WriteWithoutClose(fh, client.FileContents, t)
	client.ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, fileName, t)
}

func NewFileShouldGetSyncedToGCSAtClose(ctx context.Context, storageClient *storage.Client,
	testDirPath, fileName string, t *testing.T) {
	// Create a local file.
	_, fh := client.CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t)

	// Writing contents to local file shouldn't create file on GCS.
	testDirName := client.GetDirName(testDirPath)
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, fileName, t)

	// Close the file and validate if the file is created on GCS.
	client.CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, fileName, client.FileContents, t)
}
