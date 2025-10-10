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
	"sync/atomic"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// ReaderType enum values.
const (
	MB = 1 << 20
)

// readInfo Stores information for this read request.
type ReadInfo struct {
	// readType stores the read type evaluated for this request.
	ReadType int64
	// expectedOffset stores the expected offset for this request. Will be
	// used to determine if re-evaluation of readType is required or not with range reader.
	ExpectedOffset int64
	// seekRecorded tells whether a seek has been performed for this read request.
	SeekRecorded bool
}

// ReadPatternTracker tracks the read pattern (sequential vs random) across multiple readers
// It is safe for concurrent use by multiple goroutines.
type ReadPatternTracker struct {
	// ReadType of the reader. Will be sequential by default.
	readType atomic.Int64

	// Specifies the next expected offset for the reads. Used to distinguish between
	// sequential and random reads.
	expectedOffset atomic.Int64

	// seeks represents the number of random reads performed by the reader.
	seeks atomic.Uint64

	// totalReadBytes is the total number of bytes read by the reader.
	totalReadBytes atomic.Uint64

	sequentialReadSizeMb int64
}

// NewReadPatternTracker creates a new ReadPatternTracker with default configuration
func NewReadPatternTracker(sequentialReadSizeMb int64) *ReadPatternTracker {
	state := &ReadPatternTracker{
		readType:       atomic.Int64{},
		expectedOffset: atomic.Int64{},
		seeks:         atomic.Uint64{},
		totalReadBytes: atomic.Uint64{},
		sequentialReadSizeMb: sequentialReadSizeMb,
	}

	state.seeks.Store(1) // Start with 1 seek to avoid division by zero in average read size calculation.
	state.expectedOffset.Store(0) // Start with 0 expected offset.
	state.readType.Store(metrics.ReadTypeSequential) // Start with sequential read type.
	return state
}

// RecordSeek records a seek operation at the given offset.
func (rpt *ReadPatternTracker) RecordSeek(offset int64, size int64) {
	rpt.GetReadInfo(offset, false)
}

// RecordRead records a read operation of the given size at the given offset.
// Call it after the read is done.
func (rpt *ReadPatternTracker) RecordRead(offset int64, sizeRead int64) {
	rpt.totalReadBytes.Add(uint64(sizeRead))
	rpt.expectedOffset.Store(offset + sizeRead)
}

// isSeekNeeded determines if the current read at `offset` should be considered a
// seek, given the previous read pattern & the expected offset.
func (gr *ReadPatternTracker) isSeekNeeded(offset int64) bool {
	if gr.expectedOffset.Load() == 0 {
		return false
	}

	// Read from unexpected offset in random read is considered a seek.
	if gr.readType.Load() == metrics.ReadTypeRandom {
		return gr.expectedOffset.Load() != offset
	}

	// In sequential read, read backward or too far (> maxReadSize) forward is considered a seek.
	// This allows for some level of kernel readahead in sequential reads.
	maxReadSize := gr.sequentialReadSizeMb * MB
	if gr.readType.Load() == metrics.ReadTypeSequential {
		return offset < gr.expectedOffset.Load() || offset > gr.expectedOffset.Load()+maxReadSize
	}

	return false
}	

// getReadInfo determines the read strategy (sequential or random) for a read
// request at a given offset and returns read metadata. It also updates the
// reader's internal state based on the read pattern.
// seekRecorded parameter describes whether a seek has already been recorded for this request.
func (gr *ReadPatternTracker) GetReadInfo(offset int64, seekRecorded bool) ReadInfo {
	prreadType := gr.readType.Load()
	readType := prreadType

	expOffset := gr.expectedOffset.Load()
	numSeeks := gr.seeks.Load()


	if !seekRecorded && gr.isSeekNeeded(offset) {
		numSeeks = gr.seeks.Add(1)
		seekRecorded = true
	}

	if numSeeks >= minSeeksForRandom {
		readType = metrics.ReadTypeRandom
	}
	averageReadBytes := gr.totalReadBytes.Load()
	if numSeeks > 0 {
		averageReadBytes /= numSeeks
	}

	if averageReadBytes >= maxReadSize {
		readType = metrics.ReadTypeSequential
	}

	if readType != prreadType {
		gr.readType.Store(readType)
		logger.Infof("Read pattern changed to %s. Average read size: %d bytes over %d seeks", readType, averageReadBytes, numSeeks)
	}
	return ReadInfo{
		ReadType:       readType,
		ExpectedOffset: expOffset,
		SeekRecorded:   seekRecorded,
	}
}

func (rpt *ReadPatternTracker) SeqReadIO() int64 {
	if seeks := rpt.seeks.Load(); seeks >= minSeeksForRandom {
		averageReadBytes := rpt.totalReadBytes.Load() / seeks
		if averageReadBytes < maxReadSize {
			randomReadSize := int64(((averageReadBytes / MB) + 1) * MB)
			if randomReadSize < minReadSize {
				randomReadSize = minReadSize
			}
			if randomReadSize > maxReadSize {
				randomReadSize = maxReadSize
			}
			rpt.readType.Store(metrics.ReadTypeRandom)
			return randomReadSize
		} else {
			rpt.readType.Store(metrics.ReadTypeSequential)
		}
	}
	return rpt.sequentialReadSizeMb * MB
}

// IsReadSequential returns true if the current read pattern is sequential
func (rpt *ReadPatternTracker) IsReadSequential() bool {
	return rpt.readType.Load() == metrics.ReadTypeSequential
}
