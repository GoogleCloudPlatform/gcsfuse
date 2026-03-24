// Copyright 2026 Google LLC
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

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

var (
	// ErrBatchNotFound is returned when a batch request is not found in the registry
	ErrBatchNotFound = errors.New("batch request not found")
	// ErrBatchAlreadyExists is returned when trying to create a batch that already exists
	ErrBatchAlreadyExists = errors.New("batch request already exists")
	// ErrInvalidBlockID is returned when block ID is invalid
	ErrInvalidBlockID = errors.New("invalid block ID")
)

// ReadRequestState represents the state of a buffered read request
type ReadRequestState int

const (
	// INITIAL state: request has been created but not yet processed
	INITIAL ReadRequestState = iota
	// WAITING state: request is waiting for more reads to form a batch
	WAITING
	// SCHEDULED state: batch has been scheduled for execution
	SCHEDULED
	// READING state: batch is currently being read from GCS
	READING
	// COMPLETED state: batch read has completed successfully
	COMPLETED
	// FAILED state: batch read has failed
	FAILED
)

func (s ReadRequestState) String() string {
	switch s {
	case INITIAL:
		return "INITIAL"
	case WAITING:
		return "WAITING"
	case SCHEDULED:
		return "SCHEDULED"
	case READING:
		return "READING"
	case COMPLETED:
		return "COMPLETED"
	case FAILED:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// BatchReadConfig holds configuration for batch read operations
type BatchReadConfig struct {
	// MaxReadSizeMb is the maximum size for a single read before batching (default: 1MB)
	MaxReadSizeMb int64
	// ReadAheadMb is the read-ahead size in MB (default: 64-128MB)
	ReadAheadMb int64
	// WaitTimeMs is the wait time in milliseconds for batch formation (default: 1ms)
	WaitTimeMs int64
	// BlockSizeMb is the size of each block for batching (default: 16MB)
	BlockSizeMb int64
	// CleanupIntervalSec is the interval for cleaning up expired requests (default: 60s)
	CleanupIntervalSec int64
}

// ReadIntent represents an intention to read a specific range
type ReadIntent struct {
	offset     int64
	size       int
	buffer     []byte
	resultChan chan *ReadResult
	timestamp  time.Time
}

// ReadResult holds the result of a read operation
type ReadResult struct {
	data []byte
	size int
	err  error
}

// BufferedReadRequest represents a batch read request with multiple read intents
type BufferedReadRequest struct {
	blockID int64
	state   ReadRequestState
	mu      sync.RWMutex

	// pendingReads contains read intents waiting to be batched
	pendingReads []*ReadIntent

	// completionChan is used to notify joined reads when batch completes
	completionChan chan *BatchReadResult

	// buffer holds the read data (entire block)
	buffer []byte

	// refCount tracks number of active references to this buffer
	refCount atomic.Int32

	// err holds any error that occurred during read
	err error

	// createdAt tracks when this request was created
	createdAt time.Time

	// completedAt tracks when this request completed
	completedAt time.Time
}

// BatchReadResult holds the result of a batch read operation
type BatchReadResult struct {
	buffer []byte
	err    error
}

// BatchReadRegistry manages batch read requests for an inode
type BatchReadRegistry struct {
	mu      sync.RWMutex
	config  *BatchReadConfig
	stopped atomic.Bool

	// requests maps block_id to BufferedReadRequest
	// block_id = offset - (offset % blockSize)
	requests map[int64]*BufferedReadRequest

	// stopChan is used to signal cleanup goroutine to stop
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewBatchReadRegistry creates a new batch read registry
func NewBatchReadRegistry(config *BatchReadConfig) *BatchReadRegistry {
	if config == nil {
		config = &BatchReadConfig{
			MaxReadSizeMb:      1,
			ReadAheadMb:        64,
			WaitTimeMs:         1,
			BlockSizeMb:        16,
			CleanupIntervalSec: 60,
		}
	}

	registry := &BatchReadRegistry{
		config:   config,
		requests: make(map[int64]*BufferedReadRequest),
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	registry.wg.Add(1)
	go registry.cleanupLoop()

	return registry
}

// CalculateBlockID calculates the block ID for a given offset
func (r *BatchReadRegistry) CalculateBlockID(offset int64) int64 {
	blockSize := r.config.BlockSizeMb * 1024 * 1024
	return offset - (offset % blockSize)
}

// RegisterReadIntent registers an intention to read a range
func (r *BatchReadRegistry) RegisterReadIntent(offset int64, size int, buffer []byte) (*ReadIntent, int64) {
	blockID := r.CalculateBlockID(offset)

	intent := &ReadIntent{
		offset:     offset,
		size:       size,
		buffer:     buffer,
		resultChan: make(chan *ReadResult, 1),
		timestamp:  time.Now(),
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Get or create the buffered read request for this block
	req, exists := r.requests[blockID]
	if !exists {
		req = &BufferedReadRequest{
			blockID:        blockID,
			state:          INITIAL,
			pendingReads:   make([]*ReadIntent, 0),
			completionChan: make(chan *BatchReadResult, 100), // Buffered to avoid blocking
			createdAt:      time.Now(),
		}
		r.requests[blockID] = req
	}

	// Add this intent to the pending reads
	req.mu.Lock()
	req.pendingReads = append(req.pendingReads, intent)
	req.mu.Unlock()

	logger.Tracef("Registered read intent at offset %d, size %d for block %d", offset, size, blockID)

	return intent, blockID
}

// QueryBatch queries for an existing batch request
func (r *BatchReadRegistry) QueryBatch(blockID int64) (*BufferedReadRequest, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	req, exists := r.requests[blockID]
	return req, exists
}

// GetAdjacentReads returns all pending read intents for a block
func (r *BatchReadRegistry) GetAdjacentReads(blockID int64) []*ReadIntent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	req, exists := r.requests[blockID]
	if !exists {
		return nil
	}

	req.mu.RLock()
	defer req.mu.RUnlock()

	// Return a copy to avoid concurrent modification issues
	intents := make([]*ReadIntent, len(req.pendingReads))
	copy(intents, req.pendingReads)
	return intents
}

// MarkBatchScheduled marks a batch as scheduled for reading
// Only allows transition from INITIAL or WAITING states to prevent multiple
// readers from scheduling the same batch simultaneously.
func (r *BatchReadRegistry) MarkBatchScheduled(blockID int64) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	req, exists := r.requests[blockID]
	if !exists {
		return ErrBatchNotFound
	}

	req.mu.Lock()
	defer req.mu.Unlock()

	// Only allow transition from INITIAL or WAITING to SCHEDULED
	// This prevents race conditions where multiple readers try to schedule the same batch
	if req.state != INITIAL && req.state != WAITING {
		return fmt.Errorf("batch already in state %s, cannot schedule", req.state.String())
	}

	req.state = SCHEDULED
	logger.Tracef("Batch %d marked as SCHEDULED", blockID)
	return nil
}

// MarkBatchReading marks a batch as currently being read
// Only allows transition from SCHEDULED state.
func (r *BatchReadRegistry) MarkBatchReading(blockID int64) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	req, exists := r.requests[blockID]
	if !exists {
		return ErrBatchNotFound
	}

	req.mu.Lock()
	defer req.mu.Unlock()

	// Only allow transition from SCHEDULED to READING
	if req.state != SCHEDULED {
		return fmt.Errorf("batch in state %s, expected SCHEDULED", req.state.String())
	}

	req.state = READING
	logger.Tracef("Batch %d marked as READING", blockID)
	return nil
}

// CompleteRead marks a batch read as completed and notifies all waiting reads
func (r *BatchReadRegistry) CompleteRead(blockID int64, buffer []byte, err error) error {
	r.mu.RLock()
	req, exists := r.requests[blockID]
	r.mu.RUnlock()

	if !exists {
		return ErrBatchNotFound
	}

	req.mu.Lock()
	req.state = COMPLETED
	req.buffer = buffer
	req.err = err
	req.completedAt = time.Now()
	req.mu.Unlock()

	// Note: refCount is managed by IncrementRefCount/DecrementRefCount calls from readers
	// We don't set it here to avoid race conditions

	logger.Tracef("Batch %d completed, err=%v", blockID, err)

	// Notify all waiting reads by closing the channel
	// All readers waiting on this channel will wake up
	close(req.completionChan)

	return nil
}

// FailRead marks a batch read as failed
func (r *BatchReadRegistry) FailRead(blockID int64, err error) error {
	r.mu.RLock()
	req, exists := r.requests[blockID]
	r.mu.RUnlock()

	if !exists {
		return ErrBatchNotFound
	}

	req.mu.Lock()
	req.state = FAILED
	req.err = err
	req.completedAt = time.Now()
	req.mu.Unlock()

	logger.Tracef("Batch %d marked as FAILED: %v", blockID, err)

	// Notify all waiting reads by closing the channel
	close(req.completionChan)

	return nil
}

// IncrementRefCount increments the reference count for a batch
// This should be called when a reader joins an existing batch
func (r *BatchReadRegistry) IncrementRefCount(blockID int64) {
	r.mu.RLock()
	req, exists := r.requests[blockID]
	r.mu.RUnlock()

	if !exists {
		logger.Warnf("Cannot increment refCount: batch %d not found", blockID)
		return
	}

	newCount := req.refCount.Add(1)
	logger.Tracef("Incremented refCount for batch %d, new count=%d", blockID, newCount)
}

// DecrementRefCount decrements the reference count and frees buffer if count reaches 0
func (r *BatchReadRegistry) DecrementRefCount(blockID int64) {
	r.mu.RLock()
	req, exists := r.requests[blockID]
	r.mu.RUnlock()

	if !exists {
		return
	}

	newCount := req.refCount.Add(-1)
	logger.Tracef("Decremented refCount for batch %d, new count=%d", blockID, newCount)

	if newCount == 0 {
		// Free the buffer and remove from registry
		req.mu.Lock()
		req.buffer = nil
		req.mu.Unlock()

		r.mu.Lock()
		delete(r.requests, blockID)
		r.mu.Unlock()

		logger.Tracef("Freed buffer and removed batch %d from registry", blockID)
	}
}

// cleanupLoop periodically cleans up expired requests
func (r *BatchReadRegistry) cleanupLoop() {
	defer r.wg.Done()

	ticker := time.NewTicker(time.Duration(r.config.CleanupIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cleanupExpiredRequests()
		case <-r.stopChan:
			return
		}
	}
}

// cleanupExpiredRequests removes old completed or failed requests
func (r *BatchReadRegistry) cleanupExpiredRequests() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	expirationThreshold := time.Duration(r.config.CleanupIntervalSec) * time.Second

	for blockID, req := range r.requests {
		req.mu.RLock()
		shouldCleanup := (req.state == COMPLETED || req.state == FAILED) &&
			!req.completedAt.IsZero() &&
			now.Sub(req.completedAt) > expirationThreshold &&
			req.refCount.Load() == 0
		req.mu.RUnlock()

		if shouldCleanup {
			delete(r.requests, blockID)
			logger.Tracef("Cleaned up expired batch %d", blockID)
		}
	}
}

// Stop stops the cleanup goroutine and releases resources
func (r *BatchReadRegistry) Stop() {
	if r.stopped.CompareAndSwap(false, true) {
		close(r.stopChan)
		r.wg.Wait()

		r.mu.Lock()
		defer r.mu.Unlock()

		// Clear all requests
		r.requests = make(map[int64]*BufferedReadRequest)
		logger.Tracef("Batch read registry stopped")
	}
}

// GetStats returns statistics about the registry
func (r *BatchReadRegistry) GetStats() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := map[string]int{
		"total":     0,
		"initial":   0,
		"waiting":   0,
		"scheduled": 0,
		"reading":   0,
		"completed": 0,
		"failed":    0,
	}

	for _, req := range r.requests {
		req.mu.RLock()
		state := req.state
		req.mu.RUnlock()

		stats["total"]++
		switch state {
		case INITIAL:
			stats["initial"]++
		case WAITING:
			stats["waiting"]++
		case SCHEDULED:
			stats["scheduled"]++
		case READING:
			stats["reading"]++
		case COMPLETED:
			stats["completed"]++
		case FAILED:
			stats["failed"]++
		}
	}

	return stats
}
