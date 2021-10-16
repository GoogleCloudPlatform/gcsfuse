// Copyright 2020 Google Inc. All Rights Reserved.
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

package readers

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/oauth2/google"
)

// Vendor reader depends on "github.com/jacobsa/gcloud/gcs"
type vendorClient struct {
	ctx    context.Context
	bucket gcs.Bucket
}

func NewVendorClient(ctx context.Context, protocol string, connections int, bucketName string) (*vendorClient, error) {
	tokenSrc, err := google.DefaultTokenSource(
		ctx,
		gcs.Scope_FullControl,
	)
	if err != nil {
		return nil, err
	}
	endpoint, _ := url.Parse("https://www.googleapis.com:443")
	config := &gcs.ConnConfig{
		Url:         endpoint,
		TokenSource: tokenSrc,
		UserAgent:   "gcsfuse/dev Benchmark (concurrent_read)",
		Transport:   getTransport(protocol, connections),
	}
	conn, err := gcs.NewConn(config)
	if err != nil {
		return nil, err
	}
	bucket, err := conn.OpenBucket(
		ctx,
		&gcs.OpenBucketOptions{
			Name: bucketName,
		},
	)
	if err != nil {
		panic(fmt.Errorf("Cannot open bucket %q: %w", bucketName, err))
	}
	return &vendorClient{ctx, bucket}, nil
}

func (c *vendorClient) NewReader(objectName string) (io.ReadCloser, error) {
	return c.bucket.NewReader(
		c.ctx,
		&gcs.ReadObjectRequest{
			Name: objectName,
		},
	)
}

func getTransport(protocol string, connections int) (transport *http.Transport) {
	switch protocol {
	case "HTTP/1.1":
		return &http.Transport{
			MaxConnsPerHost: connections,
			// This disables HTTP/2 in the transport.
			TLSNextProto: make(
				map[string]func(string, *tls.Conn) http.RoundTripper,
			),
		}
	case "HTTP/2":
		return http.DefaultTransport.(*http.Transport).Clone()
	default:
		panic(fmt.Errorf("Unsupported protocol: %q", protocol))
	}
}
