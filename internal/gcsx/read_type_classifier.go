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
}

func NewReadTypeClassifier(sequentialReadSizeMb int64) *ReadTypeClassifier {
	state := &ReadTypeClassifier{
		readType:             atomic.Int64{},
		expectedOffset:       atomic.Int64{},
		seeks:                atomic.Uint64{},
		totalReadBytes:       atomic.Uint64{},
		sequentialReadSizeMb: sequentialReadSizeMb,
	}

	// Start as sequential read type, keep the existing GCSFuse read behavior.
	state.readType.Store(metrics.ReadTypeSequential)
	return state
}

// RecordSeek should be called before any read operation.
// It records a seek operation at the given offset.
func (rpt *ReadTypeClassifier) RecordSeek(offset int64) {
	rpt.GetReadInfo(offset, false)
}

// RecordRead records a read operation of the given size at the given offset.
// Call it after the read is done.
func (rpt *ReadTypeClassifier) RecordRead(offset int64, sizeRead int64) {
	rpt.totalReadBytes.Add(uint64(sizeRead))
	rpt.expectedOffset.Store(offset + sizeRead)
}

// isSeekNeeded determines if the current read at `offset` should be considered a
// seek, given the previous read pattern & the expected offset.
func (rtc *ReadTypeClassifier) isSeekNeeded(offset int64) bool {
	if rtc.expectedOffset.Load() == 0 {
		return false
	}

	// Read from unexpected offset in random read is considered a seek.
	if rtc.readType.Load() == metrics.ReadTypeRandom {
		return rtc.expectedOffset.Load() != offset
	}

	// In sequential read, read backward or too far (> maxReadSize) forward is considered a seek.
	// This allows for some level of kernel readahead in sequential reads.
	if rtc.readType.Load() == metrics.ReadTypeSequential {
		return offset < rtc.expectedOffset.Load() || offset > rtc.expectedOffset.Load()+maxReadSize
	}

	return false
}

// GetReadInfo determines the read strategy (sequential or random) for a read
// request at a given offset and returns read metadata. It also updates the
// reader's internal state based on the read pattern.
// seekRecorded parameter describes whether a seek has already been recorded for this request.
func (rtc *ReadTypeClassifier) GetReadInfo(offset int64, seekRecorded bool) ReadInfo {
	prreadType := rtc.readType.Load()
	readType := prreadType

	expOffset := rtc.expectedOffset.Load()
	numSeeks := rtc.seeks.Load()

	if !seekRecorded && rtc.isSeekNeeded(offset) {
		numSeeks = rtc.seeks.Add(1)
		logger.Infof("Seek detected. Total seeks: %d", numSeeks)
		seekRecorded = true
	}

	if numSeeks >= minSeeksForRandom {
		readType = metrics.ReadTypeRandom
	}
	averageReadBytes := avgReadBytes(rtc.totalReadBytes.Load(), numSeeks)

	if averageReadBytes >= maxReadSize {
		readType = metrics.ReadTypeSequential
	}

	if readType != prreadType {
		rtc.readType.Store(readType)
		logger.Tracef("Read pattern changed to %d. Average read size: %d bytes over %d seeks", readType, averageReadBytes, numSeeks)
	}
	return ReadInfo{
		ReadType:       readType,
		ExpectedOffset: expOffset,
		SeekRecorded:   seekRecorded,
	}
}

// SeqReadIO returns the read size to be used for the next read operation based on
// the current read pattern. It also updates the readType if needed.
// If the read pattern is classified as random, it calculates an appropriate
// read size based on the average read size per seek, bounded by min and max read sizes.
// If the read pattern is sequential, it returns the configured sequential read size.
func (rpt *ReadTypeClassifier) SeqReadIO() int64 {
	if seeks := rpt.seeks.Load(); seeks >= minSeeksForRandom {
		averageReadBytes := avgReadBytes(rpt.totalReadBytes.Load(), seeks)
		if averageReadBytes < maxReadSize {
			randomReadSize := int64(((averageReadBytes / MB) + 1) * MB)
			logger.Infof("Random read detected. Average read size: %d bytes over %d seeks. Using read size: %d bytes", averageReadBytes, seeks, randomReadSize)
			if randomReadSize < minReadSize {
				randomReadSize = minReadSize
			}
			if randomReadSize > maxReadSize {
				randomReadSize = maxReadSize
			}
			logger.Infof("Random read detected. Average read size: %d bytes over %d seeks. Using read size: %d bytes", averageReadBytes, seeks, randomReadSize)
			rpt.readType.Store(metrics.ReadTypeRandom)
			return randomReadSize
		}
	}
	rpt.readType.Store(metrics.ReadTypeSequential)
	return rpt.sequentialReadSizeMb * MB
}

// IsReadSequential returns true if the current read pattern is sequential
func (rpt *ReadTypeClassifier) IsReadSequential() bool {
	return rpt.readType.Load() == metrics.ReadTypeSequential
}

// avgReadBytes calculates the average read bytes per seek.
// If no seeks have been recorded, it returns the total read bytes.
func avgReadBytes(totalReadBytes uint64, numSeeks uint64) uint64 {
	if numSeeks > 0 {
		return totalReadBytes / numSeeks
	}
	return totalReadBytes
}
