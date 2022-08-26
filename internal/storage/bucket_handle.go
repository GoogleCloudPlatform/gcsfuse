package storage

import (
	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
)

type bucketHandle struct {
	gcs.Bucket
	bucket *storage.BucketHandle
}
