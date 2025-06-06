package storageutil

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

func GetObjectSizeFromZeroByteReader(ctx context.Context, bh *storage.BucketHandle, objectName string) (int64, error) {
	// Get object handle
	obj := bh.Object(objectName)

	// Create a new reader
	reader, err := obj.NewRangeReader(ctx, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to create reader: %w", err)
	}
	err = reader.Close()

	// Return the size
	return reader.Attrs.Size, err
}
