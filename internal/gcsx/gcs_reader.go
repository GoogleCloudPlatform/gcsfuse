// Copyright 2025 Google LLC
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
	"context"
	"fmt"
	"io"
	"math"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

type gcsReader struct {
	object *gcs.MinObject
	bucket gcs.Bucket

	// If non-nil, an in-flight read request and a function for cancelling it.
	//
	// INVARIANT: (reader == nil) == (cancel == nil)
	reader gcs.StorageReader
	cancel func()

	// The range of the object that we expect reader to yield, when reader is
	// non-nil. When reader is nil, limit is the limit of the previous read
	// operation, or -1 if there has never been one.
	//
	// INVARIANT: start <= limit
	// INVARIANT: limit < 0 implies reader != nil
	// All these properties will be used only in case of GCS reads and not for
	// reads from cache.
	start          int64
	limit          int64
	seeks          uint64
	totalReadBytes uint64

	// ReadType of the reader. Will be sequential by default.
	readType string

	sequentialReadSizeMb int32

	// Stores the handle associated with the previously closed newReader instance.
	// This will be used while making the new connection to bypass auth and metadata
	// checks.
	readHandle []byte
}

func (gr *gcsReader) CheckInvariants() {
	// INVARIANT: (reader == nil) == (cancel == nil)
	if (gr.reader == nil) != (gr.cancel == nil) {
		panic(fmt.Sprintf("Mismatch: %v vs. %v", gr.reader == nil, gr.cancel == nil))
	}

	// INVARIANT: start <= limit
	if !(gr.start <= gr.limit) {
		panic(fmt.Sprintf("Unexpected range: [%d, %d)", gr.start, gr.limit))
	}

	// INVARIANT: limit < 0 implies reader != nil
	if gr.limit < 0 && gr.reader != nil {
		panic(fmt.Sprintf("Unexpected non-nil reader with limit == %d", gr.limit))
	}
}

func (gr *gcsReader) ReadAt(ctx context.Context, p []byte, offset int64) (ObjectData, error) {
	// When the offset is AFTER the reader position, try to seek forward, within reason.
	// This happens when the kernel page cache serves some data. It's very common for
	// concurrent reads, often by only a few 128kB fuse read requests. The aim is to
	// re-use GCS connection and avoid throwing away already read data.
	// For parallel sequential reads to a single file, not throwing away the connections
	// is a 15-20x improvement in throughput: 150-200 MB/s instead of 10 MB/s.
	if gr.reader != nil && gr.start < offset && offset-gr.start < maxReadSize {
		bytesToSkip := offset - gr.start
		p := make([]byte, bytesToSkip)
		n, _ := io.ReadFull(gr.reader, p)
		gr.start += int64(n)
	}

	// If we have an existing reader, but it's positioned at the wrong place,
	// clean it up and throw it away.
	// We will also clean up the existing reader if it can't serve the entire request.
	dataToRead := math.Min(float64(offset+int64(len(p))), float64(gr.object.Size))
	if gr.reader != nil && (gr.start != offset || int64(dataToRead) > gr.limit) {
		gr.closeReader()
		gr.reader = nil
		gr.cancel = nil
		if gr.start != offset {
			// We should only increase the seek count if we have to discard the reader when it's
			// positioned at wrong place. Discarding it if can't serve the entire request would
			// result in reader size not growing for random reads scenario.
			gr.seeks++
		}
	}
	return objectData, err
}

func (gr *gcsReader) Destroy() {
	// Close out the reader, if we have one.
	if gr.reader != nil {
		gr.closeReader()
		gr.reader = nil
		gr.cancel = nil
	}
}

// closeReader fetches the readHandle before closing the reader instance.
func (gr *gcsReader) closeReader() {
	gr.readHandle = gr.reader.ReadHandle()
	err := gr.reader.Close()
	if err != nil {
		logger.Warnf("error while closing reader: %v", err)
	}
}
