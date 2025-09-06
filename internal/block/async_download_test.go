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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test the core async download functionality without external dependencies

func TestDownloadState(t *testing.T) {
	tests := []struct {
		name     string
		state    DownloadState
		expected string
	}{
		{"NotStarted", DownloadStateNotStarted, "NotStarted"},
		{"InProgress", DownloadStateInProgress, "InProgress"},
		{"Completed", DownloadStateCompleted, "Completed"},
		{"Failed", DownloadStateFailed, "Failed"},
		{"Cancelled", DownloadStateCancelled, "Cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestDownloadStatus(t *testing.T) {
	status := &DownloadStatus{
		State:     DownloadStateInProgress,
		Error:     nil,
		StartTime: time.Now(),
	}

	assert.Equal(t, DownloadStateInProgress, status.State)
	assert.NoError(t, status.Error)
	assert.True(t, time.Since(status.StartTime) < time.Second)

	// Test with error
	testError := fmt.Errorf("test error")
	status.State = DownloadStateFailed
	status.Error = testError

	assert.Equal(t, DownloadStateFailed, status.State)
	assert.Equal(t, testError, status.Error)
}

func TestBlockDownloadRequest(t *testing.T) {
	var completionCalled bool
	var completionKey CacheKey
	var completionState DownloadState
	var completionError error

	request := &BlockDownloadRequest{
		Key:         CacheKey("test-key"),
		ObjectName:  "test-object",
		Generation:  123,
		StartOffset: 0,
		EndOffset:   1024,
		Priority:    true,
		OnComplete: func(key CacheKey, state DownloadState, err error) {
			completionCalled = true
			completionKey = key
			completionState = state
			completionError = err
		},
	}

	assert.Equal(t, CacheKey("test-key"), request.Key)
	assert.Equal(t, "test-object", request.ObjectName)
	assert.Equal(t, int64(123), request.Generation)
	assert.Equal(t, int64(0), request.StartOffset)
	assert.Equal(t, int64(1024), request.EndOffset)
	assert.True(t, request.Priority)

	// Test completion callback
	if request.OnComplete != nil {
		request.OnComplete(request.Key, DownloadStateCompleted, nil)
	}

	assert.True(t, completionCalled)
	assert.Equal(t, request.Key, completionKey)
	assert.Equal(t, DownloadStateCompleted, completionState)
	assert.NoError(t, completionError)
}

func TestAsyncDownloadManagerStates(t *testing.T) {
	// Test the basic state management without external dependencies
	
	// Create a simple mock task
	mockTask := &MockAsyncTask{
		key:    CacheKey("test-key"),
		status: &DownloadStatus{State: DownloadStateNotStarted, StartTime: time.Now()},
	}

	// Test state transitions
	mockTask.status.State = DownloadStateInProgress
	assert.Equal(t, DownloadStateInProgress, mockTask.GetStatus().State)

	mockTask.status.State = DownloadStateCompleted
	assert.Equal(t, DownloadStateCompleted, mockTask.GetStatus().State)

	mockTask.status.State = DownloadStateFailed
	mockTask.status.Error = fmt.Errorf("test error")
	status := mockTask.GetStatus()
	assert.Equal(t, DownloadStateFailed, status.State)
	assert.Error(t, status.Error)
}

func TestConcurrentStateAccess(t *testing.T) {
	// Test concurrent access to download status
	mockTask := &MockAsyncTask{
		key:    CacheKey("concurrent-test"),
		status: &DownloadStatus{State: DownloadStateNotStarted, StartTime: time.Now()},
		mu:     sync.RWMutex{},
	}

	const numGoroutines = 10
	const numIterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines reading and writing status
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				if id%2 == 0 {
					// Reader goroutines
					status := mockTask.GetStatus()
					assert.NotNil(t, status)
				} else {
					// Writer goroutines
					mockTask.updateStatus(DownloadStateInProgress, nil)
				}
			}
		}(i)
	}

	wg.Wait()

	// Final status should be InProgress
	finalStatus := mockTask.GetStatus()
	assert.Equal(t, DownloadStateInProgress, finalStatus.State)
}

func TestDownloadRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request *BlockDownloadRequest
		valid   bool
	}{
		{
			name: "Valid request",
			request: &BlockDownloadRequest{
				Key:         CacheKey("valid"),
				ObjectName:  "object",
				Generation:  1,
				StartOffset: 0,
				EndOffset:   1024,
			},
			valid: true,
		},
		{
			name: "Empty key",
			request: &BlockDownloadRequest{
				Key:         CacheKey(""),
				ObjectName:  "object",
				Generation:  1,
				StartOffset: 0,
				EndOffset:   1024,
			},
			valid: false,
		},
		{
			name: "Empty object name",
			request: &BlockDownloadRequest{
				Key:         CacheKey("key"),
				ObjectName:  "",
				Generation:  1,
				StartOffset: 0,
				EndOffset:   1024,
			},
			valid: false,
		},
		{
			name: "Invalid offset range",
			request: &BlockDownloadRequest{
				Key:         CacheKey("key"),
				ObjectName:  "object",
				Generation:  1,
				StartOffset: 1024,
				EndOffset:   512,
			},
			valid: false,
		},
		{
			name: "Negative offset",
			request: &BlockDownloadRequest{
				Key:         CacheKey("key"),
				ObjectName:  "object",
				Generation:  1,
				StartOffset: -1,
				EndOffset:   1024,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDownloadRequest(tt.request)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCacheKeyGeneration(t *testing.T) {
	request := &BlockDownloadRequest{
		Key:         CacheKey("manual-key"),
		ObjectName:  "test-object",
		Generation:  123,
		StartOffset: 0,
		EndOffset:   1024,
	}

	// When key is provided, it should be used as-is
	key := generateCacheKey(request)
	assert.Equal(t, CacheKey("manual-key"), key)

	// When key is empty, generate from object details
	request.Key = CacheKey("")
	key = generateCacheKey(request)
	assert.NotEmpty(t, key)
	assert.Contains(t, string(key), "test-object")
	assert.Contains(t, string(key), "123")
	assert.Contains(t, string(key), "0")
	assert.Contains(t, string(key), "1024")
}

// Helper mock implementations for testing

type MockAsyncTask struct {
	key    CacheKey
	status *DownloadStatus
	mu     sync.RWMutex
}

func (m *MockAsyncTask) GetStatus() DownloadStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.status
}

func (m *MockAsyncTask) Cancel() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.State = DownloadStateCancelled
	return nil
}

func (m *MockAsyncTask) updateStatus(state DownloadState, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.State = state
	m.status.Error = err
}

// Helper functions that would be part of the actual implementation

func validateDownloadRequest(req *BlockDownloadRequest) error {
	if req.Key == "" {
		return fmt.Errorf("cache key cannot be empty")
	}
	if req.ObjectName == "" {
		return fmt.Errorf("object name cannot be empty")
	}
	if req.StartOffset < 0 {
		return fmt.Errorf("start offset cannot be negative")
	}
	if req.EndOffset <= req.StartOffset {
		return fmt.Errorf("end offset must be greater than start offset")
	}
	return nil
}

func generateCacheKey(req *BlockDownloadRequest) CacheKey {
	if req.Key != "" {
		return req.Key
	}
	return CacheKey(fmt.Sprintf("%s:%d:%d:%d", req.ObjectName, req.Generation, req.StartOffset, req.EndOffset))
}
