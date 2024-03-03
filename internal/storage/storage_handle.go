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
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/storage"
	"github.com/googleapis/gax-go/v2"
	mountpkg "github.com/googlecloudplatform/gcsfuse/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"golang.org/x/net/context"
	option "google.golang.org/api/option"

	// Side effect to run grpc client with direct-path on gcp machine.
	_ "google.golang.org/grpc/balancer/rls"
	_ "google.golang.org/grpc/xds/googledirectpath"
)

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

// Followed https://pkg.go.dev/cloud.google.com/go/storage#hdr-Experimental_gRPC_API to create the gRPC client.
func createGRPCClientHandle(ctx context.Context, clientConfig storageutil.StorageClientConfig) (sc *storage.Client, err error) {
	if clientConfig.ClientProtocol != mountpkg.GRPC {
		return nil, errors.New("wrong client-protocol requested")
	}

	if err := os.Setenv("GOOGLE_CLOUD_ENABLE_DIRECT_PATH_XDS", "true"); err != nil {
		log.Fatalf("error setting direct path env var: %v", err)
	}

	sc, err = storage.NewClient(ctx, option.WithGRPCConnectionPool(clientConfig.GrpcConnectionPoolSize))

	if err := os.Unsetenv("GOOGLE_CLOUD_ENABLE_DIRECT_PATH_XDS"); err != nil {
		log.Fatalf("error while unsetting direct path env var: %v", err)
	}

	return
}

func createHTTPClientHandle(ctx context.Context, clientConfig storageutil.StorageClientConfig) (sc *storage.Client, err error) {
	var clientOpts []option.ClientOption

	// Add WithHttpClient option.
	if clientConfig.ClientProtocol == mountpkg.HTTP1 || clientConfig.ClientProtocol == mountpkg.HTTP2 {
		var httpClient *http.Client
		httpClient, err = storageutil.CreateHttpClient(&clientConfig)
		if err != nil {
			err = fmt.Errorf("while creating http endpoint: %w", err)
			return
		}

		clientOpts = append(clientOpts, option.WithHTTPClient(httpClient))
	} else {
		return nil, errors.New("wrong client-protocol requested")
	}

	// Create client with JSON read flow, if EnableJasonRead flag is set.
	if clientConfig.ExperimentalEnableJsonRead {
		clientOpts = append(clientOpts, storage.WithJSONReads())
	}

	// Add Custom endpoint option.
	if clientConfig.CustomEndpoint != nil {
		clientOpts = append(clientOpts, option.WithEndpoint(clientConfig.CustomEndpoint.String()))
	}

	return storage.NewClient(ctx, clientOpts...)
}

// NewStorageHandle returns the handle of http or grpc Go storage client based on the
// provided StorageClientConfig.ClientProtocol.
// Please check out the StorageClientConfig to know about the parameters used in
// http and gRPC client.
func NewStorageHandle(ctx context.Context, clientConfig storageutil.StorageClientConfig) (sh StorageHandle, err error) {
	var sc *storage.Client
	if clientConfig.ClientProtocol == mountpkg.GRPC {
		sc, err = createGRPCClientHandle(ctx, clientConfig)
	} else {
		sc, err = createHTTPClientHandle(ctx, clientConfig)
	}

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
			Max:        clientConfig.MaxRetrySleep,
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
