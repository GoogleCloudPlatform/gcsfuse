// Copyright 2023 Google LLC
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

	"context"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// Create a bucket that limits the rate at which it calls the wrapped bucket
// using opThrottle, limits the bandwidth with which it reads from the
// wrapped bucket using egressThrottle, and limits the bandwidth with which it
// writes to the wrapped bucket using ingressThrottle.
func NewThrottledBucket(
	opThrottle Throttle,
	egressThrottle Throttle,
	ingressThrottle Throttle,
	wrapped gcs.Bucket) (b gcs.Bucket) {
	b = &throttledBucket{
		opThrottle:      opThrottle,
		egressThrottle:  egressThrottle,
		ingressThrottle: ingressThrottle,
		wrapped:         wrapped,
	}
	return
}

////////////////////////////////////////////////////////////////////////
// throttledBucket
////////////////////////////////////////////////////////////////////////

type throttledBucket struct {
	opThrottle      Throttle
	egressThrottle  Throttle
	ingressThrottle Throttle
	wrapped         gcs.Bucket
}

func (b *throttledBucket) Name() string {
	return b.wrapped.Name()
}

func (b *throttledBucket) BucketType() gcs.BucketType {
	return b.wrapped.BucketType()
}
func (b *throttledBucket) NewReaderWithReadHandle(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rd gcs.StorageReader, err error) {
	// Wait for permission to call through.

	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	rd, err = b.wrapped.NewReaderWithReadHandle(ctx, req)
	if err != nil {
		return
	}

	// Wrap the result in a throttled layer.
	rd = &throttledGCSReader{
		Reader: ThrottledReader(ctx, rd, b.egressThrottle),
		Closer: rd,
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

	// The wrapped bucket consumes req.Contents to perform the upload, so pacing
	// reads from it bounds upload bandwidth. Copy the request so the caller's
	// struct is not mutated.
	if req.Contents != nil {
		reqCopy := *req
		reqCopy.Contents = newThrottledUploadPayloadReader(ctx, req.Contents, b.ingressThrottle)
		req = &reqCopy
	}

	// Call through.
	o, err = b.wrapped.CreateObject(ctx, req)

	return
}

func (b *throttledBucket) CreateObjectChunkWriter(ctx context.Context, req *gcs.CreateObjectRequest, chunkSize int, callBack func(bytesUploadedSoFar int64)) (wc gcs.Writer, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	wc, err = b.wrapped.CreateObjectChunkWriter(ctx, req, chunkSize, callBack)
	if err != nil {
		return
	}

	// Wrap the result in a throttled layer.
	wc = b.throttleWriter(ctx, wc)

	return
}

func (b *throttledBucket) CreateAppendableObjectWriter(ctx context.Context, req *gcs.CreateObjectChunkWriterRequest) (wc gcs.Writer, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	wc, err = b.wrapped.CreateAppendableObjectWriter(ctx, req)
	if err != nil {
		return
	}

	// Wrap the result in a throttled layer.
	wc = b.throttleWriter(ctx, wc)

	return
}

func (b *throttledBucket) FinalizeUpload(ctx context.Context, w gcs.Writer) (*gcs.MinObject, error) {
	// FinalizeUpload is not throttled to prevent permanent data loss in case the
	// limiter's burst size is exceeded.
	// Note: CreateObjectChunkWriter, a prerequisite for FinalizeUpload,
	// is throttled.
	return b.wrapped.FinalizeUpload(ctx, unwrapThrottledWriter(w))
}

func (b *throttledBucket) FlushPendingWrites(ctx context.Context, w gcs.Writer) (*gcs.MinObject, error) {
	// FlushPendingWrites is not throttled to prevent permanent data loss in case the
	// limiter's burst size is exceeded.
	// Note: CreateObjectChunkWriter, a prerequisite for FlushPendingWrites,
	// is throttled.
	return b.wrapped.FlushPendingWrites(ctx, unwrapThrottledWriter(w))
}

// Wrap wc so that its Writes are paced by the ingress throttle. Subsequent
// Writes wait on ctx, so ctx must outlive the returned writer, not just this
// call.
func (b *throttledBucket) throttleWriter(ctx context.Context, wc gcs.Writer) gcs.Writer {
	return &throttledGCSWriter{
		Writer:    wc,
		throttled: ThrottledWriter(ctx, wc, b.ingressThrottle),
	}
}

// Return the writer the wrapped bucket originally handed out. Lower layers
// must see their own writer: the fake bucket, for example, type-asserts
// *FakeObjectWriter in FinalizeUpload and FlushPendingWrites.
func unwrapThrottledWriter(w gcs.Writer) gcs.Writer {
	if tw, ok := w.(*throttledGCSWriter); ok {
		return tw.Writer
	}
	return w
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
	req *gcs.StatObjectRequest) (m *gcs.MinObject, e *gcs.ExtendedObjectAttributes, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	m, e, err = b.wrapped.StatObject(ctx, req)

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

func (b *throttledBucket) MoveObject(ctx context.Context, req *gcs.MoveObjectRequest) (*gcs.Object, error) {
	// Wait for permission to call through.
	err := b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return nil, err
	}

	// Call through.
	o, err := b.wrapped.MoveObject(ctx, req)

	return o, err
}
func (b *throttledBucket) DeleteFolder(ctx context.Context, folderName string) (err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	err = b.wrapped.DeleteFolder(ctx, folderName)

	return
}

func (b *throttledBucket) RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (o *gcs.Folder, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	o, err = b.wrapped.RenameFolder(ctx, folderName, destinationFolderId)

	return
}

func (b *throttledBucket) GetFolder(ctx context.Context, req *gcs.GetFolderRequest) (folder *gcs.Folder, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	folder, err = b.wrapped.GetFolder(ctx, req)

	return folder, err
}

func (b *throttledBucket) CreateFolder(ctx context.Context, folderName string) (folder *gcs.Folder, err error) {
	// Wait for permission to call through.
	err = b.opThrottle.Wait(ctx, 1)
	if err != nil {
		return
	}

	// Call through.
	folder, err = b.wrapped.CreateFolder(ctx, folderName)

	return folder, err
}

func (b *throttledBucket) NewMultiRangeDownloader(
	ctx context.Context, req *gcs.MultiRangeDownloaderRequest) (mrd gcs.MultiRangeDownloader, err error) {
	// Call through.
	mrd, err = b.wrapped.NewMultiRangeDownloader(ctx, req)
	return
}

func (b *throttledBucket) GCSName(obj *gcs.MinObject) string {
	return b.wrapped.GCSName(obj)
}

////////////////////////////////////////////////////////////////////////
// readerCloser
////////////////////////////////////////////////////////////////////////

// An io.ReadCloser that forwards read requests to an io.Reader and close
// , readHandle requests to gcs.StorageReader.
type throttledGCSReader struct {
	Reader io.Reader
	Closer gcs.StorageReader
}

func (rc *throttledGCSReader) Read(p []byte) (n int, err error) {
	n, err = rc.Reader.Read(p)
	return
}

func (rc *throttledGCSReader) Close() (err error) {
	err = rc.Closer.Close()
	return
}

func (rc *throttledGCSReader) ReadHandle() (rh storagev2.ReadHandle) {
	rh = rc.Closer.ReadHandle()
	return
}

////////////////////////////////////////////////////////////////////////
// throttledGCSWriter
////////////////////////////////////////////////////////////////////////

// A gcs.Writer that forwards Write to a throttled io.Writer and every other
// method to the wrapped gcs.Writer.
type throttledGCSWriter struct {
	gcs.Writer
	throttled io.Writer
}

func (tw *throttledGCSWriter) Write(p []byte) (n int, err error) {
	n, err = tw.throttled.Write(p)
	return
}
