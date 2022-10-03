package gcsx

import (
	"context"
	"testing"
	"time"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/jacobsa/gcloud/gcs"
	. "github.com/jacobsa/ogletest"
)

func TestFlags(t *testing.T) { RunTests(t) }

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
	t.fakeStorageServer, err = storage.CreateFakeStorageServer([]fakestorage.Object{GetTestFakeStorageObject()})
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
	var nilValue *bucketManager = nil
	var s storage.StorageHandle
	var b gcs.Conn

	bucketConfig := BucketConfig{
		BillingProject:                     "BillingProject",
		OnlyDir:                            "OnlyDir",
		EgressBandwidthLimitBytesPerSecond: 7,
		OpRateLimitHz:                      11,
		StatCacheCapacity:                  100,
		StatCacheTTL:                       20 * time.Second,
		EnableMonitoring:                   true,
		debugGcs:                           true,
		AppendThreshold:                    2,
		TmpObjectPrefix:                    "TmpObjectPrefix",
	}
	connection := Connection{
		wrapped: b,
	}
	storageHandle := s

	bm := NewBucketManager(bucketConfig, &connection, storageHandle)
	ExpectNe(bm, nilValue)
}

func (t *BucketManagerTest) TestSetUpBucketMethod() {
	var bm BucketManager

	syncBucket, err := bm.SetUpBucket(context.Background(), "fake@bucket")

	ExpectNe(syncBucket, nil)
	ExpectEq(err, nil)
}
