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
	"sync/atomic"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

// ReadType represents the overall read pattern
type ReadType int

const (
	// ReadTypeUnknown indicates the read pattern hasn't been determined yet
	ReadTypeUnknown ReadType = iota
	// ReadTypeSequential indicates the current pattern is predominantly sequential
	ReadTypeSequential
	// ReadTypeRandom indicates the current pattern is predominantly random
	ReadTypeRandom
)

// SharedReadState holds shared state information across all readers for a file handle.
// This tracks the overall read pattern and provides average bytes per random seek.
type SharedReadState struct {
	// mu protects the non-atomic fields in this structure
	mu sync.RWMutex

	// totalBytesRead tracks the total number of bytes read across all readers
	totalBytesRead atomic.Uint64

	// randomSeekCount tracks the number of random/non-sequential reads
	randomSeekCount atomic.Uint64

	// lastReadOffset tracks the last read offset to help detect sequential patterns
	lastReadOffset atomic.Int64

	// GUARDED_BY(mu)
	// currentReadType indicates the overall read pattern at the current moment
	currentReadType ReadType

	// GUARDED_BY(mu)
	// activeReaderType tracks which type of reader is currently handling reads
	activeReaderType string
}

// NewSharedReadState creates a new SharedReadState with default configuration
func NewSharedReadState() *SharedReadState {
	return &SharedReadState{
		currentReadType: ReadTypeUnknown,
		lastReadOffset:  atomic.Int64{},
	}
}

// RecordRead records a read operation and updates the shared state
func (s *SharedReadState) RecordRead(offset int64, size int64) {
	s.totalBytesRead.Add(uint64(size))

	lastOffset := s.lastReadOffset.Load()
	isSequential := offset == lastOffset

	if !isSequential {
		s.randomSeekCount.Add(1)
	}

	s.lastReadOffset.Store(offset + size)

	// Update current read type under lock
	s.mu.Lock()
	defer s.mu.Unlock()

	if isSequential {
		s.currentReadType = ReadTypeSequential
	} else {
		s.currentReadType = ReadTypeRandom
	}
}

// GetReadStats returns current read statistics
func (s *SharedReadState) GetReadStats() ReadStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return ReadStats{
		TotalBytesRead:      s.totalBytesRead.Load(),
		RandomSeekCount:     s.randomSeekCount.Load(),
		AverageBytesPerSeek: s.getAverageBytesPerSeek(),
		CurrentReadType:     s.currentReadType,
		ActiveReaderType:    s.activeReaderType,
	}
}

// getAverageBytesPerSeek calculates average bytes per random seek
// Must be called with read lock held
func (s *SharedReadState) getAverageBytesPerSeek() float64 {
	seekCount := s.randomSeekCount.Load()
	if seekCount == 0 {
		return 0.0
	}
	return float64(s.totalBytesRead.Load()) / float64(seekCount)
}

// ReadStats contains statistical information about read operations
type ReadStats struct {
	TotalBytesRead      uint64
	RandomSeekCount     uint64
	AverageBytesPerSeek float64
	CurrentReadType     ReadType
	ActiveReaderType    string
}

// GetActiveReaderType returns the currently active reader type
func (s *SharedReadState) GetActiveReaderType() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeReaderType
}

// SetActiveReaderType sets the currently active reader type
func (s *SharedReadState) SetActiveReaderType(readerType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeReaderType = readerType
}

// GetCurrentReadType returns the overall read pattern at the current moment
func (s *SharedReadState) GetCurrentReadType() ReadType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentReadType
}

// GetAverageBytesPerSeek returns the average bytes read per random seek
func (s *SharedReadState) GetAverageBytesPerSeek() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getAverageBytesPerSeek()
}

// ShouldRestartBufferedReader returns true if buffered reader should be restarted
// based on the read pattern change from random to sequential
func (s *SharedReadState) ShouldRestartBufferedReader() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Restart if current pattern is sequential, we previously had random reads,
	// and the average bytes per seek is more than 8 MiB
	const minAverageBytesForSequential = 8 * 1024 * 1024 // 8 MiB
	averageBytes := s.getAverageBytesPerSeek()
	logger.Infof("SharedReadState: randomSeekCount=%d, averageBytes=%.0f", s.randomSeekCount.Load(), averageBytes)

	return s.currentReadType == ReadTypeSequential &&
		s.randomSeekCount.Load() > 0 &&
		averageBytes > minAverageBytesForSequential
}

// Reset clears all accumulated state (useful for testing or when switching contexts)
func (s *SharedReadState) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.totalBytesRead.Store(0)
	s.randomSeekCount.Store(0)
	s.lastReadOffset.Store(0)
	s.currentReadType = ReadTypeUnknown
	s.activeReaderType = ""
}
