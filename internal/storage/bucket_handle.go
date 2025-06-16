// Copyright 2022 Google LLC
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
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"google.golang.org/api/iterator"
)

const FullFolderPathHNS = "projects/_/buckets/%s/folders/%s"
const FullBucketPathHNS = "projects/_/buckets/%s"

type bucketHandle struct {
	gcs.Bucket
	bucket             *storage.BucketHandle
	bucketName         string
	bucketType         *gcs.BucketType
	controlClient      StorageControlClient
	enableRapidAppends bool
}

func (bh *bucketHandle) Name() string {
	return bh.bucketName
}

func (bh *bucketHandle) BucketType() gcs.BucketType {
	return *bh.bucketType
}

func (bh *bucketHandle) NewReaderWithReadHandle(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (reader gcs.StorageReader, err error) {

	defer func() {
		err = gcs.GetGCSError(err)
	}()

	// Initialising the starting offset and the length to be read by the reader.
	start := int64(0)
	length := int64(-1)
	// Following the semantics of NewRangeReader method.
	// If length is negative, the object is read until the end.
	// If offset is negative, the object is read abs(offset) bytes from the end,
	// and length must also be negative to indicate all remaining bytes will be read.
	// Ref: https://github.com/GoogleCloudPlatform/gcsfuse/blob/34211af652dbaeb012b381a3daf3c94b95f65e00/vendor/cloud.google.com/go/storage/reader.go#L80
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

	// Insert ReadHandle into objectHandle if present.
	// Objects that have been opened can be opened again using readHandle at lower latency.
	// This produces the exact same object and generation and does not check if
	// the generation is still the newest one.
	if req.ReadHandle != nil {
		obj = obj.ReadHandle(req.ReadHandle)
	}

	// NewRangeReader creates a "storage.Reader" object which is also io.ReadCloser since it contains both Read() and Close() methods present in io.ReadCloser interface.
	storageReader, err := obj.NewRangeReader(ctx, start, length)
	if err == nil {
		reader = newGCSFullReadCloser(storageReader)
	}
	return
}

func (bh *bucketHandle) DeleteObject(ctx context.Context, req *gcs.DeleteObjectRequest) (err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	obj := bh.bucket.Object(req.Name)

	// Switching to the requested generation of the object. By default, generation
	// is 0 which signifies the latest generation. Note: GCS will delete the
	// live object even if generation is not set in request. We are passing 0
	// generation explicitly to satisfy idempotency condition.
	obj = obj.Generation(req.Generation)

	// Putting condition that the object's MetaGeneration should match the requested MetaGeneration for deletion to occur.
	if req.MetaGenerationPrecondition != nil && *req.MetaGenerationPrecondition != 0 {
		obj = obj.If(storage.Conditions{MetagenerationMatch: *req.MetaGenerationPrecondition})
	}

	err = obj.Delete(ctx)
	if err != nil {
		err = fmt.Errorf("error in deleting object: %w", err)
	}
	return
}

func (bh *bucketHandle) StatObject(ctx context.Context,
	req *gcs.StatObjectRequest) (m *gcs.MinObject, e *gcs.ExtendedObjectAttributes, err error) {

	defer func() {
		err = gcs.GetGCSError(err)
	}()

	var attrs *storage.ObjectAttrs
	// Retrieving object attrs through Go Storage Client.
	attrs, err = bh.bucket.Object(req.Name).Attrs(ctx)
	if err != nil {
		err = fmt.Errorf("error in fetching object attributes: %w", err)
		return
	}
	if attrs.Finalized.IsZero() {
		if err = bh.fetchLatestSizeOfUnfinalizedObject(ctx, attrs); err != nil {
			err = fmt.Errorf("failed to fetch the latest size of unfinalized object %q: %w", attrs.Name, err)
			return
		}
	}

	// Converting attrs to type *Object
	o := storageutil.ObjectAttrsToBucketObject(attrs)
	m = storageutil.ConvertObjToMinObject(o)
	if req.ReturnExtendedObjectAttributes {
		e = storageutil.ConvertObjToExtendedObjectAttributes(o)
	}

	return
}

// Note: This is not production ready code and will be removed once StatObject
// requests return correct attr values for appendable objects.
func (bh *bucketHandle) fetchLatestSizeOfUnfinalizedObject(ctx context.Context, attrs *storage.ObjectAttrs) error {
	if bh.BucketType().Zonal && bh.enableRapidAppends {
		// Get object handle
		obj := bh.bucket.Object(attrs.Name)
		// Create a new reader
		reader, err := obj.NewRangeReader(ctx, 0, 0)
		if err != nil {
			return fmt.Errorf("failed to create zero-byte reader for object %q: %v", attrs.Name, err)
		}
		err = reader.Close()
		if err != nil {
			logger.Warnf("failed to close zero-byte reader for object %q: %v", attrs.Name, err)
		}

		// Set the size
		attrs.Size = reader.Attrs.Size
		return nil
	}
	return nil
}

func (bh *bucketHandle) getObjectHandleWithPreconditionsSet(req *gcs.CreateObjectRequest) *storage.ObjectHandle {
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
	return obj
}

func (bh *bucketHandle) CreateObject(ctx context.Context, req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	obj := bh.getObjectHandleWithPreconditionsSet(req)

	// Creating a NewWriter with requested attributes, using Go Storage Client.
	// Chuck size for resumable upload is default i.e. 16MB.
	wc := obj.NewWriter(ctx)
	wc.ChunkTransferTimeout = time.Duration(req.ChunkTransferTimeoutSecs) * time.Second
	wc = storageutil.SetAttrsInWriter(wc, req)
	wc.ProgressFunc = func(bytesUploadedSoFar int64) {
		logger.Tracef("gcs: Req %#16x: -- CreateObject(%q): %20v bytes uploaded so far", ctx.Value(gcs.ReqIdField), req.Name, bytesUploadedSoFar)
	}
	// All objects in zonal buckets must be appendable.
	wc.Append = bh.BucketType().Zonal
	// FinalizeOnClose should be true for all writes for now.
	wc.FinalizeOnClose = true

	// Copy the contents to the writer.
	if _, err = io.Copy(wc, req.Contents); err != nil {
		err = fmt.Errorf("error in io.Copy: %w", err)
		return
	}

	// We can't use defer to close the writer, because we need to close the
	// writer successfully before calling Attrs() method of writer.
	if err = wc.Close(); err != nil {
		err = fmt.Errorf("error in closing writer : %w", err)
		return
	}

	attrs := wc.Attrs() // Retrieving the attributes of the created object.
	// Converting attrs to type *Object.
	o = storageutil.ObjectAttrsToBucketObject(attrs)
	return
}

func (bh *bucketHandle) CreateObjectChunkWriter(ctx context.Context, req *gcs.CreateObjectRequest, chunkSize int, callBack func(bytesUploadedSoFar int64)) (gcs.Writer, error) {
	obj := bh.getObjectHandleWithPreconditionsSet(req)

	wc := &ObjectWriter{obj.NewWriter(ctx)}
	wc.ChunkSize = chunkSize
	wc.Writer = storageutil.SetAttrsInWriter(wc.Writer, req)
	wc.ChunkTransferTimeout = time.Duration(req.ChunkTransferTimeoutSecs) * time.Second
	if callBack == nil {
		callBack = func(bytesUploadedSoFar int64) {
			logger.Tracef("gcs: Req %#16x: -- UploadBlock(%q): %20v bytes uploaded so far", ctx.Value(gcs.ReqIdField), req.Name, bytesUploadedSoFar)
		}
	}
	wc.ProgressFunc = callBack
	// All objects in zonal buckets must be appendable.
	wc.Append = bh.BucketType().Zonal
	// FinalizeOnClose should be true for all writes for now.
	wc.FinalizeOnClose = true

	return wc, nil
}

func (bh *bucketHandle) CreateAppendableObjectWriter(ctx context.Context,
	req *gcs.CreateObjectChunkWriterRequest) (gcs.Writer, error) {
	obj := bh.getObjectHandleWithPreconditionsSet(&req.CreateObjectRequest)
	// To create the takeover writer, the objectHandle.Generation must be set.
	obj = obj.Generation(*req.CreateObjectRequest.GenerationPrecondition)
	callBack := func(bytesUploadedSoFar int64) {
		logger.Tracef("gcs: Req %#16x: -- UploadBlock(%q): %20v bytes uploaded so far", ctx.Value(gcs.ReqIdField), req.Name, bytesUploadedSoFar)
	}

	opts := storage.AppendableWriterOpts{
		ChunkSize:       req.ChunkSize,
		ProgressFunc:    callBack,
		FinalizeOnClose: false,
	}

	tw, off, err := obj.NewWriterFromAppendableObject(ctx, &opts) // Takeover writer tw created from offset off.

	if err != nil {
		err = fmt.Errorf("error while creating appendable object writer : %w", err)
		return nil, err
	}

	if off != req.Offset {
		err = fmt.Errorf("takeover offset for the created appendable object writer does not match the requested offset")
		return nil, err
	}
	w := &ObjectWriter{tw}
	return w, err
}

func (bh *bucketHandle) FinalizeUpload(ctx context.Context, w gcs.Writer) (o *gcs.MinObject, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	if err = w.Close(); err != nil {
		err = fmt.Errorf("error in closing writer : %w", err)
		return
	}

	attrs := w.Attrs() // Retrieving the attributes of the created object.
	// Converting attrs to type *MinObject.
	o = storageutil.ObjectAttrsToMinObject(attrs)
	return
}

func (bh *bucketHandle) FlushPendingWrites(ctx context.Context, w gcs.Writer) (o *gcs.MinObject, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	_, err = w.Flush()
	if err != nil {
		err = fmt.Errorf("error in FlushPendingWrites : %w", err)
		return
	}

	attrs := w.Attrs() // Retrieving the attributes of the created object.
	// Converting attrs to type *MinObject.
	o = storageutil.ObjectAttrsToMinObject(attrs)
	if o == nil {
		return nil, fmt.Errorf("FlushPendingWrites: nil object returned after w.Flush()")
	}
	return
}

func (bh *bucketHandle) CopyObject(ctx context.Context, req *gcs.CopyObjectRequest) (o *gcs.Object, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	srcObj := bh.bucket.Object(req.SrcName)
	dstObj := bh.bucket.Object(req.DstName)

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
		err = fmt.Errorf("error in copying object: %w", err)
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

func (bh *bucketHandle) ListObjects(ctx context.Context, req *gcs.ListObjectsRequest) (listing *gcs.Listing, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	// Converting *ListObjectsRequest to type *storage.Query as expected by the Go Storage Client.
	query := &storage.Query{
		Delimiter:                req.Delimiter,
		Prefix:                   req.Prefix,
		Projection:               getProjectionValue(req.ProjectionVal),
		IncludeTrailingDelimiter: req.IncludeTrailingDelimiter,
		IncludeFoldersAsPrefixes: req.IncludeFoldersAsPrefixes,
		//MaxResults: , (Field not present in storage.Query of Go Storage Library but present in ListObjectsQuery in Jacobsa code.)
	}
	minObjAttrs := []string{"Name", "Size", "Generation", "Metageneration", "Updated", "Metadata", "ContentEncoding", "CRC32C"}
	if bh.BucketType().Zonal {
		// For regional buckets, partial response API fails to populate the Finalized field.(b/398916957)
		// For objects in regional buckets, this field will be *unset*.
		minObjAttrs = append(minObjAttrs, "Finalized")
	}
	err = query.SetAttrSelection(minObjAttrs)

	if err != nil {
		err = fmt.Errorf("error while setting attribute selection for List Object query :%w", err)
		return
	}

	itr := bh.bucket.Objects(ctx, query) // Returning iterator to the list of objects.
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
			err = fmt.Errorf("error in iterating through objects: %w", err)
			return
		}
		if attrs.Finalized.IsZero() {
			if err = bh.fetchLatestSizeOfUnfinalizedObject(ctx, attrs); err != nil {
				err = fmt.Errorf("failed to fetch the latest size of unfinalized object %q: %w", attrs.Name, err)
				return
			}
		}

		// Prefix attribute will be set for the objects returned as part of Prefix[] array in list response.
		// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/cloud.google.com/go/storage/storage.go#L1304
		// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/cloud.google.com/go/storage/http_client.go#L370
		if attrs.Prefix != "" {
			list.CollapsedRuns = append(list.CollapsedRuns, attrs.Prefix)
		} else {
			// Converting attrs to *Object type.
			currMinObject := storageutil.ObjectAttrsToMinObject(attrs)
			list.MinObjects = append(list.MinObjects, currMinObject)
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

func (bh *bucketHandle) UpdateObject(ctx context.Context, req *gcs.UpdateObjectRequest) (o *gcs.Object, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	obj := bh.bucket.Object(req.Name)

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

	if err != nil {
		err = fmt.Errorf("error in updating object: %w", err)
		return
	}

	// Converting objAttrs to type *Object
	o = storageutil.ObjectAttrsToBucketObject(attrs)
	return
}

func (bh *bucketHandle) ComposeObjects(ctx context.Context, req *gcs.ComposeObjectsRequest) (o *gcs.Object, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	dstObj := bh.bucket.Object(req.DstName)

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
		currSrcObj := bh.bucket.Object(src.Name)
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
		err = fmt.Errorf("error in composing object: %w", err)
		return
	}

	// Converting attrs to type *Object.
	o = storageutil.ObjectAttrsToBucketObject(attrs)
	return
}

func (bh *bucketHandle) DeleteFolder(ctx context.Context, folderName string) (err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	var callOptions []gax.CallOption

	err = bh.controlClient.DeleteFolder(ctx, &controlpb.DeleteFolderRequest{
		Name: fmt.Sprintf(FullFolderPathHNS, bh.bucketName, folderName),
	}, callOptions...)
	return
}

func (bh *bucketHandle) MoveObject(ctx context.Context, req *gcs.MoveObjectRequest) (o *gcs.Object, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	obj := bh.bucket.Object(req.SrcName)

	// Switching to the requested generation of source object.
	if req.SrcGeneration != 0 {
		obj = obj.Generation(req.SrcGeneration)
	}

	// Putting a condition that the metaGeneration of source should match *req.SrcMetaGenerationPrecondition for move operation to occur.
	if req.SrcMetaGenerationPrecondition != nil {
		obj = obj.If(storage.Conditions{MetagenerationMatch: *req.SrcMetaGenerationPrecondition})
	}

	dstMoveObject := storage.MoveObjectDestination{
		Object:     req.DstName,
		Conditions: nil,
	}

	attrs, err := obj.Move(ctx, dstMoveObject)
	if err != nil {
		err = fmt.Errorf("error in moving object: %w", err)
		return
	}

	// Converting objAttrs to type *Object
	o = storageutil.ObjectAttrsToBucketObject(attrs)
	return
}

func (bh *bucketHandle) RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (folder *gcs.Folder, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	var controlFolder *controlpb.Folder
	req := &controlpb.RenameFolderRequest{
		Name:                fmt.Sprintf(FullFolderPathHNS, bh.bucketName, folderName),
		DestinationFolderId: destinationFolderId,
	}
	resp, err := bh.controlClient.RenameFolder(ctx, req)
	if err != nil {
		err = fmt.Errorf("error in renaming folder: %w", err)
		return
	}

	// Wait blocks until the long-running operation is completed,
	// returning the response and any errors encountered.
	controlFolder, err = resp.Wait(ctx)
	if err != nil {
		err = fmt.Errorf("error in getting result from renaming folder response: %w", err)
		return
	}

	folder = gcs.GCSFolder(bh.bucketName, controlFolder)
	return
}

// TODO: Consider adding this method to the bucket interface if additional
// layout options are needed in the future.
func (bh *bucketHandle) getStorageLayout() (*controlpb.StorageLayout, error) {
	var callOptions []gax.CallOption
	stoargeLayout, err := bh.controlClient.GetStorageLayout(context.Background(), &controlpb.GetStorageLayoutRequest{
		Name:      fmt.Sprintf("projects/_/buckets/%s/storageLayout", bh.bucketName),
		Prefix:    "",
		RequestId: "",
	}, callOptions...)

	return stoargeLayout, err
}

func (bh *bucketHandle) GetFolder(ctx context.Context, folderName string) (folder *gcs.Folder, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	var callOptions []gax.CallOption
	var clientFolder *controlpb.Folder
	clientFolder, err = bh.controlClient.GetFolder(ctx, &controlpb.GetFolderRequest{
		Name: fmt.Sprintf(FullFolderPathHNS, bh.bucketName, folderName),
	}, callOptions...)

	if err != nil {
		err = fmt.Errorf("error getting metadata for folder: %s, %w", folderName, err)
		return
	}

	folder = gcs.GCSFolder(bh.bucketName, clientFolder)
	return
}

func (bh *bucketHandle) CreateFolder(ctx context.Context, folderName string) (folder *gcs.Folder, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	req := &controlpb.CreateFolderRequest{
		Parent:    fmt.Sprintf(FullBucketPathHNS, bh.bucketName),
		FolderId:  folderName,
		Recursive: true,
	}

	clientFolder, err := bh.controlClient.CreateFolder(ctx, req)
	if err != nil {
		err = fmt.Errorf("error in creating folder: %w", err)
		return
	}

	folder = gcs.GCSFolder(bh.bucketName, clientFolder)
	return
}

func (bh *bucketHandle) NewMultiRangeDownloader(
	ctx context.Context, req *gcs.MultiRangeDownloaderRequest) (mrd gcs.MultiRangeDownloader, err error) {
	defer func() {
		err = gcs.GetGCSError(err)
	}()

	obj := bh.bucket.Object(req.Name)

	// Switching to the requested generation of object.
	if req.Generation != 0 {
		obj = obj.Generation(req.Generation)
	}

	if req.ReadCompressed {
		obj = obj.ReadCompressed(true)
	}

	mrd, err = obj.NewMultiRangeDownloader(ctx)
	return
}

func (bh *bucketHandle) GCSName(obj *gcs.MinObject) string {
	return obj.Name
}

func isStorageConditionsNotEmpty(conditions storage.Conditions) bool {
	return conditions != (storage.Conditions{})
}
