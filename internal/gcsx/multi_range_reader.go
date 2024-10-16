// Copyright 2015 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcsx

import (
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/poc"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"golang.org/x/net/context"
)

// MB is 1 Megabyte

// NewMultiRangeReader create a multi range reader for the supplied object record that
// reads using the given bucket.
func NewMultiRangeReader(
	o *gcs.MinObject,
	bucket gcs.Bucket,
	sequentialReadSizeMb int32,
	fileCacheHandler *file.CacheHandler,
	cacheFileForRangeRead bool,
) *MultiRangeReader {
	return &MultiRangeReader{
		randomReader: randomReader{
			object:                o,
			bucket:                bucket,
			start:                 -1,
			limit:                 -1,
			seeks:                 0,
			totalReadBytes:        0,
			sequentialReadSizeMb:  sequentialReadSizeMb,
			fileCacheHandler:      fileCacheHandler,
			cacheFileForRangeRead: cacheFileForRangeRead,
		},
	}
}

type MultiRangeReader struct {
	randomReader
	mrd poc.MultiRangeDownloader
	// read handle data type can be anything
	readHandle string
}

func (mrr *MultiRangeReader) ReadAt(
	ctx context.Context,
	p []byte,
	offset int64) (n int, cacheHit bool, err error) {

	if offset >= int64(mrr.object.Size) {
		err = io.EOF
		return
	}

	// TODO: refactor into single method for caching -----------------------
	n, cacheHit, err = mrr.tryReadingFromFileCache(ctx, p, offset)
	if err != nil {
		err = fmt.Errorf("ReadAt: while reading from cache: %w", err)
		return
	}
	// Data was served from cache.
	if cacheHit || n == len(p) || (n < len(p) && uint64(offset)+uint64(n) == mrr.object.Size) {
		return
	}
	// ---------------------------------------------------------------------------
	for len(p) > 0 {
		// Have we blown past the end of the object?
		if offset >= int64(mrr.object.Size) {
			err = io.EOF
			return
		}

		// When the offset is AFTER the reader position, try to seek forward, within reason.
		// This happens when the kernel page cache serves some data. It's very common for
		// concurrent reads, often by only a few 128kB fuse read requests. The aim is to
		// re-use GCS connection and avoid throwing away already read data.
		// For parallel sequential reads to a single file, not throwing away the connections
		// is a 15-20x improvement in throughput: 150-200 MB/s instead of 10 MB/s.
		mrr.seekReaderToPosition(offset)

		readType := util.Sequential
		// If we don't have a reader, start a read operation.
		if mrr.reader == nil {
			readType, err = mrr.startRead(ctx, offset, int64(len(p)))
			if err != nil {
				err = fmt.Errorf("startRead: %w", err)
				return
			}
		}

		// Now we have a reader positioned at the correct place. Consume as much from
		// it as possible.
		var tmp int
		tmp, err = mrr.readFull(ctx, p, readType)

		n += tmp
		p = p[tmp:]
		mrr.start += int64(tmp)
		offset += int64(tmp)
		mrr.totalReadBytes += uint64(tmp)

		// Sanity check.
		if !mrr.sanityCheck() {
			return
		}

		// Are we finished with storage reader now
		if mrr.start == mrr.limit {
			//set readHandle here
			//mrr.readHandle = mrr.reader.getHandle()
			mrr.reader.Close()
			mrr.reader = nil
			mrr.cancel = nil

		}

		// Handle errors.
		switch {
		case err == io.EOF || err == io.ErrUnexpectedEOF:
			// For a non-empty buffer, ReadFull returns EOF or ErrUnexpectedEOF only
			// if the reader peters out early. That's fine, but it means we should
			// have hit the limit above.
			if mrr.reader != nil {
				err = fmt.Errorf("reader returned %d too few bytes", mrr.limit-mrr.start)
				return
			}

			err = nil

		case err != nil:
			// Propagate other errors.
			err = fmt.Errorf("readFull: %w", err)
			return
		}
	}

	return
}

func (mrr *MultiRangeReader) sanityCheck() bool {
	if mrr.start > mrr.limit {
		fmt.Errorf("reader returned %d too many bytes", mrr.start-mrr.limit)

		// Don't attempt to reuse the reader when it's behaving wackily.
		mrr.reader.Close()
		mrr.reader = nil
		mrr.cancel = nil
		mrr.start = -1
		mrr.limit = -1

		return false
	}
	return true
}

