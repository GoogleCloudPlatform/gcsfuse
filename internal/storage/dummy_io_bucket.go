// Copyright 2025 Google LLC
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
	"context"
	"errors"
	"io"
	"time"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

type DummyIOBucketParams struct {
	ReaderLatency time.Duration
	PerMBLatency  time.Duration
}

// dummyIOBucket is a wrapper over gcs.Bucket that implements gcs.Bucket interface.
// It directly delegates all calls to the wrapped bucket, and performs dummy IO for
// read and write operations.
type dummyIOBucket struct {
	wrapped       gcs.Bucket
	readerLatency time.Duration
	perMBLatency  time.Duration
}

// NewDummyIOBucket creates a new dummyIOBucket wrapping the given gcs.Bucket.
// If the wrapped bucket is nil, it returns nil.
func NewDummyIOBucket(wrapped gcs.Bucket, params DummyIOBucketParams) gcs.Bucket {
	if wrapped == nil {
		return nil
	}

	return &dummyIOBucket{
		wrapped:       wrapped,
		readerLatency: params.ReaderLatency,
		perMBLatency:  params.PerMBLatency,
	}
}

// Name returns the name of the bucket.
func (d *dummyIOBucket) Name() string {
	return d.wrapped.Name()
}

// BucketType returns the type of the bucket.
func (d *dummyIOBucket) BucketType() gcs.BucketType {
	return d.wrapped.BucketType()
}

// NewReaderWithReadHandle creates a reader for reading object contents.
// Returns a dummy reader that serves zeros efficiently instead of reading from GCS.
func (d *dummyIOBucket) NewReaderWithReadHandle(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (gcs.StorageReader, error) {

	if req.Range == nil {
		return nil, errors.New("range must be specified for dummy IO bucket")
	}

	rangeLen := int64(req.Range.Limit) - int64(req.Range.Start)
	if rangeLen <= 0 {
		return nil, errors.New("invalid range: limit is less than start")
	}

	// Simulate network latency if specified.
	if d.readerLatency > 0 {
		time.Sleep(d.readerLatency)
	}

	return newDummyReader(uint64(rangeLen), d.perMBLatency), nil
}

// NewMultiRangeDownloader creates a multi-range downloader for object contents.
// TODO: Add custom logic for Read path if needed
func (d *dummyIOBucket) NewMultiRangeDownloader(
	ctx context.Context,
	req *gcs.MultiRangeDownloaderRequest) (gcs.MultiRangeDownloader, error) {
	return d.wrapped.NewMultiRangeDownloader(ctx, req)
}

// CreateObject creates or overwrites an object.
// TODO: Add custom logic for Write path if needed
func (d *dummyIOBucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	return d.wrapped.CreateObject(ctx, req)
}

// CreateObjectChunkWriter creates a writer for resumable uploads.
// TODO: Add custom logic for Write path if needed
func (d *dummyIOBucket) CreateObjectChunkWriter(
	ctx context.Context,
	req *gcs.CreateObjectRequest,
	chunkSize int,
	callBack func(bytesUploadedSoFar int64)) (gcs.Writer, error) {
	return d.wrapped.CreateObjectChunkWriter(ctx, req, chunkSize, callBack)
}

// CreateAppendableObjectWriter creates a writer to append to an existing object.
// TODO: Add custom logic for Write path if needed
func (d *dummyIOBucket) CreateAppendableObjectWriter(
	ctx context.Context,
	req *gcs.CreateObjectChunkWriterRequest) (gcs.Writer, error) {
	return d.wrapped.CreateAppendableObjectWriter(ctx, req)
}

// FinalizeUpload completes the write operation and creates the object on GCS.
// TODO: Add custom logic for Write path if needed
func (d *dummyIOBucket) FinalizeUpload(
	ctx context.Context,
	writer gcs.Writer) (*gcs.MinObject, error) {
	return d.wrapped.FinalizeUpload(ctx, writer)
}

// FlushPendingWrites flushes pending data in the writer buffer for zonal buckets.
// TODO: Add custom logic for Write path if needed
func (d *dummyIOBucket) FlushPendingWrites(
	ctx context.Context,
	writer gcs.Writer) (*gcs.MinObject, error) {
	return d.wrapped.FlushPendingWrites(ctx, writer)
}

// CopyObject copies an object to a new name.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (*gcs.Object, error) {
	return d.wrapped.CopyObject(ctx, req)
}

