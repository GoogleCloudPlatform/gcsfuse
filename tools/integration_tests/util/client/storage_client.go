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
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	storagev1 "google.golang.org/api/storage/v1"
)

func CreateStorageClient(ctx context.Context) (*storage.Client, error) {
	// Create new storage client.
	client, err := storage.NewClient(ctx, option.WithEndpoint("storage.apis-tpczero.goog:443"), option.WithTokenSource(getTokenSrc(path.Join(os.Getenv("HOME"),"kay.json"))))
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}
	return client, nil
}


func getTokenSrc(path string) (tokenSrc oauth2.TokenSource) {
	contents, err := os.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("ReadFile(%q): %w", path, err)
		return
	}

	// Create a config struct based on its contents.
	ts, err := google.JWTAccessTokenSourceWithScope(contents, storagev1.DevstorageFullControlScope)
	if err != nil {
		err = fmt.Errorf("JWTConfigFromJSON: %w", err)
		return
	}
	return ts
}

// ReadObjectFromGCS downloads the object from GCS and returns the data.
func ReadObjectFromGCS(ctx context.Context, client *storage.Client, object string) (string, error) {
	bucket, object := setup.GetBucketAndObjectBasedOnTypeOfMount(object)

	// Create storage reader to read from GCS.
	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("Object(%q).NewReader: %w", object, err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("io.ReadAll failed: %v", err)
	}

	return string(content), nil
}

// ReadChunkFromGCS downloads the object chunk from GCS and returns the data.
func ReadChunkFromGCS(ctx context.Context, client *storage.Client, object string,
	offset, size int64) (string, error) {
	bucket, object := setup.GetBucketAndObjectBasedOnTypeOfMount(object)

	// Create storage reader to read from GCS.
	rc, err := client.Bucket(bucket).Object(object).NewRangeReader(ctx, offset, size)
	if err != nil {
		return "", fmt.Errorf("Object(%q).NewReader: %w", object, err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("io.ReadAll failed: %v", err)
	}

	return string(content), nil
}

func WriteToObject(ctx context.Context, client *storage.Client, object, content string, precondition storage.Conditions) error {
	bucket, object := setup.GetBucketAndObjectBasedOnTypeOfMount(object)

	o := client.Bucket(bucket).Object(object)
	if !reflect.DeepEqual(precondition, storage.Conditions{}) {
		o = o.If(precondition)
	}

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

// CreateObjectOnGCS creates an object with given name and content on GCS.
func CreateObjectOnGCS(ctx context.Context, client *storage.Client, object, content string) error {
	return WriteToObject(ctx, client, object, content, storage.Conditions{DoesNotExist: true})
}

// CreateStorageClientWithTimeOut creates storage client with a configurable timeout and return a function to cancel the storage client
func CreateStorageClientWithTimeOut(ctx *context.Context, storageClient **storage.Client, time time.Duration) func() error {
	var err error
	var cancel context.CancelFunc
	*ctx, cancel = context.WithTimeout(*ctx, time)
	*storageClient, err = CreateStorageClient(*ctx)
	if err != nil {
		log.Fatalf("client.CreateStorageClient: %v", err)
	}
	// Return func to close storage client and release resources.
	return func() error {
		err := (*storageClient).Close()
		if err != nil {
			return fmt.Errorf("Failed to close storage client: %v", err)
		}
		defer cancel()
		return nil
	}
}

// DownloadObjectFromGCS downloads an object to a local file.
func DownloadObjectFromGCS(gcsFile string, destFileName string, t *testing.T) error {
	bucket, gcsFile := setup.GetBucketAndObjectBasedOnTypeOfMount(gcsFile)

	ctx := context.Background()
	var storageClient *storage.Client
	closeStorageClient := CreateStorageClientWithTimeOut(&ctx, &storageClient, time.Minute*5)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			t.Errorf("closeStorageClient failed: %v", err)
		}
	}()
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

func DeleteObjectOnGCS(ctx context.Context, client *storage.Client, objectName string) error {
	bucket, _ := setup.GetBucketAndObjectBasedOnTypeOfMount("")

	// Get handle to the object
	object := client.Bucket(bucket).Object(objectName)

	// Delete the object
	err := object.Delete(ctx)
	if err != nil {
		return err
	}
	return nil
}

func DeleteAllObjectsWithPrefix(ctx context.Context, client *storage.Client, prefix string) error {
	bucket, _ := setup.GetBucketAndObjectBasedOnTypeOfMount("")

	// Get an object iterator
	query := &storage.Query{Prefix: prefix}
	objectItr := client.Bucket(bucket).Objects(ctx, query)

	// Iterate through objects with the specified prefix and delete them
	for {
		attrs, err := objectItr.Next()
		if err == iterator.Done {
			break
		}
		if err := DeleteObjectOnGCS(ctx, client, attrs.Name); err != nil {
			return err
		}
	}
	return nil
}

func StatObject(ctx context.Context, client *storage.Client, object string) (*storage.ObjectAttrs, error) {
	bucket, object := setup.GetBucketAndObjectBasedOnTypeOfMount(object)

	attrs, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		return nil, err
	}
	return attrs, nil
}