func (rr *MultiRangeReader) Object() (o *gcs.MinObject) {
	o = rr.object
	return
}

func (rr *MultiRangeReader) Destroy() {
	// Close out the reader, if we have one.
	if rr.reader != nil {
		err := rr.reader.Close()
		rr.reader = nil
		rr.cancel = nil
		if err != nil {
			logger.Warnf("rr.Destroy(): while closing reader: %v", err)
		}
	}

	if rr.fileCacheHandle != nil {
		logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", rr.fileCacheHandle, rr.bucket.Name(), rr.object.Name)
		err := rr.fileCacheHandle.Close()
		if err != nil {
			logger.Warnf("rr.Destroy(): while closing cacheFileHandle: %v", err)
		}
		rr.fileCacheHandle = nil
	}
}

// Like io.ReadFull, but deals with the cancellation issues.
//
// REQUIRES: rr.reader != nil
func (mrr *MultiRangeReader) readFull(
	ctx context.Context,
	p []byte, readType string) (n int, err error) {
	// Start a goroutine that will cancel the read operation we block on below if
	// the calling context is cancelled, but only if this method has not already
	// returned (to avoid souring the reader for the next read if this one is
	// successful, since the calling context will eventually be cancelled).
	readDone := make(chan struct{})
	defer close(readDone)

	go func() {
		select {
		case <-readDone:
			return

		case <-ctx.Done():
			select {
			case <-readDone:
				return

			default:
				mrr.cancel()
			}
		}
	}()

	// Call through.
	if readType == util.Random {
		// download a range
		//mrr.mrd.Add(p, start, end, callback)
		return
	}
	n, err = io.ReadFull(mrr.reader, p)

	return
}

// Ensure that rr.reader is set up for a range for which [start, start+size) is
// a prefix. Irrespective of the size requested, we try to fetch more data
// from GCS defined by sequentialReadSizeMb flag to serve future read requests.
func (mrr *MultiRangeReader) startRead(
	ctx context.Context,
	start int64,
	size int64) (readType string, err error) {
	// Make sure start and size are legal.
	if start < 0 || uint64(start) > mrr.object.Size || size < 0 {
		err = fmt.Errorf(
			"range [%d, %d) is illegal for %d-byte object",
			start,
			start+size,
			mrr.object.Size)
		return
	}

	// GCS requests are expensive. Prefer to issue read requests defined by
	// sequentialReadSizeMb flag. Sequential reads will simply sip from the fire house
	// with each call to ReadAt. In practice, GCS will fill the TCP buffers
	// with about 6 MB of data. Requests from outside GCP will be charged
	// about 6MB of egress data, even if less data is read. Inside GCP
	// regions, GCS egress is free. This logic should limit the number of
	// GCS read requests, which are not free.

	// But if we notice random read patterns after a minimum number of seeks,
	// optimise for random reads. Random reads will read data in chunks of
	// (average read size in bytes rounded up to the next MB).
	end := int64(mrr.object.Size)
	readType = util.Sequential
	if mrr.seeks >= minSeeksForRandom {
		readType = util.Random
		end = mrr.endOffsetForRandomRead(end, start)
	}

	end = mrr.endOffsetWithinMaxLimit(end, start)

	// Begin the read.
	ctx, cancel := context.WithCancel(context.Background())
	if readType == util.Random {
		multiDownloader, err := mrr.bucket.NewMultiRangeDownloader(ctx, mrr.object.Name)
		if err != nil {
			err = fmt.Errorf("NewReader: %w", err)
			return
		}
		mrr.mrd = *multiDownloader
	} else {
		rc, err := mrr.bucket.NewReader(
			ctx,
			&gcs.ReadObjectRequest{
				Name:       mrr.object.Name,
				Generation: mrr.object.Generation,
				Range: &gcs.ByteRange{
					Start: uint64(start),
					Limit: uint64(end),
				},
				ReadCompressed: mrr.object.HasContentEncodingGzip(),
			})

		if err != nil {
			err = fmt.Errorf("NewReader: %w", err)
			return
		}
		mrr.reader = rc
	}

	mrr.cancel = cancel
	mrr.start = start
	mrr.limit = end

	requestedDataSize := end - start
	monitor.CaptureGCSReadMetrics(ctx, readType, requestedDataSize)

	return
}
