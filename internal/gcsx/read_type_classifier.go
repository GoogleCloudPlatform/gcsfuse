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

	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// ReaderType enum values.
const (
	MB = 1 << 20
)

// ReadInfo Stores information for this read request.
type ReadInfo struct {
	// ReadType stores the read type evaluated for this request.
	ReadType int64

	// ExpectedOffset stores the expected offset for this request. Will be
	// used to determine if re-evaluation of ReadType is required or not with range reader.
	ExpectedOffset int64

	// SeekRecorded tells whether a seek has been performed for this read request.
	SeekRecorded bool
}

// ReadTypeClassifier tracks the read access pattern (sequential vs random) across multiple readers.
// It uses heuristics based on the number of seeks and average read size to classify the read pattern.
// It is safe for concurrent use by multiple goroutines.
type ReadTypeClassifier struct {
	// ReadType of the reader. Will be sequential by default.
	readType atomic.Int64

	// Specifies the next expected offset for the reads. Used to distinguish between
	// sequential and random reads.
	expectedOffset atomic.Int64

	// seeks represents the number of random reads performed by the reader.
	seeks atomic.Uint64

	// totalReadBytes is the total number of bytes read by the reader.
	totalReadBytes atomic.Uint64

	// sequentialReadSizeMb is the configured sequential read size in MB.
	sequentialReadSizeMb int64

	// initialOffset stores the first read offset, helps in determining the initial prefetch size.
	initialOffset int64
}

func NewReadTypeClassifier(sequentialReadSizeMb int64, initialOffset int64) *ReadTypeClassifier {
	state := &ReadTypeClassifier{
		readType:             atomic.Int64{},
		expectedOffset:       atomic.Int64{},
		seeks:                atomic.Uint64{},
		totalReadBytes:       atomic.Uint64{},
		sequentialReadSizeMb: sequentialReadSizeMb,
		initialOffset:        initialOffset,
	}

	// Start as sequential read type, keep the existing GCSFuse read behavior.
	state.readType.Store(metrics.ReadTypeSequential)
	return state
}

// RecordSeek checks if the read at the given offset is a seek and updates the internal state accordingly.
// Call it before starting the read operation.
func (rtc *ReadTypeClassifier) RecordSeek(offset int64) {
	rtc.GetReadInfo(offset, false)
}

// RecordRead records a read operation of the given size at the given offset.
// This must be called after the read operation.
func (rtc *ReadTypeClassifier) RecordRead(offset int64, sizeRead int64) {
	rtc.totalReadBytes.Add(uint64(sizeRead))
	rtc.expectedOffset.Store(offset + sizeRead)
}

// isSeekNeeded determines if the current read at `offset` should be considered a
// seek, given the previous read pattern & the expected offset.
func (rtc *ReadTypeClassifier) isSeekNeeded(offset int64) bool {
	expectedOffset := rtc.expectedOffset.Load()
	readType := rtc.readType.Load()

	if expectedOffset == 0 {
		return false
	}

	// Read from unexpected offset in random read is considered a seek.
	if readType == metrics.ReadTypeRandom {
		return expectedOffset != offset
	}

	// In sequential read, read backward or too far (> maxReadSize) forward is considered a seek.
	// This allows for some level of kernel readahead in sequential reads.
	if readType == metrics.ReadTypeSequential {
		return offset < expectedOffset || offset > expectedOffset+maxReadSize
	}

	return false
}

// GetReadInfo determines the read strategy (sequential or random) for a read
// request at a given offset and returns read metadata. It also updates the
// internal state `readType` based on the read pattern.
// seekRecorded parameter describes whether a seek has already been recorded for this request.
func (rtc *ReadTypeClassifier) GetReadInfo(offset int64, seekRecorded bool) ReadInfo {
	previousReadType := rtc.readType.Load()
	expOffset := rtc.expectedOffset.Load()
	numSeeks := rtc.seeks.Load()
	currentTotalReadBytes := rtc.totalReadBytes.Load()

	if !seekRecorded && rtc.isSeekNeeded(offset) {
		numSeeks = rtc.seeks.Add(1)
		seekRecorded = true
	}

	readType := metrics.ReadTypeRandom
	averageReadBytes := avgReadBytes(currentTotalReadBytes, numSeeks)

	// Classify as Sequential if:
	// 1. The average read size is large enough.
	// 2. OR we haven't performed any seeks yet AND the first read was at offset 0.
	if averageReadBytes >= maxReadSize || (numSeeks == 0 && rtc.initialOffset == 0) {
		readType = metrics.ReadTypeSequential
	}

	if readType != previousReadType {
		rtc.readType.Store(readType)
	}

	return ReadInfo{
		ReadType:       readType,
		ExpectedOffset: expOffset,
		SeekRecorded:   seekRecorded,
	}
}

// ComputeSeqPrefetchWindowAndAdjustType computes the sequential IO size heuristically based on
// the current read pattern. It also updates the readType if needed.
// If the read pattern is classified as random, it calculates an appropriate
// read size based on the average read size per seek, bounded by min and max read sizes.
// If the read pattern is sequential, it returns the configured sequential read size.
// Note: The returned prefetch window size is not limited by the object size, caller should
// handle that separately.
func (rtc *ReadTypeClassifier) ComputeSeqPrefetchWindowAndAdjustType() int64 {
	currentReadType := rtc.readType.Load()
	seeks := rtc.seeks.Load()

	// Evaluate for Random read type if seeks > 0 or the first read was non-zero.
	// A non-zero initial offset implies random access even with zero seeks, so we check average read bytes.
	if seeks > 0 || rtc.initialOffset > 0 {
		averageReadBytes := avgReadBytes(rtc.totalReadBytes.Load(), seeks)

		if averageReadBytes < maxReadSize {
			randomReadSize := ((averageReadBytes + MB - 1) / MB) * MB
			// Clamp to [minReadSize, maxReadSize]
			randomReadSize = min(max(randomReadSize, minReadSize), maxReadSize)
			if currentReadType != metrics.ReadTypeRandom {
				rtc.readType.Store(metrics.ReadTypeRandom)
			}
			return int64(randomReadSize)
		}
	}
	if currentReadType != metrics.ReadTypeSequential {
		rtc.readType.Store(metrics.ReadTypeSequential)
	}
	return rtc.sequentialReadSizeMb * MB
}

// IsReadSequential returns true if the current read pattern is sequential
func (rtc *ReadTypeClassifier) IsReadSequential() bool {
	return rtc.readType.Load() == metrics.ReadTypeSequential
}

// NextExpectedOffset returns the next expected offset for a sequential read.
func (rtc *ReadTypeClassifier) NextExpectedOffset() int64 {
	return rtc.expectedOffset.Load()
}

// avgReadBytes calculates the average read bytes per seek.
// If no seeks have been recorded, it returns the total read bytes.
func avgReadBytes(totalReadBytes uint64, numSeeks uint64) uint64 {
	if numSeeks > 0 {
		return totalReadBytes / numSeeks
	}
	return totalReadBytes
}

// GetSeeks returns the current number of seeks recorded.
// This method is intended for testing purposes.
func (rtc *ReadTypeClassifier) GetSeeks() uint64 {
	return rtc.seeks.Load()
}
