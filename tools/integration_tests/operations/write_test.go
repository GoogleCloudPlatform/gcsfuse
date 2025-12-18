// Copyright 2023 Google LLC
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
	"path"
	"reflect"
	"syscall"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const tempFileName = "tmpFile"
const appendContent = "Content"
const tempFileContent = "line 1\nline 2\n"

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////
func validateExtendedObjectAttributesNonEmpty(objectName string, t *testing.T) *storage.ObjectAttrs {
	ctx := context.Background()
	var storageClient *storage.Client
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			t.Errorf("closeStorageClient failed: %v", err)
		}
	}()

	attrs, err := client.StatObject(ctx, storageClient, objectName)
	if err != nil {
		t.Errorf("Could not fetch object attributes: %v", err)
	}
	o := storageutil.ObjectAttrsToBucketObject(attrs)
	e := storageutil.ConvertObjToExtendedObjectAttributes(o)

	if e == nil || reflect.DeepEqual(*e, gcs.ExtendedObjectAttributes{}) {
		t.Errorf("Received nil/empty extended object attributes.")
	}
	return attrs
}

func validateObjectAttributes(attr1, attr2 *storage.ObjectAttrs, t *testing.T) {
	const contentType = "text/plain; charset=utf-8"
	const componentCount = 0
	const sizeBeforeOperation = int64(len(tempFileContent))
	const sizeAfterOperation = sizeBeforeOperation + int64(len(appendContent))
	storageClass := "STANDARD"
	if attr1 == nil || attr2 == nil {
		t.Fatalf("attr1 or attr2 is nil. attr1: %v, attr2: %v", attr1, attr2)
	}

	if setup.IsZonalBucketRun() {
		storageClass = "RAPID"
	}

	if attr1.ContentType != contentType || attr2.ContentType != contentType {
		t.Errorf("Expected content type: %s, Got: %s, %s", contentType, attr1.ContentType, attr2.ContentType)
	}
	if attr1.ComponentCount != componentCount || attr2.ComponentCount != componentCount {
		t.Errorf("Expected component count: %d, Got: %d, %d", componentCount, attr1.ComponentCount, attr2.ComponentCount)
	}
	if attr1.Name != attr2.Name {
		t.Errorf("Object name mismatch: %s, %s", attr1.Name, attr2.Name)
	}
	if attr1.Bucket != attr2.Bucket {
		t.Errorf("Bucket name mismatch: %s, %s", attr1.Bucket, attr2.Bucket)
	}
	if attr1.EventBasedHold != false || attr2.EventBasedHold != false {
		t.Errorf("Expected EventBasedHold: false, Got: %v %v", attr1.EventBasedHold, attr2.EventBasedHold)
	}
	if attr1.Size != sizeBeforeOperation {
		t.Errorf("Expected size before file operation: %d, Got: %d", sizeBeforeOperation, attr1.Size)
	}
	if attr2.Size != sizeAfterOperation {
		t.Errorf("Expected size after file operation: %d, Got: %d", sizeAfterOperation, attr2.Size)
	}
	if reflect.DeepEqual(attr1.MD5, []byte{}) || reflect.DeepEqual(attr2.MD5, []byte{}) {
		t.Error("Expected MD5 attributes to be non empty")
	}
	if attr1.CRC32C == 0 || attr2.CRC32C == 0 {
		t.Error("Expected CRC32 attributes to be non 0")
	}
	if attr1.MediaLink == "" || attr2.MediaLink == "" {
		if setup.IsZonalBucketRun() {
			t.Logf("media link is empty, but it is a known limitation in RAPID/zonal buckets.")
		} else {
			t.Errorf("Expected media link to be non empty")
		}
	}
	if attr1.StorageClass != storageClass || attr2.StorageClass != storageClass {
		t.Errorf("Expected storage class to be %q, but found attr1.StorageClass = %q (bucketName = %q), attr2.StorageClass = %q (bucketName = %q)", storageClass, attr1.StorageClass, attr1.Bucket, attr2.StorageClass, attr2.Bucket)
	}
	attr1MTime, _ := time.Parse(time.RFC3339Nano, attr1.Metadata[gcs.MtimeMetadataKey])
	attr2MTime, _ := time.Parse(time.RFC3339Nano, attr2.Metadata[gcs.MtimeMetadataKey])
	if attr2MTime.Before(attr1MTime) {
		t.Errorf("Unexpected MTime received. After operation1: %v, After operation2: %v", attr1MTime, attr2MTime)
	}
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func TestWriteAtEndOfFile(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	fileName := path.Join(testDir, tempFileName)

	operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, t)

	err := operations.WriteFileInAppendMode(fileName, "line 3\n")
	if err != nil {
		t.Errorf("AppendString: %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 1\nline 2\nline 3\n")
	// Validate that extended object attributes are non nil/ non-empty.
	validateExtendedObjectAttributesNonEmpty(path.Join(DirForOperationTests, tempFileName), t)
}

func TestWriteAtStartOfFile(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	fileName := path.Join(testDir, tempFileName)

	operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, t)

	err := operations.WriteFile(fileName, "line 4\n")
	if err != nil {
		t.Errorf("WriteString-Start: %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 4\nline 2\n")
	// Validate that extended object attributes are non nil/ non-empty.
	validateExtendedObjectAttributesNonEmpty(path.Join(DirForOperationTests, tempFileName), t)
}

func TestWriteAtRandom(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	fileName := path.Join(testDir, tempFileName)

	operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, t)

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
	operations.CloseFileShouldNotThrowError(t, f)

	setup.CompareFileContents(t, fileName, "line 1\nline 5\n")
	// Validate that extended object attributes are non nil/ non-empty.
	validateExtendedObjectAttributesNonEmpty(path.Join(DirForOperationTests, tempFileName), t)
}