// ComposeObjects composes one or more source objects into a single destination object.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (*gcs.Object, error) {
	return d.wrapped.ComposeObjects(ctx, req)
}

// StatObject returns current information about the object.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) StatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	return d.wrapped.StatObject(ctx, req)
}

// ListObjects lists the objects in the bucket that meet the criteria.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (*gcs.Listing, error) {
	return d.wrapped.ListObjects(ctx, req)
}

// UpdateObject updates the object specified by request.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (*gcs.Object, error) {
	return d.wrapped.UpdateObject(ctx, req)
}

// DeleteObject deletes an object.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) DeleteObject(
	ctx context.Context,
	req *gcs.DeleteObjectRequest) error {
	return d.wrapped.DeleteObject(ctx, req)
}

// MoveObject moves an object to a new name.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) MoveObject(
	ctx context.Context,
	req *gcs.MoveObjectRequest) (*gcs.Object, error) {
	return d.wrapped.MoveObject(ctx, req)
}

// DeleteFolder deletes a folder.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) DeleteFolder(ctx context.Context, folderName string) error {
	return d.wrapped.DeleteFolder(ctx, folderName)
}

// GetFolder retrieves folder information.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) GetFolder(ctx context.Context, folderName string) (*gcs.Folder, error) {
	return d.wrapped.GetFolder(ctx, folderName)
}

// RenameFolder atomically renames a folder for Hierarchical bucket.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) RenameFolder(
	ctx context.Context,
	folderName string,
	destinationFolderId string) (*gcs.Folder, error) {
	return d.wrapped.RenameFolder(ctx, folderName, destinationFolderId)
}

// CreateFolder creates a new folder.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) CreateFolder(ctx context.Context, folderName string) (*gcs.Folder, error) {
	return d.wrapped.CreateFolder(ctx, folderName)
}

// GCSName returns the original GCS name for the object.
// Directly delegates to wrapped bucket.
func (d *dummyIOBucket) GCSName(object *gcs.MinObject) string {
	return d.wrapped.GCSName(object)
}

////////////////////////////////////////////////////////////////////////
// dummyReader
////////////////////////////////////////////////////////////////////////

// dummyReader is an efficient reader that serves dummy data.
// It implements the StorageReader interface and returns zeros for all reads.
// Reading beyond the specified length returns io.EOF.
// Also, it always returns a non-nil read handle.
type dummyReader struct {
	totalLen       uint64 // Total length of data to serve
	bytesRead      uint64 // Number of bytes already read
	readHandle     storagev2.ReadHandle
	perByteLatency time.Duration
}

// newDummyReader creates a new dummyReader with the specified total length.
func newDummyReader(totalLen uint64, perMBLatency time.Duration) *dummyReader {
	return &dummyReader{
		totalLen:       totalLen,
		bytesRead:      0,
		readHandle:     []byte{}, // Always return a non-nil read handle
		perByteLatency: time.Duration(perMBLatency.Microseconds() / (1024 * 1024)),
	}
}

// Read reads up to len(p) bytes into p, filling it with zeros.
// Returns io.EOF when the total length has been reached.
func (dr *dummyReader) Read(p []byte) (n int, err error) {
	// If we've already read all the data, return EOF
	if dr.bytesRead >= dr.totalLen {
		return 0, io.EOF
	}

	// Calculate how many bytes we can still read
	remaining := dr.totalLen - dr.bytesRead

	// Determine how many bytes to read in this call
	toRead := uint64(len(p))
	if toRead > remaining {
		toRead = remaining
	}

	// Simulate per-MB latency if specified
	if dr.perByteLatency > 0 {
		time.Sleep(time.Duration(toRead) * dr.perByteLatency)
	}

	// Fill the buffer with zeros (dummy data).
	for i := uint64(0); i < toRead; i++ {
		p[i] = 0
	}

	dr.bytesRead += toRead

	// If we've read all the data, return EOF along with the last bytes
	if dr.bytesRead >= dr.totalLen {
		return int(toRead), io.EOF
	}

	return int(toRead), nil
}

// Close closes the reader. For dummy reader, this is a no-op.
func (dr *dummyReader) Close() error {
	return nil
}

// ReadHandle returns the read handle. For dummy reader, this returns a nil handle.
func (dr *dummyReader) ReadHandle() storagev2.ReadHandle {
	return dr.readHandle
}
