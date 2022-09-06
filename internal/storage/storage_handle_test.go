package storage

import (
	"context"
	"testing"
	"time"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/oauth2"
)

const validBucketName string = "will-be-present-in-fake-server"
const invalidBucketName string = "will-not-be-present-in-fake-server"

func TestStorageHandle(t *testing.T) { RunTests(t) }

type StorageHandleTest struct {
}

func init() { RegisterTestSuite(&StorageHandleTest{}) }

func createFakeServer() (fakeServer *fakestorage.Server, err error) {
	fakeServer, err = fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: []fakestorage.Object{
			{
				ObjectAttrs: fakestorage.ObjectAttrs{
					BucketName: validBucketName,
				},
			},
		},
		Host: "127.0.0.1",
		Port: 8081,
	})
	return
}

func (t *StorageHandleTest) TestBucketHandleWhenBucketExists() {
	server, err := createFakeServer()
	AssertEq(nil, err)
	defer server.Stop()

	fakeClient := server.Client()
	fakeStorageClient := &storageClient{client: fakeClient}
	bucketHandle, err := fakeStorageClient.BucketHandle(validBucketName)

	AssertEq(nil, err)
	AssertNe(nil, bucketHandle)
}

func (t *StorageHandleTest) TestBucketHandleWhenBucketDoesNotExist() {
	server, err := createFakeServer()
	AssertEq(nil, err)
	defer server.Stop()

	fakeClient := server.Client()
	fakeStorageClient := &storageClient{client: fakeClient}
	bucketHandle, err := fakeStorageClient.BucketHandle(invalidBucketName)

	AssertNe(nil, err)
	AssertEq(nil, bucketHandle)
}

func getDefaultStorageClientConfig() (clientConfig storageClientConfig) {
	return storageClientConfig{
		disableHTTP2:        true,
		maxConnsPerHost:     10,
		maxIdleConnsPerHost: 100,
		tokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{}),
		timeOut:             800 * time.Millisecond,
		maxRetryDuration:    30 * time.Second,
		retryMultiplier:     2,
	}
}

func (t *StorageHandleTest) invokeAndVerifyStorageHandle(sc storageClientConfig) {
	handleCreated, err := NewStorageHandle(context.Background(), sc)
	AssertEq(nil, err)
	AssertNe(nil, handleCreated)
}

func (t *StorageHandleTest) TestNewStorageHandleHttp2Disabled() {
	sc := getDefaultStorageClientConfig() // by default http2 disabled

	t.invokeAndVerifyStorageHandle(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleHttp2Enabled() {
	sc := getDefaultStorageClientConfig()
	sc.disableHTTP2 = false

	t.invokeAndVerifyStorageHandle(sc)
}

func (t *StorageHandleTest) TestNewStorageHandleWithZeroMaxConnsPerHost() {
	sc := getDefaultStorageClientConfig()
	sc.maxConnsPerHost = 0

	t.invokeAndVerifyStorageHandle(sc)
}
