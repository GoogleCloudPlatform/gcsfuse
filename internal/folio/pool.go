// Copyright 2025 Google Inc. All Rights Reserved.

package folio

import (
	"fmt"
	"sync"
	"syscall"
)

const (
	// Block sizes
	Block64KB = 64 * 1024   // 64 KB
	Block1MB  = 1024 * 1024 // 1 MB

	// Chunk size for physical allocation (e.g., 128 MB per chunk)
	ChunkSize = 128 * 1024 * 1024
)

// BlockSize represents the size of a memory block
type BlockSize int

const (
	Size64KB BlockSize = Block64KB
	Size1MB  BlockSize = Block1MB
)

// chunk represents a large physical memory allocation
type chunk struct {
	data      []byte     // mmap-backed memory
	size      int        // total size of chunk
	blockSize int        // size of each logical block
	numBlocks int        // number of blocks in this chunk
	bitmap    []uint64   // bitmap for tracking allocated blocks
	mu        sync.Mutex // protects bitmap
}

// Pool manages a pool of fixed-size memory blocks backed by large mmap allocations
type Pool struct {
	blockSize      int
	chunks         []*chunk
	freeBlocks     chan *Block
	reservedMemory int64 // total memory reserved in bytes
	totalCapacity  int64 // total capacity from all chunks in bytes
	allocatedBytes int64 // current allocated memory in bytes (blocks in use, not in free channel)
	mu             sync.RWMutex
}

// Block represents a logical memory block from the pool
type Block struct {
	Data     []byte
	chunk    *chunk
	blockIdx int
}

// NewPool creates a new memory pool with the specified block size and optional reserved memory.
// reservedMemory parameter (if provided) specifies bytes to reserve, not block count.
func NewPool(blockSize BlockSize, reservedMemory ...int64) (*Pool, error) {
	var reserved int64
	if len(reservedMemory) > 0 {
		reserved = reservedMemory[0]
		if reserved < 0 {
			return nil, fmt.Errorf("reserved memory cannot be negative: %d", reserved)
		}
	}

	p := &Pool{
		blockSize:      int(blockSize),
		chunks:         make([]*chunk, 0),
		freeBlocks:     make(chan *Block, 100), // Buffered channel for quick allocation
		reservedMemory: reserved,
		totalCapacity:  0,
		allocatedBytes: 0,
	}

	// Pre-allocate one chunk
	if err := p.allocateChunk(); err != nil {
		return nil, fmt.Errorf("failed to allocate initial chunk: %w", err)
	}

	// If reserved memory is requested, pre-allocate blocks to satisfy it
	if reserved > 0 {
		if err := p.reserveMemory(reserved); err != nil {
			return nil, fmt.Errorf("failed to reserve %d bytes: %w", reserved, err)
		}
	}

	return p, nil
}

// allocateChunk creates a new chunk using mmap
func (p *Pool) allocateChunk() error {
	numBlocks := ChunkSize / p.blockSize
	if numBlocks == 0 {
		return fmt.Errorf("block size %d is too large for chunk size %d", p.blockSize, ChunkSize)
	}

	// Allocate memory using mmap with anonymous mapping
	data, err := syscall.Mmap(
		-1,                                   // fd: -1 for anonymous mapping
		0,                                    // offset
		ChunkSize,                            // length
		syscall.PROT_READ|syscall.PROT_WRITE, // protection
		syscall.MAP_ANON|syscall.MAP_PRIVATE, // flags
	)
	if err != nil {
		return fmt.Errorf("mmap failed: %w", err)
	}

	// Create bitmap for tracking block allocation
	// Each uint64 can track 64 blocks
	bitmapSize := (numBlocks + 63) / 64

	c := &chunk{
		data:      data,
		size:      ChunkSize,
		blockSize: p.blockSize,
		numBlocks: numBlocks,
		bitmap:    make([]uint64, bitmapSize),
	}

	p.mu.Lock()
	p.chunks = append(p.chunks, c)
	p.totalCapacity += int64(numBlocks) * int64(p.blockSize)
	p.mu.Unlock()

	// Pre-populate free blocks - allocate more to reduce bitmap scanning
	// Allocate up to 20% of blocks or 50, whichever is smaller
	prepopCount := minInt(minInt(50, numBlocks/5), numBlocks)
	if prepopCount > 0 {
		prepopBlocks := c.allocateBatch(prepopCount)
		for _, block := range prepopBlocks {
			select {
			case p.freeBlocks <- block:
			default:
				// Channel full, mark block as free
				c.freeBlock(block.blockIdx)
			}
		}
	}

	return nil
}

