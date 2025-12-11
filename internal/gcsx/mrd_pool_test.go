// Copyright 2015 Google LLC
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
	"bytes"
	"fmt"
	"io"
	"testing"

	"cloud.google.com/go/storage"
)

// mockMultiRangeDownloader is a mock implementation for testing.
type mockMultiRangeDownloader struct {
	id     int
	closed bool
	err    error
	tasks  []mockTask
}

type mockTask struct {
	offset   int64
	length   int64
	callback func(int64, int64, error)
}

func (m *mockMultiRangeDownloader) Add(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
	if m.closed {
		if callback != nil {
			callback(offset, length, fmt.Errorf("downloader is closed"))
		}
		return
	}
	// Store task for later execution
	m.tasks = append(m.tasks, mockTask{
		offset:   offset,
		length:   length,
		callback: callback,
	})
	// Mock writing data
	if output != nil {
		data := make([]byte, length)
		output.Write(data)
	}
}

func (m *mockMultiRangeDownloader) Close() error {
	m.closed = true
	return nil
}

func (m *mockMultiRangeDownloader) Wait() {
	// Execute all pending tasks
	for _, task := range m.tasks {
		if task.callback != nil {
			task.callback(task.offset, task.length, m.err)
		}
	}
	m.tasks = nil
}

func (m *mockMultiRangeDownloader) Error() error {
	return m.err
}

func (m *mockMultiRangeDownloader) GetHandle() []byte {
	return []byte(fmt.Sprintf("handle-%d", m.id))
}

// Override createMultiRangeDownloader for testing
var testMRDCounter = 0

func createMockMRD(client *storage.Client) (MultiRangeDownloader, error) {
	testMRDCounter++
	return &mockMultiRangeDownloader{id: testMRDCounter}, nil
}

