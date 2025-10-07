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

// Package main demonstrates a read pattern that starts with random reads,
// switches to sequential reads after 7-8 random reads, and then returns to random reads.
// This example shows how the GCS reader behavior adapts to different access patterns.
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx/read_manager"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/timeutil"
	"golang.org/x/sync/semaphore"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
)

const (
	// File size constants
	fileSize         = 1024 * 1024 * 1024 // 1GB file
	readBufferSize   = 1024 * 1024        // 1MB read buffer
	randomReadCount  = 8                  // Number of random reads before switching to sequential
	sequentialReads  = 400                // Number of sequential reads
	finalRandomReads = 5                  // Number of random reads at the end
)

// ReadPatternExample demonstrates the mixed read pattern behavior
type ReadPatternExample struct {
	bucket      gcs.Bucket
	object      *gcs.MinObject
	readManager gcsx.ReadManager
	buffer      []byte
}

// NewReadPatternExample creates a new example with a fake GCS setup
func NewReadPatternExample() (*ReadPatternExample, error) {
	// Create a fake bucket for testing
	realClock := timeutil.RealClock()
	bucket := fake.NewFakeBucket(realClock, "test-bucket", gcs.BucketType{})
	bucket = storage.NewDebugBucket(bucket) // Wrap with debug bucket to log operations

	// Create test data
	testData := make([]byte, fileSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// Create object in the fake bucket
	objectName := "test-read-pattern-object"
	_, err := bucket.CreateObject(context.Background(), &gcs.CreateObjectRequest{
		Name:     objectName,
		Contents: bytes.NewReader(testData),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create test object: %w", err)
	}

	// Get the object metadata
	obj, _, err := bucket.StatObject(context.Background(), &gcs.StatObjectRequest{
		Name: objectName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	// Create configuration for the read manager
	config := &cfg.Config{
		Read: cfg.ReadConfig{
			EnableBufferedRead:   true,
			MaxBlocksPerHandle:   10,
			BlockSizeMb:          16,
			StartBlocksPerHandle: 2,
			MinBlocksPerHandle:   2,
			RandomSeekThreshold:  3, // Low threshold to demonstrate reader switching quickly
		},
	}

	// Create a worker pool
	workerPool, err := workerpool.NewStaticWorkerPoolForCurrentCPU(100)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker pool: %w", err)
	}

	// Create a metric handle (no-op for this example)
	metricHandle := metrics.NewNoopMetrics()

	// Create global semaphore for block management
	globalMaxBlocksSem := semaphore.NewWeighted(100)

	// Create read manager configuration
	rmConfig := &read_manager.ReadManagerConfig{
		MetricHandle:         metricHandle,
		SequentialReadSizeMB: 8,
		Config:               config,
		GlobalMaxBlocksSem:   globalMaxBlocksSem,
		WorkerPool:           workerPool,
	}

	// Create the read manager
	readManager := read_manager.NewReadManager(obj, bucket, rmConfig)

	return &ReadPatternExample{
		bucket:      bucket,
		object:      obj,
		readManager: readManager,
		buffer:      make([]byte, readBufferSize),
	}, nil
}

// performRandomRead performs a random read at the specified offset
func (e *ReadPatternExample) performRandomRead(ctx context.Context, offset int64, readNum int) error {
	response, err := e.readManager.ReadAt(ctx, e.buffer, offset)
	if err != nil {
		return fmt.Errorf("random read %d failed at offset %d: %w", readNum, offset, err)
	}

	fmt.Printf("Random Read %d: offset=%d, bytes_read=%d\n", readNum, offset, response.Size)
	return nil
}

// performSequentialRead performs sequential reads starting from the given offset
func (e *ReadPatternExample) performSequentialRead(ctx context.Context, startOffset int64, readNum int) (int64, error) {
	response, err := e.readManager.ReadAt(ctx, e.buffer, startOffset)
	if err != nil {
		return startOffset, fmt.Errorf("sequential read %d failed at offset %d: %w", readNum, startOffset, err)
	}

	fmt.Printf("Sequential Read %d: offset=%d, bytes_read=%d\n", readNum, startOffset, response.Size)
	return startOffset + int64(response.Size), nil
}

// RunExample executes the complete read pattern example
func (e *ReadPatternExample) RunExample() error {
	ctx := context.Background()
	rand.Seed(time.Now().UnixNano())

	fmt.Println("=== Starting Read Pattern Example ===")
	fmt.Printf("File size: %d bytes\n", e.object.Size)
	fmt.Printf("Read buffer size: %d bytes\n", readBufferSize)
	fmt.Println()

	// Phase 1: Random reads (7-8 reads)
	fmt.Println("Phase 1: Performing random reads...")
	for i := 1; i <= randomReadCount; i++ {
		// Generate random offset, ensuring we don't read beyond file size
		maxOffset := int64(e.object.Size) - int64(readBufferSize)
		if maxOffset <= 0 {
			maxOffset = 0
		}
		randomOffset := rand.Int63n(maxOffset + 1)

		if err := e.performRandomRead(ctx, randomOffset, i); err != nil {
			return err
		}

		// Small delay to simulate real-world access patterns
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Println()

	// Phase 2: Sequential reads
	fmt.Println("Phase 2: Switching to sequential reads...")
	sequentialOffset := int64(0) // Start from beginning for sequential reads

	for i := 1; i <= sequentialReads; i++ {
		nextOffset, err := e.performSequentialRead(ctx, sequentialOffset, i)
		if err != nil {
			return err
		}
		sequentialOffset = nextOffset

		// Check if we've reached the end of the file
		if sequentialOffset >= int64(e.object.Size) {
			fmt.Printf("Reached end of file at offset %d\n", sequentialOffset)
			break
		}

		// Small delay between sequential reads
		time.Sleep(5 * time.Millisecond)
	}

	fmt.Println()

	// Phase 3: Return to random reads
	fmt.Println("Phase 3: Returning to random reads...")
	for i := 1; i <= finalRandomReads; i++ {
		// Generate random offset
		maxOffset := int64(e.object.Size) - int64(readBufferSize)
		if maxOffset <= 0 {
			maxOffset = 0
		}
		randomOffset := rand.Int63n(maxOffset + 1)

		if err := e.performRandomRead(ctx, randomOffset, randomReadCount+sequentialReads+i); err != nil {
			return err
		}

		// Small delay
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Println()
	fmt.Println("=== Read Pattern Example Completed Successfully ===")

	return nil
}

// Close cleans up resources
func (e *ReadPatternExample) Close() error {
	if closer, ok := e.readManager.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

func main() {
	fmt.Println("GCS Read Pattern Example")
	fmt.Println("This example demonstrates:")
	fmt.Println("1. Random reads (7-8 times)")
	fmt.Println("2. Sequential reads (multiple)")
	fmt.Println("3. Random reads again")
	fmt.Println("Watch how the reader adapts to different access patterns.")
	fmt.Println()

	// Create the example
	example, err := NewReadPatternExample()
	if err != nil {
		log.Fatalf("Failed to create example: %v", err)
	}
	defer example.Close()

	// Run the example
	if err := example.RunExample(); err != nil {
		log.Fatalf("Example failed: %v", err)
	}

	fmt.Println("Example completed successfully!")
}
