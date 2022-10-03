package gcsx

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/jacobsa/gcloud/gcs"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/reqtrace"
	"google.golang.org/api/googleapi"
)

func TestFlags(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type BucketManagerTest struct {
	fakeStorageServer *fakestorage.Server
	bucketHandle      *storage.BucketHandle
}

type fakeStorageHandle struct {
	fakeClient *storage.Storageclient
}

type fakeConn struct {
	fakeClient          *http.Client
	fakeUrl             *url.URL
	fakeUserAgent       string
	fakeMaxBackoffSleep time.Duration
	fakeDebugLogger     *log.Logger
}

func (c *fakeConn) OpenBucket(
	ctx context.Context,
	options *gcs.OpenBucketOptions) (b gcs.Bucket, err error) {
	b = gcs.Newbukcet(c.fakeClient, c.fakeUrl, c.fakeUserAgent, options.Name, options.BillingProject)

	// Enable retry loops if requested.
	// Enable retry loops if requested.
	if c.fakeMaxBackoffSleep > 0 {
		// TODO(jacobsa): Show the retries as distinct spans in the trace.
		b = gcs.NewRetryBucket(c.fakeMaxBackoffSleep, b)
	}

	// Enable tracing if appropriate.
	if reqtrace.Enabled() {
		b = gcs.GetWrappedWithReqtraceBucket(b)
	}

	// Print debug output if requested.
	if c.fakeDebugLogger != nil {
		b = gcs.NewDebugBucket(b, c.fakeDebugLogger)
	}

	// Attempt to make an innocuous request to the bucket, snooping for HTTP 403
	// errors that indicate bad credentials. This lets us warn the user early in
	// the latter case, with a more helpful message than just "HTTP 403
	// Forbidden". Similarly for bad bucket names that don't collide with another
	// bucket.
	_, err = b.ListObjects(ctx, &gcs.ListObjectsRequest{MaxResults: 1})

	var apiError *googleapi.Error
	if errors.As(err, &apiError) {
		switch apiError.Code {
		case http.StatusForbidden:
			err = fmt.Errorf(
				"Bad credentials for bucket %q. Check the bucket name and your "+
					"credentials.",
				b.Name())

			return

		case http.StatusNotFound:
			err = fmt.Errorf("Unknown bucket %q", b.Name())
			return
		}
	}

	// Otherwise, don't interfere.
	err = nil

	return
}

func (f fakeStorageHandle) BucketHandle(bucketName string) (bh *storage.BucketHandle, err error) {
	storageBucketHandle := f.fakeClient.Client.Bucket(bucketName)
	_, err = storageBucketHandle.Attrs(context.Background())
	if err != nil {
		return
	}

	bh = &storage.BucketHandle{BucketObj: storageBucketHandle}
	return
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
	storageClient := &storage.Storageclient{Client: t.fakeStorageServer.Client()}
	fakeConnObj := fakeConn{
		fakeClient:          nil,
		fakeUrl:             nil,
		fakeUserAgent:       "fakeuserAgent",
		fakeMaxBackoffSleep: time.Second,
		fakeDebugLogger:     nil,
	}

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
		wrapped: &fakeConnObj,
	}

	bm := NewBucketManager(bucketConfig, &connection, storageClient)
	ExpectNe(bm, nilValue)
}

func (t *BucketManagerTest) TestSetUpGcsBucketMethod() {
	var bm bucketManager
	var nilBucket *storage.BucketHandle = nil

	storageClient := &storage.Storageclient{Client: t.fakeStorageServer.Client()}
	fakeStorageHandleObj := fakeStorageHandle{
		fakeClient: storageClient,
	}
	bm.storageHandle = fakeStorageHandleObj

	Bucket, err := bm.SetUpGcsBucket(context.Background(), TestBucketName)

	ExpectNe(Bucket, nilBucket)
	ExpectEq(err, nil)
}

func (t *BucketManagerTest) TestSetUpBucketMethod() {
	var bm bucketManager
	nilSync := NewSyncerBucket(0, "", nil)

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

	storageClient := &storage.Storageclient{Client: t.fakeStorageServer.Client()}
	fakeStorageHandleObj := fakeStorageHandle{
		fakeClient: storageClient,
	}
	ctx := context.Background()

	bm.storageHandle = fakeStorageHandleObj
	bm.config = bucketConfig
	bm.gcCtx = ctx

	Bucket, err := bm.SetUpBucket(context.Background(), TestBucketName)
	ExpectNe(Bucket.Syncer, nilSync.Syncer)
	ExpectEq(err, nil)
}
