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
func (b *bucketHandle) DeleteObject(ctx context.Context, req *gcs.DeleteObjectRequest) (err error) {
	obj := b.bucket.Object(req.Name)

	// Switching to the requested generation of the object.
	if req.Generation != 0 {
		obj = obj.Generation(req.Generation)
	}
	// Putting condition that the object's MetaGeneration should match the requested MetaGeneration for deletion to occur.
	if req.MetaGenerationPrecondition != nil && *req.MetaGenerationPrecondition != 0 {
		obj = obj.If(storage.Conditions{MetagenerationMatch: *req.MetaGenerationPrecondition})
	}

	// Deleting object through Go Storage Client.
	return obj.Delete(ctx)
}
