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
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
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

type StorageClientConfig struct {
	DisableHTTP2        bool
	MaxConnsPerHost     int
	MaxIdleConnsPerHost int
	TokenSrc            oauth2.TokenSource
	HttpClientTimeout   time.Duration
	MaxRetryDuration    time.Duration
	RetryMultiplier     float64
	UserAgent           string
}

// NewStorageHandle returns the handle of Go storage client containing
// customized http client. We can configure the http client using the
// storageClientConfig parameter.
func NewStorageHandle(ctx context.Context, clientConfig StorageClientConfig) (sh StorageHandle, err error) {
	var transport *http.Transport
	// Disabling the http2 makes the client more performant.
	if clientConfig.DisableHTTP2 {
		transport = &http.Transport{
			MaxConnsPerHost:     clientConfig.MaxConnsPerHost,
			MaxIdleConnsPerHost: clientConfig.MaxIdleConnsPerHost,
			// This disables HTTP/2 in transport.
			TLSNextProto: make(
				map[string]func(string, *tls.Conn) http.RoundTripper,
			),
		}
	} else {
		// For http2, change in MaxConnsPerHost doesn't affect the performance.
		transport = &http.Transport{
			DisableKeepAlives: true,
			MaxConnsPerHost:   clientConfig.MaxConnsPerHost,
			ForceAttemptHTTP2: true,
		}
	}

	// Custom http client for Go Client.
	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Base:   transport,
			Source: clientConfig.TokenSrc,
		},
		Timeout: clientConfig.HttpClientTimeout,
	}

	// Setting UserAgent through RoundTripper middleware
	httpClient.Transport = &userAgentRoundTripper{
		wrapped:   httpClient.Transport,
		UserAgent: clientConfig.UserAgent,
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
			Max:        clientConfig.MaxRetryDuration,
			Multiplier: clientConfig.RetryMultiplier,
		}),
		storage.WithErrorFunc(storageutil.ShouldRetry))

	sh = &storageClient{client: sc}
	return
}

func (sh *storageClient) BucketHandle(bucketName string) (bh *bucketHandle, err error) {
	storageBucketHandle := sh.client.Bucket(bucketName)
	obj, err := storageBucketHandle.Attrs(context.Background())
	if err != nil {
		return
	}

	bh = &bucketHandle{bucket: storageBucketHandle, bucketName: obj.Name}
	return
}
