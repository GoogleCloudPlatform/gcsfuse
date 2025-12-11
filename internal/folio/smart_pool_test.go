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

func TestSmartPool_Basic(t *testing.T) {
	sp, err := NewSmartPool()
	if err != nil {
		t.Fatalf("Failed to create smart pool: %v", err)
	}
	defer sp.Close()

	// Test small allocation (should use 64KB blocks)
	alloc, err := sp.Allocate(100 * 1024) // 100KB
	if err != nil {
		t.Fatalf("Failed to allocate 100KB: %v", err)
	}

	if alloc.TotalSize != 100*1024 {
		t.Errorf("Expected total size 102400, got %d", alloc.TotalSize)
	}

	// Should use 2 x 64KB blocks for 100KB
	if alloc.NumBlocks() != 2 {
		t.Errorf("Expected 2 blocks for 100KB, got %d", alloc.NumBlocks())
	}

	sp.Release(alloc)
}

func TestSmartPool_LargeAllocation(t *testing.T) {
	sp, err := NewSmartPool()
	if err != nil {
		t.Fatalf("Failed to create smart pool: %v", err)
	}
	defer sp.Close()

	// Test large allocation (should use 1MB blocks + 64KB blocks)
	size := 5*1024*1024 + 128*1024 // 5MB + 128KB
	alloc, err := sp.Allocate(size)
	if err != nil {
		t.Fatalf("Failed to allocate %d bytes: %v", size, err)
	}

	if alloc.TotalSize != size {
		t.Errorf("Expected total size %d, got %d", size, alloc.TotalSize)
	}

	// Should use 5 x 1MB blocks + 2 x 64KB blocks
	expectedBlocks := 5 + 2
	if alloc.NumBlocks() != expectedBlocks {
		t.Errorf("Expected %d blocks, got %d", expectedBlocks, alloc.NumBlocks())
	}

	sp.Release(alloc)
}

func TestSmartPool_AllocateExact(t *testing.T) {
	sp, err := NewSmartPool()
	if err != nil {
		t.Fatalf("Failed to create smart pool: %v", err)
	}
	defer sp.Close()

	// Test exact 1MB allocation
	alloc, err := sp.AllocateExact(3 * 1024 * 1024) // Exactly 3MB
	if err != nil {
		t.Fatalf("Failed to allocate exact 3MB: %v", err)
	}

	if alloc.NumBlocks() != 3 {
		t.Errorf("Expected 3 blocks for 3MB, got %d", alloc.NumBlocks())
	}

	sp.Release(alloc)

	// Test exact 64KB allocation
	alloc, err = sp.AllocateExact(5 * 64 * 1024) // Exactly 320KB
	if err != nil {
		t.Fatalf("Failed to allocate exact 320KB: %v", err)
	}

	if alloc.NumBlocks() != 5 {
		t.Errorf("Expected 5 blocks for 320KB, got %d", alloc.NumBlocks())
	}

	sp.Release(alloc)
}

func TestSmartPool_WriteRead(t *testing.T) {
	sp, err := NewSmartPool()
	if err != nil {
		t.Fatalf("Failed to create smart pool: %v", err)
	}
	defer sp.Close()

	// Allocate memory spanning multiple blocks
	size := 2*1024*1024 + 100*1024 // 2MB + 100KB
	alloc, err := sp.Allocate(size)
	if err != nil {
		t.Fatalf("Failed to allocate: %v", err)
	}
	defer sp.Release(alloc)

	// Create test data
	testData := make([]byte, size)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// Write to blocks
	if err := alloc.WriteToBlocks(testData); err != nil {
		t.Fatalf("Failed to write to blocks: %v", err)
	}

	// Read back from blocks
	readData := make([]byte, size)
	n, err := alloc.ReadFromBlocks(readData)
	if err != nil {
		t.Fatalf("Failed to read from blocks: %v", err)
	}

	if n != size {
		t.Errorf("Expected to read %d bytes, got %d", size, n)
	}

	// Verify data
	for i := 0; i < size; i++ {
		if readData[i] != testData[i] {
			t.Errorf("Data mismatch at offset %d: expected %d, got %d", i, testData[i], readData[i])
			break
		}
	}
}

