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

package helpers

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	client2 "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	TempFileStrLine = "This is a test file"
	TmpDirectory    = "/tmp"
)

// Creates a temporary file (name-collision-safe) in /tmp with given content size in bytes.
// If gzipCompress is true, output file is a gzip-compressed file.
// contentSize is the size of the uncompressed content. In case gzipCompress is true, the actual output file size will be
// different from contentSize (typically gzip-compressed file size < contentSize).
// Caller is responsible for deleting the created file when done using it.
// Failure cases:
// 1. contentSize <= 0
// 2. os.CreateTemp() returned error or nil handle
// 3. gzip.NewWriter() returned nil handle
// 4. Failed to write the content to the created temp file
func CreateLocalTempFile(contentSize int, gzipCompress bool) (string, error) {
	// fail if contentSize <= 0
	if contentSize <= 0 {
		return "", fmt.Errorf("unsupported fileSize: %d", contentSize)
	}

	// Create text-content of given size.
	// strings.builder is used as opposed to string appends
	// as this is much more efficient when multiple concatenations
	// are required.
	var contentBuilder strings.Builder
	const tempStr = TempFileStrLine + "\n"

	for ; contentSize >= len(tempStr); contentSize -= len(tempStr) {
		contentBuilder.WriteString(tempStr)
	}

	if contentSize > 0 {
		contentBuilder.WriteString(tempStr[0:contentSize])
	}

	// reset contentSize
	contentSize = contentBuilder.Len()

	// create appropriate name template for temp file
	filenameTemplate := "testfile-*.txt"
	if gzipCompress {
		filenameTemplate += ".gz"
	}

	// create a temp file
	f, err := os.CreateTemp(TmpDirectory, filenameTemplate)
	if err != nil {
		return "", err
	} else if f == nil {
		return "", fmt.Errorf("nil file handle returned from os.CreateTemp")
	}
	defer operations.CloseFile(f)
	filepath := f.Name()

	content := contentBuilder.String()

	if gzipCompress {
		w := gzip.NewWriter(f)
		if w == nil {
			return "", fmt.Errorf("failed to open a gzip writer handle")
		}
		defer func() {
			err := w.Close()
			if err != nil {
				fmt.Printf("Failed to close file %s: %v", filepath, err)
			}
		}()

		// write the content created above as gzip
		n, err := w.Write([]byte(content))
		if err != nil {
			return "", err
		} else if n != contentSize {
			return "", fmt.Errorf("failed to write to gzip file %s. Content-size: %d bytes, wrote = %d bytes", filepath, contentSize, n)
		}
	} else {
		// write the content created above as text
		n, err := f.WriteString(content)
		if err != nil {
			return "", err
		} else if n != contentSize {
			return "", fmt.Errorf("failed to write to text file %s. Content-size: %d bytes, wrote = %d bytes", filepath, contentSize, n)
		}
	}

	return filepath, nil
}

// Downloads given gzipped GCS object (with path without 'gs://') to local disk.
// Fails if the object doesn't exist or permission to read object is not
// available.
// Uses go storage client library to download object. Use of gsutil/gcloud is not
// possible as they both always read back objects with content-encoding: gzip as
// uncompressed/decompressed irrespective of any argument passed.
func DownloadGzipGcsObjectAsCompressed(bucketName, objPathInBucket string) (string, error) {
	gcsObjectPath := path.Join(setup.TestBucket(), objPathInBucket)
	gcsObjectSize, err := operations.GetGcsObjectSize(gcsObjectPath)
	if err != nil {
		return "", fmt.Errorf("failed to get size of gcs object %s: %w", gcsObjectPath, err)
	}

	tempfile, err := CreateLocalTempFile(1, false)
	if err != nil {
		return "", fmt.Errorf("failed to create tempfile for downloading gcs object: %w", err)
	}

	ctx := context.Background()
	client, err := client2.CreateStorageClient(ctx)
	if err != nil || client == nil {
		return "", fmt.Errorf("failed to create storage client: %w", err)
	}
	defer client.Close()

	bktName := setup.TestBucket()
	bkt := client.Bucket(bktName)
	if bkt == nil {
		return "", fmt.Errorf("failed to access bucket %s: %w", bktName, err)
	}

	obj := bkt.Object(objPathInBucket)
	if obj == nil {
		return "", fmt.Errorf("failed to access object %s from bucket %s: %w", objPathInBucket, bktName, err)
	}

	obj = obj.ReadCompressed(true)
	if obj == nil {
		return "", fmt.Errorf("failed to access object %s from bucket %s as compressed: %w", objPathInBucket, bktName, err)
	}

	r, err := obj.NewReader(ctx)
	if r == nil || err != nil {
		return "", fmt.Errorf("failed to read object %s from bucket %s: %w", objPathInBucket, bktName, err)
	}
	defer r.Close()

	gcsObjectData, err := io.ReadAll(r)
	if len(gcsObjectData) < gcsObjectSize || err != nil {
		return "", fmt.Errorf("failed to read object %s from bucket %s (expected read-size: %d, actual read-size: %d): %w", objPathInBucket, bktName, gcsObjectSize, len(gcsObjectData), err)
	}

	err = os.WriteFile(tempfile, gcsObjectData, fs.FileMode(os.O_CREATE|os.O_WRONLY|os.O_TRUNC))
	if err != nil || client == nil {
		return "", fmt.Errorf("failed to write to tempfile %s: %w", tempfile, err)
	}

	return tempfile, nil
}
