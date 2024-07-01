// Copyright 2023 Google Inc. All Rights Reserved.
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

package storageutil

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/auth"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const urlSchemeSeparator = "://"

type StorageClientConfig struct {
	/** Common client parameters. */

	// ClientProtocol decides the go-sdk client to create.
	ClientProtocol    cfg.Protocol
	UserAgent         string
	CustomEndpoint    *url.URL
	KeyFile           string
	TokenUrl          string
	ReuseTokenFromUrl bool
	MaxRetrySleep     time.Duration
	RetryMultiplier   float64

	/** HTTP client parameters. */
	MaxConnsPerHost            int
	MaxIdleConnsPerHost        int
	HttpClientTimeout          time.Duration
	ExperimentalEnableJsonRead bool
	AnonymousAccess            bool

	/** Grpc client parameters. */
	GrpcConnPoolSize int

	// Enabling new API flow for HNS bucket.
	EnableHNS config.EnableHNS
}

func CreateHttpClient(storageClientConfig *StorageClientConfig) (httpClient *http.Client, err error) {
	var transport *http.Transport
	// Using http1 makes the client more performant.
	if storageClientConfig.ClientProtocol == cfg.HTTP1 {
		transport = &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			MaxConnsPerHost:     storageClientConfig.MaxConnsPerHost,
			MaxIdleConnsPerHost: storageClientConfig.MaxIdleConnsPerHost,
			// This disables HTTP/2 in transport.
			TLSNextProto: make(
				map[string]func(string, *tls.Conn) http.RoundTripper,
			),
		}
	} else {
		// For http2, change in MaxConnsPerHost doesn't affect the performance.
		transport = &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
			MaxConnsPerHost:   storageClientConfig.MaxConnsPerHost,
			ForceAttemptHTTP2: true,
		}
	}

	if storageClientConfig.AnonymousAccess {
		// UserAgent will not be added if authentication is disabled.
		// Bypassing authentication prevents the creation of an HTTP transport
		// because it requires a token source.
		// Setting a dummy token would conflict with the "WithoutAuthentication" option.
		// While the "WithUserAgent" option could set a custom User-Agent, it's incompatible
		// with the "WithHTTPClient" option, preventing the direct injection of a user agent
		// when authentication is skipped.
		httpClient = &http.Client{
			Timeout: storageClientConfig.HttpClientTimeout,
		}
	} else {
		var tokenSrc oauth2.TokenSource
		tokenSrc, err = CreateTokenSource(storageClientConfig)
		if err != nil {
			err = fmt.Errorf("while fetching tokenSource: %w", err)
			return
		}

		// Custom http client for Go Client.
		httpClient = &http.Client{
			Transport: &oauth2.Transport{
				Base:   transport,
				Source: tokenSrc,
			},
			Timeout: storageClientConfig.HttpClientTimeout,
		}
		// Setting UserAgent through RoundTripper middleware
		httpClient.Transport = &userAgentRoundTripper{
			wrapped:   httpClient.Transport,
			UserAgent: storageClientConfig.UserAgent,
		}
	}
	return httpClient, err
}

// It creates the token-source from the provided
// key-file or using ADC search order (https://cloud.google.com/docs/authentication/application-default-credentials#order).
func CreateTokenSource(storageClientConfig *StorageClientConfig) (tokenSrc oauth2.TokenSource, err error) {
	return auth.GetTokenSource(context.Background(), storageClientConfig.KeyFile, storageClientConfig.TokenUrl, storageClientConfig.ReuseTokenFromUrl)
}

// StripScheme strips the scheme part of given url.
func StripScheme(url string) string {
	if strings.Contains(url, urlSchemeSeparator) {
		url = strings.SplitN(url, urlSchemeSeparator, 2)[1]
	}
	return url
}
