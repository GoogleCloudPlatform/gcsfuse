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

// NewStorageHandle returns the handle of Go storage client containing
// customized http client. We can configure the http client using the
// storageClientConfig parameter.
func NewStorageHandle(ctx context.Context,
	clientConfig storageClientConfig) (sh *storageClient, err error) {
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

	sc.SetRetry(
		storage.WithBackoff(gax.Backoff{
			Max:        clientConfig.maxRetryDuration,
			Multiplier: clientConfig.retryMultiplier,
		}),
		storage.WithPolicy(storage.RetryAlways))

	sh = &storageClient{client: sc}
	return
}
