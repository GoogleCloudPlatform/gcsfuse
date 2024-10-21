package readers

import (
	"cloud.google.com/go/storage"
)

// NewSingleRangeReader creates a new SingleRangeReader instance with range reading capabilities
// This is used for sequential and random for non zonal buckets and sequnetial for zonal buckets
type AdaptivePrefetchReader struct {
	bucket *storage.BucketHandle
}

func (sr *AdaptivePrefetchReader) NewAdaptivePrefetchReader(object *storage.ObjectHandle, start int, end int) (*AdaptivePrefetchReader, error) {

	// Create new range reader

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

	return nil, nil
}

func (sr *AdaptivePrefetchReader) Read(readBuffer []byte, offset int64, limit int) error {
	// readBuffer is the common buffer being passed from Read Manager and will be relayed to go sdk
	return nil
}
