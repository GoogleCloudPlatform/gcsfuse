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
	reader, err := obj.NewRangeReader(ctx, 0, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to create reader: %w", err)
	}
	defer func(reader *storage.Reader) {
		err := reader.Close()
		if err != nil {

		}
	}(reader)

	// Get the object attributes from the reader
	attrs := reader.Attrs
	// Return the size
	return attrs.Size, nil
}
