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

package bufferedread

import "github.com/googlecloudplatform/gcsfuse/v3/common"

// readPatternDetector encapsulates the logic for detecting random read patterns.
type readPatternDetector struct {
	randomSeekCount      int64
	randomReadsThreshold int64
	blockSize            int64
}

// newReadPatternDetector creates a new detector with a given threshold and block size.
func newReadPatternDetector(threshold, blockSize int64) *readPatternDetector {
	return &readPatternDetector{
		randomReadsThreshold: threshold,
		blockSize:            blockSize,
	}
}

// patternDetectorCheck holds the inputs for a pattern detection check.
type patternDetectorCheck struct {
	Offset int64
	Queue  common.Queue[*blockQueueEntry]
}

// isRandomSeek checks if a read at a given offset constitutes a random seek
// based on the state of the prefetch queue.
func (d *readPatternDetector) isRandomSeek(check *patternDetectorCheck) bool {
	if check.Offset == 0 {
		return false
	}
	if !check.Queue.IsEmpty() {
		start := check.Queue.Peek().block.AbsStartOff()
		end := start + int64(check.Queue.Len())*d.blockSize
		if check.Offset >= start && check.Offset < end {
			return false
		}
	}
	return true
}

// check determines if a read is random, updates the internal seek count, and
// returns whether a fallback to a different reader is recommended.
func (d *readPatternDetector) check(check *patternDetectorCheck) (isRandom, shouldFallback bool) {
	if !d.isRandomSeek(check) {
		return false, false
	}
	d.randomSeekCount++
	return true, d.randomSeekCount > d.randomReadsThreshold
}

// isAboveThreshold returns true if the random seek count has exceeded the configured threshold.
func (d *readPatternDetector) isAboveThreshold() bool {
	return d.randomSeekCount > d.randomReadsThreshold
}

// threshold returns the configured random read threshold.
func (d *readPatternDetector) threshold() int64 {
	return d.randomReadsThreshold
}
