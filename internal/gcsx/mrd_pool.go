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
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

const (
	smallFileThresholdMiB  = 100
	mediumFileThresholdMiB = 500
)

// ObjectAccessLatency records the latency of accessing a GCS object.
type ObjectAccessLatency struct {
	Latency time.Duration
}

// MRDEntry holds a single MultiRangeDownloader instance and a mutex to protect access to it.
type MRDEntry struct {
	mrd gcs.MultiRangeDownloader
	mu  sync.RWMutex
	// To store latencies for each read.
	latencies []ObjectAccessLatency
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
	// stopCreation is used to signal background creation goroutine to stop without
	// canceling the context, enabling graceful shutdown.
	stopCreation chan struct{}
	// creationWg is used to wait for the background creation of MRDs to finish.
	creationWg sync.WaitGroup
}

// determinePoolSize sets the pool size to 1 if the object size is smaller than
// smallFileThresholdMiB.
func (mrdPoolConfig *MRDPoolConfig) determinePoolSize() {
	if mrdPoolConfig.object.Size < smallFileThresholdMiB*MiB {
		mrdPoolConfig.PoolSize = 1
		return
	}
	if mrdPoolConfig.object.Size < mediumFileThresholdMiB*MiB {
		mrdPoolConfig.PoolSize = 2
		return
	}
}

// NewMRDPool initializes a new MRDPool.
// It creates the first MRD synchronously to ensure immediate availability and starts a background goroutine to create the remaining MRDs.
func NewMRDPool(config *MRDPoolConfig, handle []byte) (*MRDPool, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	p := &MRDPool{
		poolConfig:   config,
		ctx:          context.Background(),
		stopCreation: make(chan struct{}),
	}
	p.poolConfig.determinePoolSize()
	logger.Tracef("Initializing MRD Pool with size: %d", p.poolConfig.PoolSize)
	p.entries = make([]MRDEntry, p.poolConfig.PoolSize)

	// Create the first MRD synchronously.
	mrd, err := config.bucket.NewMultiRangeDownloader(p.ctx, &gcs.MultiRangeDownloaderRequest{
		Name:           config.object.Name,
		Generation:     config.object.Generation,
		ReadCompressed: config.object.HasContentEncodingGzip(),
		ReadHandle:     handle,
	})
	if err != nil {
		var notFoundError *gcs.NotFoundError
		if errors.As(err, &notFoundError) {
			return nil, &gcsfuse_errors.FileClobberedError{
				Err:        fmt.Errorf("NewMRDPool: %w", err),
				ObjectName: config.object.Name,
			}
		}
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
		// Check if we should stop creating MRDs (graceful shutdown initiated)
		select {
		case <-p.stopCreation:
			return
		default:
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
			// Use TryRLock to avoid deadlock if multiple entries are being recreated simultaneously.
			if p.entries[i].mu.TryRLock() {
				if p.entries[i].mrd != nil {
					handle = p.entries[i].mrd.GetHandle()
					p.entries[i].mu.RUnlock()
					break
				}
				p.entries[i].mu.RUnlock()
			}
		}
	}

	mrd, err := p.poolConfig.bucket.NewMultiRangeDownloader(p.ctx, &gcs.MultiRangeDownloaderRequest{
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

// Close shuts down the MRDPool gracefully.
// It signals background creation to stop, waits for pending creations to finish,
// waits for active downloads on existing MRDs to complete, and then closes all MRDs.
// The context used for MRD creation is never canceled, ensuring in-flight range
// requests complete without interruption.
// It returns a handle from one of the closed MRDs for potential future use.
func (p *MRDPool) Close() (handle []byte) {
	// Signal background creation to stop
	close(p.stopCreation)
	// Wait for background creation to finish
	p.creationWg.Wait()

	// Wait for all MRDs to complete their work and close them
	for i := range p.entries {
		entry := &p.entries[i]
		entry.mu.Lock()
		if entry.mrd != nil {
			// Wait for in-flight downloads to complete
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

// Return the max size of the pool.
func (p *MRDPool) Size() uint64 {
	return uint64(p.poolConfig.PoolSize)
}

func (e *MRDEntry) logLatencyStats(mrdIndex int) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.latencies) == 0 {
		return
	}

	// Sort latencies in ascending order.
	sort.Slice(e.latencies, func(i, j int) bool {
		return e.latencies[i].Latency < e.latencies[j].Latency
	})

	// Calculate percentiles.
	p50 := e.latencies[len(e.latencies)/2].Latency
	p90 := e.latencies[len(e.latencies)*9/10].Latency
	p99 := e.latencies[len(e.latencies)*99/100].Latency
	p100 := e.latencies[len(e.latencies)-1].Latency

	logger.Infof("MRD %d latency stats: P50: %v, P90: %v, P99: %v, P100: %v",
		mrdIndex, p50, p90, p99, p100)
}
