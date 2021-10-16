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
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// Google reader depends on "cloud.google.com/go/storage"
type googleClient struct {
	ctx    context.Context
	bucket *storage.BucketHandle
}

func NewGoogleClient(ctx context.Context, protocol string, connections int, bucketName string) (*googleClient, error) {
	client, err := getStorageClient(ctx, protocol, connections)
	if err != nil {
		return nil, err
	}
	bucket := client.Bucket(bucketName)
	return &googleClient{ctx, bucket}, nil
}

func (c *googleClient) NewReader(objectName string) (io.ReadCloser, error) {
	return c.bucket.Object(objectName).NewReader(c.ctx)
}

func getStorageClient(ctx context.Context, protocol string, connections int) (*storage.Client, error) {
	tokenSrc, err := google.DefaultTokenSource(ctx, gcs.Scope_FullControl)
	if err != nil {
		return nil, err
	}
	return storage.NewClient(
		ctx,
		option.WithUserAgent(userAgent),
		option.WithHTTPClient(&http.Client{
			Transport: &oauth2.Transport{
				Base:   getTransport(protocol, connections),
				Source: tokenSrc,
			},
		}),
	)
}
