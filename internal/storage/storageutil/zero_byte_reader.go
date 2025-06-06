package storageutil

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

func GetObjectSizeFromZeroByteReader(ctx context.Context, bucketName, objectName string) (int64, error) {
	// Create a new storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to create storage client: %w", err)
	}
	defer func(client *storage.Client) {
		err := client.Close()
		if err != nil {

		}
	}(client)

	// Get object handle
	obj := client.Bucket(bucketName).Object(objectName)

	// Create a new reader
	reader, err := obj.NewReader(ctx)
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
