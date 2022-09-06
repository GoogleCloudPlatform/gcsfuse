package storage

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
	"google.golang.org/api/iterator"
)

type bucketHandle struct {
	gcs.Bucket
	bucket *storage.BucketHandle
}

//func(bh *bucketHandle) Name string {
//}

func (bh *bucketHandle) NewReader(
		ctx context.Context,
		req *gcs.ReadObjectRequest) (rc io.ReadCloser, err error) {
	// make a call to gcs
	return
}
func (b *bucket) DeleteObject(
		ctx context.Context,
		req *DeleteObjectRequest) (err error) {

	// Construct an appropriate URL (cf. http://goo.gl/TRQJjZ).
	ctx = context.Background()
	client, err := storage.NewClient(ctx)
	name := b.Name()
	bucket := client.Bucket(name)

	it := bucket.Objects(ctx, nil)
	for {
		objAttrs, err := it.Next()
		if err != nil && err != iterator.Done {
			fmt.Println("Error Occurred while deleting object")
		}
		if err == iterator.Done {
			break
		}
		if req.Name == objAttrs.Name {
			if err := bucket.Object(objAttrs.Name).Delete(ctx); err != nil {
				fmt.Println("Error Occurred while deleting object")
			}
		}
	}

	// Propagate other errors.
	if err != nil {
		return
	}

	return
}
