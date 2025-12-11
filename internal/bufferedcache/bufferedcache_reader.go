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

package bufferedcache

import (
	"context"
	"fmt"
	"sync/atomic"

	cachefolio "github.com/googlecloudplatform/gcsfuse/v3/internal/cache/folio"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/folio"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
)

// BufferedCacheReaderConfig holds configuration for the buffered cache reader.
type BufferedCacheReaderConfig struct {
	// PageSize is the alignment size for window boundaries (typically 4KB)
	PageSize int64

	// MaxWindow is the maximum readahead window size
	MaxWindow int64

	// MergeGap is the maximum gap between reads to consider them related
	MergeGap int64

	// UseOsWindow determines if OS-requested readahead should be used
	UseOsWindow bool
}

// DefaultConfig returns a default configuration for the buffered cache reader.
func DefaultConfig() *BufferedCacheReaderConfig {
	return &BufferedCacheReaderConfig{
		PageSize:    4096,              // 4KB
		MaxWindow:   128 * 1024 * 1024, // 128MB
		MergeGap:    64 * 1024,         // 64KB
		UseOsWindow: false,
	}
}

// BufferedCacheReader manages readahead windows and folio allocation for file reads.
type BufferedCacheReader struct {
	object   *gcs.MinObject
	bucket   gcs.Bucket
	config   *BufferedCacheReaderConfig
	inode    *inode.Inode
	pool     *folio.SmartPool
	lruCache *cachefolio.LRUCache

	// Window state
	windowStart     int64
	windowEnd       int64
	triggerStart    int64
	prevWindowStart int64

	prevEndOffset int64
	numReads      int64
	ioDepth       atomic.Int64
}

// Ensure BufferedCacheReader implements gcsx.Reader
var _ gcsx.Reader = &BufferedCacheReader{}

// NewBufferedCacheReader creates a new BufferedCacheReader instance.
// Returns nil and error if lruCache is nil.
func NewBufferedCacheReader(object *gcs.MinObject, bucket gcs.Bucket, config *BufferedCacheReaderConfig, inode *inode.Inode, pool *folio.SmartPool, lruCache *cachefolio.LRUCache) (*BufferedCacheReader, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if lruCache == nil {
		return nil, fmt.Errorf("lruCache is required but not provided")
	}

	return &BufferedCacheReader{
		object:   object,
		bucket:   bucket,
		config:   config,
		inode:    inode,
		pool:     pool,
		lruCache: lruCache,
	}, nil
}

// describe returns a description of the current readahead state.
func (ra *BufferedCacheReader) describe() any {
	return &struct {
		WindowStart, WindowEnd, WindowSize      int64
		TriggerStart, PrevWindowStart, NumReads int64
	}{
		WindowStart:     ra.windowStart,
		WindowEnd:       ra.windowEnd,
		WindowSize:      ra.windowEnd - ra.windowStart,
		TriggerStart:    ra.triggerStart,
		PrevWindowStart: ra.prevWindowStart,
		NumReads:        ra.numReads,
	}
}

// nextWindowSize calculates the next readahead window size based on current size.
// Similar to how Linux scales the window size.
func (ra *BufferedCacheReader) nextWindowSize(size int64) int64 {
	windowSize := roundNextPowerOfTwo(size)
	if size <= ra.config.MaxWindow/16 {
		windowSize *= 4
	} else if size <= ra.config.MaxWindow/4 {
		windowSize *= 2
	} else {
		windowSize = ra.config.MaxWindow
	}
	return roundUp(windowSize, ra.config.PageSize)
}

// Peek examines the next read operation and determines readahead window updates.
// It populates the ReadAhead struct with the calculated window boundaries.
//
// This function runs synchronously in the main connection loop to detect
// sequential reads. It updates the readahead window and returns a context
// with the window information for the Read() function to use asynchronously.
func (ra *BufferedCacheReader) Peek(ctx context.Context, req *gcsx.ReadRequest, fileSize int64) context.Context {
	offset := req.Offset
	size := int64(len(req.Buffer))

	ra.numReads++
	readGap := offset - ra.prevEndOffset
	endOffset := offset + size
	ra.prevEndOffset = endOffset
	ra.ioDepth.Add(1)

	// Initialize ReadAhead if not already set
	if req.ReadAhead == nil {
		req.ReadAhead = &gcsx.ReadAhead{}
	}

	// Check if size is larger than the window
	if size > ra.config.MaxWindow {
		// Read is larger than the window, no readahead
		req.ReadAhead.WindowStart = offset
		req.ReadAhead.WindowEnd = offset + size
		return ctx
	}

	if offset == 0 && ra.numReads == 1 {
		// First read starting at offset 0 - likely sequential
		ra.windowStart = 0
		ra.windowEnd = ra.nextWindowSize(size)
		ra.triggerStart = ra.windowEnd / 2
		ra.prevWindowStart = 0

	} else if overlap(ra.triggerStart, ra.windowEnd, offset, endOffset) {
		// Reading in trigger window - move to next window
		ra.prevWindowStart = ra.windowStart
		nextSize := ra.nextWindowSize(ra.windowEnd - ra.windowStart)
		ra.windowStart = ra.windowEnd
		ra.windowEnd = ra.windowStart + nextSize
		ra.triggerStart = ra.windowStart

	} else if overlap(ra.prevWindowStart, ra.windowEnd, offset, endOffset) {
		// Reading inside current or previous window, but not trigger window
		// No change to window

	} else if readGap >= 0 && readGap < ra.config.MergeGap {
		// Outside window but adjacent to previous read - start new window
		ra.windowStart = roundDown(offset, ra.config.PageSize)
		nextSize := ra.nextWindowSize(endOffset - ra.windowStart)
		ra.windowEnd = ra.windowStart + nextSize
		ra.triggerStart = (ra.windowEnd - ra.windowStart) / 2
		ra.prevWindowStart = ra.windowStart

	} else {
		// Random read unrelated to previous - keep window but no readahead
		req.ReadAhead.WindowStart = offset
		req.ReadAhead.WindowEnd = offset + size
		return ctx
	}

	// Clamp window to file size
	if ra.windowStart >= fileSize {
		ra.windowStart = 0
		ra.windowEnd = 0
		ra.triggerStart = 0
		ra.prevWindowStart = 0
	} else if ra.windowEnd > fileSize {
		ra.windowEnd = roundUp(fileSize, ra.config.PageSize)
	}

	// Populate the ReadAhead struct with the calculated window
	req.ReadAhead.WindowStart = ra.windowStart
	req.ReadAhead.WindowEnd = ra.windowEnd

	return ctx
}

