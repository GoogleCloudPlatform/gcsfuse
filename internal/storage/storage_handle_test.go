package storage

import (
	"context"
	"testing"
	"time"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"golang.org/x/oauth2"
)

const correctBucketName string = "will-be-present-in-fake-server"
const wrongBucketName string = "will-not-be-present-in-fake-server"

func createFakeServer() (fakeServer *fakestorage.Server, err error) {
	fakeServer, err = fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: []fakestorage.Object{
			{
				ObjectAttrs: fakestorage.ObjectAttrs{
					BucketName: correctBucketName,
				},
			},
		},
		Host: "127.0.0.1",
		Port: 8081,
	})
	return
}

func TestBucketHandleWhenBucketExists(t *testing.T) {
	server, err := createFakeServer()
	if err != nil {
		t.Fatalf("Server creation failed")
	}
	defer server.Stop()

	fakeClient := server.Client()
	fakeStorageClient := &StorageClient{client: fakeClient}
	bucketHandle, err := fakeStorageClient.BucketHandle(correctBucketName)

	if err != nil {
		t.Fatalf("BucketHandle creation failed")
	}
	if bucketHandle == nil {
		t.Errorf("BucketHandle should be non null")
	}
}

func TestBucketHandleWhenBucketDoesNotExist(t *testing.T) {
	server, err := createFakeServer()
	if err != nil {
		t.Fatalf("Server creation failed")
	}
	defer server.Stop()

	fakeClient := server.Client()
	fakeStorageClient := &StorageClient{client: fakeClient}
	bucketHandle, err := fakeStorageClient.BucketHandle(wrongBucketName)

	if err == nil || bucketHandle != nil {
		t.Fatalf("BucketHandle should be nil")
	}
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

func invokeAndVerifyStorageHandle(t *testing.T, sc storageClientConfig) {
	handleCreated, err := NewStorageHandle(context.Background(), sc)

	if err != nil {
		t.Errorf("Handle creation failure")
	}
	if handleCreated == nil {
		t.Fatalf("Storage handle is null")
	}
	if nil == handleCreated.client {
		t.Fatalf("Storage client handle is null")
	}
}

func TestNewStorageHandleHttp2Disabled(t *testing.T) {
	sc := getDefaultStorageClientConfig() // by default http2 disabled

	invokeAndVerifyStorageHandle(t, sc)
}

func TestNewStorageHandleHttp2Enabled(t *testing.T) {
	sc := getDefaultStorageClientConfig()
	sc.disableHTTP2 = false

	invokeAndVerifyStorageHandle(t, sc)
}

func TestNewStorageHandleWithZeroMaxConnsPerHost(t *testing.T) {
	sc := getDefaultStorageClientConfig()
	sc.maxConnsPerHost = 0

	invokeAndVerifyStorageHandle(t, sc)
}
