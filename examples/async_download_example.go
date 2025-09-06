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

// Package block provides an example of using async download functionality
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
)

// Example: How to use the async download functionality with worker pools
func main() {
	// This example demonstrates how to use the async download system
	// Note: In real usage, you would have actual GCS bucket and worker pool instances

	fmt.Println("=== Async Download Functionality Example ===")
	fmt.Println()

	// 1. Create a download request
	request := &block.BlockDownloadRequest{
		Key:         block.CacheKey("example-block-1"),
		ObjectName:  "my-object.txt",
		Generation:  123456,
		StartOffset: 0,
		EndOffset:   1024,
		Priority:    false, // Set to true for high-priority downloads
		OnComplete: func(key block.CacheKey, state block.DownloadState, err error) {
			if err != nil {
				fmt.Printf("Download for %s failed: %v\n", key, err)
			} else {
				fmt.Printf("Download for %s completed successfully with state: %s\n", key, state)
			}
		},
	}

	fmt.Printf("Created download request:\n")
	fmt.Printf("  Key: %s\n", request.Key)
	fmt.Printf("  Object: %s\n", request.ObjectName)
	fmt.Printf("  Generation: %d\n", request.Generation)
	fmt.Printf("  Range: %d-%d bytes\n", request.StartOffset, request.EndOffset)
	fmt.Printf("  Priority: %t\n", request.Priority)
	fmt.Println()

	// 2. Example of how async download manager would be used
	fmt.Println("Async Download Manager Usage Pattern:")
	fmt.Println("1. AsyncDownloadManager schedules downloads using WorkerPool")
	fmt.Println("2. Downloads can be prioritized (priority vs normal queue)")
	fmt.Println("3. Download state is tracked throughout the process")
	fmt.Println("4. Completion callbacks notify when downloads finish")
	fmt.Println("5. Downloads can be cancelled if needed")
	fmt.Println("6. BlockCache integration provides seamless access")
	fmt.Println()

	// 3. Show download state progression
	fmt.Println("Download State Progression:")
	states := []block.DownloadState{
		block.DownloadStateNotStarted,
		block.DownloadStateInProgress,
		block.DownloadStateCompleted,
	}

	for i, state := range states {
		fmt.Printf("  %d. %s\n", i+1, state.String())
		time.Sleep(500 * time.Millisecond) // Simulate progression
	}
	fmt.Println()

	// 4. Example of priority vs normal downloads
	fmt.Println("Priority vs Normal Downloads:")
	
	normalRequest := &block.BlockDownloadRequest{
		Key:         block.CacheKey("normal-download"),
		ObjectName:  "normal-file.txt",
		Generation:  100,
		StartOffset: 0,
		EndOffset:   512,
		Priority:    false,
	}

	priorityRequest := &block.BlockDownloadRequest{
		Key:         block.CacheKey("priority-download"),
		ObjectName:  "urgent-file.txt", 
		Generation:  200,
		StartOffset: 0,
		EndOffset:   512,
		Priority:    true,
	}

	fmt.Printf("Normal Download: %s (Priority: %t)\n", normalRequest.Key, normalRequest.Priority)
	fmt.Printf("Priority Download: %s (Priority: %t)\n", priorityRequest.Key, priorityRequest.Priority)
	fmt.Println()

	// 5. Show how worker pool integration works
	fmt.Println("Worker Pool Integration:")
	fmt.Println("- AsyncDownloadManager uses WorkerPool.Schedule() to queue tasks")
	fmt.Println("- Priority downloads use urgent scheduling")
	fmt.Println("- Normal downloads use regular scheduling") 
	fmt.Println("- Multiple downloads can run concurrently")
	fmt.Println("- Tasks implement workerpool.Task interface")
	fmt.Println()

	// 6. Block cache integration example
	fmt.Println("Block Cache Integration:")
	fmt.Println("- GetOrScheduleDownload() checks cache first")
	fmt.Println("- If block exists: returns immediately")
	fmt.Println("- If block missing: schedules async download and returns task")
	fmt.Println("- Caller can wait for download or proceed asynchronously")
	fmt.Println("- Downloaded blocks are stored in cache for future access")
	fmt.Println()

	// 7. Error handling and cancellation
	fmt.Println("Error Handling & Cancellation:")
	fmt.Println("- Downloads can fail due to network issues, GCS errors, etc.")
	fmt.Println("- Failed downloads call OnComplete callback with error")
	fmt.Println("- Downloads can be cancelled using CancelDownload(key)")
	fmt.Println("- Context cancellation propagates to ongoing downloads")
	fmt.Println("- Cleanup functions remove completed/failed downloads")
	fmt.Println()

	fmt.Println("=== Implementation Details ===")
	fmt.Println()
	fmt.Println("Key Components:")
	fmt.Println("1. AsyncBlockDownloadTask - Individual download task")
	fmt.Println("2. AsyncDownloadManager - Manages multiple downloads")
	fmt.Println("3. BlockDownloadRequest - Download specification")
	fmt.Println("4. DownloadState - State tracking (NotStarted, InProgress, etc.)")
	fmt.Println("5. Worker Pool Integration - Concurrent execution")
	fmt.Println("6. Block Cache Integration - Seamless caching")
	fmt.Println()

	fmt.Println("Benefits:")
	fmt.Println("✓ Non-blocking downloads improve performance")
	fmt.Println("✓ Priority queuing for urgent requests")
	fmt.Println("✓ Concurrent downloads using worker pools")
	fmt.Println("✓ Automatic caching of downloaded blocks")
	fmt.Println("✓ Robust error handling and cancellation")
	fmt.Println("✓ State tracking and progress monitoring")
	fmt.Println("✓ Callback-based completion notification")
	fmt.Println()

	log.Println("Example completed successfully!")
}

