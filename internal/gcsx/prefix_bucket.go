// Copyright 2015 Google LLC
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
	"strings"
	"unicode/utf8"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
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
	wrapped gcs.Bucket) (b gcs.Bucket, err error) {
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
	wrapped gcs.Bucket
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

func (b *prefixBucket) BucketType() gcs.BucketType {
	return b.wrapped.BucketType()
}

func (b *prefixBucket) NewReaderWithReadHandle(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rd gcs.StorageReader, err error) {
	// Modify the request and call through.
	mReq := new(gcs.ReadObjectRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	rd, err = b.wrapped.NewReaderWithReadHandle(ctx, mReq)
	return
}

func (b *prefixBucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {
	// Modify the request and call through.
	mReq := new(gcs.CreateObjectRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	o, err = b.wrapped.CreateObject(ctx, mReq)

	// Modify the returned object.
	if o != nil {
		o.Name = b.localName(o.Name)
	}

	return
}

func (b *prefixBucket) CreateObjectChunkWriter(ctx context.Context, req *gcs.CreateObjectRequest, chunkSize int, callBack func(bytesUploadedSoFar int64)) (gcs.Writer, error) {
	// Modify the request and call through.
	mReq := new(gcs.CreateObjectRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	wc, err := b.wrapped.CreateObjectChunkWriter(ctx, mReq, chunkSize, callBack)
	if err != nil {
		return nil, err
	}

	return wc, err
}

func (b *prefixBucket) CreateAppendableObjectWriter(ctx context.Context, req *gcs.CreateObjectChunkWriterRequest) (gcs.Writer, error) {
	// Modify the request and call through.
	mReq := new(gcs.CreateObjectChunkWriterRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	wc, err := b.wrapped.CreateAppendableObjectWriter(ctx, mReq)
	if err != nil {
		return nil, err
	}

	return wc, err
}

func (b *prefixBucket) FinalizeUpload(ctx context.Context, w gcs.Writer) (o *gcs.MinObject, err error) {
	o, err = b.wrapped.FinalizeUpload(ctx, w)
	// Modify the returned object.
	if o != nil {
		o.Name = b.localName(o.Name)
	}
	return
}

func (b *prefixBucket) FlushPendingWrites(ctx context.Context, w gcs.Writer) (o *gcs.MinObject, err error) {
	o, err = b.wrapped.FlushPendingWrites(ctx, w)
	// Modify the returned object.
	if o != nil {
		o.Name = b.localName(o.Name)
	}
	return
}

func (b *prefixBucket) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (o *gcs.Object, err error) {
	// Modify the request and call through.
	mReq := new(gcs.CopyObjectRequest)
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
	req *gcs.ComposeObjectsRequest) (o *gcs.Object, err error) {
	// Modify the request and call through.
	mReq := new(gcs.ComposeObjectsRequest)
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
	req *gcs.StatObjectRequest) (m *gcs.MinObject, e *gcs.ExtendedObjectAttributes, err error) {
	// Modify the request and call through.
	mReq := new(gcs.StatObjectRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	m, e, err = b.wrapped.StatObject(ctx, mReq)

	// Modify the returned object.
	if m != nil {
		m.Name = b.localName(m.Name)
	}

	return
}

func (b *prefixBucket) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (l *gcs.Listing, err error) {
	// Modify the request and call through.
	mReq := new(gcs.ListObjectsRequest)
	*mReq = *req
	mReq.Prefix = b.prefix + mReq.Prefix

	l, err = b.wrapped.ListObjects(ctx, mReq)

	// Modify the returned listing.
	if l != nil {
		for _, o := range l.MinObjects {
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
	req *gcs.UpdateObjectRequest) (o *gcs.Object, err error) {
	// Modify the request and call through.
	mReq := new(gcs.UpdateObjectRequest)
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
	req *gcs.DeleteObjectRequest) (err error) {
	// Modify the request and call through.
	mReq := new(gcs.DeleteObjectRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	err = b.wrapped.DeleteObject(ctx, mReq)
	return
}

func (b *prefixBucket) MoveObject(ctx context.Context, req *gcs.MoveObjectRequest) (*gcs.Object, error) {
	// Modify the request and call through.
	mReq := new(gcs.MoveObjectRequest)
	*mReq = *req
	mReq.SrcName = b.wrappedName(req.SrcName)
	mReq.DstName = b.wrappedName(req.DstName)

	o, err := b.wrapped.MoveObject(ctx, mReq)

	// Modify the returned object.
	if o != nil {
		o.Name = b.localName(o.Name)
	}

	return o, err
}

func (b *prefixBucket) DeleteFolder(ctx context.Context, folderName string) (err error) {
	mFolderName := b.wrappedName(folderName)
	return b.wrapped.DeleteFolder(ctx, mFolderName)
}

func (b *prefixBucket) GetFolder(ctx context.Context, req *gcs.GetFolderRequest) (folder *gcs.Folder, err error) {
	mReq := new(gcs.GetFolderRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	f, err := b.wrapped.GetFolder(ctx, mReq)

	// Modify the returned folder.
	if f != nil {
		f.Name = b.localName(f.Name)
	}

	return f, err
}

func (b *prefixBucket) CreateFolder(ctx context.Context, folderName string) (*gcs.Folder, error) {
	mFolderName := b.wrappedName(folderName)
	f, err := b.wrapped.CreateFolder(ctx, mFolderName)

	// Modify the returned folder.
	if f != nil {
		f.Name = b.localName(mFolderName)
	}

	return f, err
}

func (b *prefixBucket) RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (*gcs.Folder, error) {
	mFolderName := b.wrappedName(folderName)
	mDestinationFolderId := b.wrappedName(destinationFolderId)
	f, err := b.wrapped.RenameFolder(ctx, mFolderName, mDestinationFolderId)

	// Modify the returned folder.
	if f != nil {
		f.Name = b.localName(f.Name)
	}

	return f, err
}

func (b *prefixBucket) NewMultiRangeDownloader(
	ctx context.Context, req *gcs.MultiRangeDownloaderRequest) (mrd gcs.MultiRangeDownloader, err error) {
	// Modify the request and call through.
	mReq := new(gcs.MultiRangeDownloaderRequest)
	*mReq = *req
	mReq.Name = b.wrappedName(req.Name)

	mrd, err = b.wrapped.NewMultiRangeDownloader(ctx, mReq)
	return
}

func (b *prefixBucket) GCSName(object *gcs.MinObject) string {
	return b.wrappedName(b.wrapped.GCSName(object))
}
