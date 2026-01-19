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
)

// Constants for the metadata prefetch state.
const (
	prefetchReady uint32 = iota
	prefetchInProgress
)

type MetadataPrefetcher struct {
	// Variables for metadata prefetching.
	enabled          bool
	metadataCacheTTL time.Duration
	state            atomic.Uint32 // 0=Ready, 1=InProgress
	ctx              context.Context
	cancel           context.CancelFunc
	maxPrefetchCount int64
	// isLargeDir indicates if the directory size exceeds maxPrefetchCount.
	// If true, we start prefetching from the looked-up object's offset.
	isLargeDir atomic.Bool

	// listCallFunc allows the prefetcher to perform GCS List call and hydrate metadata cache.
	listCallFunc func(ctx context.Context, tok string, startOffset string, limit int) (map[Name]*Core, []string, string, error)
}

func NewMetadataPrefetcher(
	cfg *cfg.Config,
	listFunc func(context.Context, string, string, int) (map[Name]*Core, []string, string, error),
) *MetadataPrefetcher {
	// Initialize a new context for metadata prefetch worker so it can run in background.
	ctx, cancel := context.WithCancel(context.Background())
	return &MetadataPrefetcher{
		enabled:          cfg.MetadataCache.ExperimentalDirMetadataPrefetch,
		metadataCacheTTL: time.Duration(cfg.MetadataCache.TtlSecs) * time.Second,
		ctx:              ctx,
		cancel:           cancel,
		maxPrefetchCount: cfg.MetadataCache.ExperimentalMetadataPrefetchLimit,
		listCallFunc:     listFunc,
		// state is 0 (prefetchReady) by default.
	}
}

// Run attempts to prefetch metadata for the directory if a prefetch is due.
// It uses an atomic state to prevent concurrent execution.
func (p *MetadataPrefetcher) Run(fullObjectName string) {
	// Do not trigger prefetching if:
	// 1. metadata prefetch config is disabled.
	// 2. metadata cache ttl is 0 (disabled).
	// 3. prefetch state is in progress already.
	if !p.enabled || p.metadataCacheTTL == 0 || !p.state.CompareAndSwap(prefetchReady, prefetchInProgress) {
		// Another prefetch is already in progress. Abort.
		return
	}

	// Run in background to avoid blocking the main Lookup call.
	go func() {
		// Reset to Ready state when the worker finishes.
		defer p.state.Store(prefetchReady)

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
			case <-p.ctx.Done():
				logger.Debugf("Metadata prefetch for directory %s aborted: context cancelled.", dirName)
				return
			default:
			}

			// Calculate how many results to ask for in this batch.
			remaining := p.maxPrefetchCount - totalPrefetched
			batchSize := min(remaining, MaxResultsForListObjectsCall)

			// Perform network I/O without holding the inode lock.
			cores, _, newTok, err := p.listCallFunc(p.ctx, continuationToken, startOffset, int(batchSize))
			if err != nil {
				logger.Warnf("Prefetch failed for %s: %v", dirName, err)
				return // Abort.
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
	}()
}

func (p *MetadataPrefetcher) Cancel() {
	if p.enabled && p.cancel != nil {
		// cancel any in progress prefetch when the inode is getting destroyed.
		p.cancel()
	}
}
