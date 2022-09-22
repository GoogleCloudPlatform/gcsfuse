// Copyright 2022 Google Inc. All Rights Reserved.
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

// For now, we are not writing the unit test, which requires multiple
// version of same object. As this is not supported by fake-storage-server.
// Although API is exposed to enable the object versioning for a bucket,
// but it returns "method not allowed" when we call it.

package storage

import (
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

type bucketHandle struct {
	gcs.Bucket
	bucket *storage.BucketHandle
}

func (bh *bucketHandle) NewReader(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rc io.ReadCloser, err error) {
	// Initialising the starting offset and the length to be read by the reader.
	start := int64((*req.Range).Start)
	end := int64((*req.Range).Limit)
	length := int64(end - start)

	obj := bh.bucket.Object(req.Name)

	// Switching to the requested generation of object.
	if req.Generation != 0 {
		obj = obj.Generation(req.Generation)
	}

	// Creating a NewRangeReader instance.
	r, err := obj.NewRangeReader(ctx, start, length)
	if err != nil {
		err = fmt.Errorf("error in creating a NewRangeReader instance: %v", err)
		return
	}

	// Converting io.Reader to io.ReadCloser by adding a no-op closer method
	// to match the return type interface.
	rc = io.NopCloser(r)
	return
}
func (b *bucketHandle) DeleteObject(ctx context.Context, req *gcs.DeleteObjectRequest) error {
	obj := b.bucket.Object(req.Name)

	// Switching to the requested generation of the object.
	if req.Generation != 0 {
		obj = obj.Generation(req.Generation)
	}
	// Putting condition that the object's MetaGeneration should match the requested MetaGeneration for deletion to occur.
	if req.MetaGenerationPrecondition != nil && *req.MetaGenerationPrecondition != 0 {
		obj = obj.If(storage.Conditions{MetagenerationMatch: *req.MetaGenerationPrecondition})
	}

	return obj.Delete(ctx)
}

func (b *bucketHandle) StatObject(ctx context.Context, req *gcs.StatObjectRequest) (o *gcs.Object, err error) {
	var attrs *storage.ObjectAttrs
	// Retrieving object attrs through Go Storage Client.
	attrs, err = b.bucket.Object(req.Name).Attrs(ctx)

	// If error is of type storage.ErrObjectNotExist
	if err == storage.ErrObjectNotExist {
		err = &gcs.NotFoundError{Err: err} // Special case error that object not found in the bucket.
		return
	}
	if err != nil {
		err = fmt.Errorf("Error in fetching object attributes: %v", err)
		return
	}

	// Converting attrs to type *Object
	o = storageutil.ObjectAttrsToBucketObject(attrs)

	return
}

func (bh *bucketHandle) CreateObject(ctx context.Context, req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {
	obj := bh.bucket.Object(req.Name)

	if req.GenerationPrecondition != nil && *req.GenerationPrecondition != 0 {
		if req.MetaGenerationPrecondition != nil && *req.MetaGenerationPrecondition != 0 {
			obj = obj.If(storage.Conditions{GenerationMatch: *req.GenerationPrecondition, MetagenerationMatch: *req.MetaGenerationPrecondition})
		} else {
			obj = obj.If(storage.Conditions{GenerationMatch: *req.GenerationPrecondition})
		}
	}

	// Creating a NewWriter with requested attributes, using Go Storage Client.
	// Chuck size for resumable upload is default i.e. 16MB.
	wc := obj.NewWriter(ctx)
	wc.ChunkSize = 0 // This enables one shot upload.
	wc = storageutil.SetAttrsInWriter(wc, req)

	// Copy the contents to the writer.
	if _, err = io.Copy(wc, req.Contents); err != nil {
		err = fmt.Errorf("error in io.Copy: %w", err)
		return
	}

	// We can't use defer to close the writer, because we need to close the
	// writer successfully before calling Attrs() method of writer.
	if err = wc.Close(); err != nil {
		err = fmt.Errorf("error in closing writer: %v", err)
		return
	}

	attrs := wc.Attrs() // Retrieving the attributes of the created object.

	// Converting attrs to type *Object.
	o = storageutil.ObjectAttrsToBucketObject(attrs)
	return
}