// Additional helper functions that would be used in practice:

// validateAndPrepareRequest ensures the download request is valid
func validateAndPrepareRequest(req *block.BlockDownloadRequest) error {
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

// monitorDownloadProgress shows how to monitor an ongoing download
func monitorDownloadProgress(ctx context.Context, task interface{ GetStatus() block.DownloadStatus }) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Monitoring cancelled")
			return
		case <-ticker.C:
			status := task.GetStatus()
			switch status.State {
			case block.DownloadStateCompleted:
				fmt.Printf("Download completed successfully!\n")
				return
			case block.DownloadStateFailed:
				fmt.Printf("Download failed: %v\n", status.Error)
				return
			case block.DownloadStateCancelled:
				fmt.Printf("Download was cancelled\n")
				return
			case block.DownloadStateInProgress:
				fmt.Printf("Download in progress... (started %v ago)\n", time.Since(status.StartTime))
			}
		}
	}
}

// Example usage pattern for real applications:
/*
func useAsyncDownloads(ctx context.Context, cache *block.BlockCache) error {
	// 1. Create download request
	request := &block.BlockDownloadRequest{
		Key:         block.CacheKey("my-block"),
		ObjectName:  "path/to/object",
		Generation:  gen,
		StartOffset: offset,
		EndOffset:   offset + blockSize,
		Priority:    false,
		OnComplete: func(key block.CacheKey, state block.DownloadState, err error) {
			// Handle completion
			if err != nil {
				log.Printf("Download failed: %v", err)
			} else {
				log.Printf("Download completed: %s", key)
			}
		},
	}

	// 2. Try to get from cache or schedule download
	block, task, err := cache.GetOrScheduleDownload(ctx, request)
	if err != nil {
		return err
	}

	if block != nil {
		// Block was in cache, use immediately
		defer cache.Release(block)
		// Use block data...
		return nil
	}

	if task != nil {
		// Block not in cache, download was scheduled
		// Option 1: Wait for download
		for {
			status := task.GetStatus()
			if status.State == block.DownloadStateCompleted {
				// Try to get block again
				block, _, err := cache.GetOrScheduleDownload(ctx, request)
				if err == nil && block != nil {
					defer cache.Release(block)
					// Use block data...
					return nil
				}
			} else if status.State == block.DownloadStateFailed {
				return status.Error
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	return nil
}
*/
