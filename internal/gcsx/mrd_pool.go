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

package gcsx

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

const smallFileThresholdMiB = 500

// MRDEntry holds a single MultiRangeDownloader instance and a mutex to protect access to it.
type MRDEntry struct {
	mrd gcs.MultiRangeDownloader
	mu  sync.RWMutex
}

// MRDPoolConfig contains configuration for the MRD pool.
type MRDPoolConfig struct {
	// PoolSize is the number of MultiRangeDownloader instances in the pool
	PoolSize int

	object *gcs.MinObject
	bucket gcs.Bucket
	Handle []byte
}

// MRDPool manages a pool of MultiRangeDownloader instances to allow concurrent downloads.
type MRDPool struct {
	poolConfig  *MRDPoolConfig
	entries     []MRDEntry
	current     atomic.Uint64
	currentSize atomic.Uint64
	ctx         context.Context
	// cancelFunc is used to cancel the background creation of MRDs.
	cancelFunc context.CancelFunc
	// creationWg is used to wait for the background creation of MRDs to finish.
	creationWg sync.WaitGroup
}

// determinePoolSize sets the pool size to 1 if the object size is smaller than
// smallFileThresholdMiB.
func (mrdPoolConfig *MRDPoolConfig) determinePoolSize() {
	if mrdPoolConfig.object.Size < smallFileThresholdMiB*MiB {
		mrdPoolConfig.PoolSize = 1
	}
}

// NewMRDPool initializes a new MRDPool.
// It creates the first MRD synchronously to ensure immediate availability and starts a background goroutine to create the remaining MRDs.
func NewMRDPool(config *MRDPoolConfig, handle []byte) (*MRDPool, error) {
	p := &MRDPool{
		poolConfig: config,
	}
	p.poolConfig.determinePoolSize()
	p.entries = make([]MRDEntry, p.poolConfig.PoolSize)
	p.ctx, p.cancelFunc = context.WithCancel(context.Background())

	// Create the first MRD synchronously.
	mrd, err := config.bucket.NewMultiRangeDownloader(p.ctx, &gcs.MultiRangeDownloaderRequest{
		Name:           config.object.Name,
		Generation:     config.object.Generation,
		ReadCompressed: config.object.HasContentEncodingGzip(),
		ReadHandle:     handle,
	})
	if err != nil {
		return nil, err
	}
	p.entries[0].mrd = mrd
	p.currentSize.Store(1)

	// Create the rest of the MRDs asynchronously.
	if p.poolConfig.PoolSize > 1 {
		mrdHandle := mrd.GetHandle()
		p.creationWg.Add(1)
		go func() {
			defer p.creationWg.Done()
			p.createRemainingMRDs(mrdHandle)
		}()
	}

	return p, nil
}

// createRemainingMRDs creates the remaining MultiRangeDownloader instances in the background.
// It populates the pool entries and increments the currentSize counter.
func (p *MRDPool) createRemainingMRDs(handle []byte) {
	for i := 1; i < p.poolConfig.PoolSize; i++ {
		if p.ctx.Err() != nil {
			return
		}
		mrd, err := p.poolConfig.bucket.NewMultiRangeDownloader(p.ctx, &gcs.MultiRangeDownloaderRequest{
			Name:           p.poolConfig.object.Name,
			Generation:     p.poolConfig.object.Generation,
			ReadCompressed: p.poolConfig.object.HasContentEncodingGzip(),
			ReadHandle:     handle,
		})
		if err == nil {
			p.entries[i].mu.Lock()
			p.entries[i].mrd = mrd
			p.entries[i].mu.Unlock()
		} else {
			logger.Warnf("Error in creating MRD. Would be retried once before using the MRD")
		}
		p.currentSize.Add(1)
	}
}

// Next returns the next available MRDEntry from the pool using a round-robin strategy based on the number of currently initialized MRDs.
// Please check returned MRD is non nil and valid (i.e. not in an error state) before using it.
func (p *MRDPool) Next() *MRDEntry {
	limit := p.currentSize.Load()
	// Use post-increment style to get 0-based index for round-robin.
	idx := (p.current.Add(1) - 1) % limit
	return &p.entries[idx]
}

// RecreateMRD attempts to recreate a specific MRDEntry's MultiRangeDownloader.
// It uses a handle from an existing MRD or a fallback handle.
func (p *MRDPool) RecreateMRD(entry *MRDEntry, fallbackHandle []byte) error {
	entry.mu.Lock()
	defer entry.mu.Unlock()

	var handle []byte
	if entry.mrd != nil {
		handle = entry.mrd.GetHandle()
	} else if fallbackHandle != nil {
		handle = fallbackHandle
	} else {
		for i := 0; i < int(p.currentSize.Load()); i++ {
			if &p.entries[i] == entry {
				continue
			}
			p.entries[i].mu.RLock()
			if p.entries[i].mrd != nil {
				handle = p.entries[i].mrd.GetHandle()
				p.entries[i].mu.RUnlock()
				break
			}
			p.entries[i].mu.RUnlock()
		}
	}

	mrd, err := p.poolConfig.bucket.NewMultiRangeDownloader(context.Background(), &gcs.MultiRangeDownloaderRequest{
		Name:           p.poolConfig.object.Name,
		Generation:     p.poolConfig.object.Generation,
		ReadCompressed: p.poolConfig.object.HasContentEncodingGzip(),
		ReadHandle:     handle,
	})

	if err == nil {
		entry.mrd = mrd
	} else {
		return fmt.Errorf("Error in recreating MRD: %w", err)
	}
	return nil
}

// Close shuts down the MRDPool.
// It cancels any ongoing background creation, waits for pending creations to finish, waits for active downloads on existing MRDs to complete, and then closes all MRDs.
// It returns a handle from one of the closed MRDs for potential future use.
func (p *MRDPool) Close() (handle []byte) {
	if p.cancelFunc != nil {
		p.cancelFunc()
	}
	p.creationWg.Wait()

	for i := range p.entries {
		entry := &p.entries[i]
		entry.mu.Lock()
		if entry.mrd != nil {
			entry.mrd.Wait()
			if handle == nil {
				handle = entry.mrd.GetHandle()
			}
			entry.mrd.Close()
			entry.mrd = nil
		}
		entry.mu.Unlock()
	}
	return
}
