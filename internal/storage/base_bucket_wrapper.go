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

// baseBucketWrapper provides default pass-through implementations for all
// gcs.Bucket methods. Concrete wrappers can embed this struct and selectively
// override only the methods they need to customize.
//
// This pattern eliminates boilerplate delegation code and makes it clear which
// methods each wrapper is actually customizing.
//
// Example usage:
//
//	type myBucket struct {
//	    baseBucketWrapper
//	    // custom fields
//	}
//
//	func NewMyBucket(wrapped gcs.Bucket) gcs.Bucket {
//	    return &myBucket{
//	        baseBucketWrapper: baseBucketWrapper{wrapped: wrapped},
//	    }
//	}
//
//	// Only override methods that need custom behavior
//	func (b *myBucket) CreateObject(...) {...}
type baseBucketWrapper struct {
	wrapped gcs.Bucket
}

// Ensure baseBucketWrapper implements gcs.Bucket at compile time.
var _ gcs.Bucket = (*baseBucketWrapper)(nil)

// Name returns the bucket name.
func (b *baseBucketWrapper) Name() string {
	return b.wrapped.Name()
}

// BucketType returns the bucket type.
func (b *baseBucketWrapper) BucketType() gcs.BucketType {
	return b.wrapped.BucketType()
}

// NewReaderWithReadHandle delegates to the wrapped bucket.
func (b *baseBucketWrapper) NewReaderWithReadHandle(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (gcs.StorageReader, error) {
	return b.wrapped.NewReaderWithReadHandle(ctx, req)
}

// NewMultiRangeDownloader delegates to the wrapped bucket.
func (b *baseBucketWrapper) NewMultiRangeDownloader(
	ctx context.Context,
	req *gcs.MultiRangeDownloaderRequest) (gcs.MultiRangeDownloader, error) {
	return b.wrapped.NewMultiRangeDownloader(ctx, req)
}

// CreateObject delegates to the wrapped bucket.
func (b *baseBucketWrapper) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	return b.wrapped.CreateObject(ctx, req)
}

// CreateObjectChunkWriter delegates to the wrapped bucket.
func (b *baseBucketWrapper) CreateObjectChunkWriter(
	ctx context.Context,
	req *gcs.CreateObjectRequest,
	chunkSize int,
	callBack func(bytesUploadedSoFar int64)) (gcs.Writer, error) {
	return b.wrapped.CreateObjectChunkWriter(ctx, req, chunkSize, callBack)
}

// CreateAppendableObjectWriter delegates to the wrapped bucket.
func (b *baseBucketWrapper) CreateAppendableObjectWriter(
	ctx context.Context,
	req *gcs.CreateObjectChunkWriterRequest) (gcs.Writer, error) {
	return b.wrapped.CreateAppendableObjectWriter(ctx, req)
}

// FinalizeUpload delegates to the wrapped bucket.
func (b *baseBucketWrapper) FinalizeUpload(
	ctx context.Context,
	writer gcs.Writer) (*gcs.MinObject, error) {
	return b.wrapped.FinalizeUpload(ctx, writer)
}

// FlushPendingWrites delegates to the wrapped bucket.
func (b *baseBucketWrapper) FlushPendingWrites(
	ctx context.Context,
	writer gcs.Writer) (*gcs.MinObject, error) {
	return b.wrapped.FlushPendingWrites(ctx, writer)
}

// CopyObject delegates to the wrapped bucket.
func (b *baseBucketWrapper) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (*gcs.Object, error) {
	return b.wrapped.CopyObject(ctx, req)
}

// ComposeObjects delegates to the wrapped bucket.
func (b *baseBucketWrapper) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (*gcs.Object, error) {
	return b.wrapped.ComposeObjects(ctx, req)
}

// StatObject delegates to the wrapped bucket.
func (b *baseBucketWrapper) StatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	return b.wrapped.StatObject(ctx, req)
}

// ListObjects delegates to the wrapped bucket.
func (b *baseBucketWrapper) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (*gcs.Listing, error) {
	return b.wrapped.ListObjects(ctx, req)
}

// UpdateObject delegates to the wrapped bucket.
func (b *baseBucketWrapper) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (*gcs.Object, error) {
	return b.wrapped.UpdateObject(ctx, req)
}

// DeleteObject delegates to the wrapped bucket.
func (b *baseBucketWrapper) DeleteObject(
	ctx context.Context,
	req *gcs.DeleteObjectRequest) error {
	return b.wrapped.DeleteObject(ctx, req)
}

// MoveObject delegates to the wrapped bucket.
func (b *baseBucketWrapper) MoveObject(
	ctx context.Context,
	req *gcs.MoveObjectRequest) (*gcs.Object, error) {
	return b.wrapped.MoveObject(ctx, req)
}

// DeleteFolder delegates to the wrapped bucket.
func (b *baseBucketWrapper) DeleteFolder(
	ctx context.Context,
	folderName string) error {
	return b.wrapped.DeleteFolder(ctx, folderName)
}

// GetFolder delegates to the wrapped bucket.
func (b *baseBucketWrapper) GetFolder(
	ctx context.Context,
	folderName string) (*gcs.Folder, error) {
	return b.wrapped.GetFolder(ctx, folderName)
}

// CreateFolder delegates to the wrapped bucket.
func (b *baseBucketWrapper) CreateFolder(
	ctx context.Context,
	folderName string) (*gcs.Folder, error) {
	return b.wrapped.CreateFolder(ctx, folderName)
}

// RenameFolder delegates to the wrapped bucket.
func (b *baseBucketWrapper) RenameFolder(
	ctx context.Context,
	folderName string,
	destinationFolderId string) (*gcs.Folder, error) {
	return b.wrapped.RenameFolder(ctx, folderName, destinationFolderId)
}

// GCSName delegates to the wrapped bucket.
func (b *baseBucketWrapper) GCSName(object *gcs.MinObject) string {
	return b.wrapped.GCSName(object)
}
