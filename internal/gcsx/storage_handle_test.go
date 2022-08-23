package gcsx

import (
	"context"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func verifyStorageHandle(t *testing.T, handle *storageClient, err error) {
	if err != nil {
		t.Errorf("Handle creation failure")
	}
	if nil == handle {
		t.Fatalf("Storage handle is null")
	}
	if nil == handle.client {
		t.Fatalf("Storage client handle is null")
	}
}

func TestNewStorageHandleHttp2Disabled(t *testing.T) {
	sc := storageClientConfig{disableHTTP2: true,
		maxConnsPerHost:     10,
		maxIdleConnsPerHost: 100,
		tokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{}),
		timeOut:             800 * time.Millisecond}

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	verifyStorageHandle(t, handleCreated, err)
}

func TestNewStorageHandleHttp2Enabled(t *testing.T) {
	sc := storageClientConfig{disableHTTP2: false,
		maxConnsPerHost:     10,
		maxIdleConnsPerHost: 100,
		tokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{}),
		timeOut:             800 * time.Millisecond}

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	verifyStorageHandle(t, handleCreated, err)
}

func TestNewStorageHandleWithZeroMaxConnsPerHost(t *testing.T) {
	sc := storageClientConfig{disableHTTP2: true,
		maxConnsPerHost:     0,
		maxIdleConnsPerHost: 100,
		tokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{}),
		timeOut:             800 * time.Millisecond}

	handleCreated, err := NewStorageHandle(context.Background(), sc)

	verifyStorageHandle(t, handleCreated, err)
}
