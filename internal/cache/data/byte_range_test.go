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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteRangeMap_AddRange(t *testing.T) {
	tests := []struct {
		name           string
		initialRanges  []ByteRange
		addStart       uint64
		addEnd         uint64
		expectedRanges []ByteRange
		expectedAdded  uint64
	}{
		{
			name:           "add to empty map",
			initialRanges:  []ByteRange{},
			addStart:       10,
			addEnd:         20,
			expectedRanges: []ByteRange{{10, 20}},
			expectedAdded:  10,
		},
		{
			name:           "add non-overlapping range before",
			initialRanges:  []ByteRange{{20, 30}},
			addStart:       10,
			addEnd:         15,
			expectedRanges: []ByteRange{{10, 15}, {20, 30}},
			expectedAdded:  5,
		},
		{
			name:           "add non-overlapping range after",
			initialRanges:  []ByteRange{{10, 20}},
			addStart:       30,
			addEnd:         40,
			expectedRanges: []ByteRange{{10, 20}, {30, 40}},
			expectedAdded:  10,
		},
		{
			name:           "add adjacent range before (merge)",
			initialRanges:  []ByteRange{{20, 30}},
			addStart:       10,
			addEnd:         20,
			expectedRanges: []ByteRange{{10, 30}},
			expectedAdded:  10,
		},
		{
			name:           "add adjacent range after (merge)",
			initialRanges:  []ByteRange{{10, 20}},
			addStart:       20,
			addEnd:         30,
			expectedRanges: []ByteRange{{10, 30}},
			expectedAdded:  10,
		},
		{
			name:           "add overlapping range",
			initialRanges:  []ByteRange{{10, 20}},
			addStart:       15,
			addEnd:         25,
			expectedRanges: []ByteRange{{10, 25}},
			expectedAdded:  5, // only 20-25 is new
		},
		{
			name:           "add range contained in existing",
			initialRanges:  []ByteRange{{10, 30}},
			addStart:       15,
			addEnd:         25,
			expectedRanges: []ByteRange{{10, 30}},
			expectedAdded:  0, // already covered
		},
		{
			name:           "add range that contains existing",
			initialRanges:  []ByteRange{{15, 25}},
			addStart:       10,
			addEnd:         30,
			expectedRanges: []ByteRange{{10, 30}},
			expectedAdded:  10, // 10-15 and 25-30 are new
		},
		{
			name:           "merge multiple ranges",
			initialRanges:  []ByteRange{{10, 20}, {30, 40}},
			addStart:       15,
			addEnd:         35,
			expectedRanges: []ByteRange{{10, 40}},
			expectedAdded:  10, // 20-30 is new
		},
		{
			name:           "add invalid range (start >= end)",
			initialRanges:  []ByteRange{{10, 20}},
			addStart:       30,
			addEnd:         30,
			expectedRanges: []ByteRange{{10, 20}},
			expectedAdded:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			brm := NewByteRangeMap()
			// Add initial ranges
			for _, r := range tt.initialRanges {
				brm.AddRange(r.Start, r.End)
			}

			// Add the test range
			added := brm.AddRange(tt.addStart, tt.addEnd)

			// Check results
			assert.Equal(t, tt.expectedAdded, added, "bytes added mismatch")
			assert.Equal(t, tt.expectedRanges, brm.Ranges(), "ranges mismatch")
		})
	}
}

func TestByteRangeMap_ContainsRange(t *testing.T) {
	brm := NewByteRangeMap()
	brm.AddRange(10, 20)
	brm.AddRange(30, 40)
	brm.AddRange(50, 60)

	tests := []struct {
		name     string
		start    uint64
		end      uint64
		expected bool
	}{
		{"fully contained in first range", 12, 18, true},
		{"exact match first range", 10, 20, true},
		{"fully contained in middle range", 32, 38, true},
		{"spans two ranges", 15, 35, false},
		{"starts before first range", 5, 15, false},
		{"ends after last range", 55, 65, false},
		{"completely outside ranges", 70, 80, false},
		{"empty range", 25, 25, true}, // empty ranges are considered contained
		{"starts in range, ends outside", 15, 25, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := brm.ContainsRange(tt.start, tt.end)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestByteRangeMap_GetMissingRanges(t *testing.T) {
	brm := NewByteRangeMap()
	brm.AddRange(10, 20)
	brm.AddRange(30, 40)
	brm.AddRange(50, 60)

	tests := []struct {
		name     string
		start    uint64
		end      uint64
		expected []ByteRange
	}{
		{
			name:     "fully covered range",
			start:    12,
			end:      18,
			expected: nil,
		},
		{
			name:     "gap between two ranges",
			start:    15,
			end:      35,
			expected: []ByteRange{{20, 30}},
		},
		{
			name:     "multiple gaps",
			start:    5,
			end:      65,
			expected: []ByteRange{{5, 10}, {20, 30}, {40, 50}, {60, 65}},
		},
		{
			name:     "completely missing range",
			start:    70,
			end:      80,
			expected: []ByteRange{{70, 80}},
		},
		{
			name:     "starts in gap, ends in range",
			start:    22,
			end:      35,
			expected: []ByteRange{{22, 30}},
		},
		{
			name:     "starts in range, ends in gap",
			start:    15,
			end:      25,
			expected: []ByteRange{{20, 25}},
		},
		{
			name:     "empty range",
			start:    25,
			end:      25,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := brm.GetMissingRanges(tt.start, tt.end)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestByteRangeMap_TotalBytes(t *testing.T) {
	brm := NewByteRangeMap()

	assert.Equal(t, uint64(0), brm.TotalBytes(), "empty map should have 0 bytes")

	brm.AddRange(10, 20) // 10 bytes
	assert.Equal(t, uint64(10), brm.TotalBytes())

	brm.AddRange(30, 50) // 20 bytes
	assert.Equal(t, uint64(30), brm.TotalBytes())

	brm.AddRange(15, 35) // merges ranges, adds 10 bytes (20-30)
	assert.Equal(t, uint64(40), brm.TotalBytes()) // 10-50 = 40 bytes
}

func TestByteRangeMap_Clear(t *testing.T) {
	brm := NewByteRangeMap()
	brm.AddRange(10, 20)
	brm.AddRange(30, 40)

	assert.Equal(t, uint64(20), brm.TotalBytes())

	brm.Clear()

	assert.Equal(t, uint64(0), brm.TotalBytes())
	assert.Empty(t, brm.Ranges())
}

func TestByteRangeMap_ConcurrentAccess(t *testing.T) {
	brm := NewByteRangeMap()

	// This test just ensures no race conditions occur
	// Run with -race flag to detect issues
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := uint64(0); i < 100; i += 10 {
			brm.AddRange(i, i+5)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := uint64(0); i < 100; i += 10 {
			brm.ContainsRange(i, i+5)
			brm.GetMissingRanges(i, i+10)
			brm.TotalBytes()
		}
		done <- true
	}()

	<-done
	<-done
}
