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

type storageHandle struct {
	client *storage.Client
}

type storageClientConfig struct {
	disableHttp2        bool
	maxConnsPerHost     int
	maxIdleConnsPerHost int
	tokenSrc            oauth2.TokenSource
}

// GetStorageClientHandle returns the handle of Go storage client containing
// customized http client. We can configure the http client using the
// storageClientConfig parameter.
func GetStorageClientHandle(ctx context.Context,
	sc storageClientConfig) (sh *storageHandle, err error) {
	var storageClient *storage.Client

	var tr *http.Transport
	// Choosing between HTTP1 and HTTP2. HTTP2 is more performant.
	if sc.disableHttp2 {
		tr = &http.Transport{
			MaxConnsPerHost:     sc.maxConnsPerHost,
			MaxIdleConnsPerHost: sc.maxIdleConnsPerHost,
			// This disables HTTP/2 in transport.
			TLSNextProto: make(
				map[string]func(string, *tls.Conn) http.RoundTripper,
			),
		}
	} else {
		tr = &http.Transport{
			DisableKeepAlives: true,
			MaxConnsPerHost:   sc.maxConnsPerHost, // not affect the performance for http2
			ForceAttemptHTTP2: true,
		}
	}

	// Custom http Client for Go Client.
	httpClient := &http.Client{Transport: &oauth2.Transport{
		Base:   tr,
		Source: sc.tokenSrc,
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
