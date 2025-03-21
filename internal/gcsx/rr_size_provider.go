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
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

// randomReaderReadSizeProvider implements the ReadSizeProvider interface for randomReader.
type randomReaderReadSizeProvider struct {
	seek               int64
	totalReadBytes     int64
	readType           string
	objectSize         int64
	sequentialReadSize int32
}

// NewRandomReaderReadSizeProvider creates a new ReadSizeProvider for the given randomReader.
func NewRandomReaderReadSizeProvider(objectSize int64) ReadSizeProvider {
	return &randomReaderReadSizeProvider{
		seek:               0,
		totalReadBytes:     0,
		readType:           util.Sequential,
		objectSize:         objectSize,
		sequentialReadSize: 200 * MB,
	}
}

// GetNextReadSize returns the size of the next read request, given the current offset.
func (rrs *randomReaderReadSizeProvider) GetNextReadSize(offset int64) (size int64, err error) {
	// Make sure start is legal.
	if offset < 0 || uint64(offset) >= uint64(rrs.objectSize) {
		err = fmt.Errorf(
			"offset %d is illegal for %d-byte object",
			offset,
			rrs.objectSize)
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
	end := int64(rrs.objectSize)
	if rrs.seek >= minSeeksForRandom {
		rrs.readType = util.Random
		averageReadBytes := rrs.totalReadBytes / rrs.seek
		if averageReadBytes < maxReadSize {
			randomReadSize := int64(((averageReadBytes / MB) + 1) * MB)
			if randomReadSize < minReadSize {
				randomReadSize = minReadSize
			}
			if randomReadSize > maxReadSize {
				randomReadSize = maxReadSize
			}
			end = offset + randomReadSize
		}
	}
	if end > int64(rrs.objectSize) {
		end = int64(rrs.objectSize)
	}

	// To avoid overloading GCS and to have reasonable latencies, we will only
	// fetch data of max size defined by sequentialReadSizeMb.
	if end-offset > int64(rrs.sequentialReadSize) {
		end = offset + int64(rrs.sequentialReadSize)
	}

	size = end - offset
	return
}

func (rrs *randomReaderReadSizeProvider) ReadType() string {
	return rrs.readType
}

func (rrs *randomReaderReadSizeProvider) ProvideFeedback(f *Feedback) {
	rrs.totalReadBytes = f.TotalReadBytes
	if !f.ReadCompletely {
		rrs.seek++
	}
}
