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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

type bucketHandle struct {
	gcs.Bucket
	bucket     *storage.BucketHandle
	bucketName string
	bucketType string
}

func (bh *bucketHandle) Name() string {
	return bh.bucketName
}

func (bh *bucketHandle) Type() string {
	return bh.bucketType
}

func (bh *bucketHandle) NewReader(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (io.ReadCloser, error) {
	// Initialising the starting offset and the length to be read by the reader.
	start := int64(0)
	length := int64(-1)
	// Following the semantics of NewReader method. Passing start, length as 0,-1 reads the entire file.
	// https://github.com/GoogleCloudPlatform/gcsfuse/blob/34211af652dbaeb012b381a3daf3c94b95f65e00/vendor/cloud.google.com/go/storage/reader.go#L75
	if req.Range != nil {
		start = int64((*req.Range).Start)
		end := int64((*req.Range).Limit)
		length = end - start
	}

	obj := bh.bucket.Object(req.Name)

	// Switching to the requested generation of object.
	if req.Generation != 0 {
		obj = obj.Generation(req.Generation)
	}

	if req.ReadCompressed {
		obj = obj.ReadCompressed(true)
	}

	// NewRangeReader creates a "storage.Reader" object which is also io.ReadCloser since it contains both Read() and Close() methods present in io.ReadCloser interface.
	return obj.NewRangeReader(ctx, start, length)
}
func (b *bucketHandle) DeleteObject(ctx context.Context, req *gcs.DeleteObjectRequest) error {
	obj := b.bucket.Object(req.Name)

	// Switching to the requested generation of the object. By default, generation
	// is 0 which signifies the latest generation. Note: GCS will delete the
	// live object even if generation is not set in request. We are passing 0
	// generation explicitly to satisfy idempotency condition.
	obj = obj.Generation(req.Generation)

	// Putting condition that the object's MetaGeneration should match the requested MetaGeneration for deletion to occur.
	if req.MetaGenerationPrecondition != nil && *req.MetaGenerationPrecondition != 0 {
		obj = obj.If(storage.Conditions{MetagenerationMatch: *req.MetaGenerationPrecondition})
	}

	err := obj.Delete(ctx)
	// If storage object does not exist, httpclient is returning ErrObjectNotExist error instead of googleapi error
	// https://github.com/GoogleCloudPlatform/gcsfuse/blob/7ad451c6f2ead7992e030503e5b66c555b2ebf71/vendor/cloud.google.com/go/storage/http_client.go#L399
	if err != nil {
		switch ee := err.(type) {
		case *googleapi.Error:
			if ee.Code == http.StatusPreconditionFailed {
				err = &gcs.PreconditionError{Err: ee}
			}
		default:
			if err == storage.ErrObjectNotExist {
				err = &gcs.NotFoundError{Err: storage.ErrObjectNotExist}
			} else {
				err = fmt.Errorf("Error in deleting object: %w", err)
			}
		}
	}
	return err

}

func (b *bucketHandle) StatObject(ctx context.Context,
	req *gcs.StatObjectRequest) (m *gcs.MinObject, e *gcs.ExtendedObjectAttributes, err error) {
	var attrs *storage.ObjectAttrs
	// Retrieving object attrs through Go Storage Client.
	attrs, err = b.bucket.Object(req.Name).Attrs(ctx)

	// If error is of type storage.ErrObjectNotExist
	if err == storage.ErrObjectNotExist {
		err = &gcs.NotFoundError{Err: err} // Special case error that object not found in the bucket.
		return
	}
	if err != nil {
		err = fmt.Errorf("Error in fetching object attributes: %w", err)
		return
	}

	// Converting attrs to type *Object
	o := storageutil.ObjectAttrsToBucketObject(attrs)
	m = storageutil.ConvertObjToMinObject(o)
	if req.ReturnExtendedObjectAttributes {
		e = storageutil.ConvertObjToExtendedObjectAttributes(o)
	}

	return
}

func (bh *bucketHandle) CreateObject(ctx context.Context, req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {
	obj := bh.bucket.Object(req.Name)

	// GenerationPrecondition - If non-nil, the object will be created/overwritten
	// only if the current generation for the object name is equal to the given value.
	// Zero means the object does not exist.
	// MetaGenerationPrecondition - If non-nil, the object will be created/overwritten
	// only if the current metaGeneration for the object name is equal to the given value.
	// Zero means the object does not exist.
	preconditions := storage.Conditions{}

	if req.GenerationPrecondition != nil {
		if *req.GenerationPrecondition == 0 {
			preconditions.DoesNotExist = true
		} else {
			preconditions.GenerationMatch = *req.GenerationPrecondition
		}
	}

	if req.MetaGenerationPrecondition != nil && *req.MetaGenerationPrecondition != 0 {
		preconditions.MetagenerationMatch = *req.MetaGenerationPrecondition
	}

	// Setting up the conditions on the object if it's not empty i.e, atleast
	// if one of the condition is set.
	if isStorageConditionsNotEmpty(preconditions) {
		obj = obj.If(preconditions)
	}

	// Creating a NewWriter with requested attributes, using Go Storage Client.
	// Chuck size for resumable upload is default i.e. 16MB.
	wc := obj.NewWriter(ctx)
	wc = storageutil.SetAttrsInWriter(wc, req)

	// Copy the contents to the writer.
	if _, err = io.Copy(wc, req.Contents); err != nil {
		err = fmt.Errorf("error in io.Copy: %w", err)
		return
	}

	// We can't use defer to close the writer, because we need to close the
	// writer successfully before calling Attrs() method of writer.
	if err = wc.Close(); err != nil {
		var gErr *googleapi.Error
		if errors.As(err, &gErr) {
			if gErr.Code == http.StatusPreconditionFailed {
				err = &gcs.PreconditionError{Err: err}
				return
			}
		}
		err = fmt.Errorf("error in closing writer : %w", err)
		return
	}

	attrs := wc.Attrs() // Retrieving the attributes of the created object.
	// Converting attrs to type *Object.
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

	// Putting a condition that the metaGeneration of source should match *req.SrcMetaGenerationPrecondition for copy operation to occur.
	if req.SrcMetaGenerationPrecondition != nil {
		srcObj = srcObj.If(storage.Conditions{MetagenerationMatch: *req.SrcMetaGenerationPrecondition})
	}

	objAttrs, err := dstObj.CopierFrom(srcObj).Run(ctx)

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
			err = fmt.Errorf("Error in copying object: %w", err)
		}
		return
	}
	// Converting objAttrs to type *Object
	o = storageutil.ObjectAttrsToBucketObject(objAttrs)
	return
}