func TestCreateFile(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	fileName := path.Join(testDir, tempFileName)

	operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, t)

	// Stat the file to check if it exists.
	if _, err := os.Stat(fileName); err != nil {
		t.Errorf("File not found, %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 1\nline 2\n")
	// Validate that extended object attributes are non nil/ non-empty.
	validateExtendedObjectAttributesNonEmpty(path.Join(DirForOperationTests, tempFileName), t)
}

func TestAppendFileOperationsDoesNotChangeObjectAttributes(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	// Create file.
	fileName := path.Join(testDir, tempFileName)

	operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, t)
	attr1 := validateExtendedObjectAttributesNonEmpty(path.Join(DirForOperationTests, tempFileName), t)
	// Append to the file.
	err := operations.WriteFileInAppendMode(fileName, appendContent)
	if err != nil {
		t.Errorf("Could not append to file: %v", err)
	}
	attr2 := validateExtendedObjectAttributesNonEmpty(path.Join(DirForOperationTests, tempFileName), t)

	// Validate object attributes are as expected.
	validateObjectAttributes(attr1, attr2, t)
}

func TestWriteAtFileOperationsDoesNotChangeObjectAttributes(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	// Create file.
	fileName := path.Join(testDir, tempFileName)

	operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, t)
	attr1 := validateExtendedObjectAttributesNonEmpty(path.Join(DirForOperationTests, tempFileName), t)
	// Over-write the file.
	fh, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_DIRECT, operations.FilePermission_0600)
	if err != nil {
		t.Fatalf("Could not open file %s after creation.", fileName)
	}
	operations.WriteAt(tempFileContent+appendContent, 0, fh, t)
	operations.CloseFileShouldNotThrowError(t, fh)
	attr2 := validateExtendedObjectAttributesNonEmpty(path.Join(DirForOperationTests, tempFileName), t)

	// Validate object attributes are as expected.
	validateObjectAttributes(attr1, attr2, t)
}
