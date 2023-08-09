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

// Provides integration tests for gzip objects in gcsfuse mounts.
package gzip_test

import (
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/gzip/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

// Size of the overwritten content created in bytes.
const OverwittenFileSize = 1000

// Verify that the passed file exists on the GCS test-bucket and in the mounted bucket
// and its size in the mounted directory matches that of the GCS object. Also verify
// that the passed file in the mounted bucket matches the corresponding
// GCS object in content.
// GCS object.
func verifyFullFileOverwrite(t *testing.T, filename string) {
	mountedFilePath := path.Join(setup.MntDir(), TestBucketPrefixPath, filename)
	gcsObjectPath := path.Join(setup.TestBucket(), TestBucketPrefixPath, filename)
	gcsObjectSize, err := operations.GetGcsObjectSize(gcsObjectPath)
	if err != nil {
		t.Fatalf("Failed to get size of gcs object %s: %v\n", gcsObjectPath, err)
	}

	fi, err := operations.StatFile(mountedFilePath)
	if err != nil || fi == nil {
		t.Fatalf("Failed to get stat info of mounted file %s: %v\n", mountedFilePath, err)
	}

	if (*fi).Size() != int64(gcsObjectSize) {
		t.Fatalf("Size of file mounted through gcsfuse (%s, %d) doesn't match the size of the file on GCS (%s, %d)",
			mountedFilePath, (*fi).Size(), gcsObjectPath, gcsObjectSize)
	}

	// No need to worry about gzipping the overwritten data, because it's
	// expensive to invoke a gzip-writer and unnecessary in this case.
	// All we are interested in testing is that the content of the overwritten
	// gzip file matches in size with that of the source file that was used to
	// overwrite it.
	tempfile, err := helpers.CreateLocalTempFile(OverwittenFileSize, false)
	if err != nil {
		t.Fatalf("Failed to create local temp file for overwriting existing gzip object: %v", err)
	}
	defer operations.RemoveFile(tempfile)

	err = operations.CopyFileAllowOverwrite(tempfile, mountedFilePath)
	if err != nil {
		t.Fatalf("Failed to copy/overwrite temp file %s to existing gzip object/file %s: %v", tempfile, mountedFilePath, err)
	}

	gcsObjectSize, err = operations.GetGcsObjectSize(gcsObjectPath)
	if err != nil {
		t.Fatalf("Failed to get size of gcs object %s: %v\n", gcsObjectPath, err)
	}

	if gcsObjectSize != OverwittenFileSize {
		t.Fatalf("Size of overwritten gcs object (%s, %d) doesn't match that of the expected overwrite size (%s, %d)", gcsObjectPath, gcsObjectSize, tempfile, OverwittenFileSize)
	}
}

func TestGzipEncodedTextFileWithNoTransformFullFileOverwrite(t *testing.T) {
	verifyFullFileOverwrite(t, TextContentWithContentEncodingWithNoTransformToOverwrite)
}

func TestGzipEncodedTextFileWithoutNoTransformFullFileOverwrite(t *testing.T) {
	verifyFullFileOverwrite(t, TextContentWithContentEncodingWithoutNoTransformToOverwrite)
}

func TestGzipUnencodedGzipFileFullFileOverwrite(t *testing.T) {
	verifyFullFileOverwrite(t, GzipContentWithoutContentEncodingToOverwrite)
}

func TestGzipEncodedGzipFileWithNoTransformFullFileOverwrite(t *testing.T) {
	verifyFullFileOverwrite(t, GzipContentWithContentEncodingWithNoTransformToOverwrite)
}

func TestGzipEncodedGzipFileWithoutNoTransformFullFileOverwrite(t *testing.T) {
	verifyFullFileOverwrite(t, GzipContentWithContentEncodingWithoutNoTransformToOverwrite)
}
