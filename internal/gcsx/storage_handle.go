package gcsx

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

type StorageHandle interface {
	BucketHandle(bucketName string) (bh *bucketHandle, err error)
}

type storageHandle struct {
	client *storage.Client
}

func NewStorageHandle(ctx context.Context, tokenSrc oauth2.TokenSource) (sh *storageHandle, err error) {
	var storageClient *storage.Client

	// Creating client through Go Storage Client Library for the storageClient parameter of bucket.
	var tr *http.Transport

	// Choosing between HTTP1 and HTTP2.
	if true {
		tr = &http.Transport{
			MaxConnsPerHost:     10,
			MaxIdleConnsPerHost: 100,
			// This disables HTTP/2 in transport.
			TLSNextProto: make(
				map[string]func(string, *tls.Conn) http.RoundTripper,
			),
		}
	} else {
		tr = &http.Transport{
			DisableKeepAlives: true,
			MaxConnsPerHost:   10, // Not affecting the performance when HTTP 2.0 is enabled.
			ForceAttemptHTTP2: true,
		}
	}

	// Custom http Client for Go Client.
	httpClient := &http.Client{Transport: &oauth2.Transport{
		Base:   tr,
		Source: tokenSrc,
	},
		Timeout: 800 * time.Millisecond,
	}

	// check retry strategy should be enabled here.
	storageClient, err = storage.NewClient(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		err = fmt.Errorf("go storage client creation: %v", err)
	}
	sh = &storageHandle{storageClient}

	return
}

func (sh *storageHandle) BucketHandle(bucketName string) (bh *bucketHandle,
	err error) {
	bh = &bucketHandle{bucket: sh.client.Bucket(bucketName)}

	// TODO: Add retry strategy
	// We need to make a call to gcs ListObjects method to check if the bucket
	// name is valid and the client has access
	return
}
