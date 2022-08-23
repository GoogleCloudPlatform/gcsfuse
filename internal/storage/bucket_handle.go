package storage

import (
	"crypto/md5"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	storagev1 "google.golang.org/api/storage/v1"
)

// Moto: We will implement Go Client Library functionality with existing
// gcs.Bucket interface provided by the jacobsa/gcloud, will get rid of that
// when Go Client Storage will be fully functional and robust.

type bucketHandle struct {
	gcs.Bucket
	bucket *storage.BucketHandle
}

func (bh *bucketHandle) Name() (name string) {
	attrs, err := bh.bucket.Attrs(context.Background())
	if err == nil {
		name = attrs.Name
	}
	return
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
		err = fmt.Errorf("Error in creating a NewRangeReader instance: %v", err)
		return
	}

	rc = io.NopCloser(r) // Converting io.Reader to io.ReadCloser.

	return
}

func (bh *bucketHandle) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (listing *gcs.Listing, err error) {

	// Explicitly converting Projection Value because the ProjectionVal interface of jacobsa/gcloud and Go Client API are not coupled correctly.
	var convertedProjection storage.Projection = storage.Projection(1) // Stores the Projection Value according to the Go Client API Interface.
	switch int(req.ProjectionVal) {
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

	// Converting *ListObjectsRequest to type *storage.Query as expected by the Go Storage Client.
	query := &storage.Query{
		Delimiter:                req.Delimiter,
		Prefix:                   req.Prefix,
		Projection:               convertedProjection,
		IncludeTrailingDelimiter: req.IncludeTrailingDelimiter,
		//MaxResults: , (Field not present in storage.Query of Go Storage Library but present in ListObjectsQuery in Jacobsa code.)
	}
	itr := bh.bucket.Objects(ctx, query) // Returning iterator to the list of objects.
	var list gcs.Listing

	// Iterating through all the objects in the bucket and one by one adding them to the list.
	for {
		var attrs *storage.ObjectAttrs
		attrs, err = itr.Next()
		if iterator.Done == err {
			err = nil
			break
		}
		if err != nil {
			err = fmt.Errorf("Error in iterating through objects: %v", err)
			return
		}

		// Converting attrs to *Object type.
		currObject := ObjectAttrsToBucketObject(attrs)
		list.Objects = append(list.Objects, currObject)
	}

	listing = &list
	return
}

// ObjectAttrsToBucketObject Convert the object attrs return by the Go Client to Object struct type present in object.go file.
func ObjectAttrsToBucketObject(attrs *storage.ObjectAttrs) *gcs.Object {
	// Converting []ACLRule returned by the Go Client into []*storagev1.ObjectAccessControl which complies with GCSFuse type.
	var Acl []*storagev1.ObjectAccessControl
	for _, element := range attrs.ACL {
		currACL := &storagev1.ObjectAccessControl{
			Entity:   string(element.Entity),
			EntityId: element.EntityID,
			Role:     string(element.Role),
			Domain:   element.Domain,
			Email:    element.Email,
			ProjectTeam: &storagev1.ObjectAccessControlProjectTeam{
				ProjectNumber: element.ProjectTeam.ProjectNumber,
				Team:          element.ProjectTeam.Team,
			},
		}
		Acl = append(Acl, currACL)
	}

	// Converting MD5[] slice to MD5[md5.Size] type fixed array as accepted by GCSFuse.
	var MD5 [md5.Size]byte
	copy(MD5[:], attrs.MD5)

	// Setting the parameters in Object and doing conversions as necessary.
	return &gcs.Object{
		Name:            attrs.Name,
		ContentType:     attrs.ContentType,
		ContentLanguage: attrs.ContentLanguage,
		CacheControl:    attrs.CacheControl,
		Owner:           attrs.Owner,
		Size:            uint64(attrs.Size),
		ContentEncoding: attrs.ContentEncoding,
		MD5:             &MD5,
		CRC32C:          &attrs.CRC32C,
		MediaLink:       attrs.MediaLink,
		Metadata:        attrs.Metadata,
		Generation:      attrs.Generation,
		MetaGeneration:  attrs.Metageneration,
		StorageClass:    attrs.StorageClass,
		Deleted:         attrs.Deleted,
		Updated:         attrs.Updated,
		//ComponentCount: , (Field not found in attrs returned by Go Client.)
		ContentDisposition: attrs.ContentDisposition,
		CustomTime:         string(attrs.CustomTime.Format(time.RFC3339)),
		EventBasedHold:     attrs.EventBasedHold,
		Acl:                Acl,
	}
}

func (bh *bucketHandle) StatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (o *gcs.Object, err error) {

	var attrs *storage.ObjectAttrs = nil
	// Retrieving object attrs through Go Storage Client.
	attrs, err = bh.bucket.Object(req.Name).Attrs(ctx)

	// If error is of type storage.ErrObjectNotExist, then we have to retry once by appending '/' to the object name.
	// We are retyring to handle the case when the object is a directory.
	// Since directories in GCS bucket are denoted with a an extra '/' at the end of their name. But in the request we are only provided with their name without '/'.
	if err == storage.ErrObjectNotExist {
		dirName := req.Name + "/"
		attrs, err = bh.bucket.Object(dirName).Attrs(ctx)
		if err == storage.ErrObjectNotExist {
			err = &gcs.NotFoundError{Err: err} // Special case error that object not found in the bucket.
			return
		}
	}
	if err != nil {
		err = fmt.Errorf("Error in returning object attributes: %v", err)
		return
	}

	// Converting attrs to type *Object
	o = ObjectAttrsToBucketObject(attrs)

	return
}

func (bh *bucketHandle) DeleteObject(
	ctx context.Context,
	req *gcs.DeleteObjectRequest) (err error) {

	obj := bh.bucket.Object(req.Name)

	// Switching to the requested generation of the object.
	if req.Generation != 0 {
		obj = obj.Generation(req.Generation)
	}

	// Putting condition that the object's MetaGeneration should match the requested MetaGeneration for deletion to occur.
	if req.MetaGenerationPrecondition != nil && *req.MetaGenerationPrecondition != 0 {
		obj = obj.If(storage.Conditions{MetagenerationMatch: *req.MetaGenerationPrecondition})
	}

	// Deleting object through Go Storage Client.
	err = obj.Delete(ctx)
	if err != nil {
		err = fmt.Errorf("Error in deleting the object through Go storage client: %v", err)
		return
	}

	return
}

func (bh *bucketHandle) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (o *gcs.Object, err error) {
	dstObj := bh.bucket.Object(req.DstName)

	// Putting Generation and MetaGeneration conditions on Destination Object.
	if req.DstGenerationPrecondition != nil {
		if req.DstMetaGenerationPrecondition != nil {
			dstObj = dstObj.If(storage.Conditions{GenerationMatch: *req.DstGenerationPrecondition, MetagenerationMatch: *req.DstMetaGenerationPrecondition})
		} else {
			dstObj = dstObj.If(storage.Conditions{GenerationMatch: *req.DstGenerationPrecondition})
		}
	} else if req.DstMetaGenerationPrecondition != nil {
		dstObj = dstObj.If(storage.Conditions{MetagenerationMatch: *req.DstMetaGenerationPrecondition})
	}

	// Converting the req.Sources list to a list of storage.ObjectHandle as expected by the Go Storage Client.
	var srcObjList []*storage.ObjectHandle
	for _, src := range req.Sources {
		currSrcObj := bh.bucket.Object(src.Name)
		// Switching to requested Generation of the object.
		if src.Generation != 0 {
			currSrcObj = currSrcObj.Generation(src.Generation)
		}
		srcObjList = append(srcObjList, currSrcObj)
	}

	// Composing Source Objects to Destination Object using Composer created through Go Storage Client.
	attrs, err := dstObj.ComposerFrom(srcObjList...).Run(ctx)
	if err != nil {
		err = fmt.Errorf("Error in composing objects through Go Storage Client: %v", err)
		return
	}

	// Converting attrs to type *Object.
	o = ObjectAttrsToBucketObject(attrs)
	return

}

func (bh *bucketHandle) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (o *gcs.Object, err error) {

	srcObj := bh.bucket.Object(req.SrcName)
	dstObj := bh.bucket.Object(req.DstName)

	// Switching to the requested Generation of Source Object.
	if req.SrcGeneration != 0 {
		srcObj = srcObj.Generation(req.SrcGeneration)
	}

	// Putting a condition that the MetaGeneration of source should match *req.SrcMetaGenerationPrecondition for copying operation to occur.
	if req.SrcMetaGenerationPrecondition != nil {
		srcObj = srcObj.If(storage.Conditions{MetagenerationMatch: *req.SrcMetaGenerationPrecondition})
	}

	// Copying Source Object to the Destination Object through a Copier created by Go Storage Client.
	objAttrs, err := dstObj.CopierFrom(srcObj).Run(ctx)
	if err != nil {
		err = fmt.Errorf("Error in copying using Go Storage Client: %v", err)
		return
	}

	// Converting objAttrs to type *Object
	o = ObjectAttrsToBucketObject(objAttrs)
	return
}

func (bh *bucketHandle) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {

	obj := bh.bucket.Object(req.Name)

	// Putting conditions on Generation and MetaGeneration of the object for upload to occur.
	if req.GenerationPrecondition != nil {
		if *req.GenerationPrecondition == 0 {
			// Passing because GenerationPrecondition = 0 means object does not exist in the GCS Bucket yet.
		} else if req.MetaGenerationPrecondition != nil && *req.MetaGenerationPrecondition != 0 {
			obj = obj.If(storage.Conditions{GenerationMatch: *req.GenerationPrecondition, MetagenerationMatch: *req.MetaGenerationPrecondition})
		} else {
			obj = obj.If(storage.Conditions{GenerationMatch: *req.GenerationPrecondition})
		}
	}

	// Creating a NewWriter with requested attributes, using Go Storage Client.
	// Chuck size for resumable upload is deafult i.e. 16MB.
	wc := obj.NewWriter(ctx)
	wc.ChunkSize = 0 // This will enable one shot upload and thus increase performance. JSON API Client also performs one-shot upload.
	//wc = gcs.SetAttrs(wc, req)

	// Copying contents from the request to the Writer. These contents will be copied to the newly created object / already existing object.
	if _, err = io.Copy(wc, req.Contents); err != nil {
		err = fmt.Errorf("Error in io.Copy: %v", err)
		return
	}

	// Closing the Writer.
	if err = wc.Close(); err != nil {
		err = fmt.Errorf("Error in closing writer: %v", err)
		return
	}

	attrs := wc.Attrs() // Retrieving the attributes of the created object.

	// Converting attrs to type *Object.
	o = ObjectAttrsToBucketObject(attrs)
	return
}

func (bh *bucketHandle) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (o *gcs.Object, err error) {

	obj := bh.bucket.Object(req.Name)

	// Switching to requested Generation of object.
	if req.Generation != 0 {
		obj = obj.Generation(req.Generation)
	}

	// Putting condition to ensure MetaGeneration of object matches *req.MetaGenerationPrecondition for update to occur.
	if req.MetaGenerationPrecondition != nil && *req.MetaGenerationPrecondition != 0 {
		obj = obj.If(storage.Conditions{MetagenerationMatch: *req.MetaGenerationPrecondition})
	}

	// Creating update query consisting of attributes to update in the object.
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

	// Updating parameters using Update method of Go Storage Client.
	attrs, err := obj.Update(ctx, updateQuery)
	if err == storage.ErrObjectNotExist {
		err = &gcs.NotFoundError{Err: err} // Handling special case of object not found.
		return
	} else if err != nil {
		err = fmt.Errorf("Error in updating object through Go Storage Client: %v", err)
		return
	}

	// Convert attrs to type *Object.
	o = ObjectAttrsToBucketObject(attrs)

	return
}
