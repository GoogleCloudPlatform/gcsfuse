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
	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	. "github.com/jacobsa/ogletest"
	"github.com/stretchr/testify/mock"
)

func TestBucketManager(t *testing.T) { RunTests(t) }

const TestBucketName string = "gcsfuse-default-bucket"
const invalidBucketName string = "will-not-be-present-in-fake-server"

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type BucketManagerTest struct {
	bucket        gcs.Bucket
	storageHandle storage.StorageHandle
	fakeStorage   storage.FakeStorage
	mockClient    *storage.MockStorageControlClient
}

var _ SetUpInterface = &BucketManagerTest{}
var _ TearDownInterface = &BucketManagerTest{}

func init() { RegisterTestSuite(&BucketManagerTest{}) }

func (t *BucketManagerTest) SetUp(_ *TestInfo) {
	var err error
	t.mockClient = new(storage.MockStorageControlClient)
	t.fakeStorage = storage.NewFakeStorageWithMockClient(t.mockClient, cfg.HTTP2)
	t.storageHandle = t.fakeStorage.CreateStorageHandle()
	t.mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(&controlpb.StorageLayout{
			HierarchicalNamespace: &controlpb.StorageLayout_HierarchicalNamespace{Enabled: true},
			LocationType:          "zone",
		}, nil)
	ctx := context.Background()
	t.bucket, err = t.storageHandle.BucketHandle(ctx, TestBucketName, "")

	AssertNe(nil, t.bucket)
	AssertEq(nil, err)
}

func (t *BucketManagerTest) TearDown() {
	t.fakeStorage.ShutDown()
}

func (t *BucketManagerTest) TestNewBucketManagerMethod() {
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

	bm := NewBucketManager(bucketConfig, t.storageHandle)

	ExpectNe(nil, bm)
}

func (t *BucketManagerTest) TestSetUpBucketMethod() {
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
	ctx := context.Background()
	bm.storageHandle = t.storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	bucket, err := bm.SetUpBucket(context.Background(), TestBucketName, false, common.NewNoopMetrics())

	ExpectNe(nil, bucket.Syncer)
	ExpectEq(nil, err)
}

func (t *BucketManagerTest) TestSetUpBucketMethod_IsMultiBucketMountTrue() {
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
	ctx := context.Background()
	bm.storageHandle = t.storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	bucket, err := bm.SetUpBucket(context.Background(), TestBucketName, true, common.NewNoopMetrics())

	ExpectNe(nil, bucket.Syncer)
	ExpectEq(nil, err)
}

func (t *BucketManagerTest) TestSetUpBucketMethodWhenBucketDoesNotExist() {
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
	ctx := context.Background()
	bm.storageHandle = t.storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	bucket, err := bm.SetUpBucket(context.Background(), invalidBucketName, false, common.NewNoopMetrics())

	AssertNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), "error in iterating through objects: storage: bucket doesn't exist"))
	ExpectNe(nil, bucket.Syncer)
}

func (t *BucketManagerTest) TestSetUpBucketMethodWhenBucketDoesNotExist_IsMultiBucketMountTrue() {
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
	ctx := context.Background()
	bm.storageHandle = t.storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	bucket, err := bm.SetUpBucket(context.Background(), invalidBucketName, true, common.NewNoopMetrics())

	AssertNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), "error in iterating through objects: storage: bucket doesn't exist"))
	ExpectNe(nil, bucket.Syncer)
}
