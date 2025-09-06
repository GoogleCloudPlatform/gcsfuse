// Copyright 2024 Google LLC
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

package block

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// DownloadState represents the state of an async download operation
type DownloadState int

const (
	DownloadStateNotStarted DownloadState = iota
	DownloadStateInProgress
	DownloadStateCompleted
	DownloadStateFailed
	DownloadStateCancelled
)

// String returns the string representation of DownloadState
func (ds DownloadState) String() string {
	switch ds {
	case DownloadStateNotStarted:
		return "NotStarted"
	case DownloadStateInProgress:
		return "InProgress"
	case DownloadStateCompleted:
		return "Completed"
	case DownloadStateFailed:
		return "Failed"
	case DownloadStateCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// BlockDownloadRequest represents a request to download data for a block
type BlockDownloadRequest struct {
	// Block metadata
	Key         CacheKey
	ObjectName  string
	Generation  int64
	StartOffset int64
	EndOffset   int64

	// Download context
	Priority bool // If true, schedule as urgent task

	// Completion callback (optional)
	OnComplete func(key CacheKey, state DownloadState, err error)
}

// DownloadStatus represents the current status of a download operation
type DownloadStatus struct {
	State      DownloadState
	Error      error
	StartTime  time.Time
	FinishTime time.Time
	BytesRead  int64
}

// AsyncBlockDownloadTask implements workerpool.Task for downloading data to blocks
type AsyncBlockDownloadTask struct {
	workerpool.Task

	request      *BlockDownloadRequest
	ctx          context.Context
	cancel       context.CancelFunc
	bucket       gcs.Bucket
	blockCache   *BlockCache
	metricHandle metrics.MetricHandle

	// Track download status
	statusMu sync.RWMutex
	status   DownloadStatus
}

// NewAsyncBlockDownloadTask creates a new async block download task
func NewAsyncBlockDownloadTask(
	ctx context.Context,
	request *BlockDownloadRequest,
	bucket gcs.Bucket,
	blockCache *BlockCache,
	metricHandle metrics.MetricHandle,
) *AsyncBlockDownloadTask {
	taskCtx, cancel := context.WithCancel(ctx)
	
	return &AsyncBlockDownloadTask{
		request:      request,
		ctx:          taskCtx,
		cancel:       cancel,
		bucket:       bucket,
		blockCache:   blockCache,
		metricHandle: metricHandle,
		status: DownloadStatus{
			State:     DownloadStateNotStarted,
			StartTime: time.Now(),
		},
	}
}

// GetStatus returns the current download status (thread-safe)
func (t *AsyncBlockDownloadTask) GetStatus() DownloadStatus {
	t.statusMu.RLock()
	defer t.statusMu.RUnlock()
	return t.status
}

// updateStatus updates the download status (thread-safe)
func (t *AsyncBlockDownloadTask) updateStatus(state DownloadState, err error, bytesRead int64) {
	t.statusMu.Lock()
	defer t.statusMu.Unlock()
	
	t.status.State = state
	t.status.Error = err
	t.status.BytesRead = bytesRead
	
	if state == DownloadStateCompleted || state == DownloadStateFailed || state == DownloadStateCancelled {
		t.status.FinishTime = time.Now()
	}
}

// Cancel cancels the download task
func (t *AsyncBlockDownloadTask) Cancel() {
	t.updateStatus(DownloadStateCancelled, context.Canceled, 0)
	t.cancel()
}

// Execute implements the workerpool.Task interface
func (t *AsyncBlockDownloadTask) Execute() {
	logger.Tracef("AsyncDownload: Starting download for block %s (object: %s, offset: %d-%d)",
		t.request.Key, t.request.ObjectName, t.request.StartOffset, t.request.EndOffset)

	t.updateStatus(DownloadStateInProgress, nil, 0)
	
	startTime := time.Now()
	var downloadErr error
	var bytesRead int64
	
	defer func() {
		duration := time.Since(startTime)
		
		// Determine final state
		var finalState DownloadState
		var status string
		
		if t.ctx.Err() == context.Canceled {
			finalState = DownloadStateCancelled
			status = "cancelled"
		} else if downloadErr != nil {
			finalState = DownloadStateFailed
			status = "failed"
		} else {
			finalState = DownloadStateCompleted
			status = "successful"
		}
		
		t.updateStatus(finalState, downloadErr, bytesRead)
		
		// Log completion
		logger.Tracef("AsyncDownload: Finished download for block %s (%s) in %v, bytes: %d",
			t.request.Key, status, duration, bytesRead)
		
		// Record metrics
		if t.metricHandle != nil {
			t.metricHandle.BufferedReadDownloadBlockLatency(t.ctx, duration, status)
		}
		
		// Call completion callback if provided
		if t.request.OnComplete != nil {
			t.request.OnComplete(t.request.Key, finalState, downloadErr)
		}
	}()

	// Get or create the block from cache
	block, err := t.blockCache.Get(t.request.Key)
	if err != nil {
		downloadErr = fmt.Errorf("failed to get block from cache: %w", err)
		return
	}
	
	// Always release the block when done
	defer t.blockCache.Release(block)

	// Check if context was cancelled before starting download
	if t.ctx.Err() != nil {
		downloadErr = t.ctx.Err()
		return
	}

	// Create GCS reader for the specified range
	reader, err := t.bucket.NewReaderWithReadHandle(t.ctx, &gcs.ReadObjectRequest{
		Name:       t.request.ObjectName,
		Generation: t.request.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(t.request.StartOffset),
			Limit: uint64(t.request.EndOffset),
		},
	})
	if err != nil {
		downloadErr = fmt.Errorf("failed to create GCS reader: %w", err)
		return
	}
	defer reader.Close()

	// Download data to the block
	written, err := io.Copy(block, reader)
	if err != nil {
		downloadErr = fmt.Errorf("failed to copy data to block: %w", err)
		return
	}
	
	bytesRead = written
	logger.Debugf("AsyncDownload: Successfully downloaded %d bytes for block %s", bytesRead, t.request.Key)
}

// AsyncDownloadManager manages async download operations for the block cache
type AsyncDownloadManager struct {
	mu           sync.RWMutex
	workerPool   workerpool.WorkerPool
	bucket       gcs.Bucket
	blockCache   *BlockCache
	metricHandle metrics.MetricHandle
	
	// Track active downloads by cache key
	activeDownloads map[CacheKey]*AsyncBlockDownloadTask
}

// NewAsyncDownloadManager creates a new async download manager
func NewAsyncDownloadManager(
	workerPool workerpool.WorkerPool,
	bucket gcs.Bucket,
	blockCache *BlockCache,
	metricHandle metrics.MetricHandle,
) *AsyncDownloadManager {
	return &AsyncDownloadManager{
		workerPool:      workerPool,
		bucket:         bucket,
		blockCache:     blockCache,
		metricHandle:   metricHandle,
		activeDownloads: make(map[CacheKey]*AsyncBlockDownloadTask),
	}
}

// ScheduleDownload schedules an asynchronous download for a block
func (m *AsyncDownloadManager) ScheduleDownload(ctx context.Context, request *BlockDownloadRequest) (*AsyncBlockDownloadTask, error) {
	if request == nil {
		return nil, fmt.Errorf("download request cannot be nil")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if there's already an active download for this key
	if existingTask, exists := m.activeDownloads[request.Key]; exists {
		status := existingTask.GetStatus()
		if status.State == DownloadStateInProgress {
			logger.Debugf("Download already in progress for block %s", request.Key)
			return existingTask, nil
		}
		// Clean up completed/failed downloads
		delete(m.activeDownloads, request.Key)
	}
	
	// Create new download task
	task := NewAsyncBlockDownloadTask(ctx, request, m.bucket, m.blockCache, m.metricHandle)
	
	// Track the active download
	m.activeDownloads[request.Key] = task
	
	// Schedule the task with the worker pool
	m.workerPool.Schedule(request.Priority, task)
	
	logger.Debugf("Scheduled async download for block %s (priority: %t)", request.Key, request.Priority)
	return task, nil
}

// CancelDownload cancels an active download
func (m *AsyncDownloadManager) CancelDownload(key CacheKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	task, exists := m.activeDownloads[key]
	if !exists {
		return fmt.Errorf("no active download found for key %s", key)
	}
	
	task.Cancel()
	delete(m.activeDownloads, key)
	
	logger.Debugf("Cancelled download for block %s", key)
	return nil
}

// GetDownloadStatus returns the status of a download operation
func (m *AsyncDownloadManager) GetDownloadStatus(key CacheKey) (*DownloadStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	task, exists := m.activeDownloads[key]
	if !exists {
		return nil, fmt.Errorf("no download found for key %s", key)
	}
	
	status := task.GetStatus()
	return &status, nil
}

// ListActiveDownloads returns a list of all active download keys
func (m *AsyncDownloadManager) ListActiveDownloads() []CacheKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	keys := make([]CacheKey, 0, len(m.activeDownloads))
	for key := range m.activeDownloads {
		keys = append(keys, key)
	}
	return keys
}

// CleanupCompletedDownloads removes completed/failed downloads from tracking
func (m *AsyncDownloadManager) CleanupCompletedDownloads() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	cleaned := 0
	for key, task := range m.activeDownloads {
		status := task.GetStatus()
		if status.State == DownloadStateCompleted || 
		   status.State == DownloadStateFailed || 
		   status.State == DownloadStateCancelled {
			delete(m.activeDownloads, key)
			cleaned++
		}
	}
	
	if cleaned > 0 {
		logger.Debugf("Cleaned up %d completed downloads", cleaned)
	}
	return cleaned
}

// Shutdown cancels all active downloads and cleans up
func (m *AsyncDownloadManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for key, task := range m.activeDownloads {
		task.Cancel()
		logger.Debugf("Cancelled download for block %s during shutdown", key)
	}
	
	// Clear all downloads
	m.activeDownloads = make(map[CacheKey]*AsyncBlockDownloadTask)
}
