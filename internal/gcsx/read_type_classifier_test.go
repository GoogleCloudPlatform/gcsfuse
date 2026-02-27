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
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
)

func TestReadTypeClassifier_InitialState(t *testing.T) {
	readTypeClassifier := NewReadTypeClassifier(sequentialReadSizeInMb, 0)

	assert.Equal(t, metrics.ReadTypeSequential, readTypeClassifier.readType.Load())
	assert.Equal(t, int64(0), readTypeClassifier.expectedOffset.Load())
	assert.Equal(t, uint64(0), readTypeClassifier.seeks.Load())
	assert.Equal(t, uint64(0), readTypeClassifier.totalReadBytes.Load())
	assert.Equal(t, int64(0), readTypeClassifier.initialOffset)
}

func TestReadTypeClassifier_IsSeekNeeded(t *testing.T) {
	testCases := []struct {
		name           string
		readType       int64
		offset         int64
		expectedOffset int64
		want           bool
	}{
		{
			name:           "First read, expectedOffset is 0",
			readType:       metrics.ReadTypeSequential,
			offset:         100,
			expectedOffset: 0,
			want:           false,
		},
		{
			name:           "Random read, same offset",
			readType:       metrics.ReadTypeRandom,
			offset:         100,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Random read, different offset",
			readType:       metrics.ReadTypeRandom,
			offset:         200,
			expectedOffset: 100,
			want:           true,
		},
		{
			name:           "Sequential read, same offset",
			readType:       metrics.ReadTypeSequential,
			offset:         100,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Sequential read, small forward jump within maxReadSize",
			readType:       metrics.ReadTypeSequential,
			offset:         100 + maxReadSize/2,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Sequential read, forward jump to boundary of maxReadSize",
			readType:       metrics.ReadTypeSequential,
			offset:         100 + maxReadSize,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Sequential read, large forward jump beyond maxReadSize",
			readType:       metrics.ReadTypeSequential,
			offset:         100 + maxReadSize + 1,
			expectedOffset: 100,
			want:           true,
		},
		{
			name:           "Sequential read, backward jump",
			readType:       metrics.ReadTypeSequential,
			offset:         99,
			expectedOffset: 100,
			want:           true,
		},
		{
			name:           "Unknown read type",
			readType:       -1, // An invalid read type
			offset:         200,
			expectedOffset: 100,
			want:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			classifier := NewReadTypeClassifier(sequentialReadSizeInMb, 0)
			classifier.readType.Store(tc.readType)
			classifier.expectedOffset.Store(tc.expectedOffset)

			got := classifier.isSeekNeeded(tc.offset)

			assert.Equal(t, tc.want, got)
		})
	}
}

// This test also covers RecordSeek functionality.
func TestReadTypeClassifier_GetReadInfo(t *testing.T) {
	testCases := []struct {
		name                  string
		offset                int64
		seekRecorded          bool
		initialReadType       int64
		initialExpOffset      int64
		initialNumSeeks       uint64
		initialTotalReadBytes uint64
		initialOffset         int64
		expectedReadType      int64
		expectedNumSeeks      uint64
	}{
		{
			name:                  "First Read",
			offset:                0,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      0,
			initialNumSeeks:       0,
			initialTotalReadBytes: 0,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0,
		},
		{
			name:                  "First Read at non-zero offset",
			offset:                100,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      0,
			initialNumSeeks:       0,
			initialTotalReadBytes: 0,
			initialOffset:         100,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      0,
		},
		{
			name:                  "Sequential Read",
			offset:                10,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      10,
			initialNumSeeks:       0,
			initialTotalReadBytes: 100,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0,
		},
		{
			name:                  "Sequential read with small forward jump and high average read bytes is still sequential",
			offset:                100,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      10,
			initialNumSeeks:       0,
			initialTotalReadBytes: 10000000,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0,
		},
		{
			name:                  "Sequential read with large forward jump is a seek and switches to random",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       0,
			initialTotalReadBytes: 50 * 1024,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      1,
		},
		{
			name:                  "Sequential read with backward jump is a seek and switches to random",
			offset:                49,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       0,
			initialTotalReadBytes: 50 * 1024,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      1,
		},
		{
			name:                  "Contiguous random read is not a seek",
			offset:                50,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       1,
			initialTotalReadBytes: 50 * 1024,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      1,
		},
		{
			name:                  "Non-contiguous random read is a seek",
			offset:                100,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       1,
			initialTotalReadBytes: 50 * 1024,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      2,
		},
		{
			name:                  "Switches to random read on seek",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       0,
			initialTotalReadBytes: 1000,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      1,
		},
		{
			name:                  "Switches back to sequential with high average read bytes",
			offset:                100,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       1,
			initialTotalReadBytes: maxReadSize * 2,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      2,
		},
		{
			name:                  "Seek recorded: sequential large forward jump",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       0,
			initialTotalReadBytes: 50 * 1024,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0, // Not incremented
		},
		{
			name:                  "Seek recorded: sequential backward jump switches to random",
			offset:                49,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       1,
			initialTotalReadBytes: 50 * 1024,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      1, // Not incremented
		},
		{
			name:                  "Seek recorded: non-contiguous random read",
			offset:                100,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       1,
			initialTotalReadBytes: 50 * 1024,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      1, // Not incremented
		},
		{
			name:                  "Seek recorded: switches to random",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       1,
			initialTotalReadBytes: 1000,
			initialOffset:         0,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      1, // Not incremented
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			readTypeClassifier := NewReadTypeClassifier(sequentialReadSizeInMb, 0)
			readTypeClassifier.readType.Store(tc.initialReadType)
			readTypeClassifier.expectedOffset.Store(tc.initialExpOffset)
			readTypeClassifier.seeks.Store(tc.initialNumSeeks)
			readTypeClassifier.totalReadBytes.Store(tc.initialTotalReadBytes)
			readTypeClassifier.initialOffset = tc.initialOffset

			readInfo := readTypeClassifier.GetReadInfo(tc.offset, tc.seekRecorded)

			assert.Equal(t, tc.expectedReadType, readInfo.ReadType, "Read type mismatch")
			assert.Equal(t, tc.expectedNumSeeks, readTypeClassifier.seeks.Load(), "Number of seeks mismatch")
		})
	}
}

