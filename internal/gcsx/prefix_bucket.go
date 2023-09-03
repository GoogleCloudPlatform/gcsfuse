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
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/googlecloudplatform/gcsfuse/internal/storage/bucket"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/object"
	"golang.org/x/net/context"
)

// NewPrefixBucket creates a view on the wrapped bucket that pretends as if only
// the objects whose names contain the supplied string as a strict prefix exist,
// and that strips the prefix from the names of those objects before exposing them.
//
// In order to preserve the invariant that object names are valid UTF-8, prefix
// must be valid UTF-8.
func NewPrefixBucket(
		prefix string,
		wrapped bucket.Bucket) (b bucket.Bucket, err error) {
	if !utf8.ValidString(prefix) {
		err = errors.New("prefix is not valid UTF-8")
		return
	}

	b = &prefixBucket{
		prefix:  prefix,
		wrapped: wrapped,
	}

	return
}

type prefixBucket struct {
	prefix  string
	wrapped bucket.Bucket
}

func (b *prefixBucket) wrappedName(n string) string {
	return b.prefix + n
}

func (b *prefixBucket) localName(n string) string {
	return strings.TrimPrefix(n, b.prefix)
}

func (b *prefixBucket) Name() string {
	return b.wrapped.Name()
}

func (b *prefixBucket) NewReader(
		ctx context.Context,
		req *object.ReadObjectRequest) (rc io.ReadCloser, err error) {
	// Modify the request and call through.
	mReq := new(object.ReadObjectRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	rc, err = b.wrapped.NewReader(ctx, mReq)
	return
}

func (b *prefixBucket) CreateObject(
		ctx context.Context,
		req *object.CreateObjectRequest) (o *object.Object, err error) {
	// Modify the request and call through.
	mReq := new(object.CreateObjectRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	o, err = b.wrapped.CreateObject(ctx, mReq)

	// Modify the returned object.
	if o != nil {
		o.Name = b.localName(o.Name)
	}

	return
}

func (b *prefixBucket) CopyObject(
		ctx context.Context,
		req *object.CopyObjectRequest) (o *object.Object, err error) {
	// Modify the request and call through.
	mReq := new(object.CopyObjectRequest)
	*mReq = *req
	mReq.SrcName = b.wrappedName(req.SrcName)
	mReq.DstName = b.wrappedName(req.DstName)

	o, err = b.wrapped.CopyObject(ctx, mReq)

	// Modify the returned object.
	if o != nil {
		o.Name = b.localName(o.Name)
	}

	return
}

func (b *prefixBucket) ComposeObjects(
		ctx context.Context,
		req *object.ComposeObjectsRequest) (o *object.Object, err error) {
	// Modify the request and call through.
	mReq := new(object.ComposeObjectsRequest)
	*mReq = *req
	mReq.DstName = b.wrappedName(req.DstName)

	mReq.Sources = nil
	for _, s := range req.Sources {
		s.Name = b.wrappedName(s.Name)
		mReq.Sources = append(mReq.Sources, s)
	}

	o, err = b.wrapped.ComposeObjects(ctx, mReq)

	// Modify the returned object.
	if o != nil {
		o.Name = b.localName(o.Name)
	}

	return
}

func (b *prefixBucket) StatObject(
		ctx context.Context,
		req *object.StatObjectRequest) (o *object.Object, err error) {
	// Modify the request and call through.
	mReq := new(object.StatObjectRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	o, err = b.wrapped.StatObject(ctx, mReq)

	// Modify the returned object.
	if o != nil {
		o.Name = b.localName(o.Name)
	}

	return
}

func (b *prefixBucket) ListObjects(
		ctx context.Context,
		req *object.ListObjectsRequest) (l *object.Listing, err error) {
	// Modify the request and call through.
	mReq := new(object.ListObjectsRequest)
	*mReq = *req
	mReq.Prefix = b.prefix + mReq.Prefix

	l, err = b.wrapped.ListObjects(ctx, mReq)

	// Modify the returned listing.
	if l != nil {
		for _, o := range l.Objects {
			o.Name = b.localName(o.Name)
		}

		for i, n := range l.CollapsedRuns {
			l.CollapsedRuns[i] = strings.TrimPrefix(n, b.prefix)
		}
	}

	return
}

func (b *prefixBucket) UpdateObject(
		ctx context.Context,
		req *object.UpdateObjectRequest) (o *object.Object, err error) {
	// Modify the request and call through.
	mReq := new(object.UpdateObjectRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	o, err = b.wrapped.UpdateObject(ctx, mReq)

	// Modify the returned object.
	if o != nil {
		o.Name = b.localName(o.Name)
	}

	return
}

func (b *prefixBucket) DeleteObject(
		ctx context.Context,
		req *object.DeleteObjectRequest) (err error) {
	// Modify the request and call through.
	mReq := new(object.DeleteObjectRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	err = b.wrapped.DeleteObject(ctx, mReq)
	return
}
