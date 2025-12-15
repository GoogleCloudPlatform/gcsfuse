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

package storage

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
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
	v ...any) {
	logger.Tracef("gcs: Req %#16x: %s", id, fmt.Sprintf(format, v...))
}

func (b *debugBucket) startRequest(
	format string,
	v ...any) (id uint64, desc string, start time.Time) {
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
	wrapped   gcs.StorageReader
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

func (dr *debugReader) ReadHandle() storagev2.ReadHandle {
	return dr.wrapped.ReadHandle()
}

////////////////////////////////////////////////////////////////////////
// Bucket interface
////////////////////////////////////////////////////////////////////////

func (b *debugBucket) Name() string {
	return b.wrapped.Name()
}

func (b *debugBucket) BucketType() gcs.BucketType {
	return b.wrapped.BucketType()
}

func setupReader(ctx context.Context, b *debugBucket, req *gcs.ReadObjectRequest, method string) (gcs.StorageReader, error) {
	id, desc, start := b.startRequest("%s(%q, %v)", method, req.Name, req.Range)

	// Call through.
	rc, err := b.wrapped.NewReaderWithReadHandle(ctx, req)
	if err != nil {
		b.finishRequest(id, desc, start, &err)
		return rc, err
	}

	// Return a special reader that prings debug info.
	rc = &debugReader{
		bucket:    b,
		requestID: id,
		desc:      desc,
		startTime: start,
		wrapped:   rc,
	}
	return rc, err
}

func (b *debugBucket) NewReaderWithReadHandle(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rd gcs.StorageReader, err error) {
	rd, err = setupReader(ctx, b, req, "ReadWithReadHandle")
	return
}

func (b *debugBucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {
	id, desc, start := b.startRequest("CreateObject(%q)", req.Name)
	defer b.finishRequest(id, desc, start, &err)
	if req.CallBack == nil {
		req.CallBack = func(bytesUploadedSoFar int64) {
			logger.Tracef("gcs: Req %#16x: -- UploadBlock(%q): %20v bytes uploaded so far", id, req.Name, bytesUploadedSoFar)
		}
	}
	o, err = b.wrapped.CreateObject(ctx, req)
	return
}

func (b *debugBucket) CreateObjectChunkWriter(ctx context.Context, req *gcs.CreateObjectRequest, chunkSize int, callBack func(bytesUploadedSoFar int64)) (wc gcs.Writer, err error) {
	id, desc, start := b.startRequest("CreateObjectChunkWriter(%q)", req.Name)
	defer b.finishRequest(id, desc, start, &err)
	if callBack == nil {
		callBack = func(bytesUploadedSoFar int64) {
			logger.Tracef("gcs: Req %#16x: -- UploadBlock(%q): %20v bytes uploaded so far", id, req.Name, bytesUploadedSoFar)
		}
	}
	wc, err = b.wrapped.CreateObjectChunkWriter(ctx, req, chunkSize, callBack)
	return
}

func (b *debugBucket) CreateAppendableObjectWriter(ctx context.Context,
	req *gcs.CreateObjectChunkWriterRequest) (wc gcs.Writer, err error) {
	id, desc, start := b.startRequest("CreateAppendableObjectWriter(%q, %d)", req.Name, req.Offset)
	defer b.finishRequest(id, desc, start, &err)
	if req.CallBack == nil {
		req.CallBack = func(bytesUploadedSoFar int64) {
			logger.Tracef("gcs: Req %#16x: -- UploadBlock(%q): %20v bytes uploaded so far", id, req.Name, bytesUploadedSoFar)
		}
	}
	wc, err = b.wrapped.CreateAppendableObjectWriter(ctx, req)
	return
}

func (b *debugBucket) FinalizeUpload(ctx context.Context, w gcs.Writer) (o *gcs.MinObject, err error) {
	id, desc, start := b.startRequest("FinalizeUpload(%q)", w.ObjectName())
	defer b.finishRequest(id, desc, start, &err)

	o, err = b.wrapped.FinalizeUpload(ctx, w)
	return
}

func (b *debugBucket) FlushPendingWrites(ctx context.Context, w gcs.Writer) (o *gcs.MinObject, err error) {
	id, desc, start := b.startRequest("FlushPendingWrites(%q)", w.ObjectName())
	defer b.finishRequest(id, desc, start, &err)

	o, err = b.wrapped.FlushPendingWrites(ctx, w)
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

func (b *debugBucket) MoveObject(ctx context.Context, req *gcs.MoveObjectRequest) (*gcs.Object, error) {
	var err error
	var o *gcs.Object
	id, desc, start := b.startRequest("MoveObject(%q, %q)", req.SrcName, req.DstName)

	defer b.finishRequest(id, desc, start, &err)

	o, err = b.wrapped.MoveObject(ctx, req)
	return o, err
}

func (b *debugBucket) DeleteFolder(ctx context.Context, folderName string) (err error) {
	id, desc, start := b.startRequest("DeleteFolder(%q)", folderName)
	defer b.finishRequest(id, desc, start, &err)

	err = b.wrapped.DeleteFolder(ctx, folderName)
	return err
}

func (b *debugBucket) GetFolder(ctx context.Context, folderName string) (folder *gcs.Folder, err error) {
	id, desc, start := b.startRequest("GetFolder(%q)", folderName)
	defer b.finishRequest(id, desc, start, &err)

	folder, err = b.wrapped.GetFolder(ctx, folderName)
	return
}

func (b *debugBucket) CreateFolder(ctx context.Context, folderName string) (folder *gcs.Folder, err error) {
	id, desc, start := b.startRequest("CreateFolder(%q)", folderName)
	defer b.finishRequest(id, desc, start, &err)

	folder, err = b.wrapped.CreateFolder(ctx, folderName)
	return
}

func (b *debugBucket) RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (o *gcs.Folder, err error) {
	id, desc, start := b.startRequest("RenameFolder(%q, %q)", folderName, destinationFolderId)
	defer b.finishRequest(id, desc, start, &err)

	o, err = b.wrapped.RenameFolder(ctx, folderName, destinationFolderId)
	return o, err
}

type debugMultiRangeDownloader struct {
	object    string
	bucket    *debugBucket
	requestID uint64
	desc      string
	startTime time.Time
	wrapped   gcs.MultiRangeDownloader
}

func (dmrd *debugMultiRangeDownloader) Add(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
	id, desc, start := dmrd.bucket.startRequest("MultiRangeDownloader.Add(%s, [%v,%v))", dmrd.object, offset, offset+length)
	wrapperCallback := func(offset int64, length int64, err error) {
		defer dmrd.bucket.finishRequest(id, desc, start, &err)
		if callback != nil {
			callback(offset, length, err)
		}
	}
	dmrd.wrapped.Add(output, offset, length, wrapperCallback)
}

func (dmrd *debugMultiRangeDownloader) Close() (err error) {
	id, desc, start := dmrd.bucket.startRequest("MultiRangeDownloader.Close()")
	defer dmrd.bucket.finishRequest(id, desc, start, &err)
	err = dmrd.wrapped.Close()
	return
}

func (dmrd *debugMultiRangeDownloader) Wait() {
	id, desc, start := dmrd.bucket.startRequest("MultiRangeDownloader.Wait()")
	var err error
	defer dmrd.bucket.finishRequest(id, desc, start, &err)
	dmrd.wrapped.Wait()
}

func (dmrd *debugMultiRangeDownloader) Error() (err error) {
	err = dmrd.wrapped.Error()
	return
}

func (dmrd *debugMultiRangeDownloader) GetHandle() []byte {
	id, desc, start := dmrd.bucket.startRequest("MultiRangeDownloader.GetHandle()")
	var err error
	defer dmrd.bucket.finishRequest(id, desc, start, &err)
	return dmrd.wrapped.GetHandle()
}

func (b *debugBucket) NewMultiRangeDownloader(
	ctx context.Context, req *gcs.MultiRangeDownloaderRequest) (mrd gcs.MultiRangeDownloader, err error) {
	id, desc, start := b.startRequest("NewMultiRangeDownloader(%q)", req.Name)
	defer b.finishRequest(id, desc, start, &err)

	// Call through.
	mrd, err = b.wrapped.NewMultiRangeDownloader(ctx, req)
	if err != nil {
		return
	}

	// Return a special reader that prints debug info.
	mrd = &debugMultiRangeDownloader{
		object:    req.Name,
		bucket:    b,
		requestID: id,
		desc:      desc,
		startTime: start,
		wrapped:   mrd,
	}
	return
}

func (b *debugBucket) GCSName(obj *gcs.MinObject) string {
	return obj.Name
}
