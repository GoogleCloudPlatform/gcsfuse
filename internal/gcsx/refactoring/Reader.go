package refactoring

import (
	"context"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

// File Handle has this instance and will call its read method to read for any type of bucket
// Reader is the main read orchestrator which will be orchestrating all flows for reading.
// This holds the state for current reads, cache config and common buffer to be used for zonal as well as
// non-zonal flows
type Reader struct {
	// this field represents single range reader both with and without read handle
	singleRangeReader *SingleRangeReader
	currentReadState  ReadStateAndReader
	readerFactory     ReaderFactory
	object            *gcs.MinObject
	cacheConfig       CacheConfig
	bucket            gcs.Bucket
	localBuffer       []byte
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
func (reader *Reader) read(ctx context.Context, offset int64, limit int) []byte {

	// check if data can be served from local buffer

	// logic to read from cache is moved to separate file cache config
	reader.cacheConfig.tryReadingFromFileCache(ctx, offset)

	//readSize tell how much data is added in buffer to be returned to jacobsa,
	// this will be same len(p) > 0, changed it to this to make the loop more intuitive
	var readSize int = 0
	for readSize < limit {

		// Check based on seek if its random or sequntial operation

		// New read or in case of random read
		if reader.singleRangeReader == nil {
			end := -1
			// Update end with same logic as rr
			reader.singleRangeReader, _ = reader.readerFactory.NewSingleRangeReader(nil, int(offset), end)
			//read from single rang reader
		}

		// SEQUENTIAL FLOW
		// try to seek this single reader to current offset similar to rr.seekReaderToPosition(offset)
		readType := util.Sequential
		// read type will become  random based on seeks or if offset != current
		//if rr.seeks >= minSeeksForRandom {
		//	readType = util.Random
		//	end = rr.endOffsetForRandomRead(end, start)
		// reader.singleRangeReader = nil
		//}

		if readType == "random" {
			if string(reader.bucket.BucketType()) == "Zonal" {
				reader.readerFactory.GetMultiRangeReader(nil)
				//read data into localBuffer
				//increase readSize by number of bytes read
			} else {
				// set end offset
				end := -1
				reader.singleRangeReader, _ = reader.readerFactory.NewSingleRangeReader(nil, int(offset), end)
				//read data into localBuffer
				//increase readSize by number of bytes read
			}
		} else {
			// set end offset
			end := -1
			reader.singleRangeReader, _ = reader.readerFactory.NewSingleRangeReader(nil, int(offset), end)
			//read data into localBuffer
			//increase readSize by number of bytes read
		}
	}
	return reader.localBuffer
}