func TestReadTypeClassifier_RecordRead(t *testing.T) {
	testCases := []struct {
		name                  string
		initialExpectedOffset int64
		initialTotalReadBytes uint64
		offset                int64
		sizeRead              int64
		expectedOffset        int64
		expectedTotalBytes    uint64
	}{
		{
			name:                  "First read",
			initialExpectedOffset: 0,
			initialTotalReadBytes: 0,
			offset:                0,
			sizeRead:              10 * MB,
			expectedOffset:        10 * MB,
			expectedTotalBytes:    10 * MB,
		},
		{
			name:                  "Subsequent read",
			initialExpectedOffset: 10 * MB,
			initialTotalReadBytes: 10 * MB,
			offset:                10 * MB,
			sizeRead:              5 * MB,
			expectedOffset:        15 * MB,
			expectedTotalBytes:    15 * MB,
		},
		{
			name:                  "Any random read",
			initialExpectedOffset: 15 * MB,
			initialTotalReadBytes: 15 * MB,
			offset:                15 * MB,
			sizeRead:              20 * MB,
			expectedOffset:        35 * MB,
			expectedTotalBytes:    35 * MB,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			classifier := NewReadTypeClassifier(sequentialReadSizeInMb, 0)
			classifier.expectedOffset.Store(tc.initialExpectedOffset)
			classifier.totalReadBytes.Store(tc.initialTotalReadBytes)

			classifier.RecordRead(tc.offset, tc.sizeRead)

			assert.Equal(t, tc.expectedOffset, classifier.expectedOffset.Load(), "Expected offset mismatch")
			assert.Equal(t, tc.expectedTotalBytes, classifier.totalReadBytes.Load(), "Total read bytes mismatch")
		})
	}
}

