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

package gcs

import (
	"io"

	"golang.org/x/net/context"
)

// ChunkUploader interface represents asynchronous object
// uploaders which upload a chunk of an object's data in a
// call using GCS' resumable upload API.
//
// On closing, they return a GCS object (gcs.Object)
// and and error object for error-handling.
type ChunkUploader interface {
	// UploadChunkAsync uploads the given chunk to gcs.
	// Progress should be tracked using the progress func
	// passed during ChunkUploader instance creation.
	//
	// Unlike the io.Writer interface, it doesn't return number-of-bytes
	// uploaded in this call, as the write is asynchronous; instead
	// this interface instead has
	// BytesWrittenSoFar function below, which returns the total number
	// of bytes successfully uploaded so far.
	UploadChunkAsync(contents io.Reader) error

	// Close finalizes the upload and returns the created object.
	// Error is returned in case of failures.
	Close() (*Object, error)

	// Returns the number of bytes successfully uploaded so far
	// by this uploader.
	BytesUploadedSoFar() int64
}

// Bucket represents a GCS bucket, pre-bound with a bucket name and necessary
// authorization information.
//
// Each method that may block accepts a context object that is used for
// deadlines and cancellation. Users need not package authorization information
// into the context object (using cloud.WithContext or similar).
//
// All methods are safe for concurrent access.
type Bucket interface {
	Name() string

	// Create a reader for the contents of a particular generation of an object.
	// On a nil error, the caller must arrange for the reader to be closed when
	// it is no longer needed.
	//
	// Non-existent objects cause either this method or the first read from the
	// resulting reader to return an error of type *NotFoundError.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/get
	NewReader(
		ctx context.Context,
		req *ReadObjectRequest) (io.ReadCloser, error)

	// Create or overwrite an object according to the supplied request. The new
	// object is guaranteed to exist immediately for the purposes of reading (and
	// eventually for listing) after this method returns a nil error. It is
	// guaranteed not to exist before req.Contents returns io.EOF.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/insert
	//     https://cloud.google.com/storage/docs/json_api/v1/how-tos/upload
	CreateObject(
		ctx context.Context,
		req *CreateObjectRequest) (*Object, error)

	// CreateChunkUploader creates a chunkUploader instance
	// according to supplied createChunkUploaderRequest. This needs
	// to be used when caller wants control on the resumable
	// upload api.
	//
	// This is an alternative to createObject which also uploads
	// data using resumable upload api, but caller has no control
	// on the resumable api.
	//
	// While creating a chunk-uploader, you can specify a callback called
	// progressFunc which carries (n int64) the total number of bytes
	// successfully uploaded by the uploader so far.
	//
	// The progress function callback is expected exactly once for each upload.
	//
	// Sample usage:
	// uploader, err := bucket.CreateChunkUploader(ctx, createChunkUploaderReq,
	//						chunkSize,
	//						func(n int64) {
	//							log("n bytes successfully uploaded so far")
	//						}
	// )
	// // check err
	// for n times {
	// 		err = uploader.UploadChunkAsync(buffer)
	// 		// check err
	// }
	// obj, err := uploader.Close()
	// // check err
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/resumable-uploads#go
	CreateChunkUploader(
		ctx context.Context,
		req *CreateChunkUploaderRequest,
		writeChunkSize int,
		progressFunc func(int64)) (ChunkUploader, error)

	// Copy an object to a new name, preserving all metadata. Any existing
	// generation of the destination name will be overwritten.
	//
	// Returns a record for the new object.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/copy
	CopyObject(
		ctx context.Context,
		req *CopyObjectRequest) (*Object, error)

	// Compose one or more source objects into a single destination object by
	// concatenating. Any existing generation of the destination name will be
	// overwritten.
	//
	// Returns a record for the new object.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/compose
	ComposeObjects(
		ctx context.Context,
		req *ComposeObjectsRequest) (*Object, error)

	// Return current information about the object with the given name.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/get
	StatObject(
		ctx context.Context,
		req *StatObjectRequest) (*Object, error)

	// List the objects in the bucket that meet the criteria defined by the
	// request, returning a result object that contains the results and
	// potentially a cursor for retrieving the next portion of the larger set of
	// results.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/list
	ListObjects(
		ctx context.Context,
		req *ListObjectsRequest) (*Listing, error)

	// Update the object specified by newAttrs.Name, patching using the non-zero
	// fields of newAttrs.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/patch
	UpdateObject(
		ctx context.Context,
		req *UpdateObjectRequest) (*Object, error)

	// Delete an object. Non-existence of the object is not treated as an error.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/delete
	DeleteObject(
		ctx context.Context,
		req *DeleteObjectRequest) error
}
