// Copyright 2022 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googleapis/gax-go/v2"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

type StorageHandle interface {
	BucketHandle(bucketName string) (bh *bucketHandle, err error)
}

type storageClient struct {
	client *storage.Client
}

type storageClientConfig struct {
	disableHTTP2        bool
	maxConnsPerHost     int
	maxIdleConnsPerHost int
	tokenSrc            oauth2.TokenSource
	timeOut             time.Duration
	maxRetryDuration    time.Duration
	retryMultiplier     float64
}

// NewStorageHandle returns the handle of Go storage client containing
// customized http client. We can configure the http client using the
// storageClientConfig parameter.
func NewStorageHandle(ctx context.Context, clientConfig storageClientConfig) (sh StorageHandle, err error) {
	var transport *http.Transport
	// Disabling the http2 makes the client more performant.
	if clientConfig.disableHTTP2 {
		transport = &http.Transport{
			MaxConnsPerHost:     clientConfig.maxConnsPerHost,
			MaxIdleConnsPerHost: clientConfig.maxIdleConnsPerHost,
			// This disables HTTP/2 in transport.
			TLSNextProto: make(
				map[string]func(string, *tls.Conn) http.RoundTripper,
			),
		}
	} else {
		// For http2, change in MaxConnsPerHost doesn't affect the performance.
		transport = &http.Transport{
			DisableKeepAlives: true,
			MaxConnsPerHost:   clientConfig.maxConnsPerHost,
			ForceAttemptHTTP2: true,
		}
	}

	// Custom http client for Go Client.
	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Base:   transport,
			Source: clientConfig.tokenSrc,
		},
		Timeout: clientConfig.timeOut,
	}

	var sc *storage.Client
	sc, err = storage.NewClient(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		err = fmt.Errorf("go storage client creation failed: %w", err)
		return
	}

	// RetryAlways causes all operations to be retried when the service returns a transient error, regardless of
	// idempotency considerations.
	sc.SetRetry(
		storage.WithBackoff(gax.Backoff{
			Max:        clientConfig.maxRetryDuration,
			Multiplier: clientConfig.retryMultiplier,
		}),
		storage.WithPolicy(storage.RetryAlways))

	sh = &storageClient{client: sc}
	return
}

func (sh *storageClient) BucketHandle(bucketName string) (bh *bucketHandle, err error) {
	storageBucketHandle := sh.client.Bucket(bucketName)
	_, err = storageBucketHandle.Attrs(context.Background())
	if err != nil {
		return
	}

	bh = &bucketHandle{bucket: storageBucketHandle}
	return
}
