package gcsx

import (
	"context"
	"testing"
	"time"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestFlags(t *testing.T) { RunTests(t) }

const TestBucketName string = "gcsfuse-default-bucket"

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type BucketManagerTest struct {
	fakeStorageServer *fakestorage.Server
	bucketHandle      *storage.BucketHandle
}

var _ SetUpInterface = &BucketManagerTest{}
var _ TearDownInterface = &BucketManagerTest{}

func init() { RegisterTestSuite(&BucketManagerTest{}) }

func (t *BucketManagerTest) SetUp(_ *TestInfo) {
	var err error

	t.fakeStorageServer, err = storage.CreateFakeStorageServer([]fakestorage.Object{storage.GetTestFakeStorageObject()})

	AssertEq(nil, err)

	storageClient := &storage.Storageclient{Client: t.fakeStorageServer.Client()}
	t.bucketHandle, err = storageClient.BucketHandle(TestBucketName)

	AssertEq(nil, err)
	AssertNe(nil, t.bucketHandle)
}

func (t *BucketManagerTest) TearDown() {
	t.fakeStorageServer.Stop()
}

func (t *BucketManagerTest) TestNewBucketManagerMethod() {
	storageClient := &storage.Storageclient{Client: t.fakeStorageServer.Client()}
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
		EnableStorageClientLibrary:         true,
	}

	bm := NewBucketManager(bucketConfig, nil, storageClient)

	ExpectNe(nil, bm)
}

func (t *BucketManagerTest) TestSetupGcsBucketWhenEnableStorageClientLibraryIsTrue() {
	var bm bucketManager
	bm.storageHandle = &storage.Storageclient{Client: t.fakeStorageServer.Client()}
	bm.config.EnableStorageClientLibrary = true
	bm.config.DebugGCS = true

	bucket, err := bm.SetUpGcsBucket(context.Background(), TestBucketName)

	ExpectNe(nil, bucket)
	ExpectEq(nil, err)
}

func (t *BucketManagerTest) TestSetupGcsBucketWhenEnableStorageClientLibraryIsFalse() {
	var bm bucketManager
	bm.storageHandle = &storage.Storageclient{Client: t.fakeStorageServer.Client()}
	bm.config.EnableStorageClientLibrary = false
	bm.config.BillingProject = "BillingProject"
	bm.conn = &Connection{
		wrapped: gcsfake.NewConn(timeutil.RealClock()),
	}

	bucket, err := bm.SetUpGcsBucket(context.Background(), "fake@bucket")

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
		EnableStorageClientLibrary:         true,
	}
	ctx := context.Background()
	bm.storageHandle = &storage.Storageclient{Client: t.fakeStorageServer.Client()}
	bm.config = bucketConfig
	bm.gcCtx = ctx
	bm.conn = &Connection{
		wrapped: gcsfake.NewConn(timeutil.RealClock()),
	}

	bucket, err := bm.SetUpBucket(context.Background(), TestBucketName)

	ExpectNe(nil, bucket.Syncer)
	ExpectEq(nil, err)
}
