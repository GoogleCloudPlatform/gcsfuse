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
	ctx context.Context,
) *MultiRangeReader {
	mrd, _ := poc.NewMultiRangeDownloader(ctx)
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
		mrd: mrd,
	}
}

type MultiRangeReader struct {
	randomReader
	mrd *poc.MultiRangeDownloader
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

		mrr.seekReaderToPosition(offset)

		readType := util.Sequential
		if mrr.seeks >= minSeeksForRandom {
			readType = util.Random
			if mrr.mrd == nil {
				err = mrr.startRandomRead(ctx, offset, int64(len(p)))
				if err != nil {
					err = fmt.Errorf("startRead: %w", err)
					return
				}
			}

		}

		// If we don't have a reader and read type is sequential, start a read operation for sequential.
		if readType == util.Sequential && mrr.reader == nil {
			err = mrr.startRead(ctx, offset, int64(len(p)))
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

func (mrr *MultiRangeReader) startRandomRead(ctx context.Context, start int64, size int64) error {
	err := mrr.sanityCheckForOffset(start, size)
	if err != nil {
		return err
	}
	end := int64(mrr.object.Size)
	end = mrr.endOffsetForRandomRead(end, start)
	end = mrr.endOffsetWithinMaxLimit(end, start)

	ctx, cancel := context.WithCancel(context.Background())
	rc, err := mrr.bucket.NewMultiRangeDownloader(
		ctx,
		mrr.object.Name)

	if err != nil {
		err = fmt.Errorf("New multi range downloader: %w", err)
		return err
	}

	mrr.mrd = rc
	mrr.cancel = cancel
	mrr.start = start
	mrr.limit = end

	requestedDataSize := end - start
	monitor.CaptureGCSReadMetrics(ctx, util.Random, requestedDataSize)

	return nil
}
