package refactoring

import (
	"context"
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// File Handle has this instance and will call its read method to read for any type of bucket
// ReadManager is the main read orchestrator which will be orchestrating all flows for reading.
// This has list of all readers
type ReadManager struct {
	// this field represents list of types of reader
	readers []Reader
}

func NewReadManager(
	bucket gcs.Bucket,
	cacheHandler *file.CacheHandler,
	fileCacheHandle *file.CacheHandle,
	obj *gcs.MinObject,
	seqReadSizeMb int32,
	inode *inode.FileInode,
) *ReadManager {
	// Initialize the readerFactory
	readerFactory := ReaderFactory{
		bucket:                bucket,
		readerInstanceMap:     make(map[string]string), // Initialize empty map
		fileCacheHandler:      cacheHandler,
		cacheFileForRangeRead: false, // or true, depending on the use case
		fileCacheHandle:       fileCacheHandle,
		o:                     obj,
		sequentialReadSizeMb:  seqReadSizeMb,
		inode:                 inode,
	}

	var readers []Reader

	// Add readers based on flag or default
	cacheReader, err := readerFactory.GetReader("cache")
	if err != nil {
		readers = append(readers, *cacheReader)
	}
	// in case adaptive prefetch flag is true, append it to list
	readerFactory.GetReader("adaptivePrefetch")

	// add the default reader
	defaultReader, _ := readerFactory.GetReader("adaptivePrefetch")
	readers = append(readers, *defaultReader)

	// Instantiate ReadManager with an empty slice of Readers and the initialized factory
	return &ReadManager{
		readers: readers,
	}
}

// dst is the buffer passed form read handle, it can be jacobsa buffer or
// it can be buffer created by read handle in case vectored read is ON
func (rm *ReadManager) read(ctx context.Context, offset int64, dst []byte) (int, error) {
	// Iterate through the slice and call the Read method
	n := 0
	for _, reader := range rm.readers {
		n, err := reader.Read(ctx, dst, offset)

		if err != nil {
			fmt.Printf("Error reading from %s: %v\n", reader, err)
			continue // print error and continue to the next reader
		}

		// Exit the loop if the data is read
		if n > 0 {
			break
		}
	}
	return n, nil
}
