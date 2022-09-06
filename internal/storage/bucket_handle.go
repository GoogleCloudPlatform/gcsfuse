package storage

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
)

type bucketHandle struct {
	gcs.Bucket
	bucket *storage.BucketHandle
}

func (bh *bucketHandle) NewReader(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rc io.ReadCloser, err error) {
	return
}
func (b *bucketHandle) DeleteObject(ctx context.Context, req *gcs.DeleteObjectRequest) (err error) {
	ctx = context.Background()

	if err := b.bucket.Object(req.Name).Delete(ctx); err != nil {
		fmt.Println("Error Occurred while deleting object")
	}
	// Propagate other errors.
	if err != nil {
		return
	}
	return
}
