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

// Provides integration tests for write flows.
package operations_test

import (
	"context"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func validateExtendedObjectAttributesWrittenSuccessfully(objectName string, t *testing.T) {
	ctx := context.Background()
	var storageClient *storage.Client
	closeStorageClient := client.CreateStorageClientWithTimeOut(&ctx, &storageClient, time.Minute*5, t)
	defer closeStorageClient()

	attrs, err := client.StatObject(ctx, storageClient, objectName)
	t.Logf("Object name: %s", objectName)
	if err != nil {
		t.Errorf("Could not read file from GCS: %v", err)
	}
	o := storageutil.ObjectAttrsToBucketObject(attrs)
	e := storageutil.ConvertObjToExtendedObjectAttributes(o)

	if e == nil || reflect.DeepEqual(*e, gcs.ExtendedObjectAttributes{}) {
		t.Errorf("Received nil/empty extended object attributes.")
	}
}

func TestWriteAtEndOfFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	fileName := setup.CreateTempFile()
	objectName := "tmpFile"

	err := operations.WriteFileInAppendMode(fileName, "line 3\n")
	if err != nil {
		t.Errorf("AppendString: %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 1\nline 2\nline 3\n")
	// Validate that extended object attributes are non nil/ non-empty.
	validateExtendedObjectAttributesWrittenSuccessfully(objectName, t)
}

func TestWriteAtStartOfFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	fileName := setup.CreateTempFile()
	objectName := "tmpFile"

	err := operations.WriteFile(fileName, "line 4\n")
	if err != nil {
		t.Errorf("WriteString-Start: %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 4\nline 2\n")
	// Validate that extended object attributes are non nil/ non-empty.
	validateExtendedObjectAttributesWrittenSuccessfully(objectName, t)
}

func TestWriteAtRandom(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	fileName := setup.CreateTempFile()
	objectName := "tmpFile"

	f, err := os.OpenFile(fileName, os.O_WRONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Open file for write at random: %v", err)
	}

	// Write at 7th byte which corresponds to the start of 2nd line
	// thus changing line2\n with line5\n.
	if _, err = f.WriteAt([]byte("line 5\n"), 7); err != nil {
		t.Errorf("WriteString-Random: %v", err)
	}
	// Closing file at the end
	operations.CloseFile(f)

	setup.CompareFileContents(t, fileName, "line 1\nline 5\n")
	// Validate that extended object attributes are non nil/ non-empty.
	validateExtendedObjectAttributesWrittenSuccessfully(objectName, t)
}

func TestCreateFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	fileName := setup.CreateTempFile()
	objectName := "tmpFile"

	// Stat the file to check if it exists.
	if _, err := os.Stat(fileName); err != nil {
		t.Errorf("File not found, %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 1\nline 2\n")
	// Validate that extended object attributes are non nil/ non-empty.
	validateExtendedObjectAttributesWrittenSuccessfully(objectName, t)
}
