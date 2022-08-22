package gcsx

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
		var attrs *storage.ObjectAttrs = nil
		attrs, err = itr.Next()
		if err == iterator.Done {
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

// Convert the object attrs return by the Go Client to Object struct type present in object.go file.
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