func getProjectionValue(req gcs.Projection) storage.Projection {
	// Explicitly converting Projection Value because the ProjectionVal interface of jacobsa/gcloud and Go Client API are not coupled correctly.
	var convertedProjection storage.Projection // Stores the Projection Value according to the Go Client API Interface.
	switch int(req) {
	// Projection Value 0 in jacobsa/gcloud maps to Projection Value 1 in Go Client API, that is for "full".
	case 0:
		convertedProjection = storage.Projection(1)
	// Projection Value 1 in jacobsa/gcloud maps to Projection Value 2 in Go Client API, that is for "noAcl".
	case 1:
		convertedProjection = storage.Projection(2)
	// Default Projection value in jacobsa/gcloud library is 0 that maps to 1 in Go Client API interface, and that is for "full".
	default:
		convertedProjection = storage.Projection(1)
	}
	return convertedProjection
}

func (b *bucketHandle) ListObjects(ctx context.Context, req *gcs.ListObjectsRequest) (listing *gcs.Listing, err error) {
	// Converting *ListObjectsRequest to type *storage.Query as expected by the Go Storage Client.
	query := &storage.Query{
		Delimiter:                req.Delimiter,
		Prefix:                   req.Prefix,
		Projection:               getProjectionValue(req.ProjectionVal),
		IncludeTrailingDelimiter: req.IncludeTrailingDelimiter,
		IncludeFoldersAsPrefixes: req.IncludeFoldersAsPrefixes,
		//MaxResults: , (Field not present in storage.Query of Go Storage Library but present in ListObjectsQuery in Jacobsa code.)
	}
	itr := b.bucket.Objects(ctx, query) // Returning iterator to the list of objects.
	pi := itr.PageInfo()
	pi.MaxSize = req.MaxResults
	pi.Token = req.ContinuationToken
	var list gcs.Listing

	// Iterating through all the objects in the bucket and one by one adding them to the list.
	for {
		var attrs *storage.ObjectAttrs

		attrs, err = itr.Next()
		if err == iterator.Done {
			err = nil
			break
		}
		if err != nil {
			err = fmt.Errorf("Error in iterating through objects: %w", err)
			return
		}

		// Prefix attribute will be set for the objects returned as part of Prefix[] array in list response.
		// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/cloud.google.com/go/storage/storage.go#L1304
		// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/cloud.google.com/go/storage/http_client.go#L370
		if attrs.Prefix != "" {
			list.CollapsedRuns = append(list.CollapsedRuns, attrs.Prefix)
		} else {
			// Converting attrs to *Object type.
			currObject := storageutil.ObjectAttrsToBucketObject(attrs)
			list.Objects = append(list.Objects, currObject)
		}

		// itr.next returns all the objects present in the bucket. Hence adding a
		// check to break after iterating over the current page. pi.Remaining()
		// function returns number of items (items + prefixes) remaining in current
		// page to be iterated by iterator (itr). The func returns (number of items in current page - 1)
		// after first itr.Next() call and becomes 0 when iteration is done.
		// If req.MaxResults is 0, then wait till iterator is done. This is similar
		// to https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/github.com/jacobsa/gcloud/gcs/bucket.go#L164
		if req.MaxResults != 0 && (pi.Remaining() == 0) {
			break
		}
	}

	list.ContinuationToken = itr.PageInfo().Token
	listing = &list
	return
}

