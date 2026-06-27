// Copyright 2015 Google LLC
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

package gcsx

import (
	"context"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const TestBucketName string = "gcsfuse-default-bucket"
const invalidBucketName string = "will-not-be-present-in-fake-server"

func setupTest(t *testing.T) (gcs.Bucket, storage.StorageHandle, *storage.MockStorageControlClient) {
	mockClient := new(storage.MockStorageControlClient)
	fakeStorage := storage.NewFakeStorageWithMockClient(mockClient, cfg.HTTP2)
	storageHandle := fakeStorage.CreateStorageHandle()

	mockClient.On("GetStorageLayout", mock.Anything, mock.MatchedBy(func(req *controlpb.GetStorageLayoutRequest) bool {
		return strings.Contains(req.Name, TestBucketName)
	}), mock.Anything).
		Return(&controlpb.StorageLayout{
			HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
			LocationType:          "zone",
		}, nil)

	ctx := context.Background()
	bucket, err := storageHandle.BucketHandle(ctx, TestBucketName, "")

	require.NotNil(t, bucket)
	require.NoError(t, err)

	t.Cleanup(func() {
		fakeStorage.ShutDown()
	})

	return bucket, storageHandle, mockClient
}

func TestNewBucketManager(t *testing.T) {
	_, storageHandle, _ := setupTest(t)
	bucketConfig := BucketConfig{
		BillingProject:                     "BillingProject",
		OnlyDir:                            "OnlyDir",
		EgressBandwidthLimitBytesPerSecond: 7,
		OpRateLimitHz:                      11,
		StatCacheMaxSizeMB:                 1,
		StatCacheTTL:                       20 * time.Second,
		EnableMonitoring:                   true,
		AppendThreshold:                    2,
		TmpObjectPrefix:                    "TmpObjectPrefix",
	}

	bm := NewBucketManager(bucketConfig, storageHandle)

	assert.NotNil(t, bm)
}

func TestSetUpBucket(t *testing.T) {
	_, storageHandle, _ := setupTest(t)
	var bm bucketManager
	bucketConfig := BucketConfig{
		BillingProject:                     "BillingProject",
		OnlyDir:                            "OnlyDir",
		EgressBandwidthLimitBytesPerSecond: 7,
		OpRateLimitHz:                      11,
		StatCacheMaxSizeMB:                 1,
		StatCacheTTL:                       20 * time.Second,
		EnableMonitoring:                   true,
		AppendThreshold:                    2,
		TmpObjectPrefix:                    "TmpObjectPrefix",
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	bm.storageHandle = storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	bucket, err := bm.SetUpBucket(ctx, TestBucketName, false, metrics.NewNoopMetrics())

	require.NoError(t, err)
	assert.NotNil(t, bucket.Syncer)
}

func TestSetUpBucket_IsMultiBucketMountTrue(t *testing.T) {
	_, storageHandle, _ := setupTest(t)
	var bm bucketManager
	bucketConfig := BucketConfig{
		BillingProject:                     "BillingProject",
		OnlyDir:                            "OnlyDir",
		EgressBandwidthLimitBytesPerSecond: 7,
		OpRateLimitHz:                      11,
		StatCacheMaxSizeMB:                 1,
		StatCacheTTL:                       20 * time.Second,
		EnableMonitoring:                   true,
		AppendThreshold:                    2,
		TmpObjectPrefix:                    "TmpObjectPrefix",
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	bm.storageHandle = storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	bucket, err := bm.SetUpBucket(ctx, TestBucketName, true, metrics.NewNoopMetrics())

	require.NoError(t, err)
	assert.NotNil(t, bucket.Syncer)
}

func TestSetUpBucketWhenBucketDoesNotExist(t *testing.T) {
	_, storageHandle, mockClient := setupTest(t)
	var bm bucketManager
	bucketConfig := BucketConfig{
		BillingProject:                     "BillingProject",
		OnlyDir:                            "OnlyDir",
		EgressBandwidthLimitBytesPerSecond: 7,
		OpRateLimitHz:                      11,
		StatCacheMaxSizeMB:                 1,
		StatCacheTTL:                       20 * time.Second,
		EnableMonitoring:                   true,
		AppendThreshold:                    2,
		TmpObjectPrefix:                    "TmpObjectPrefix",
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	bm.storageHandle = storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	// Configure mock to return NotFound error for invalidBucketName layout check
	mockClient.On("GetStorageLayout", mock.Anything, mock.MatchedBy(func(req *controlpb.GetStorageLayoutRequest) bool {
		return strings.Contains(req.Name, invalidBucketName)
	}), mock.Anything).
		Return(nil, status.Errorf(codes.NotFound, "The specified bucket does not exist."))

	bucket, err := bm.SetUpBucket(ctx, invalidBucketName, false, metrics.NewNoopMetrics())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "storageLayout call failed:")
	assert.Contains(t, err.Error(), "code = NotFound desc = The specified bucket does not exist.")
	assert.Nil(t, bucket.Syncer)
}

func TestSetUpBucketWhenBucketDoesNotExist_IsMultiBucketMountTrue(t *testing.T) {
	_, storageHandle, mockClient := setupTest(t)
	var bm bucketManager
	bucketConfig := BucketConfig{
		BillingProject:                     "BillingProject",
		OnlyDir:                            "OnlyDir",
		EgressBandwidthLimitBytesPerSecond: 7,
		OpRateLimitHz:                      11,
		StatCacheMaxSizeMB:                 1,
		StatCacheTTL:                       20 * time.Second,
		EnableMonitoring:                   true,
		AppendThreshold:                    2,
		TmpObjectPrefix:                    "TmpObjectPrefix",
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	bm.storageHandle = storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	// Configure mock to return NotFound error for invalidBucketName layout check
	mockClient.On("GetStorageLayout", mock.Anything, mock.MatchedBy(func(req *controlpb.GetStorageLayoutRequest) bool {
		return strings.Contains(req.Name, invalidBucketName)
	}), mock.Anything).
		Return(nil, status.Errorf(codes.NotFound, "The specified bucket does not exist."))

	bucket, err := bm.SetUpBucket(ctx, invalidBucketName, true, metrics.NewNoopMetrics())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "storageLayout call failed:")
	assert.Contains(t, err.Error(), "code = NotFound desc = The specified bucket does not exist.")
	assert.Nil(t, bucket.Syncer)
}