func TestReadTypeClassifier_ComputeSeqPrefetchWindowAndAdjustType(t *testing.T) {
	testCases := []struct {
		name                      string
		initialNumSeeks           uint64
		initialTotalReadBytes     uint64
		initialOffset             int64
		sequentialReadSizeMb      int64
		expectedSeqPrefetchWindow int64
	}{
		{
			name:                      "Sequential Read, No seek",
			initialNumSeeks:           0,
			initialTotalReadBytes:     0,
			initialOffset:             0,
			sequentialReadSizeMb:      22,
			expectedSeqPrefetchWindow: 22 * MB,
		},
		{
			name:                      "Sequential Read, 1 seek but high average read size",
			initialNumSeeks:           1,
			initialTotalReadBytes:     100 * MB,
			initialOffset:             0,
			sequentialReadSizeMb:      22,
			expectedSeqPrefetchWindow: 22 * MB,
		},
		{
			name:                      "Sequential Read, multiple seeks but low average read size",
			initialNumSeeks:           2,
			initialTotalReadBytes:     10 * MB,
			initialOffset:             0,
			sequentialReadSizeMb:      22,
			expectedSeqPrefetchWindow: 5 * MB, // Avg is 5MB.
		},
		{
			name:                      "Random Read, multiple seeks and low average read size",
			initialNumSeeks:           2,
			initialTotalReadBytes:     5 * MB,
			initialOffset:             0,
			sequentialReadSizeMb:      22,
			expectedSeqPrefetchWindow: 3 * MB, // Avg is 2.5MB, rounded up to 3MB.
		},
		{
			name:                      "Random Read, multiple seeks and very low average read size",
			initialNumSeeks:           2,
			initialTotalReadBytes:     500 * 1024, // 500KB
			initialOffset:             0,
			sequentialReadSizeMb:      22,
			expectedSeqPrefetchWindow: minReadSize,
		},
		{
			name:                      "Random Read, multiple seeks and moderate average read size",
			initialNumSeeks:           2,
			initialTotalReadBytes:     3 * MB,
			initialOffset:             0,
			sequentialReadSizeMb:      22,
			expectedSeqPrefetchWindow: 2 * MB, // Avg is 1.5MB, rounded up to 2MB.
		},
		{
			name:                      "Random Read, multiple seeks and high average read size",
			initialNumSeeks:           2,
			initialTotalReadBytes:     100 * MB,
			initialOffset:             0,
			sequentialReadSizeMb:      22,
			expectedSeqPrefetchWindow: 22 * MB, // Avg is ~33MB, more than maxReadSize so capped to 22MB.
		},
		{
			name:                      "Sequential read, Different sequential read size configured",
			initialNumSeeks:           0,
			initialTotalReadBytes:     0,
			initialOffset:             0,
			sequentialReadSizeMb:      10,
			expectedSeqPrefetchWindow: 10 * MB,
		},
		{
			name:                      "Random Read, multiple seeks and high average read size, 10MB sequential read size",
			initialNumSeeks:           2,
			initialTotalReadBytes:     100 * MB,
			initialOffset:             0,
			sequentialReadSizeMb:      10,
			expectedSeqPrefetchWindow: 10 * MB,
		},
		{
			name:                      "Random Read, 1 seek and low average read size",
			initialNumSeeks:           1,
			initialTotalReadBytes:     1 * MB,
			initialOffset:             0,
			sequentialReadSizeMb:      22,
			expectedSeqPrefetchWindow: 1 * MB,
		},
		{
			name:                      "First read non-zero offset (Random type set, seeks 0)",
			initialNumSeeks:           0,
			initialTotalReadBytes:     0,
			initialOffset:             100,
			sequentialReadSizeMb:      22,
			expectedSeqPrefetchWindow: minReadSize,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			classifier := NewReadTypeClassifier(tc.sequentialReadSizeMb, 0)
			classifier.seeks.Store(tc.initialNumSeeks)
			classifier.totalReadBytes.Store(tc.initialTotalReadBytes)
			classifier.initialOffset = tc.initialOffset

			seqReadIO := classifier.ComputeSeqPrefetchWindowAndAdjustType()

			assert.Equal(t, tc.expectedSeqPrefetchWindow, seqReadIO, "SeqIO size mismatch")
		})
	}
}

func TestReadTypeClassifier_IsSequentialRead(t *testing.T) {
	testCases := []struct {
		name           string
		readType       int64
		SequentialRead bool
	}{
		{
			name:           "ReadTypeSequential",
			readType:       metrics.ReadTypeSequential,
			SequentialRead: true,
		},
		{
			name:           "ReadTypeRandom",
			readType:       metrics.ReadTypeRandom,
			SequentialRead: false,
		},
		{
			name:           "ReadTypeUnknown",
			readType:       -1,
			SequentialRead: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			classifier := NewReadTypeClassifier(sequentialReadSizeInMb, 0)
			classifier.readType.Store(tc.readType)

			assert.Equal(t, tc.SequentialRead, classifier.IsReadSequential())
		})
	}
}

func Test_avgReadBytes(t *testing.T) {
	testCases := []struct {
		name                 string
		totalReadBytes       uint64
		numSeeks             uint64
		expectedAvgReadBytes uint64
	}{
		{
			name:                 "No seeks",
			totalReadBytes:       100 * MB,
			numSeeks:             0,
			expectedAvgReadBytes: 100 * MB,
		},
		{
			name:                 "One seek",
			totalReadBytes:       100 * MB,
			numSeeks:             1,
			expectedAvgReadBytes: 100 * MB,
		},
		{
			name:                 "Multiple seeks",
			totalReadBytes:       300 * MB,
			numSeeks:             3,
			expectedAvgReadBytes: 100 * MB,
		},
		{
			name:                 "Multiple seeks with remainder",
			totalReadBytes:       350 * MB,
			numSeeks:             3,
			expectedAvgReadBytes: 122333866, // Integer division
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			avg := avgReadBytes(tc.totalReadBytes, tc.numSeeks)

			assert.Equal(t, tc.expectedAvgReadBytes, avg)
		})
	}
}

