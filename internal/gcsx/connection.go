// Copyright 2015 Google Inc. All Rights Reserved.
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

package gcsx

import (
	"fmt"
	"strings"
	"syscall"

	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Connection struct {
	wrapped gcs.Conn
	gcs     *storage.Client
}

func NewConnection(cfg *gcs.ConnConfig) (c *Connection, err error) {
	wrapped, err := gcs.NewConn(cfg)
	if err != nil {
		err = fmt.Errorf("Cannot create Conn: %w", err)
		return
	}

	gcs, err := storage.NewClient(
		context.Background(),
		option.WithEndpoint(fmt.Sprintf("%s/storage/v1/", cfg.Url.String())),
		option.WithTokenSource(cfg.TokenSource))
	if err != nil {
		err = fmt.Errorf("Cannot create GCS client: %w", err)
		return
	}

	c = &Connection{
		wrapped: wrapped,
		gcs:     gcs,
	}
	return
}

func (c *Connection) OpenBucket(
	ctx context.Context,
	options *gcs.OpenBucketOptions) (b gcs.Bucket, err error) {
	b, err = c.wrapped.OpenBucket(ctx, options)

	// The gcs.Conn.OpenBucket returns converted errors without the underlying
	// googleapi.Error, which is impossible to use errors.As to match the error
	// type. To interpret the errors in syscall, here we use string matching.
	if err != nil {
		if strings.Contains(err.Error(), "Bad credentials") {
			return b, fmt.Errorf("Bad credentials for bucket %q: %w", options.Name, syscall.EACCES)
		}
		if strings.Contains(err.Error(), "Unknown bucket") {
			return b, fmt.Errorf("Unknown bucket %q: %w", options.Name, syscall.ENOENT)
		}
	}

	return
}

func (c *Connection) ListBuckets(
	ctx context.Context,
	projectId string) (names []string, err error) {
	it := c.gcs.Buckets(ctx, projectId)
	for {
		bucket, nextErr := it.Next()
		switch nextErr {
		case nil:
			names = append(names, bucket.Name)
		case iterator.Done:
			return
		default:
			err = nextErr
			return
		}
	}
}
