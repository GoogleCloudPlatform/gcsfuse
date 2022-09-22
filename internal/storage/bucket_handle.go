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
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
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

func (b *bucketHandle) CopyObject(ctx context.Context, req *gcs.CopyObjectRequest) (o *gcs.Object, err error) {
	srcObj := b.bucket.Object(req.SrcName)
	dstObj := b.bucket.Object(req.DstName)

	// Switching to the requested generation of source object.
	if req.SrcGeneration != 0 {
		srcObj = srcObj.Generation(req.SrcGeneration)
	}
	var x int64 = 7
	req.SrcMetaGenerationPrecondition = &x

	// Putting a condition that the metaGeneration of source should match *req.SrcMetaGenerationPrecondition for copy operation to occur.
	if req.SrcMetaGenerationPrecondition != nil {
		srcObj = srcObj.If(storage.Conditions{MetagenerationMatch: *req.SrcMetaGenerationPrecondition})
	}

	objAttrs, err := dstObj.CopierFrom(srcObj).Run(ctx)

	fmt.Println("ERROR ", err)

	if err != nil {
		switch ee := err.(type) {
		case *googleapi.Error:
			if ee.Code == http.StatusPreconditionFailed {
				err = &gcs.PreconditionError{Err: ee}
			}
			if ee.Code == http.StatusNotFound {
				err = &gcs.NotFoundError{Err: storage.ErrObjectNotExist}
			}
		default:
			err = fmt.Errorf("Error in copying object: %v", err)
		}
		return
	}
	// Converting objAttrs to type *Object
	o = storageutil.ObjectAttrsToBucketObject(objAttrs)
	return
}