// Acquire gets a block from the pool
func (p *Pool) Acquire() (*Block, error) {
	// Try to get from free blocks channel first
	select {
	case block := <-p.freeBlocks:
		// Block is being reused, don't update allocatedBytes
		return block, nil
	default:
		// No free blocks in channel, try to allocate from existing chunks
	}

	p.mu.RLock()
	chunks := p.chunks
	p.mu.RUnlock()

	// Try to allocate from existing chunks
	for _, c := range chunks {
		if block := c.allocateBlock(); block != nil {
			// New allocation, update allocated bytes
			p.mu.Lock()
			p.allocatedBytes += int64(p.blockSize)
			p.mu.Unlock()
			return block, nil
		}
	}

	// All chunks are full, allocate a new chunk
	if err := p.allocateChunk(); err != nil {
		return nil, fmt.Errorf("failed to allocate new chunk: %w", err)
	}

	// Try again with the new chunk
	p.mu.RLock()
	newChunk := p.chunks[len(p.chunks)-1]
	p.mu.RUnlock()

	block := newChunk.allocateBlock()
	if block == nil {
		return nil, fmt.Errorf("failed to allocate block from new chunk")
	}

	return block, nil
}

// AcquireBatch efficiently acquires multiple blocks from the pool
func (p *Pool) AcquireBatch(count int) ([]*Block, error) {
	if count <= 0 {
		return nil, nil
	}

	blocks := make([]*Block, 0, count)

	// First, try to get as many as possible from the free blocks channel
	for i := 0; i < count; i++ {
		select {
		case block := <-p.freeBlocks:
			blocks = append(blocks, block)
		default:
			goto allocateFromChunks
		}
	}

	// If we got all blocks from channel, return early (reused blocks, no allocation tracking needed)
	if len(blocks) == count {
		return blocks, nil
	}

allocateFromChunks:
	remaining := count - len(blocks)
	newlyAllocated := 0

	p.mu.RLock()
	chunks := p.chunks
	p.mu.RUnlock()

	// Try to allocate remaining blocks from existing chunks using batch allocation
	for _, c := range chunks {
		if remaining == 0 {
			break
		}
		batchBlocks := c.allocateBatch(remaining)
		blocks = append(blocks, batchBlocks...)
		newlyAllocated += len(batchBlocks)
		remaining -= len(batchBlocks)
	}

	if remaining == 0 {
		// Update allocated bytes for newly allocated blocks
		if newlyAllocated > 0 {
			p.mu.Lock()
			p.allocatedBytes += int64(newlyAllocated) * int64(p.blockSize)
			p.mu.Unlock()
		}
		return blocks, nil
	}

	// Need more chunks
	for remaining > 0 {
		if err := p.allocateChunk(); err != nil {
			// Cleanup already acquired blocks
			for _, block := range blocks {
				p.Release(block)
			}
			return nil, fmt.Errorf("failed to allocate new chunk: %w", err)
		}

		p.mu.RLock()
		newChunk := p.chunks[len(p.chunks)-1]
		p.mu.RUnlock()

		// Allocate from new chunk using batch
		batchBlocks := newChunk.allocateBatch(remaining)
		blocks = append(blocks, batchBlocks...)
		newlyAllocated += len(batchBlocks)
		remaining -= len(batchBlocks)
	}

	// Update allocated bytes for all newly allocated blocks
	if newlyAllocated > 0 {
		p.mu.Lock()
		p.allocatedBytes += int64(newlyAllocated) * int64(p.blockSize)
		p.mu.Unlock()
	}

	return blocks, nil
}

// Release returns a block to the pool
func (p *Pool) Release(block *Block) {
	if block == nil || block.chunk == nil {
		return
	}

	// Note: We don't zero out the block data for performance reasons.
	// The caller should be responsible for not leaking sensitive data.
	// If zeroing is required, use ReleaseAndClear() instead.

	// Mark block as free in bitmap
	block.chunk.freeBlock(block.blockIdx)

	// Try to return to free blocks channel
	select {
	case p.freeBlocks <- block:
	default:
		// Channel full, block is already marked as free in bitmap
	}
}

// reserveMemory pre-allocates blocks to satisfy the reserved memory requirement
func (p *Pool) reserveMemory(bytes int64) error {
	if bytes <= 0 {
		return nil
	}

	// Calculate number of blocks needed
	blockCount := int((bytes + int64(p.blockSize) - 1) / int64(p.blockSize))

	// Calculate number of chunks needed for this reservation
	blocksPerChunk := ChunkSize / p.blockSize
	chunksNeeded := (blockCount + blocksPerChunk - 1) / blocksPerChunk

	// Allocate the necessary chunks
	// Note: One chunk is already allocated in NewPool
	for i := 1; i < chunksNeeded; i++ {
		if err := p.allocateChunk(); err != nil {
			return fmt.Errorf("failed to allocate chunk %d for reservation: %w", i, err)
		}
	}

	// All chunks are allocated, now acquire the blocks and put them in free pool
	blocks, err := p.AcquireBatch(blockCount)
	if err != nil {
		return fmt.Errorf("failed to acquire blocks for reserved memory: %w", err)
	}

	// Put them in the free blocks channel so they're available for use
	for _, block := range blocks {
		select {
		case p.freeBlocks <- block:
		default:
			// Channel full, block is already allocated in chunk
			// This shouldn't happen with our buffer size, but handle it gracefully
		}
	}

	return nil
}