func TestSmartPool_GetContiguousView(t *testing.T) {
	sp, err := NewSmartPool()
	if err != nil {
		t.Fatalf("Failed to create smart pool: %v", err)
	}
	defer sp.Close()

	size := 200 * 1024 // 200KB, will span multiple 64KB blocks
	alloc, err := sp.Allocate(size)
	if err != nil {
		t.Fatalf("Failed to allocate: %v", err)
	}
	defer sp.Release(alloc)

	// Write test pattern
	for i, block := range alloc.Blocks {
		pattern := byte(i + 1)
		for j := range block.Data {
			block.Data[j] = pattern
		}
	}

	// Get contiguous view
	view := alloc.GetContiguousView()
	if len(view) != size {
		t.Errorf("Expected view size %d, got %d", size, len(view))
	}

	// Verify pattern in contiguous view
	for i := 0; i < size; i++ {
		blockIdx := i / Block64KB
		expectedPattern := byte(blockIdx + 1)
		if view[i] != expectedPattern {
			t.Errorf("Pattern mismatch at offset %d: expected %d, got %d", i, expectedPattern, view[i])
			break
		}
	}
}

func TestSmartPool_MultipleAllocations(t *testing.T) {
	sp, err := NewSmartPool()
	if err != nil {
		t.Fatalf("Failed to create smart pool: %v", err)
	}
	defer sp.Close()

	// Create multiple allocations of different sizes
	allocations := make([]*Allocation, 0)

	sizes := []int{
		50 * 1024,              // 50KB
		1024 * 1024,            // 1MB
		3*1024*1024 + 200*1024, // 3.2MB
		128 * 1024,             // 128KB
		10 * 1024 * 1024,       // 10MB
	}

	for _, size := range sizes {
		alloc, err := sp.Allocate(size)
		if err != nil {
			t.Fatalf("Failed to allocate %d bytes: %v", size, err)
		}
		allocations = append(allocations, alloc)
	}

	// Verify all allocations are valid
	for i, alloc := range allocations {
		if alloc.TotalSize != sizes[i] {
			t.Errorf("Allocation %d: expected size %d, got %d", i, sizes[i], alloc.TotalSize)
		}
		if alloc.NumBlocks() == 0 {
			t.Errorf("Allocation %d has no blocks", i)
		}
	}

	// Release all allocations
	for _, alloc := range allocations {
		sp.Release(alloc)
	}

	// Verify pools can be reused
	alloc, err := sp.Allocate(1024 * 1024)
	if err != nil {
		t.Fatalf("Failed to allocate after release: %v", err)
	}
	sp.Release(alloc)
}

func TestSmartPool_Stats(t *testing.T) {
	sp, err := NewSmartPool()
	if err != nil {
		t.Fatalf("Failed to create smart pool: %v", err)
	}
	defer sp.Close()

	// Initial stats
	stats := sp.Stats()
	if len(stats.PoolStats) != 2 {
		t.Errorf("Expected 2 pools (default 1MB and 64KB), got %d", len(stats.PoolStats))
	}

	for _, poolStat := range stats.PoolStats {
		if poolStat.NumChunks == 0 {
			t.Error("Expected at least one chunk initially")
		}
	}

	// Track initial used blocks
	initialUsed := make([]int, len(stats.PoolStats))
	for i, poolStat := range stats.PoolStats {
		initialUsed[i] = poolStat.UsedBlocks
	}

	// Allocate and check stats
	alloc, err := sp.Allocate(2*1024*1024 + 100*1024) // 2MB + 100KB
	if err != nil {
		t.Fatalf("Failed to allocate: %v", err)
	}

	stats = sp.Stats()
	// Should have allocated 2 x 1MB blocks + 2 x 64KB blocks
	t.Logf("After allocation: Pool 0 (1MB): %d used, Pool 1 (64KB): %d used",
		stats.PoolStats[0].UsedBlocks, stats.PoolStats[1].UsedBlocks)

	sp.Release(alloc)

	stats = sp.Stats()
	t.Logf("After release: Pool 0 (1MB): %d used, Pool 1 (64KB): %d used",
		stats.PoolStats[0].UsedBlocks, stats.PoolStats[1].UsedBlocks)
}

