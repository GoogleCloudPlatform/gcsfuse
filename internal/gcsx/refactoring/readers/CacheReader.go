package readers

import (
	"context"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
)

type FileCacheReader struct {
	fileCacheHandler      *file.CacheHandler
	cacheFileForRangeRead bool
	fileCacheHandle       *file.CacheHandle
}

func (fcr *FileCacheReader) tryReadingFromFileCache(ctx context.Context, offset int64) (data []byte, cacheHit bool, err error) {
	// try read from cache using cacheReader and if cache hit true return data in data array
	return
}

func (fcr *FileCacheReader) Read(readBuffer []byte, offset int64, limit int) error {
	// readBuffer is the common buffer being passed from Read Manager and will be used to add data from disk
	return nil
}
