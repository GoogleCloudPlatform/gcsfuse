package refactoring

import (
	"context"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
)

type CacheConfig struct {
	fileCacheHandler      *file.CacheHandler
	cacheFileForRangeRead bool
	fileCacheHandle       *file.CacheHandle
}

func (cc *CacheConfig) tryReadingFromFileCache(ctx context.Context, offset int64) (data []byte, cacheHit bool, err error) {
	// try read from cache using cacheConfig and if cache hit true return data in data array
	return
}
