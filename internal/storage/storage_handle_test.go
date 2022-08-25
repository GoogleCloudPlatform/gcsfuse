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
