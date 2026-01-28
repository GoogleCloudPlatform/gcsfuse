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
	"slices"
	"sync"
)

const DefaultChunkSize = 1024 * 1024 // 1MB

// ByteRangeMap tracks which chunks have been downloaded in a sparse file.
// The chunk size should match the actual download chunk size for efficient tracking.
type ByteRangeMap struct {
	mu         sync.RWMutex
	chunkSize  uint64
	fileSize   uint64          // Total size of the file
	chunks     map[uint64]bool // chunk ID -> downloaded
	totalBytes uint64          // Total bytes downloaded
}

// NewByteRangeMap creates a new empty ByteRangeMap with the specified chunk size and file size.
// The chunkSize should match the download chunk size (e.g., DownloadChunkSizeMb * 1MB).
func NewByteRangeMap(chunkSize, fileSize uint64) *ByteRangeMap {
	if chunkSize == 0 {
		chunkSize = DefaultChunkSize
	}
	return &ByteRangeMap{
		chunkSize: chunkSize,
		fileSize:  fileSize,
		chunks:    make(map[uint64]bool),
	}
}

// chunkID returns the chunk ID for a given byte offset
func (brm *ByteRangeMap) chunkID(offset uint64) uint64 {
	return offset / brm.chunkSize
}

// chunkSizeOf returns the size of a specific chunk, handling the last chunk which might be smaller.
func (brm *ByteRangeMap) chunkSizeOf(chunkID uint64) uint64 {
	chunkStart := chunkID * brm.chunkSize
	if chunkStart >= brm.fileSize {
		return 0
	}
	chunkEnd := chunkStart + brm.chunkSize
	if chunkEnd > brm.fileSize {
		return brm.fileSize - chunkStart
	}
	return brm.chunkSize
}

// AddRange marks all chunks in the range [start, end) as downloaded.
// Returns the total number of new bytes added.
func (brm *ByteRangeMap) AddRange(start, end uint64) uint64 {
	brm.mu.Lock()
	defer brm.mu.Unlock()

	if start >= end {
		return 0
	}

	startChunk := brm.chunkID(start)
	endChunk := brm.chunkID(end - 1) // inclusive end

	bytesAdded := uint64(0)
	for chunkID := startChunk; chunkID <= endChunk; chunkID++ {
		if !brm.chunks[chunkID] {
			brm.chunks[chunkID] = true
			bytesAdded += brm.chunkSizeOf(chunkID)
		}
	}

	brm.totalBytes += bytesAdded
	return bytesAdded
}

// ContainsRange checks if all chunks covering [start, end) have been downloaded
func (brm *ByteRangeMap) ContainsRange(start, end uint64) bool {
	brm.mu.RLock()
	defer brm.mu.RUnlock()

	if start >= end {
		return true
	}

	startChunk := brm.chunkID(start)
	endChunk := brm.chunkID(end - 1)

	for chunkID := startChunk; chunkID <= endChunk; chunkID++ {
		if !brm.chunks[chunkID] {
			return false
		}
	}
	return true
}

// GetMissingChunks returns the IDs of chunks that haven't been downloaded.
func (brm *ByteRangeMap) GetMissingChunks(start, end uint64) []uint64 {
	brm.mu.RLock()
	defer brm.mu.RUnlock()

	if start >= end {
		return nil
	}

	var missing []uint64
	startChunk := brm.chunkID(start)
	endChunk := brm.chunkID(end - 1)

	var currentStart uint64 = ^uint64(0) // Sentinel value

	for chunkID := startChunk; chunkID <= endChunk; chunkID++ {
		if !brm.chunks[chunkID] {
			missing = append(missing, chunkID)
		}
	}

	// Handle the last segment if it extends to the end of the requested range
	if currentStart != ^uint64(0) {
		lastChunkEnd := (endChunk + 1) * brm.chunkSize
		if lastChunkEnd > brm.fileSize {
			lastChunkEnd = brm.fileSize
		}
		missing = append(missing, ByteRange{Start: currentStart, End: lastChunkEnd})
	}

	return missing
}

// TotalBytes returns the total number of bytes downloaded (sum of chunk sizes)
func (brm *ByteRangeMap) TotalBytes() uint64 {
	brm.mu.RLock()
	defer brm.mu.RUnlock()
	return brm.totalBytes
}

// Clear removes all chunk records
func (brm *ByteRangeMap) Clear() {
	brm.mu.Lock()
	defer brm.mu.Unlock()
	brm.chunks = make(map[uint64]bool)
	brm.totalBytes = 0
}

// Chunks returns a sorted list of all downloaded chunk IDs. (for debugging/testing)
func (brm *ByteRangeMap) Chunks() []uint64 {
	brm.mu.RLock()
	defer brm.mu.RUnlock()

	chunks := make([]uint64, 0, len(brm.chunks))
	for id := range brm.chunks {
		chunks = append(chunks, id)
	}
	slices.Sort(chunks)
	return chunks
}
