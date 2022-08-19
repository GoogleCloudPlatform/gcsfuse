package gcsx

import (
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// Moto: We will implement Go Client Library functionality with existing
// gcs.Bucket interface provided by the jacobsa/gcloud, will get rid of that
// when Go Client Storage will be fully functional and robust.

type bucketHandle struct {
	gcs.Bucket
	bucket *storage.BucketHandle
}

func (bh *bucketHandle) Name() string {
	return bh.Name()
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
