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

package data

import (
	"sort"
	"sync"
)

// ByteRange represents a contiguous range of bytes [Start, End)
type ByteRange struct {
	Start uint64
	End   uint64 // exclusive
}

// ByteRangeMap tracks which byte ranges have been downloaded in a sparse file.
// It maintains a set of non-overlapping, sorted ranges and provides methods to
// add new ranges and query whether specific ranges are downloaded.
type ByteRangeMap struct {
	mu     sync.RWMutex
	ranges []ByteRange // sorted, non-overlapping ranges
}

// NewByteRangeMap creates a new empty ByteRangeMap
func NewByteRangeMap() *ByteRangeMap {
	return &ByteRangeMap{
		ranges: make([]ByteRange, 0),
	}
}

// AddRange adds a new byte range to the map, merging with existing ranges if they overlap or are adjacent.
// Returns the total number of new bytes added (may be less than range size if it overlaps with existing ranges).
func (brm *ByteRangeMap) AddRange(start, end uint64) uint64 {
	brm.mu.Lock()
	defer brm.mu.Unlock()

	if start >= end {
		return 0
	}

	// If no existing ranges, just add it
	if len(brm.ranges) == 0 {
		brm.ranges = append(brm.ranges, ByteRange{Start: start, End: end})
		return end - start
	}

	// Find the first range that might overlap or be adjacent
	// A range overlaps or is adjacent if: existing.End >= start
	firstIdx := -1
	lastIdx := -1

	for i, r := range brm.ranges {
		// Check if this range overlaps or is adjacent to our new range
		if r.End >= start && r.Start <= end {
			if firstIdx == -1 {
				firstIdx = i
			}
			lastIdx = i
		}
	}

	// No overlaps, find where to insert
	if firstIdx == -1 {
		// Find insertion point
		insertIdx := len(brm.ranges)
		for i, r := range brm.ranges {
			if start < r.Start {
				insertIdx = i
				break
			}
		}

		// Insert at the right position
		newRanges := make([]ByteRange, 0, len(brm.ranges)+1)
		newRanges = append(newRanges, brm.ranges[:insertIdx]...)
		newRanges = append(newRanges, ByteRange{Start: start, End: end})
		newRanges = append(newRanges, brm.ranges[insertIdx:]...)
		brm.ranges = newRanges

		return end - start
	}

	// Calculate the merged range
	mergedStart := min(start, brm.ranges[firstIdx].Start)
	mergedEnd := max(end, brm.ranges[lastIdx].End)

	// Calculate bytes added
	// Total new bytes = merged range size - sum of all overlapping existing ranges
	existingBytes := uint64(0)
	for i := firstIdx; i <= lastIdx; i++ {
		existingBytes += brm.ranges[i].End - brm.ranges[i].Start
	}
	mergedBytes := mergedEnd - mergedStart
	bytesAdded := mergedBytes - existingBytes

	// Build new ranges slice with the merged range
	newRanges := make([]ByteRange, 0, len(brm.ranges)-(lastIdx-firstIdx))
	newRanges = append(newRanges, brm.ranges[:firstIdx]...)
	newRanges = append(newRanges, ByteRange{Start: mergedStart, End: mergedEnd})
	newRanges = append(newRanges, brm.ranges[lastIdx+1:]...)

	brm.ranges = newRanges
	return bytesAdded
}

// ContainsRange checks if the entire range [start, end) has been downloaded
func (brm *ByteRangeMap) ContainsRange(start, end uint64) bool {
	brm.mu.RLock()
	defer brm.mu.RUnlock()

	if start >= end {
		return true
	}

	// Binary search to find the first range that might contain our start
	idx := sort.Search(len(brm.ranges), func(i int) bool {
		return brm.ranges[i].End > start
	})

	if idx >= len(brm.ranges) {
		return false
	}

	// Check if the found range contains the entire requested range
	return brm.ranges[idx].Start <= start && brm.ranges[idx].End >= end
}

// GetMissingRanges returns a list of byte ranges within [start, end) that haven't been downloaded yet
func (brm *ByteRangeMap) GetMissingRanges(start, end uint64) []ByteRange {
	brm.mu.RLock()
	defer brm.mu.RUnlock()

	if start >= end {
		return nil
	}

	var missing []ByteRange
	current := start

	for _, r := range brm.ranges {
		// If this range is completely after our request, we're done
		if r.Start >= end {
			break
		}

		// If this range is completely before our current position, skip it
		if r.End <= current {
			continue
		}

		// If there's a gap between current and this range, it's missing
		if current < r.Start && r.Start < end {
			missing = append(missing, ByteRange{
				Start: current,
				End:   min(r.Start, end),
			})
		}

		// Move current forward
		current = max(current, r.End)

		// If we've covered the entire requested range, we're done
		if current >= end {
			break
		}
	}

	// If there's still uncovered range at the end
	if current < end {
		missing = append(missing, ByteRange{
			Start: current,
			End:   end,
		})
	}

	return missing
}

// TotalBytes returns the total number of bytes covered by all ranges
func (brm *ByteRangeMap) TotalBytes() uint64 {
	brm.mu.RLock()
	defer brm.mu.RUnlock()

	var total uint64
	for _, r := range brm.ranges {
		total += r.End - r.Start
	}
	return total
}

// Clear removes all ranges
func (brm *ByteRangeMap) Clear() {
	brm.mu.Lock()
	defer brm.mu.Unlock()
	brm.ranges = make([]ByteRange, 0)
}

// Ranges returns a copy of all ranges (for debugging/testing)
func (brm *ByteRangeMap) Ranges() []ByteRange {
	brm.mu.RLock()
	defer brm.mu.RUnlock()

	result := make([]ByteRange, len(brm.ranges))
	copy(result, brm.ranges)
	return result
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