// ReservedMemory returns the amount of memory reserved at creation time in bytes
func (p *Pool) ReservedMemory() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.reservedMemory
}

// AllocatedMemory returns the total capacity allocated in bytes (includes reserved memory)
func (p *Pool) AllocatedMemory() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.totalCapacity
}

// AvailableMemory returns the amount of memory that can still be allocated
// This is effectively unlimited for mmap-based pools, but respects reserved memory
func (p *Pool) AvailableMemory() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// For mmap pools, we report reserved memory as the baseline
	// Actual available memory is much larger (system dependent)
	if p.allocatedBytes < p.reservedMemory {
		return p.reservedMemory - p.allocatedBytes
	}
	return 0 // Beyond reserved, we can still allocate but report 0 to indicate we've used reservation
}

// ReleaseAndClear returns a block to the pool after clearing its data
func (p *Pool) ReleaseAndClear(block *Block) {
	if block == nil || block.chunk == nil {
		return
	}

	// Clear the block data for security
	for i := range block.Data {
		block.Data[i] = 0
	}

	// Mark block as free in bitmap
	block.chunk.freeBlock(block.blockIdx)

	// Try to return to free blocks channel
	select {
	case p.freeBlocks <- block:
	default:
		// Channel full, block is already marked as free in bitmap
	}
}

// Close releases all mmap allocations
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for _, c := range p.chunks {
		if err := syscall.Munmap(c.data); err != nil {
			lastErr = err
		}
	}

	p.chunks = nil
	close(p.freeBlocks)

	return lastErr
}

// Stats returns statistics about the pool
func (p *Pool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		BlockSize:   p.blockSize,
		NumChunks:   len(p.chunks),
		TotalBlocks: 0,
		UsedBlocks:  0,
		FreeBlocks:  0,
	}

	for _, c := range p.chunks {
		c.mu.Lock()
		stats.TotalBlocks += c.numBlocks
		used := c.countUsedBlocks()
		stats.UsedBlocks += used
		stats.FreeBlocks += c.numBlocks - used
		c.mu.Unlock()
	}

	return stats
}

// PoolStats contains statistics about a pool
type PoolStats struct {
	BlockSize   int
	NumChunks   int
	TotalBlocks int
	UsedBlocks  int
	FreeBlocks  int
}

// allocateBatch allocates multiple blocks from this chunk
func (c *chunk) allocateBatch(count int) []*Block {
	if count <= 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	blocks := make([]*Block, 0, count)
	allocated := 0

	// Find free blocks using bitmap
	for i := 0; i < len(c.bitmap) && allocated < count; i++ {
		if c.bitmap[i] != ^uint64(0) { // Not all bits are set
			// Find free bits in this uint64
			for bit := 0; bit < 64 && allocated < count; bit++ {
				blockIdx := i*64 + bit
				if blockIdx >= c.numBlocks {
					return blocks
				}

				mask := uint64(1) << bit
				if c.bitmap[i]&mask == 0 { // Block is free
					// Mark as used
					c.bitmap[i] |= mask

					// Create block
					offset := blockIdx * c.blockSize
					block := &Block{
						Data:     c.data[offset : offset+c.blockSize],
						chunk:    c,
						blockIdx: blockIdx,
					}
					blocks = append(blocks, block)
					allocated++
				}
			}
		}
	}

	return blocks
}

// allocateBlock allocates a block from this chunk
func (c *chunk) allocateBlock() *Block {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find a free block using bitmap
	for i := 0; i < len(c.bitmap); i++ {
		if c.bitmap[i] != ^uint64(0) { // Not all bits are set
			// Find first free bit
			for bit := 0; bit < 64; bit++ {
				blockIdx := i*64 + bit
				if blockIdx >= c.numBlocks {
					return nil
				}

				mask := uint64(1) << bit
				if c.bitmap[i]&mask == 0 { // Block is free
					// Mark as allocated
					c.bitmap[i] |= mask

					// Calculate offset in the data slice
					offset := blockIdx * c.blockSize

					return &Block{
						Data:     c.data[offset : offset+c.blockSize],
						chunk:    c,
						blockIdx: blockIdx,
					}
				}
			}
		}
	}

	return nil
}

// freeBlock marks a block as free
func (c *chunk) freeBlock(blockIdx int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bitmapIdx := blockIdx / 64
	bit := blockIdx % 64
	mask := uint64(1) << bit

	// Clear the bit to mark as free
	c.bitmap[bitmapIdx] &^= mask
}

// countUsedBlocks counts the number of allocated blocks
func (c *chunk) countUsedBlocks() int {
	count := 0
	for i := 0; i < len(c.bitmap); i++ {
		count += popcount(c.bitmap[i])
	}
	return count
}

// popcount counts the number of set bits
func popcount(x uint64) int {
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