func TestSmartPool_CustomBlockSizes(t *testing.T) {
	// Create a smart pool with custom block sizes: 4MB, 256KB, 16KB
	sp, err := NewSmartPool(4*1024*1024, 256*1024, 16*1024)
	if err != nil {
		t.Fatalf("Failed to create smart pool with custom sizes: %v", err)
	}
	defer sp.Close()

	// Test allocation that should use 4MB blocks
	alloc, err := sp.Allocate(10 * 1024 * 1024) // 10MB
	if err != nil {
		t.Fatalf("Failed to allocate 10MB: %v", err)
	}
	// Should use 2 x 4MB + 2 x 1MB (wait, we have 4MB, 256KB, 16KB)
	// Should use 2 x 4MB + 8 x 256KB
	t.Logf("10MB allocated with %d blocks", alloc.NumBlocks())
	sp.Release(alloc)

	// Test allocation using 256KB blocks
	alloc, err = sp.Allocate(512 * 1024) // 512KB
	if err != nil {
		t.Fatalf("Failed to allocate 512KB: %v", err)
	}
	// Should use 2 x 256KB
	if alloc.NumBlocks() != 2 {
		t.Errorf("Expected 2 blocks for 512KB, got %d", alloc.NumBlocks())
	}
	sp.Release(alloc)

	// Test small allocation using 16KB blocks
	alloc, err = sp.Allocate(50 * 1024) // 50KB
	if err != nil {
		t.Fatalf("Failed to allocate 50KB: %v", err)
	}
	// Should use 4 x 16KB (since it's less than 4x16KB=64KB)
	if alloc.NumBlocks() != 4 {
		t.Errorf("Expected 4 blocks for 50KB, got %d", alloc.NumBlocks())
	}
	sp.Release(alloc)
}

func TestSmartPool_EdgeCases(t *testing.T) {
	sp, err := NewSmartPool()
	if err != nil {
		t.Fatalf("Failed to create smart pool: %v", err)
	}
	defer sp.Close()

	// Test zero size
	_, err = sp.Allocate(0)
	if err == nil {
		t.Error("Expected error for zero size allocation")
	}

	// Test negative size
	_, err = sp.Allocate(-1)
	if err == nil {
		t.Error("Expected error for negative size allocation")
	}

	// Test very small size (1 byte)
	alloc, err := sp.Allocate(1)
	if err != nil {
		t.Fatalf("Failed to allocate 1 byte: %v", err)
	}
	if alloc.NumBlocks() != 1 {
		t.Errorf("Expected 1 block for 1 byte, got %d", alloc.NumBlocks())
	}
	sp.Release(alloc)

	// Test exact block boundary
	alloc, err = sp.Allocate(Block64KB)
	if err != nil {
		t.Fatalf("Failed to allocate exactly 64KB: %v", err)
	}
	if alloc.NumBlocks() != 1 {
		t.Errorf("Expected 1 block for exactly 64KB, got %d", alloc.NumBlocks())
	}
	sp.Release(alloc)
}

func BenchmarkSmartPool_Allocate_Small(b *testing.B) {
	sp, err := NewSmartPool()
	if err != nil {
		b.Fatalf("Failed to create smart pool: %v", err)
	}
	defer sp.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		alloc, err := sp.Allocate(100 * 1024) // 100KB
		if err != nil {
			b.Fatalf("Failed to allocate: %v", err)
		}
		sp.Release(alloc)
	}
}

func BenchmarkSmartPool_Allocate_Large(b *testing.B) {
	sp, err := NewSmartPool()
	if err != nil {
		b.Fatalf("Failed to create smart pool: %v", err)
	}
	defer sp.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		alloc, err := sp.Allocate(5*1024*1024 + 128*1024) // 5MB + 128KB
		if err != nil {
			b.Fatalf("Failed to allocate: %v", err)
		}
		sp.Release(alloc)
	}
}

func BenchmarkSmartPool_WriteRead(b *testing.B) {
	sp, err := NewSmartPool()
	if err != nil {
		b.Fatalf("Failed to create smart pool: %v", err)
	}
	defer sp.Close()

	size := 2*1024*1024 + 100*1024 // 2MB + 100KB
	alloc, err := sp.Allocate(size)
	if err != nil {
		b.Fatalf("Failed to allocate: %v", err)
	}
	defer sp.Release(alloc)

	testData := make([]byte, size)
	readData := make([]byte, size)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := alloc.WriteToBlocks(testData); err != nil {
			b.Fatalf("Failed to write: %v", err)
		}
		if _, err := alloc.ReadFromBlocks(readData); err != nil {
			b.Fatalf("Failed to read: %v", err)
		}
	}
}
