package refactoring

import (
	"context"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// Reader is the main read orchestrator which will be orchestrating all flows for reading.
// This holds the state for current reads, cache config and common buffer to be used for zonal as well as
// non-zonal flows
type Reader struct {
	currenReader     DataReader
	currentReadState ReadStateAndReader
	readerFactory    ReaderFactory
	object           *gcs.MinObject
	cacheConfig      CacheConfig
}

type CacheConfig struct {
	fileCacheHandler      *file.CacheHandler
	cacheFileForRangeRead bool
	fileCacheHandle       *file.CacheHandle
}

// This struct holds the current state as well as current reader
type ReadStateAndReader struct {
	start                int64
	limit                int64
	seeks                uint64
	totalReadBytes       uint64
	sequentialReadSizeMb int32
	readType             string // default read type can be sequential
}

// Not passing buffer here to avoid side effect, the returned buffer has data and passed buffer dst can point to it
func (reader *Reader) read(ctx context.Context, offset int64, dst []byte) {

	// try to read from cache before trying to read from any kid of reader
	reader.tryReadingFromFileCache(ctx, offset)

	//readSize tell how much data is added in buffer to be returned to jacobsa,
	// this will be same len(p) > 0, changed it to this to make the loop more intuitive
	var readSize int = 0
	for readSize < len(dst) {

		// If we don't have any current reader, start a read operation assuming sequential read.
		if reader.currenReader == nil {
			reader.currenReader = reader.readerFactory.GetReader("sequential")

		}

		// rr.seekReaderToPosition(offset)

		// read data here
	}
}

func (reader *Reader) tryReadingFromFileCache(ctx context.Context, offset int64) (data []byte, cacheHit bool, err error) {
	// try read from cache using cacheConfig and if cache hit true return data in data array
	return
}
