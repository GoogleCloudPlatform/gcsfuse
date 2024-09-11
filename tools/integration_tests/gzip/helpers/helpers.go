// Copyright 2023 Google LLC
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
)

func CreateDataOfSize(contentSize int) (string, error) {
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
		n, err := contentBuilder.WriteString(tempStr)
		if err != nil {
			return "", fmt.Errorf("failed to write to string builder: %w", err)
		}
		if n != len(tempStr) {
			return "", fmt.Errorf("unexpected number of bytes written: expected %d, got %d", len(tempStr), n)
		}
	}

	if contentSize > 0 {
		n, err := contentBuilder.WriteString(tempStr[0:contentSize])
		if err != nil {
			return "", fmt.Errorf("failed to write to string builder: %w", err)
		}
		if n != contentSize {
			return "", fmt.Errorf("unexpected number of bytes written: expected %d, got %d", contentSize, n)
		}
	}

	content := contentBuilder.String()
	return content, nil
}

// Downloads given gzipped GCS object (with path without 'gs://') to local disk.
// Fails if the object doesn't exist or permission to read object is not
// available.
// Uses go storage client library to download object. Use of gsutil/gcloud is not
// possible as they both always read back objects with content-encoding: gzip as
// uncompressed/decompressed irrespective of any argument passed.
func DownloadGzipGcsObjectAsCompressed(bucketName, objPathInBucket string) (tempfile string, err error) {
	gcsObjectPath := path.Join(setup.TestBucket(), objPathInBucket)
	gcsObjectSize, err := operations.GetGcsObjectSize(gcsObjectPath)

	if err != nil {
		err = fmt.Errorf("failed to get size of gcs object %s: %w", gcsObjectPath, err)
		return
	}

	content, err := CreateDataOfSize(1)
	if err != nil {
		err = fmt.Errorf("failed to create data: %w", err)
		return
	}
	tempfile, err = operations.CreateLocalTempFile(content, false)
	if err != nil {
		err = fmt.Errorf("failed to create tempfile for downloading gcs object: %w", err)
		return
	}
	defer func() {
		if err != nil {
			os.Remove(tempfile)
		}
	}()

	ctx := context.Background()
	client, err := client2.CreateStorageClient(ctx)
	if err != nil || client == nil {
		err = fmt.Errorf("failed to create storage client: %w", err)
		return
	}
	defer client.Close()

	bktName := setup.TestBucket()
	bkt := client.Bucket(bktName)
	if bkt == nil {
		err = fmt.Errorf("failed to access bucket %s: %w", bktName, err)
		return
	}

	obj := bkt.Object(objPathInBucket)
	if obj == nil {
		err = fmt.Errorf("failed to access object %s from bucket %s: %w", objPathInBucket, bktName, err)
		return
	}

	obj = obj.ReadCompressed(true)
	if obj == nil {
		err = fmt.Errorf("failed to access object %s from bucket %s as compressed: %w", objPathInBucket, bktName, err)
		return
	}

	r, err := obj.NewReader(ctx)
	if r == nil || err != nil {
		err = fmt.Errorf("failed to read object %s from bucket %s: %w", objPathInBucket, bktName, err)
		return
	}
	defer r.Close()

	gcsObjectData, err := io.ReadAll(r)
	if len(gcsObjectData) < gcsObjectSize || err != nil {
		err = fmt.Errorf("failed to read object %s from bucket %s (expected read-size: %d, actual read-size: %d): %w", objPathInBucket, bktName, gcsObjectSize, len(gcsObjectData), err)
		return
	}

	err = os.WriteFile(tempfile, gcsObjectData, fs.FileMode(os.O_CREATE|os.O_WRONLY|os.O_TRUNC))
	if err != nil || client == nil {
		err = fmt.Errorf("failed to write to tempfile %s: %w", tempfile, err)
		return
	}

	return tempfile, nil
}
