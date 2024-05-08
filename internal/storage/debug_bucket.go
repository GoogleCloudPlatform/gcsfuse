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

package storage

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"golang.org/x/net/context"
)

// Wrap the supplied bucket in a layer that prints debug messages.
func NewDebugBucket(
	wrapped gcs.Bucket) (b gcs.Bucket) {
	b = &debugBucket{
		wrapped: wrapped,
	}

	return
}

type debugBucket struct {
	wrapped gcs.Bucket

	nextRequestID uint64
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (b *debugBucket) mintRequestID() (id uint64) {
	id = atomic.AddUint64(&b.nextRequestID, 1) - 1
	return
}

func (b *debugBucket) requestLogf(
	id uint64,
	format string,
	v ...interface{}) {
	logger.Tracef("gcs: Req %#16x: %s", id, fmt.Sprintf(format, v...))
}

func (b *debugBucket) startRequest(
	format string,
	v ...interface{}) (id uint64, desc string, start time.Time) {
	start = time.Now()
	id = b.mintRequestID()
	desc = fmt.Sprintf(format, v...)

	b.requestLogf(id, "<- %s", desc)
	return
}

func (b *debugBucket) finishRequest(
	id uint64,
	desc string,
	start time.Time,
	err *error) {
	duration := time.Since(start)

	errDesc := "OK"
	if *err != nil {
		errDesc = (*err).Error()
	}

	b.requestLogf(id, "-> %s (%v): %s", desc, duration, errDesc)
}

////////////////////////////////////////////////////////////////////////
// Reader
////////////////////////////////////////////////////////////////////////

type debugReader struct {
	bucket    *debugBucket
	requestID uint64
	desc      string
	startTime time.Time
	wrapped   io.ReadCloser
}

func (dr *debugReader) Read(p []byte) (n int, err error) {
	n, err = dr.wrapped.Read(p)

	// Don't log EOF errors, which are par for the course.
	if err != nil && err != io.EOF {
		dr.bucket.requestLogf(dr.requestID, "-> Read error: %v", err)
	}

	return
}

func (dr *debugReader) Close() (err error) {
	defer dr.bucket.finishRequest(
		dr.requestID,
		dr.desc,
		dr.startTime,
		&err)

	err = dr.wrapped.Close()
	return
}

////////////////////////////////////////////////////////////////////////
// Bucket interface
////////////////////////////////////////////////////////////////////////

func (b *debugBucket) Name() string {
	return b.wrapped.Name()
}

func (b *debugBucket) Type() string {
	return b.wrapped.Type()
}

func (b *debugBucket) NewReader(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rc io.ReadCloser, err error) {
	id, desc, start := b.startRequest("Read(%q, %v)", req.Name, req.Range)

	// Call through.
	rc, err = b.wrapped.NewReader(ctx, req)
	if err != nil {
		b.finishRequest(id, desc, start, &err)
		return
	}

	// Return a special reader that prings debug info.
	rc = &debugReader{
		bucket:    b,
		requestID: id,
		desc:      desc,
		startTime: start,
		wrapped:   rc,
	}

	return
}

func (b *debugBucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {
	id, desc, start := b.startRequest("CreateObject(%q)", req.Name)
	defer b.finishRequest(id, desc, start, &err)

	o, err = b.wrapped.CreateObject(ctx, req)
	return
}

func (b *debugBucket) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (o *gcs.Object, err error) {
	id, desc, start := b.startRequest(
		"CopyObject(%q, %q)",
		req.SrcName,
		req.DstName)

	defer b.finishRequest(id, desc, start, &err)

	o, err = b.wrapped.CopyObject(ctx, req)
	return
}

func (b *debugBucket) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (o *gcs.Object, err error) {
	id, desc, start := b.startRequest(
		"ComposeObjects(%q)",
		req.DstName)

	defer b.finishRequest(id, desc, start, &err)

	o, err = b.wrapped.ComposeObjects(ctx, req)
	return
}

func (b *debugBucket) StatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (m *gcs.MinObject, e *gcs.ExtendedObjectAttributes, err error) {
	id, desc, start := b.startRequest("StatObject(%q)", req.Name)
	defer b.finishRequest(id, desc, start, &err)

	m, e, err = b.wrapped.StatObject(ctx, req)
	return
}

func (b *debugBucket) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (listing *gcs.Listing, err error) {
	id, desc, start := b.startRequest("ListObjects(%q)", req.Prefix)
	defer b.finishRequest(id, desc, start, &err)

	listing, err = b.wrapped.ListObjects(ctx, req)
	return
}

func (b *debugBucket) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (o *gcs.Object, err error) {
	id, desc, start := b.startRequest("UpdateObject(%q)", req.Name)
	defer b.finishRequest(id, desc, start, &err)

	o, err = b.wrapped.UpdateObject(ctx, req)
	return
}

func (b *debugBucket) DeleteObject(
	ctx context.Context,
	req *gcs.DeleteObjectRequest) (err error) {
	id, desc, start := b.startRequest("DeleteObject(%q)", req.Name)
	defer b.finishRequest(id, desc, start, &err)

	err = b.wrapped.DeleteObject(ctx, req)
	return
}
