package refactoring

import (
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// holds the responsibility of creating and maintaining single reader as well multi range reader instances
type ReaderFactory struct {
	bucket                gcs.Bucket
	readerInstanceMap     map[string]string
	fileCacheHandler      *file.CacheHandler
	cacheFileForRangeRead bool
	fileCacheHandle       *file.CacheHandle
	o                     *gcs.MinObject
	sequentialReadSizeMb  int32
	// inode is used for mrr and read handle
	inode *inode.FileInode
}

func (rf *ReaderFactory) GetReader(readerType string) (*Reader, error) {
	// return GCSRangeReader, Cache Reader pr adaptive prefetch reader based on type from readerInstanceMap
	// if not present in map, add new corresponding instance in map and return the newly created instance
	return nil, nil
}