// ReadAt performs the actual read operation, calling Peek to update readahead window.
// Implements gcsx.Reader interface.
func (ra *BufferedCacheReader) ReadAt(ctx context.Context, req *gcsx.ReadRequest) (gcsx.ReadResponse, error) {
	defer ra.ioDepth.Add(-1)

	// Early return if inode or pool is not available
	if ra.inode == nil || ra.pool == nil {
		return gcsx.ReadResponse{}, fmt.Errorf("bufferedcache reader not fully initialized")
	}

	// Determine the range to read
	offset := req.Offset
	dst := req.Buffer
	size := int64(len(dst))

	// Get file size - if inode is not available, use a large default
	fileSize := int64(1 << 62) // Default to very large size
	// TODO: Get actual file size from inode when available

	// Call Peek to update readahead window state
	_ = ra.Peek(ctx, req, fileSize)

	// Get inode ID for cache lookup
	var inodeID fuseops.InodeID
	if ra.inode != nil {
		inodeID = (*ra.inode).ID()
	}

	// Get folios from LRU cache for the requested range only
	folios, err := ra.lruCache.Get(uint64(inodeID), offset, size)
	if err != nil {
		return gcsx.ReadResponse{}, fmt.Errorf("failed to get folios from cache: %w", err)
	}
	if len(folios) == 0 {
		return gcsx.ReadResponse{}, fmt.Errorf("failed to get folios from cache: no folios returned")
	}

	// If we have a readahead window, prefetch it separately (asynchronously)
	if req.ReadAhead != nil && req.ReadAhead.WindowEnd > req.ReadAhead.WindowStart {
		windowStart := req.ReadAhead.WindowStart
		windowEnd := req.ReadAhead.WindowEnd
		go func() {
			// Prefetch readahead window in background
			_, _ = ra.lruCache.Get(uint64(inodeID), windowStart, windowEnd-windowStart)
		}()
	}

	// TODO: Initiate async downloads for the folios
	// For now, we'll just return the requested portion

	// Create a FolioRefs to manage the folios
	refs := &folio.FolioRefs{}
	for _, f := range folios {
		refs.Add(f)
	}
	defer refs.Release()

	// Wait for folios to be ready
	refs.Wait()

	// Extract the requested data from folios
	n, slices := refs.Slice(offset, offset+int64(len(dst)))
	if n == 0 {
		return gcsx.ReadResponse{Size: 0}, nil
	}

	// Copy data to destination buffer
	copied := 0
	for _, slice := range slices {
		copy(dst[copied:], slice)
		copied += len(slice)
	}

	return gcsx.ReadResponse{Size: copied}, nil
}

// CheckInvariants performs internal consistency checks on the reader state.
// Implements gcsx.Reader interface.
func (ra *BufferedCacheReader) CheckInvariants() {
	// Currently no invariants to check
}

// Destroy releases any resources held by the reader.
// Implements gcsx.Reader interface.
func (ra *BufferedCacheReader) Destroy() {
	// TODO: Release folios and cleanup resources
}

// overlap checks if two ranges [a1, a2) and [b1, b2) overlap.
func overlap(a1, a2, b1, b2 int64) bool {
	return a1 < b2 && b1 < a2
}

// roundDown rounds down to the nearest multiple of align.
func roundDown(x, align int64) int64 {
	return (x / align) * align
}

// roundUp rounds up to the nearest multiple of align.
func roundUp(x, align int64) int64 {
	return ((x + align - 1) / align) * align
}

// roundNextPowerOfTwo rounds up to the next power of two.
func roundNextPowerOfTwo(x int64) int64 {
	if x <= 0 {
		return 1
	}
	// Find the highest bit set
	power := int64(1)
	for power < x {
		power <<= 1
	}
	return power
}

// min returns the minimum of two int64 values.
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two int64 values.
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
