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

const MB = 1024 * 1024

func TestByteRangeMap_AddRange(t *testing.T) {
	tests := []struct {
		name           string
		initialRanges  [][2]uint64 // [start, end] pairs
		addStart       uint64
		addEnd         uint64
		expectedRanges []ByteRange // chunk-aligned ranges
		expectedAdded  uint64       // bytes added (chunk size multiples)
	}{
		{
			name:           "add to empty map - single chunk",
			initialRanges:  [][2]uint64{},
			addStart:       0,
			addEnd:         MB,
			expectedRanges: []ByteRange{{0, MB}},
			expectedAdded:  MB,
		},
		{
			name:           "add to empty map - partial chunk becomes full chunk",
			initialRanges:  [][2]uint64{},
			addStart:       100,
			addEnd:         200,
			expectedRanges: []ByteRange{{0, MB}},
			expectedAdded:  MB, // tracks full chunk
		},
		{
			name:           "add non-overlapping chunk",
			initialRanges:  [][2]uint64{{0, MB}},
			addStart:       2 * MB,
			addEnd:         3 * MB,
			expectedRanges: []ByteRange{{0, MB}, {2 * MB, 3 * MB}},
			expectedAdded:  MB,
		},
		{
			name:           "add already downloaded chunk",
			initialRanges:  [][2]uint64{{0, MB}},
			addStart:       100,
			addEnd:         200,
			expectedRanges: []ByteRange{{0, MB}},
			expectedAdded:  0, // chunk already tracked
		},
		{
			name:           "add range spanning two chunks",
			initialRanges:  [][2]uint64{},
			addStart:       0,
			addEnd:         2 * MB,
			expectedRanges: []ByteRange{{0, 2 * MB}},
			expectedAdded:  2 * MB,
		},
		{
			name:           "add range spanning partial chunks",
			initialRanges:  [][2]uint64{},
			addStart:       MB / 2,
			addEnd:         MB + MB/2,
			expectedRanges: []ByteRange{{0, 2 * MB}}, // both chunks 0 and 1
			expectedAdded:  2 * MB,
		},
		{
			name:           "add range that fills gap",
			initialRanges:  [][2]uint64{{0, MB}, {2 * MB, 3 * MB}},
			addStart:       MB,
			addEnd:         2 * MB,
			expectedRanges: []ByteRange{{0, 3 * MB}},
			expectedAdded:  MB,
		},
		{
			name:           "add overlapping chunks - some new",
			initialRanges:  [][2]uint64{{0, MB}},
			addStart:       0,
			addEnd:         2 * MB,
			expectedRanges: []ByteRange{{0, 2 * MB}},
			expectedAdded:  MB, // only chunk 1 is new
		},
		{
			name:           "add invalid range (start >= end)",
			initialRanges:  [][2]uint64{{0, MB}},
			addStart:       2 * MB,
			addEnd:         2 * MB,
			expectedRanges: []ByteRange{{0, MB}},
			expectedAdded:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			brm := NewByteRangeMap(DefaultChunkSize)
			// Add initial ranges
			for _, r := range tt.initialRanges {
				brm.AddRange(r[0], r[1])
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
	brm := NewByteRangeMap(DefaultChunkSize)
	brm.AddRange(0, MB)       // chunk 0
	brm.AddRange(2*MB, 3*MB)  // chunk 2
	brm.AddRange(5*MB, 6*MB)  // chunk 5

	tests := []struct {
		name     string
		start    uint64
		end      uint64
		expected bool
	}{
		{"fully contained in first chunk", 100, 200, true},
		{"exact match first chunk", 0, MB, true},
		{"fully contained in middle chunk", 2*MB + 100, 2*MB + 200, true},
		{"spans downloaded and missing chunk", MB / 2, MB + MB/2, false},
		{"starts in downloaded, ends in missing", 100, MB + 100, false},
		{"completely outside ranges", 7 * MB, 8 * MB, false},
		{"empty range", MB + MB/2, MB + MB/2, true}, // empty ranges are considered contained
		{"spans two downloaded chunks with gap", 0, 3 * MB, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := brm.ContainsRange(tt.start, tt.end)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestByteRangeMap_GetMissingRanges(t *testing.T) {
	brm := NewByteRangeMap(DefaultChunkSize)
	brm.AddRange(0, MB)       // chunk 0
	brm.AddRange(2*MB, 3*MB)  // chunk 2
	brm.AddRange(5*MB, 6*MB)  // chunk 5

	tests := []struct {
		name     string
		start    uint64
		end      uint64
		expected []ByteRange
	}{
		{
			name:     "fully covered range",
			start:    100,
			end:      200,
			expected: nil,
		},
		{
			name:     "single missing chunk",
			start:    MB,
			end:      2 * MB,
			expected: []ByteRange{{MB, 2 * MB}},
		},
		{
			name:  "multiple missing chunks",
			start: 0,
			end:   6 * MB,
			expected: []ByteRange{
				{MB, 2 * MB},     // chunk 1
				{3 * MB, 4 * MB}, // chunk 3
				{4 * MB, 5 * MB}, // chunk 4
			},
		},
		{
			name:     "completely missing range",
			start:    10 * MB,
			end:      11 * MB,
			expected: []ByteRange{{10 * MB, 11 * MB}},
		},
		{
			name:     "partial chunk request - missing",
			start:    MB + 100,
			end:      MB + 200,
			expected: []ByteRange{{MB, 2 * MB}}, // returns full chunk
		},
		{
			name:     "partial chunk request - present",
			start:    100,
			end:      200,
			expected: nil,
		},
		{
			name:     "empty range",
			start:    MB + MB/2,
			end:      MB + MB/2,
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
	brm := NewByteRangeMap(DefaultChunkSize)

	assert.Equal(t, uint64(0), brm.TotalBytes(), "empty map should have 0 bytes")

	brm.AddRange(0, MB) // 1 chunk
	assert.Equal(t, uint64(MB), brm.TotalBytes())

	brm.AddRange(2*MB, 4*MB) // 2 chunks
	assert.Equal(t, uint64(3*MB), brm.TotalBytes())

	brm.AddRange(MB, 2*MB) // 1 chunk, fills gap
	assert.Equal(t, uint64(4*MB), brm.TotalBytes()) // 4 contiguous chunks
}

func TestByteRangeMap_Clear(t *testing.T) {
	brm := NewByteRangeMap(DefaultChunkSize)
	brm.AddRange(0, MB)
	brm.AddRange(2*MB, 3*MB)

	assert.Equal(t, uint64(2*MB), brm.TotalBytes())

	brm.Clear()

	assert.Equal(t, uint64(0), brm.TotalBytes())
	assert.Empty(t, brm.Ranges())
}

func TestByteRangeMap_ConcurrentAccess(t *testing.T) {
	brm := NewByteRangeMap(DefaultChunkSize)

	// This test just ensures no race conditions occur
	// Run with -race flag to detect issues
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := uint64(0); i < 10; i++ {
			brm.AddRange(i*MB, (i+1)*MB)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := uint64(0); i < 10; i++ {
			brm.ContainsRange(i*MB, (i+1)*MB)
			brm.GetMissingRanges(i*MB, (i+2)*MB)
			brm.TotalBytes()
		}
		done <- true
	}()

	<-done
	<-done
}

func TestByteRangeMap_ChunkAlignment(t *testing.T) {
	brm := NewByteRangeMap(DefaultChunkSize)

	// Test that partial byte ranges get tracked as full chunks
	brm.AddRange(100, 200)

	// Should track chunk 0
	assert.True(t, brm.ContainsRange(0, MB))
	assert.True(t, brm.ContainsRange(100, 200))
	assert.True(t, brm.ContainsRange(0, 1000))
	assert.Equal(t, uint64(MB), brm.TotalBytes())

	// Should not contain chunk 1
	assert.False(t, brm.ContainsRange(MB, MB+1))
	assert.False(t, brm.ContainsRange(100, MB+100))
}
