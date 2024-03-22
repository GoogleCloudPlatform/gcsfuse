package gcsx

import (
	"context"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	. "github.com/jacobsa/ogletest"
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
}

var _ SetUpInterface = &BucketManagerTest{}
var _ TearDownInterface = &BucketManagerTest{}

func init() { RegisterTestSuite(&BucketManagerTest{}) }

func (t *BucketManagerTest) SetUp(_ *TestInfo) {
	t.fakeStorage = storage.NewFakeStorage()
	t.storageHandle = t.fakeStorage.CreateStorageHandle()
	t.bucket = t.storageHandle.BucketHandle(TestBucketName, "")

	AssertNe(nil, t.bucket)
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
		DebugGCS:                           true,
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
		DebugGCS:                           true,
		AppendThreshold:                    2,
		TmpObjectPrefix:                    "TmpObjectPrefix",
	}
	ctx := context.Background()
	bm.storageHandle = t.storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	bucket, err := bm.SetUpBucket(context.Background(), TestBucketName, false)

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
		DebugGCS:                           true,
		AppendThreshold:                    2,
		TmpObjectPrefix:                    "TmpObjectPrefix",
	}
	ctx := context.Background()
	bm.storageHandle = t.storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	bucket, err := bm.SetUpBucket(context.Background(), TestBucketName, true)

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
		DebugGCS:                           true,
		AppendThreshold:                    2,
		TmpObjectPrefix:                    "TmpObjectPrefix",
	}
	ctx := context.Background()
	bm.storageHandle = t.storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	bucket, err := bm.SetUpBucket(context.Background(), invalidBucketName, false)

	ExpectEq("Error in iterating through objects: storage: bucket doesn't exist", err.Error())
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
		DebugGCS:                           true,
		AppendThreshold:                    2,
		TmpObjectPrefix:                    "TmpObjectPrefix",
	}
	ctx := context.Background()
	bm.storageHandle = t.storageHandle
	bm.config = bucketConfig
	bm.gcCtx = ctx

	bucket, err := bm.SetUpBucket(context.Background(), invalidBucketName, true)

	ExpectEq("Error in iterating through objects: storage: bucket doesn't exist", err.Error())
	ExpectNe(nil, bucket.Syncer)
}
