package gcsx

import (
	"context"
	"testing"

	"golang.org/x/oauth2"
)

func TestNewStorageHandleHttp2Disabled(t *testing.T) {
	sc := storageClientConfig{disableHttp2: true,
		maxConnsPerHost:     10,
		maxIdleConnsPerHost: 100,
		tokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{})}

	handleCreated, err := GetStorageClientHandle(context.Background(), sc)

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

func TestNewStorageHandleHttp2Enabled(t *testing.T) {
	sc := storageClientConfig{disableHttp2: false,
		maxConnsPerHost:     10,
		maxIdleConnsPerHost: 100,
		tokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{})}

	handleCreated, err := GetStorageClientHandle(context.Background(), sc)

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

func TestNewStorageHandleWithZeroMaxConnsPerHost(t *testing.T) {
	sc := storageClientConfig{disableHttp2: true,
		maxConnsPerHost:     0,
		maxIdleConnsPerHost: 100,
		tokenSrc:            oauth2.StaticTokenSource(&oauth2.Token{})}

	handleCreated, err := GetStorageClientHandle(context.Background(), sc)

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
