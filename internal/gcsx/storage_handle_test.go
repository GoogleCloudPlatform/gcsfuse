package gcsx

import (
	"context"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func invokeAndVerifyStorageHandle(t *testing.T, sc storageClientConfig) {
	handleCreated, err := NewStorageHandle(context.Background(), sc)

	if err != nil {
		t.Errorf("Handle creation failure")
	}
	if nil == handleCreated {
		t.Fatalf("Storage handle is null")
	}
	if nil == handleCreated.client {
		t.Fatalf("Storage client handle is null")
	}
}

func TestNewStorageHandleHttp2Disabled(t *testing.T) {
	sc := storageClientConfig{disableHTTP2: true,
		maxConnsPerHost:     10,
		maxIdleConnsPerHost: 100,
		tokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{}),
		timeOut:             800 * time.Millisecond}

	invokeAndVerifyStorageHandle(t, sc)
}

func TestNewStorageHandleHttp2Enabled(t *testing.T) {
	sc := storageClientConfig{disableHTTP2: false,
		maxConnsPerHost:     10,
		maxIdleConnsPerHost: 100,
		tokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{}),
		timeOut:             800 * time.Millisecond}

	invokeAndVerifyStorageHandle(t, sc)
}

func TestNewStorageHandleWithZeroMaxConnsPerHost(t *testing.T) {
	sc := storageClientConfig{disableHTTP2: true,
		maxConnsPerHost:     0,
		maxIdleConnsPerHost: 100,
		tokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{}),
		timeOut:             800 * time.Millisecond}

	invokeAndVerifyStorageHandle(t, sc)
}
