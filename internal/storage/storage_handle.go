// Copyright 2022 Google Inc. All Rights Reserved.
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

package storage

import (
	"fmt"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googleapis/gax-go/v2"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

const ProdEndpoint = "https://storage.googleapis.com:443"
const CustomEndpoint = "https://localhost:9000"

type StorageHandle interface {
	// In case of non-empty billingProject, this project is set as user-project for
	// all subsequent calls on the bucket. Calls with user-project will be billed
	// to that project rather than to the bucket's owning project.
	//
	// A user-project is required for all operations on Requester Pays buckets.
	BucketHandle(bucketName string, billingProject string) (bh *bucketHandle)
}

type storageClient struct {
	client *storage.Client
}

type StorageClientConfig struct {
	MaxRetryDuration time.Duration
	RetryMultiplier  float64
	ClientOptions    []option.ClientOption
}

// NewStorageHandle returns the handle of Go storage client containing
// customized http client. We can configure the http client using the
// storageClientConfig parameter.
func NewStorageHandle(ctx context.Context, clientConfig StorageClientConfig) (sh StorageHandle, err error) {
	var sc *storage.Client
	sc, err = storage.NewClient(ctx, clientConfig.ClientOptions...)
	if err != nil {
		err = fmt.Errorf("go storage client creation failed: %w", err)
		return
	}

	// ShouldRetry function checks if an operation should be retried based on the
	// response of operation (error.Code).
	// RetryAlways causes all operations to be checked for retries using
	// ShouldRetry function.
	// Without RetryAlways, only those operations are checked for retries which
	// are idempotent.
	// https://github.com/googleapis/google-cloud-go/blob/main/storage/storage.go#L1953
	sc.SetRetry(
		storage.WithBackoff(gax.Backoff{
			Max:        clientConfig.MaxRetryDuration,
			Multiplier: clientConfig.RetryMultiplier,
		}),
		storage.WithPolicy(storage.RetryAlways),
		storage.WithErrorFunc(storageutil.ShouldRetry))

	sh = &storageClient{client: sc}
	return
}

func (sh *storageClient) BucketHandle(bucketName string, billingProject string) (bh *bucketHandle) {
	storageBucketHandle := sh.client.Bucket(bucketName)

	if billingProject != "" {
		storageBucketHandle = storageBucketHandle.UserProject(billingProject)
	}

	bh = &bucketHandle{bucket: storageBucketHandle, bucketName: bucketName}
	return
}
