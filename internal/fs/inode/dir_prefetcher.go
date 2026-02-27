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

package inode

import (
	"context"
	"path"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/jacobsa/timeutil"
	"golang.org/x/sync/semaphore"
)

// Constants for the metadata prefetch state.
const (
	prefetchReady uint32 = iota
	prefetchInProgress
)

type MetadataPrefetcher struct {
	// Variables for metadata prefetching.
	metadataCacheTTL time.Duration
	state            atomic.Uint32 // 0=Ready, 1=InProgress

	// inodeCtx is the lifecycle context of the owning inode.
	// If this is cancelled (e.g. due to rename/delete folder), all prefetch runs including the subdirectories stop
	// because they are all inherited from the same inodeCtx.
	inodeCtx context.Context

	// runCancelFunc cancels the *current* prefetch run, without killing the inodeCtx.
	runCancelFunc context.CancelFunc

	maxPrefetchCount int64
	cacheClock       timeutil.Clock
	lastPrefetchTime atomic.Pointer[time.Time]
	// isLargeDir indicates if the directory size exceeds maxPrefetchCount.
	// If true, we start prefetching from the looked-up object's offset.
	isLargeDir atomic.Bool

	// sem limits the number of concurrent prefetch goroutines.
	sem *semaphore.Weighted

	// listCallFunc allows the prefetcher to perform GCS List call and hydrate metadata cache.
	listCallFunc func(ctx context.Context, tok string, startOffset string, limit int) (map[Name]*Core, []string, string, error)

	// shouldRun is a callback that returns true if prefetching is allowed.
	// It checks if there are any active writers in the directory.
	shouldRun func() bool
}

func NewMetadataPrefetcher(
	inodeCtx context.Context, // Passed from DirInode
	cfg *cfg.Config,
	prefetchSem *semaphore.Weighted, // Shared semaphore across all MetadataPrefetchers.
	cacheClock timeutil.Clock,
	listFunc func(context.Context, string, string, int) (map[Name]*Core, []string, string, error),
	shouldRun func() bool,
) *MetadataPrefetcher {
	return &MetadataPrefetcher{
		inodeCtx:         inodeCtx,
		metadataCacheTTL: time.Duration(cfg.MetadataCache.TtlSecs) * time.Second,
		maxPrefetchCount: cfg.MetadataCache.MetadataPrefetchEntriesLimit,
		cacheClock:       cacheClock,
		sem:              prefetchSem,
		listCallFunc:     listFunc,
		shouldRun:        shouldRun,
		// state is 0 (prefetchReady) by default.
	}
}

// Run attempts to prefetch metadata for the directory if a prefetch is due.
// It uses an atomic state to prevent concurrent execution.
// This function is already protected by directory mutex so no new mutex is required here for the setup part.
func (p *MetadataPrefetcher) Run(fullObjectName string) {
	// Do not trigger prefetching if:
	// 1. The inode context is nil or already cancelled (dir inode is dead/renamed).
	// 2. If there are active writers in the directory, do not trigger prefetch.
	if p.inodeCtx == nil || p.inodeCtx.Err() != nil || !p.shouldRun() {
		return
	}

	// Do not trigger prefetch if the last prefetch result is still within the TTL.
	lastPrefetchTime := p.lastPrefetchTime.Load()
	now := p.cacheClock.Now()
	if lastPrefetchTime != nil && now.Sub(*lastPrefetchTime) < p.metadataCacheTTL {
		return
	}

	// Ensure only one prefetch runs at a time for this directory.
	if !p.state.CompareAndSwap(prefetchReady, prefetchInProgress) {
		return
	}

	// Create a new context for this specific run, derived from the Inode's lifecycle context.
	// This ensures that if the Inode is cancelled (Recursive), this run stops.
	// We also keep runCancel so we can stop prefetch for just this run (for operations only effecting the curr dir).
	ctx, cancel := context.WithCancel(p.inodeCtx)
	p.runCancelFunc = cancel

	// Run in background to avoid blocking the main Lookup call.
	go func() {
		// Reset to Ready state when the worker finishes.
		defer p.state.Store(prefetchReady)
		// Ensure context resources are released when worker finishes. Calling cancel is idempotent, so it is safe even
		// if p.Cancel() is called externally.
		defer cancel()

		// Try to acquire a semaphore. If the semaphore is full, we skip this prefetch
		// to avoid queuing stale background work.
		if !p.sem.TryAcquire(1) {
			return
		}
		defer p.sem.Release(1)

		dirName := path.Dir(fullObjectName)
		var continuationToken string
		var totalPrefetched int64 = 0

		// If the directory was previously identified as 'large', we optimize by
		// starting the listing from the current looked-up object to ensure its
		// immediate siblings (lexicographically greater) are cached.
		startOffset := ""
		if p.isLargeDir.Load() {
			startOffset = fullObjectName
		}

		for totalPrefetched < p.maxPrefetchCount {
			select {
			case <-ctx.Done():
				logger.Debugf("Metadata prefetch for directory %s aborted: context cancelled.", dirName)
				return
			default:
			}

			// Calculate how many results to ask for in this batch.
			remaining := p.maxPrefetchCount - totalPrefetched
			batchSize := min(remaining, MaxResultsForListObjectsCall)

			// Perform network I/O without holding the inode lock.
			cores, _, newTok, err := p.listCallFunc(ctx, continuationToken, startOffset, int(batchSize))
			if err != nil {
				if ctx == nil || ctx.Err() != nil {
					logger.Debugf("Metadata prefetch for directory %s aborted during list call: %v", dirName, ctx.Err())
				} else {
					logger.Warnf("Prefetch failed for %s: %v", dirName, err)
				}
				return
			}

			totalPrefetched += int64(len(cores))

			// If we hit the prefetch limit but there is still more data in GCS,
			// mark this as a large directory for future targeted prefetches.
			if totalPrefetched >= p.maxPrefetchCount {
				if newTok != "" {
					p.isLargeDir.Store(true)
				}
				break
			}

			// End of directory reached.
			if newTok == "" {
				break
			}
			continuationToken = newTok
		}
		// Update lastPrefetchTime on successful completion.
		now := p.cacheClock.Now()
		p.lastPrefetchTime.Store(&now)
	}()
}

// Cancel stops the *current* prefetch run. It does NOT cancel the inodeCtx.
// Use this for operations to stop the prefetcher for this directory.
// This function is already protected by directory mutex so no new mutex is required here.
func (p *MetadataPrefetcher) Cancel() {
	if p.runCancelFunc != nil {
		p.runCancelFunc()
		p.runCancelFunc = nil
	}
}
