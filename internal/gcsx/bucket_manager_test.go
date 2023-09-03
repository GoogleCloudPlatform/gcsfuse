package gcsx

import (
	"context"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/storage/bucket"
	. "github.com/jacobsa/ogletest"
)

func TestBucketManager(t *testing.T) { RunTests(t) }

const TestBucketName string = "gcsfuse-default-bucket"
const invalidBucketName string = "will-not-be-present-in-fake-server"

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type BucketManagerTest struct {
	bucket        bucket.Bucket
	storageHandle bucket.StorageHandle
	fakeStorage   bucket.FakeStorage
}

var _ SetUpInterface = &BucketManagerTest{}
var _ TearDownInterface = &BucketManagerTest{}

func init() { RegisterTestSuite(&BucketManagerTest{}) }

func (t *BucketManagerTest) SetUp(_ *TestInfo) {
	t.fakeStorage = bucket.NewFakeStorage()
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
		StatCacheCapacity:                  100,
		StatCacheTTL:                       20 * time.Second,
		EnableMonitoring:                   true,
		DebugGCS:                           true,
		AppendThreshold:                    2,
		TmpObjectPrefix:                    "TmpObjectPrefix",
	}

	bm := NewBucketManager(bucketConfig, t.storageHandle)

	ExpectNe(nil, bm)
}

func (t *BucketManagerTest) TestSetupGcsBucket() {
	var bm bucketManager
	bm.storageHandle = t.storageHandle
	bm.config.DebugGCS = true

	bucket, err := bm.SetUpGcsBucket(TestBucketName)

	ExpectNe(nil, bucket)
	ExpectEq(nil, err)
}

func (t *BucketManagerTest) TestSetUpBucketMethod() {
	var bm bucketManager
	bucketConfig := BucketConfig{
		BillingProject:                     "BillingProject",
		OnlyDir:                            "OnlyDir",
		EgressBandwidthLimitBytesPerSecond: 7,
		OpRateLimitHz:                      11,
		StatCacheCapacity:                  100,
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

	bucket, err := bm.SetUpBucket(context.Background(), TestBucketName)

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
		StatCacheCapacity:                  100,
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

	bucket, err := bm.SetUpBucket(context.Background(), invalidBucketName)

	ExpectEq("Error in iterating through objects: storage: bucket doesn't exist", err.Error())
	ExpectNe(nil, bucket.Syncer)
}
