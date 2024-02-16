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
	"fmt"
	"io"
	"log"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func separateBucketAndObjectName(bucket, object *string) {
	bucketAndObjectPath := strings.SplitN(*bucket, "/", 2)
	*bucket = bucketAndObjectPath[0]
	*object = path.Join(bucketAndObjectPath[1], *object)
}

func setBucketAndObjectBasedOnTypeOfMount(bucket, object *string) {
	*bucket = setup.TestBucket()
	if strings.Contains(setup.TestBucket(), "/") {
		// This case arises when we run tests on mounted directory and pass
		// bucket/directory in testbucket flag.
		separateBucketAndObjectName(bucket, object)
	}
	if setup.DynamicBucketMounted() != "" {
		*bucket = setup.DynamicBucketMounted()
	}
	if setup.OnlyDirMounted() != "" {
		*object = path.Join(setup.OnlyDirMounted(), *object)
	}
}

func CreateStorageClient(ctx context.Context) (*storage.Client, error) {
	// Create new storage client.
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}
	return client, nil
}

// ReadObjectFromGCS downloads the object from GCS and returns the data.
func ReadObjectFromGCS(ctx context.Context, client *storage.Client, object string, size int64) (string, error) {
	var bucket string
	setBucketAndObjectBasedOnTypeOfMount(&bucket, &object)

	// Create storage reader to read from GCS.
	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("Object(%q).NewReader: %w", object, err)
	}
	defer rc.Close()

	// Variable buf will contain the output from reader.
	buf := make([]byte, size)
	_, err = rc.Read(buf)
	if err != nil && !strings.Contains(err.Error(), "EOF") {
		return "", fmt.Errorf("rc.Read: %w", err)
	}

	// Remove any extra null characters from buf before returning.
	return strings.Trim(string(buf), "\x00"), nil
}

// CreateObjectOnGCS creates an object with given name and content on GCS.
func CreateObjectOnGCS(ctx context.Context, client *storage.Client, object, content string) error {
	var bucket string
	setBucketAndObjectBasedOnTypeOfMount(&bucket, &object)

	o := client.Bucket(bucket).Object(object)
	o = o.If(storage.Conditions{DoesNotExist: true})

	// Upload an object with storage.Writer.
	wc := o.NewWriter(ctx)
	if _, err := io.WriteString(wc, content); err != nil {
		return fmt.Errorf("io.WriteSTring: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %w", err)
	}

	return nil
}

// CreateStorageClientWithTimeOut creates storage client with a configurable timeout and return a function to cancel the storage client
func CreateStorageClientWithTimeOut(ctx *context.Context, storageClient **storage.Client, time time.Duration, t *testing.T) func() {
	var err error
	var cancel context.CancelFunc
	*ctx, cancel = context.WithTimeout(*ctx, time)
	*storageClient, err = CreateStorageClient(*ctx)
	if err != nil {
		log.Fatalf("client.CreateStorageClient: %v", err)
	}
	// Return func to close storage client and release resources.
	return func() {
		err := (*storageClient).Close()
		if err != nil {
			t.Log("Failed to close storage client")
		}
		defer cancel()
	}
}

// DownloadObjectFromGCS downloads an object to a local file.
func DownloadObjectFromGCS(gcsFile string, destFileName string, t *testing.T) error {
	var bucket string
	setBucketAndObjectBasedOnTypeOfMount(&bucket, &gcsFile)

	ctx := context.Background()
	var storageClient *storage.Client
	closeStorageClient := CreateStorageClientWithTimeOut(&ctx, &storageClient, time.Minute*5, t)
	defer closeStorageClient()

	f := operations.CreateFile(destFileName, setup.FilePermission_0600, t)
	defer operations.CloseFile(f)

	rc, err := storageClient.Bucket(bucket).Object(gcsFile).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("Object(%q).NewReader: %w", gcsFile, err)
	}
	defer rc.Close()

	if _, err := io.Copy(f, rc); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}

	return nil
}
