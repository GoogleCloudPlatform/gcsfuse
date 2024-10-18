package refactoring

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/poc"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// holds the responsibility of creating and maintaining single reader as well multi range reader instances
type ReaderFactory struct {
	bucket            gcs.Bucket
	singleRangeReader *SingleRangeReader
	mrr               *poc.MultiRangeDownloader
}

type SingleRangeReader struct {
	rc         *storage.Reader // Range Reader instance
	readHandle string          // field to hold the read handle (could be object name, etc.)
	bucket     *storage.BucketHandle
}

// GetMultiRangeReader creates a new MRD instance with multi range reading capabilities
func (r *ReaderFactory) GetMultiRangeReader(ctx context.Context) (*poc.MultiRangeDownloader, error) {
	if r.mrr != nil {
		return r.mrr, nil
	}
	// NewMultiRangeDownloader(ctx)

	return r.mrr, nil
}

// NewSingleRangeReader creates a new SingleRangeReader instance with range reading capabilities
func (r *ReaderFactory) NewSingleRangeReader(object *storage.ObjectHandle, start int, end int) (*SingleRangeReader, error) {
	if r.singleRangeReader != nil {
		return r.singleRangeReader, nil
	}
	//r.singleReader = r.bucket.NewReader(
	//	ctx,
	//	&gcs.ReadObjectRequest{
	//		Name:      object.ObjectName(),
	//		Generation: 0,
	//		Range: &gcs.ByteRange{
	//			Start: uint64(start),
	//			Limit: uint64(end),
	//		}
	//	})

	return r.singleRangeReader, nil
}