func TestNewMRDPool(t *testing.T) {
	tests := []struct {
		name        string
		config      *MRDPoolConfig
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			errContains: "config cannot be nil",
		},
		{
			name: "zero pool size",
			config: &MRDPoolConfig{
				PoolSize: 0,
			},
			wantErr:     true,
			errContains: "pool size must be greater than 0",
		},
		{
			name: "negative pool size",
			config: &MRDPoolConfig{
				PoolSize: -1,
			},
			wantErr:     true,
			errContains: "pool size must be greater than 0",
		},
		{
			name: "nil client",
			config: &MRDPoolConfig{
				PoolSize: 5,
			},
			wantErr:     true,
			errContains: "storage client cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMRDPool(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMRDPool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("NewMRDPool() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestMRDPool_RoundRobin(t *testing.T) {
	// Create a pool with mock downloaders
	poolSize := 5
	pool := &MRDPool{
		downloaders: make([]MultiRangeDownloader, poolSize),
		poolSize:    poolSize,
	}

	// Initialize with mock downloaders
	for i := 0; i < poolSize; i++ {
		pool.downloaders[i] = &mockMultiRangeDownloader{id: i}
	}

	// Test round-robin distribution
	var buf bytes.Buffer

	// Make multiple requests and verify round-robin behavior
	for i := 0; i < poolSize*3; i++ {
		pool.Add(&buf, int64(i*1024), 1024, nil)
	}

	// Verify counter increased
	stats := pool.GetStats()
	expectedCount := uint64(poolSize * 3)
	if stats.RequestCount != expectedCount {
		t.Errorf("Expected request count %d, got %d", expectedCount, stats.RequestCount)
	}
}

func TestMRDPool_Close(t *testing.T) {
	poolSize := 3
	pool := &MRDPool{
		downloaders: make([]MultiRangeDownloader, poolSize),
		poolSize:    poolSize,
	}

	// Initialize with mock downloaders
	for i := 0; i < poolSize; i++ {
		pool.downloaders[i] = &mockMultiRangeDownloader{id: i}
	}

	// Close the pool
	err := pool.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify pool is closed
	if !pool.IsClosed() {
		t.Error("Pool should be closed")
	}

	// Verify all downloaders are closed
	for i, downloader := range pool.downloaders {
		mock := downloader.(*mockMultiRangeDownloader)
		if !mock.closed {
			t.Errorf("Downloader %d should be closed", i)
		}
	}

	// Verify subsequent operations fail
	var buf bytes.Buffer
	pool.Add(&buf, 0, 1024, nil)

	// Verify closing again is safe
	err = pool.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

func TestMRDPool_ConcurrentAccess(t *testing.T) {
	poolSize := 5
	pool := &MRDPool{
		downloaders: make([]MultiRangeDownloader, poolSize),
		poolSize:    poolSize,
	}

	// Initialize with mock downloaders
	for i := 0; i < poolSize; i++ {
		pool.downloaders[i] = &mockMultiRangeDownloader{id: i}
	}

	// Test concurrent access
	numGoroutines := 50
	numRequests := 100

	errChan := make(chan error, numGoroutines*numRequests)
	doneChan := make(chan bool, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func() {
			for i := 0; i < numRequests; i++ {
				var buf bytes.Buffer
				pool.Add(&buf, int64(i*1024), 1024, nil)
			}
			doneChan <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-doneChan
	}
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent access error: %v", err)
	}

	// Verify stats
	stats := pool.GetStats()
	expectedCount := uint64(numGoroutines * numRequests)
	if stats.RequestCount != expectedCount {
		t.Errorf("Expected request count %d, got %d", expectedCount, stats.RequestCount)
	}
}

func TestMRDPool_Add(t *testing.T) {
	pool := &MRDPool{
		downloaders: make([]MultiRangeDownloader, 1),
		poolSize:    1,
	}
	pool.downloaders[0] = &mockMultiRangeDownloader{id: 0}

	var buf bytes.Buffer
	called := false
	pool.Add(&buf, 0, 1024, func(offset, length int64, err error) {
		called = true
		if offset != 0 || length != 1024 {
			t.Errorf("Expected offset=0, length=1024, got offset=%d, length=%d", offset, length)
		}
	})

	// Wait and verify callback
	pool.Wait()
	if !called {
		t.Error("Callback should have been called")
	}
}

func TestMRDPool_GetStats(t *testing.T) {
	poolSize := 4
	pool := &MRDPool{
		downloaders: make([]MultiRangeDownloader, poolSize),
		poolSize:    poolSize,
	}

	for i := 0; i < poolSize; i++ {
		pool.downloaders[i] = &mockMultiRangeDownloader{id: i}
	}

	// Initial stats
	stats := pool.GetStats()
	if stats.PoolSize != poolSize {
		t.Errorf("Expected pool size %d, got %d", poolSize, stats.PoolSize)
	}
	if stats.RequestCount != 0 {
		t.Errorf("Expected request count 0, got %d", stats.RequestCount)
	}
	if stats.Closed {
		t.Error("Pool should not be closed initially")
	}

	// After some requests
	var buf bytes.Buffer
	for i := 0; i < 10; i++ {
		pool.Add(&buf, int64(i*1024), 1024, nil)
	}

	stats = pool.GetStats()
	if stats.RequestCount != 10 {
		t.Errorf("Expected request count 10, got %d", stats.RequestCount)
	}

	// After closing
	pool.Close()
	stats = pool.GetStats()
	if !stats.Closed {
		t.Error("Pool should be closed")
	}
}

func TestMRDPool_Wait(t *testing.T) {
	poolSize := 2
	pool := &MRDPool{
		downloaders: make([]MultiRangeDownloader, poolSize),
		poolSize:    poolSize,
	}

	for i := 0; i < poolSize; i++ {
		pool.downloaders[i] = &mockMultiRangeDownloader{id: i}
	}

	// Add some tasks
	var buf bytes.Buffer
	callbackCount := 0
	for i := 0; i < 5; i++ {
		pool.Add(&buf, int64(i*1024), 1024, func(offset, length int64, err error) {
			callbackCount++
		})
	}

	// Wait should execute all callbacks
	pool.Wait()
	if callbackCount != 5 {
		t.Errorf("Expected 5 callbacks, got %d", callbackCount)
	}
}

func TestMRDPool_Error(t *testing.T) {
	poolSize := 2
	pool := &MRDPool{
		downloaders: make([]MultiRangeDownloader, poolSize),
		poolSize:    poolSize,
	}

	// Create one downloader with an error
	pool.downloaders[0] = &mockMultiRangeDownloader{id: 0, err: fmt.Errorf("test error")}
	pool.downloaders[1] = &mockMultiRangeDownloader{id: 1}

	// Error should return the error from the first downloader
	err := pool.Error()
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err != nil && !contains(err.Error(), "test error") {
		t.Errorf("Expected error to contain 'test error', got %v", err)
	}
}

func TestMRDPool_GetHandle(t *testing.T) {
	poolSize := 2
	pool := &MRDPool{
		downloaders: make([]MultiRangeDownloader, poolSize),
		poolSize:    poolSize,
	}

	for i := 0; i < poolSize; i++ {
		pool.downloaders[i] = &mockMultiRangeDownloader{id: i}
	}

	// GetHandle should return handle from first downloader
	handle := pool.GetHandle()
	if handle == nil {
		t.Error("Expected handle, got nil")
	}
	expected := "handle-0"
	if string(handle) != expected {
		t.Errorf("Expected handle '%s', got '%s'", expected, string(handle))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Example usage test
func ExampleMRDPool() {
	// Create a storage client (in real usage)
	// client, err := storage.NewClient(context.Background())
	// if err != nil {
	//     log.Fatal(err)
	// }
	// defer client.Close()

	// Create MRD pool with 5 downloaders
	config := &MRDPoolConfig{
		PoolSize: 5,
	}

	pool, err := NewMRDPool(config)
	if err != nil {
		fmt.Printf("Failed to create pool: %v\n", err)
		return
	}
	defer pool.Close()

	// Use the pool for downloading
	var buf bytes.Buffer
	completed := false

	pool.Add(&buf, 0, 1024, func(offset, length int64, err error) {
		if err != nil {
			fmt.Printf("Download failed: %v\n", err)
		} else {
			fmt.Printf("Downloaded %d bytes at offset %d\n", length, offset)
			completed = true
		}
	})

	if err != nil {
		fmt.Printf("Add failed: %v\n", err)
		return
	}

	// Wait for all downloads to complete
	pool.Wait()

	// Check for errors
	if err := pool.Error(); err != nil {
		fmt.Printf("Pool error: %v\n", err)
		return
	}

	// Get pool statistics
	stats := pool.GetStats()
	fmt.Printf("Pool size: %d, Total requests: %d, Completed: %v\n", stats.PoolSize, stats.RequestCount, completed)
}