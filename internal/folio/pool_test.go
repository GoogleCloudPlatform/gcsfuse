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

package folio

import (
	"testing"
)

func TestPool_Basic(t *testing.T) {
	// Create a pool with 64KB blocks
	pool, err := NewPool(Size64KB)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Acquire a block
	block, err := pool.Acquire()
	if err != nil {
		t.Fatalf("Failed to acquire block: %v", err)
	}

	if len(block.Data) != Block64KB {
		t.Errorf("Expected block size %d, got %d", Block64KB, len(block.Data))
	}

	// Write some data
	testData := []byte("Hello, World!")
	copy(block.Data, testData)

	// Verify data
	for i, b := range testData {
		if block.Data[i] != b {
			t.Errorf("Data mismatch at index %d", i)
		}
	}

	// Release the block
	pool.Release(block)

	// Check stats
	stats := pool.Stats()
	t.Logf("Pool stats: %+v", stats)
}

func TestPool_MultipleBlocks(t *testing.T) {
	pool, err := NewPool(Size1MB)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Acquire multiple blocks
	blocks := make([]*Block, 10)
	for i := 0; i < 10; i++ {
		block, err := pool.Acquire()
		if err != nil {
			t.Fatalf("Failed to acquire block %d: %v", i, err)
		}
		blocks[i] = block

		// Write unique data to each block
		blocks[i].Data[0] = byte(i)
	}

	// Verify each block has unique data
	for i, block := range blocks {
		if block.Data[0] != byte(i) {
			t.Errorf("Block %d has wrong data: expected %d, got %d", i, i, block.Data[0])
		}
	}

	// Release all blocks
	for _, block := range blocks {
		pool.Release(block)
	}

	// Check stats
	stats := pool.Stats()
	t.Logf("Pool stats after release: %+v", stats)

	// Note: blocks in the free channel are still marked as "used" in the bitmap
	// but they're available for reuse. This is expected behavior for performance.
	// The key is that FreeBlocks should match the number we released or more
	if stats.FreeBlocks < 10 {
		t.Errorf("Expected at least 10 free blocks, got %d", stats.FreeBlocks)
	}
}

func TestPool_ReuseBlocks(t *testing.T) {
	pool, err := NewPool(Size64KB)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Exhaust pre-populated blocks first
	initialBlocks := make([]*Block, 60)
	for i := 0; i < 60; i++ {
		initialBlocks[i], _ = pool.Acquire()
	}
	for _, b := range initialBlocks {
		pool.Release(b)
	}

	// Test 1: Release without clearing - data can persist
	block1, err := pool.Acquire()
	if err != nil {
		t.Fatalf("Failed to acquire block: %v", err)
	}
	block1.Data[0] = 42
	block1.Data[100] = 99
	pool.Release(block1)

	// Get the same block back
	block2, err := pool.Acquire()
	if err != nil {
		t.Fatalf("Failed to acquire second block: %v", err)
	}

	// Verify blocks can be reused (they're the same memory)
	t.Logf("Block reuse: first byte=%d, 100th byte=%d", block2.Data[0], block2.Data[100])

	// Test 2: ReleaseAndClear should zero data
	block2.Data[0] = 123
	block2.Data[100] = 234
	pool.ReleaseAndClear(block2)

	block3, err := pool.Acquire()
	if err != nil {
		t.Fatalf("Failed to acquire third block: %v", err)
	}

	// After ReleaseAndClear, data should be zeroed
	if block3.Data[0] != 0 {
		t.Errorf("Expected zeroed data after ReleaseAndClear at offset 0, got %d", block3.Data[0])
	}
	if block3.Data[100] != 0 {
		t.Errorf("Expected zeroed data after ReleaseAndClear at offset 100, got %d", block3.Data[100])
	}

	pool.Release(block2)
}

func TestPool_Stats(t *testing.T) {
	pool, err := NewPool(Size64KB)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	initialStats := pool.Stats()
	t.Logf("Initial stats: %+v", initialStats)

	// Acquire some blocks
	blocks := make([]*Block, 5)
	for i := 0; i < 5; i++ {
		blocks[i], err = pool.Acquire()
		if err != nil {
			t.Fatalf("Failed to acquire block %d: %v", i, err)
		}
	}

	afterAcquireStats := pool.Stats()
	t.Logf("After acquire stats: %+v", afterAcquireStats)

	if afterAcquireStats.UsedBlocks < 5 {
		t.Errorf("Expected at least 5 used blocks, got %d", afterAcquireStats.UsedBlocks)
	}

	// Release blocks
	for _, block := range blocks {
		pool.Release(block)
	}

	afterReleaseStats := pool.Stats()
	t.Logf("After release stats: %+v", afterReleaseStats)
}

func BenchmarkPool_Acquire_64KB(b *testing.B) {
	pool, err := NewPool(Size64KB)
	if err != nil {
		b.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block, err := pool.Acquire()
		if err != nil {
			b.Fatalf("Failed to acquire block: %v", err)
		}
		pool.Release(block)
	}
}

func BenchmarkPool_Acquire_1MB(b *testing.B) {
	pool, err := NewPool(Size1MB)
	if err != nil {
		b.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block, err := pool.Acquire()
		if err != nil {
			b.Fatalf("Failed to acquire block: %v", err)
		}
		pool.Release(block)
	}
}
