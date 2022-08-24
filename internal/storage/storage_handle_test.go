package storage

import (
	"context"
	"testing"
)

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

func TestBucketHandleIfDoesNotExist(t *testing.T) {
	handleCreated, _ := NewStorageHandle(context.Background(), getDefaultStorageClientConfig())
	bh, _ := handleCreated.BucketHandle("test")

	if bh != nil {
		t.Errorf("bucket handle is non-null")
	}
}
