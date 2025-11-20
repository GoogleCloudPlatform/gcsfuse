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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// dummyIOBucket is a wrapper over gcs.Bucket that implements gcs.Bucket interface.
// It directly delegates all calls to the wrapped bucket, and performs dummy IO for
// read and write operations.
type dummyIOBucket struct {
	wrapped gcs.Bucket
}

// NewDummyIOBucket creates a new dummyIOBucket wrapping the given gcs.Bucket.
func NewDummyIOBucket(wrapped gcs.Bucket) gcs.Bucket {
	return &dummyIOBucket{
		wrapped: wrapped,
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
// TODO: Add custom logic for Read path if needed
func (d *dummyIOBucket) NewReaderWithReadHandle(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (gcs.StorageReader, error) {
	return d.wrapped.NewReaderWithReadHandle(ctx, req)
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