func (b *bucketHandle) UpdateObject(ctx context.Context, req *gcs.UpdateObjectRequest) (o *gcs.Object, err error) {
	obj := b.bucket.Object(req.Name)

	if req.Generation != 0 {
		obj = obj.Generation(req.Generation)
	}

	if req.MetaGenerationPrecondition != nil {
		obj = obj.If(storage.Conditions{MetagenerationMatch: *req.MetaGenerationPrecondition})
	}

	updateQuery := storage.ObjectAttrsToUpdate{}

	if req.ContentType != nil {
		updateQuery.ContentType = *req.ContentType
	}

	if req.ContentEncoding != nil {
		updateQuery.ContentEncoding = *req.ContentEncoding
	}

	if req.ContentLanguage != nil {
		updateQuery.ContentLanguage = *req.ContentLanguage
	}

	if req.CacheControl != nil {
		updateQuery.CacheControl = *req.CacheControl
	}

	if req.Metadata != nil {
		updateQuery.Metadata = make(map[string]string)
		for key, element := range req.Metadata {
			if element != nil {
				updateQuery.Metadata[key] = *element
			}
		}
	}

	attrs, err := obj.Update(ctx, updateQuery)

	if err == nil {
		// Converting objAttrs to type *Object
		o = storageutil.ObjectAttrsToBucketObject(attrs)
		return
	}

	// If storage object does not exist, httpclient is returning ErrObjectNotExist error instead of googleapi error
	// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/cloud.google.com/go/storage/http_client.go#L516
	switch ee := err.(type) {
	case *googleapi.Error:
		if ee.Code == http.StatusPreconditionFailed {
			err = &gcs.PreconditionError{Err: ee}
		}
	default:
		if err == storage.ErrObjectNotExist {
			err = &gcs.NotFoundError{Err: storage.ErrObjectNotExist}
		} else {
			err = fmt.Errorf("Error in updating object: %w", err)
		}
	}

	return
}

func (b *bucketHandle) ComposeObjects(ctx context.Context, req *gcs.ComposeObjectsRequest) (o *gcs.Object, err error) {
	dstObj := b.bucket.Object(req.DstName)

	dstObjConds := storage.Conditions{}
	if req.DstMetaGenerationPrecondition != nil {
		dstObjConds.MetagenerationMatch = *req.DstMetaGenerationPrecondition
	}
	// DstGenerationPrecondition or DoesNotExist should be set in dstObj
	// preconditions to make requests Idempotent.
	// https://github.com/GoogleCloudPlatform/gcsfuse/blob/7ad451c6f2ead7992e030503e5b66c555b2ebf71/vendor/cloud.google.com/go/storage/copy.go#L230
	if req.DstGenerationPrecondition != nil {
		if *req.DstGenerationPrecondition == 0 {
			dstObjConds.DoesNotExist = true
		} else {
			dstObjConds.GenerationMatch = *req.DstGenerationPrecondition
		}
	}
	// Only set conditions on dstObj if there is at least one condition in
	// dstObjConds. Otherwise, storage client library gives empty conditions error.
	// https://github.com/GoogleCloudPlatform/gcsfuse/blob/7ad451c6f2ead7992e030503e5b66c555b2ebf71/vendor/cloud.google.com/go/storage/storage.go#L1739
	if isStorageConditionsNotEmpty(dstObjConds) {
		dstObj = dstObj.If(dstObjConds)
	}

	// Converting the req.Sources list to a list of storage.ObjectHandle as expected by the Go Storage Client.
	var srcObjList []*storage.ObjectHandle
	for _, src := range req.Sources {
		currSrcObj := b.bucket.Object(src.Name)
		// Switching to requested Generation of the object.
		// Zero src generation is the latest generation, we are skipping it because by default it will take the latest one
		if src.Generation != 0 {
			currSrcObj = currSrcObj.Generation(src.Generation)
		}
		srcObjList = append(srcObjList, currSrcObj)
	}

	// Composing Source Objects to Destination Object using Composer created through Go Storage Client.
	attrs, err := dstObj.ComposerFrom(srcObjList...).Run(ctx)
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
			err = fmt.Errorf("Error in composing object: %w", err)
		}
		return
	}

	// Converting attrs to type *Object.
	o = storageutil.ObjectAttrsToBucketObject(attrs)

	return
}

func isStorageConditionsNotEmpty(conditions storage.Conditions) bool {
	return conditions != (storage.Conditions{})
}
