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

package ratelimit

import (
	"io"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// Create a bucket that limits the rate at which it calls the wrapped bucket
// using opThrottle, and limits the bandwidth with which it reads from the
// wrapped bucket using egressThrottle.
func NewThrottledBucket(
	opThrottle Throttle,
	egressThrottle Throttle,
	wrapped gcs.Bucket) (b gcs.Bucket) {
	b = &throttledBucket{
		opThrottle:     opThrottle,
		egressThrottle: egressThrottle,
		wrapped:        wrapped,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// throttledBucket
////////////////////////////////////////////////////////////////////////

type throttledBucket struct {
	opThrottle     Throttle
	egressThrottle Throttle
	wrapped        gcs.Bucket
}

func (b *throttledBucket) Name() string {
	return b.wrapped.Name()
}

func (b *throttledBucket) NewReader(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rc io.ReadCloser, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	rc, err = b.wrapped.NewReader(ctx, req)
	if err != nil {
		return
	}

	// Wrap the result in a throttled layer.
	rc = &readerCloser{
		Reader: ThrottledReader(ctx, rc, b.egressThrottle),
		Closer: rc,
	}

	return
}

func (b *throttledBucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	o, err = b.wrapped.CreateObject(ctx, req)

	return
}

func (b *throttledBucket) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (o *gcs.Object, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	o, err = b.wrapped.CopyObject(ctx, req)

	return
}

func (b *throttledBucket) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (o *gcs.Object, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	o, err = b.wrapped.ComposeObjects(ctx, req)

	return
}

func (b *throttledBucket) StatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (o *gcs.Object, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	o, err = b.wrapped.StatObject(ctx, req)

	return
}

func (b *throttledBucket) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (listing *gcs.Listing, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	listing, err = b.wrapped.ListObjects(ctx, req)

	return
}

func (b *throttledBucket) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (o *gcs.Object, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	o, err = b.wrapped.UpdateObject(ctx, req)

	return
}

func (b *throttledBucket) DeleteObject(
	ctx context.Context,
	req *gcs.DeleteObjectRequest) (err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	err = b.wrapped.DeleteObject(ctx, req)

	return
}

////////////////////////////////////////////////////////////////////////
// readerCloser
////////////////////////////////////////////////////////////////////////

// An io.ReadCloser that forwards read requests to an io.Reader and close
// requests to an io.Closer.
type readerCloser struct {
	Reader io.Reader
	Closer io.Closer
}

func (rc *readerCloser) Read(p []byte) (n int, err error) {
	n, err = rc.Reader.Read(p)
	return
}

func (rc *readerCloser) Close() (err error) {
	err = rc.Closer.Close()
	return
}
