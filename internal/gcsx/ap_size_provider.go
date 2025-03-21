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

const gSequentialReadThreshold = 1 * MB

// apSizeProvider implements the ReadSizeProvider interface for randomReader.
type apSizeProvider struct {
	seek                 int64
	totalReadBytes       int64
	readType             string
	objectSize           int64
	maxSequentialMiB     int64
	sequentialMultiplier int32
	lastOffset           int64
	lastRequestSize      int64
}

// NewAPSizeProvider creates a new ReadSizeProvider for the given randomReader.
func NewAPSizeProvider(objectSize int64, sequentialMulplier int32) ReadSizeProvider {
	return &apSizeProvider{
		seek:                 0,
		totalReadBytes:       0,
		readType:             util.Sequential,
		objectSize:           objectSize,
		maxSequentialMiB:     1000 * MB,
		sequentialMultiplier: sequentialMulplier,
		lastRequestSize:      math.MaxInt64,
	}
}

// GetNextReadSize returns the size of the next read request, given the current offset.
func (rrs *apSizeProvider) GetNextReadSize(offset int64) (size int64, err error) {
	// Make sure start is legal.
	if offset < 0 || uint64(offset) > uint64(rrs.objectSize) {
		err = fmt.Errorf(
			"offset %d is illegal for %d-byte object",
			offset,
			rrs.objectSize)
		return
	}

	if !rrs.isSequential(offset) {
		rrs.readType = util.Random
		rrs.lastRequestSize = 1 * MB
	} else {
		rrs.readType = util.Sequential
		requestSize := rrs.lastRequestSize * int64(rrs.sequentialMultiplier)
		if requestSize > rrs.maxSequentialMiB {
			requestSize = rrs.maxSequentialMiB
		}
		rrs.lastRequestSize = requestSize
	}

	if rrs.lastRequestSize > rrs.objectSize-offset {
		rrs.lastRequestSize = rrs.objectSize - offset
	}
	return rrs.lastRequestSize, nil
}

func (rrs *apSizeProvider) ReadType() string {
	return rrs.readType
}

func (rrs *apSizeProvider) ProvideFeedback(f *Feedback) {
	rrs.totalReadBytes = f.TotalReadBytes
	if !f.ReadCompletely {
		rrs.seek++
	}
	rrs.lastOffset = f.LastOffsetRead
}

func (rrs *apSizeProvider) isSequential(offset int64) bool {
	offsetMargin := offset - rrs.lastOffset
	if offsetMargin < 0 || offsetMargin > gSequentialReadThreshold { // random read
		return false
	}
	return true
}
