package gcsx

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// An interface to generalize the MultiRangeDownloader
// structure in go-storage module to ease our testing.
type MultiRangeDownloader interface {
	Add(output io.Writer, offset, length int64, callback func(int64, int64, error))
	Close() error
	Wait()
	Error() error
	GetHandle() []byte
}

// MRDPoolConfig contains configuration for the MRD pool.
type MRDPoolConfig struct {
	// PoolSize is the number of MultiRangeDownloader instances in the pool
	PoolSize int

	object *gcs.MinObject
	bucket gcs.Bucket
	Handle []byte
}

// MRDPool manages a pool of MultiRangeDownloader instances and distributes
// requests across them using round-robin scheduling.
type MRDPool struct {
	MultiRangeDownloader
	downloaders []MultiRangeDownloader
	counter     uint64 // atomic counter for round-robin
	poolSize    int
	mu          sync.RWMutex
	closed      bool
}

// NewMRDPool creates a new pool of MultiRangeDownloader instances.
func NewMRDPool(config *MRDPoolConfig) (*MRDPool, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.PoolSize <= 0 {
		return nil, fmt.Errorf("pool size must be greater than 0")
	}

	pool := &MRDPool{
		downloaders: make([]MultiRangeDownloader, config.PoolSize),
		poolSize:    config.PoolSize,
	}

	// Initialize all MRD instances in the pool
	for i := 0; i < config.PoolSize; i++ {
		mrd, err := config.bucket.NewMultiRangeDownloader(context.Background(), &gcs.MultiRangeDownloaderRequest{
			Name:           config.object.Name,
			Generation:     config.object.Generation,
			ReadCompressed: config.object.HasContentEncodingGzip(),
			ReadHandle:     config.Handle,
		})
		if err != nil {
			// Clean up any created downloaders before returning error
			pool.Close()
			return nil, fmt.Errorf("failed to create MRD instance %d: %w", i, err)
		}
		pool.downloaders[i] = mrd
	}

	return pool, nil
}

// getNextDownloader returns the next downloader using round-robin selection.
func (p *MRDPool) getNextDownloader() (MultiRangeDownloader, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, fmt.Errorf("pool is closed")
	}

	// Use atomic operations for thread-safe round-robin
	index := atomic.AddUint64(&p.counter, 1) % uint64(p.poolSize)
	return p.downloaders[index], nil
}

// Add adds a download task to one of the MRD instances using round-robin selection.
// The output writer receives the downloaded data, and the callback is invoked
// when the download completes with the offset, length, and any error.
func (p *MRDPool) Add(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
	downloader, _ := p.getNextDownloader()


	downloader.Add(output, offset, length, callback)
}

// Wait waits for all downloads on all MRD instances to complete.
func (p *MRDPool) Wait() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return
	}

	for _, downloader := range p.downloaders {
		if downloader != nil {
			downloader.Wait()
		}
	}
}

// Error returns any error from the MRD instances.
// Returns the first error encountered, or nil if no errors.
func (p *MRDPool) Error() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return fmt.Errorf("pool is closed")
	}

	for i, downloader := range p.downloaders {
		if downloader != nil {
			if err := downloader.Error(); err != nil {
				return fmt.Errorf("downloader %d error: %w", i, err)
			}
		}
	}

	return nil
}

// GetHandle returns the handle from the first MRD instance in the pool.
// This is primarily for compatibility with the MultiRangeDownloader interface.
func (p *MRDPool) GetHandle() []byte {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed || len(p.downloaders) == 0 || p.downloaders[0] == nil {
		return nil
	}

	return p.downloaders[0].GetHandle()
}

// Close closes all MultiRangeDownloader instances in the pool.
func (p *MRDPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	var errs []error
	for i, downloader := range p.downloaders {
		if downloader != nil {
			if err := downloader.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close downloader %d: %w", i, err))
			}
		}
	}

	p.closed = true

	if len(errs) > 0 {
		return fmt.Errorf("errors closing downloaders: %v", errs)
	}

	return nil
}

// PoolSize returns the size of the pool.
func (p *MRDPool) PoolSize() int {
	return p.poolSize
}

// IsClosed returns whether the pool is closed.
func (p *MRDPool) IsClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}

// Stats returns statistics about the pool usage.
type PoolStats struct {
	PoolSize     int
	RequestCount uint64
	Closed       bool
}

// GetStats returns current pool statistics.
func (p *MRDPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PoolStats{
		PoolSize:     p.poolSize,
		RequestCount: atomic.LoadUint64(&p.counter),
		Closed:       p.closed,
	}
}