func TestReadTypeClassifier_SequentialReads(t *testing.T) {
	readTypeClassifier := NewReadTypeClassifier(sequentialReadSizeInMb, 0)

	// Simulate 4 reads of 10MB IO.
	readSizes := []int64{10 * MB, 10 * MB, 10 * MB, 10 * MB}
	var offset int64 = 0
	for _, size := range readSizes {
		readTypeClassifier.RecordSeek(offset)
		readTypeClassifier.RecordRead(offset, size)
		offset += size
		assert.Equal(t, metrics.ReadTypeSequential, readTypeClassifier.readType.Load())
		assert.Equal(t, offset, readTypeClassifier.expectedOffset.Load())
	}

	assert.Equal(t, metrics.ReadTypeSequential, readTypeClassifier.readType.Load())
	assert.Equal(t, uint64(0), readTypeClassifier.seeks.Load())
	assert.Equal(t, uint64(40*MB), readTypeClassifier.totalReadBytes.Load())
}

func TestReadTypeClassifier_RandomReads(t *testing.T) {
	classifier := NewReadTypeClassifier(sequentialReadSizeInMb, 0)

	// Simulate random reads of 5MB each at different offsets.
	readSizes := []int64{5 * MB, 5 * MB, 5 * MB, 5 * MB}
	offsets := []int64{0, 20 * MB, 10 * MB, 30 * MB}
	for i, size := range readSizes {
		classifier.RecordSeek(offsets[i])
		classifier.RecordRead(offsets[i], size)
	}

	assert.Equal(t, metrics.ReadTypeRandom, classifier.readType.Load(), "Read type mismatch")
	assert.Equal(t, uint64(3), classifier.seeks.Load(), "Seek mismatch")
	assert.Equal(t, uint64(20*MB), classifier.totalReadBytes.Load(), "Total read bytes mismatch")
}

func TestReadTypeClassifier_RandomToSequentialRead(t *testing.T) {
	classifier := NewReadTypeClassifier(sequentialReadSizeInMb, 0)
	// Start with random reads.
	randomReadSizes := []int64{2 * MB, 2 * MB, 2 * MB, 2 * MB, 2 * MB}
	randomOffsets := []int64{50 * MB, 20 * MB, 10 * MB, 30 * MB, 40 * MB}
	for i, size := range randomReadSizes {
		classifier.RecordSeek(randomOffsets[i])
		classifier.RecordRead(randomOffsets[i], size)
	}
	assert.Equal(t, uint64(4), classifier.seeks.Load(), "Seek mismatch")
	assert.Equal(t, uint64(10*MB), classifier.totalReadBytes.Load(), "Total read bytes mismatch")

	// Now do large sequential reads from different seek.
	seqReadSizes := []int64{20 * MB, 20 * MB, 20 * MB}
	var offset int64 = 100 * MB
	for _, size := range seqReadSizes {
		classifier.RecordSeek(offset)
		classifier.RecordRead(offset, size)
		offset += size
	}

	assert.Equal(t, metrics.ReadTypeSequential, classifier.readType.Load(), "Read type should switch to sequential")
	assert.Equal(t, uint64(5), classifier.seeks.Load(), "Seek count should remain the same")
	assert.Equal(t, uint64(70*MB), classifier.totalReadBytes.Load(), "Total read bytes mismatch")
}

func TestReadTypeClassifier_ConcurrentUpdates(t *testing.T) {
	classifier := NewReadTypeClassifier(sequentialReadSizeInMb, 0)
	var wg sync.WaitGroup
	numGoroutines := 10
	readsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			offset := int64(id * 100 * MB)
			for j := 0; j < readsPerGoroutine; j++ {
				size := int64(5 * MB)
				classifier.RecordSeek(offset)
				classifier.RecordRead(offset, size)
				offset += size
			}
		}(i)
	}

	wg.Wait()

	// After all concurrent updates, check that internal state is consistent.
	totalReads := int64(numGoroutines * readsPerGoroutine * 5 * MB)
	assert.Equal(t, uint64(totalReads), classifier.totalReadBytes.Load())
	// Read type could be either sequential or random depending on timing, so just check it's valid.
	readType := classifier.readType.Load()
	assert.True(t, readType == metrics.ReadTypeSequential || readType == metrics.ReadTypeRandom)
}
