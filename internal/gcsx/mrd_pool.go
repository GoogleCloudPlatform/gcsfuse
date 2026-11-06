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
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

const (
	smallFileThresholdMiB  = 100
	mediumFileThresholdMiB = 500
)

// MRDEntry holds a single MultiRangeDownloader instance and a mutex to protect access to it.
type MRDEntry struct {
	mrd gcs.MultiRangeDownloader
	mu  sync.RWMutex
}

// MRDPoolConfig contains configuration for the MRD pool.
type MRDPoolConfig struct {
	// PoolSize is the number of concurrent streams configured on the MultiRangeDownloader.
	PoolSize int

	object *gcs.MinObject
	bucket gcs.Bucket
	Handle []byte
}

// MRDPool manages a MultiRangeDownloader configured with multiple concurrent gRPC streams
// via the Go Storage SDK's WithMinConnections / WithMaxConnections options.
type MRDPool struct {
	poolConfig *MRDPoolConfig
	entries    []MRDEntry
	ctx        context.Context
}

// determinePoolSize sets the pool size to 1 if the object size is smaller than
// smallFileThresholdMiB, or 2 if smaller than mediumFileThresholdMiB.
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
// It delegates multi-connection stream management to the Go Storage SDK using
// MinConnections and MaxConnections, which opens the first stream synchronously
// and remaining streams asynchronously in the background reusing the ReadHandle.
func NewMRDPool(config *MRDPoolConfig, handle []byte) (*MRDPool, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	p := &MRDPool{
		poolConfig: config,
		ctx:        context.Background(),
	}
	p.poolConfig.determinePoolSize()
	logger.Tracef("Initializing MRD Pool with size: %d", p.poolConfig.PoolSize)
	p.entries = make([]MRDEntry, 1)

	mrd, err := config.bucket.NewMultiRangeDownloader(p.ctx, &gcs.MultiRangeDownloaderRequest{
		Name:           config.object.Name,
		Generation:     config.object.Generation,
		ReadCompressed: config.object.HasContentEncodingGzip(),
		ReadHandle:     handle,
		MinConnections: p.poolConfig.PoolSize,
		MaxConnections: p.poolConfig.PoolSize,
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

	return p, nil
}

// Next returns the MRDEntry managed by the pool.
// Please check returned MRD is non nil and valid (i.e. not in an error state) before using it.
func (p *MRDPool) Next() *MRDEntry {
	return &p.entries[0]
}

// RecreateMRD attempts to recreate the MRDEntry's MultiRangeDownloader.
// It uses a handle from the existing MRD or a fallback handle.
func (p *MRDPool) RecreateMRD(entry *MRDEntry, fallbackHandle []byte) error {
	entry.mu.Lock()
	defer entry.mu.Unlock()

	var handle []byte
	if entry.mrd != nil {
		handle = entry.mrd.GetHandle()
	} else if fallbackHandle != nil {
		handle = fallbackHandle
	}

	mrd, err := p.poolConfig.bucket.NewMultiRangeDownloader(p.ctx, &gcs.MultiRangeDownloaderRequest{
		Name:           p.poolConfig.object.Name,
		Generation:     p.poolConfig.object.Generation,
		ReadCompressed: p.poolConfig.object.HasContentEncodingGzip(),
		ReadHandle:     handle,
		MinConnections: p.poolConfig.PoolSize,
		MaxConnections: p.poolConfig.PoolSize,
	})

	if err == nil {
		entry.mrd = mrd
	} else {
		return fmt.Errorf("Error in recreating MRD: %w", err)
	}
	return nil
}

// Close shuts down the MRDPool gracefully.
// It waits for active downloads on the MRD to complete and then closes the MRD.
// The context used for MRD creation is never canceled, ensuring in-flight range
// requests complete without interruption.
// It returns a handle from the closed MRD for potential future use.
func (p *MRDPool) Close() (handle []byte) {
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

// Return the configured connection pool size.
func (p *MRDPool) Size() uint64 {
	return uint64(p.poolConfig.PoolSize)
}
