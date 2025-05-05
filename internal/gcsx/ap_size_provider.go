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
	"fmt"
	"math"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

const (
	gSequentialReadThreshold = 1 * MB
	gMaxSequentialReadSize   = 1000 * MB
)

// adaptiveReadSizeProvider implements the ReadSizeProvider interface, adapting
// read sizes based on observed access patterns (sequential vs. random).
// For sequential, starts with 1MiB and progressively increases it (multiplied by sequentialMultiplier) until it reaches gMaxSequentialReadSize.
// For random, resets the read-size back to 1 MiB.
type adaptiveReadSizeProvider struct {
	totalBytesRead       int64
	readType             string
	objectSize           int64
	sequentialMultiplier int32
	lastOffset           int64
	lastReadSize         int64 // Size of the last read request.
	seekCount            int64 // Number of non-sequential seeks.
}

func NewAdaptiveReadSizeProvider(objectSize int64, sequentialMultiplier int32) ReadSizeProvider {
	return &adaptiveReadSizeProvider{
		totalBytesRead:       0,
		readType:             util.Random,
		objectSize:           objectSize,
		sequentialMultiplier: sequentialMultiplier,
		lastReadSize:         MB, // Initialize to a large value.
		lastOffset:           math.MaxInt64,
	}
}

// GetNextReadSize returns the size of the next read request, given the current offset.
func (apsp *adaptiveReadSizeProvider) GetNextReadSize(offset int64) (size int64, err error) {
	// Make sure start is legal.
	if offset < 0 || uint64(offset) > uint64(apsp.objectSize) {
		err = fmt.Errorf(
			"offset %d is illegal for %d-byte object",
			offset,
			apsp.objectSize)
		return
	}

	if !apsp.isSequential(offset) {
		apsp.readType = util.Random
		apsp.lastReadSize = 1 * MB
	} else {
		apsp.readType = util.Sequential
		requestSize := apsp.lastReadSize * int64(apsp.sequentialMultiplier)
		if requestSize > gMaxSequentialReadSize {
			requestSize = gMaxSequentialReadSize
		}
		apsp.lastReadSize = requestSize
	}

	if apsp.lastReadSize > apsp.objectSize-offset {
		apsp.lastReadSize = apsp.objectSize - offset
	}
	return apsp.lastReadSize, nil
}

func (apsp *adaptiveReadSizeProvider) GetReadType() string {
	return apsp.readType
}

func (apsp *adaptiveReadSizeProvider) ProvideFeedback(f *ReadFeedback) {
	apsp.totalBytesRead = f.TotalBytesRead
	if !f.ReadComplete {
		apsp.seekCount++
	}
	apsp.lastOffset = f.LastOffset
}

func (apsp *adaptiveReadSizeProvider) isSequential(offset int64) bool {
	offsetMargin := offset - apsp.lastOffset
	if offsetMargin < 0 || offsetMargin > gSequentialReadThreshold { // random read
		return false
	}
	return true
}
