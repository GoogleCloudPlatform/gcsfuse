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
	"bytes"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/clock"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"golang.org/x/net/context"
)

// Timeout value which determines when the MultiRangeDownloader will be closed after
// it's refcount reaches 0.
const multiRangeDownloaderTimeout = 60 * time.Second

func NewMultiRangeDownloaderWrapper(bucket gcs.Bucket, object *gcs.MinObject, config *cfg.Config) (MultiRangeDownloaderWrapper, error) {
	return NewMultiRangeDownloaderWrapperWithClock(bucket, object, clock.RealClock{}, config)
}

func NewMultiRangeDownloaderWrapperWithClock(bucket gcs.Bucket, object *gcs.MinObject, clock clock.Clock, config *cfg.Config) (MultiRangeDownloaderWrapper, error) {
	if object == nil {
		return MultiRangeDownloaderWrapper{}, fmt.Errorf("NewMultiRangeDownloaderWrapperWithClock: Missing MinObject")
	}
	// In case of a local inode, MRDWrapper would be created with an empty minObject (i.e. with a minObject without any information)
	// and when the object is actually created, MRDWrapper would be updated using SetMinObject method.
	return MultiRangeDownloaderWrapper{
		clock:  clock,
		bucket: bucket,
		object: object,
		config: config,
	}, nil
}

type readResult struct {
	bytesRead int
	err       error
}

type mrdEntry struct {
	mrd gcs.MultiRangeDownloader
	mu  sync.RWMutex
	wg  sync.WaitGroup
}

type mrdPool struct {
	entries        []*mrdEntry
	size           int
	current        uint64
	ctx            context.Context
	cancelCreation context.CancelFunc
	creationWg     sync.WaitGroup
}

type MultiRangeDownloaderWrapper struct {
	// Holds the pool of MRDs.
	mrdPool *mrdPool

	// Bucket and object details for MultiRangeDownloader.
	// Object should not be nil.
	object *gcs.MinObject
	bucket gcs.Bucket

	// Refcount is used to determine when to close the MultiRangeDownloader.
	refCount int
	// Mutex is used to synchronize access over refCount.
	mu sync.RWMutex
	// Holds the cancel function, which can be called to cancel the cleanup function.
	cancelCleanup context.CancelFunc
	// Used for waiting for timeout (helps us in mocking the functionality).
	clock clock.Clock
	// GCSFuse mount config.
	config *cfg.Config
	// MRD Read handle. Would be updated when MRD is being closed so that it can be used
	// next time during MRD recreation.
	handle []byte
}

// SetMinObject sets the gcs.MinObject stored in the wrapper to passed value, only if it's non nil.
func (mrdWrapper *MultiRangeDownloaderWrapper) SetMinObject(minObj *gcs.MinObject) error {
	if minObj == nil {
		return fmt.Errorf("MultiRangeDownloaderWrapper::SetMinObject: Missing MinObject")
	}
	mrdWrapper.object = minObj
	return nil
}

// GetMinObject returns the minObject stored in MultiRangeDownloaderWrapper. Used only for unit testing.
func (mrdWrapper *MultiRangeDownloaderWrapper) GetMinObject() *gcs.MinObject {
	return mrdWrapper.object
}

// GetRefCount returns current refcount.
func (mrdWrapper *MultiRangeDownloaderWrapper) GetRefCount() int {
	mrdWrapper.mu.RLock()
	defer mrdWrapper.mu.RUnlock()
	return mrdWrapper.refCount
}

// IncrementRefCount increments the refcount and cancel any running cleanup function.
// This method should be called exactly once per user of this wrapper.
// It has to be called before using the MultiRangeDownloader.
func (mrdWrapper *MultiRangeDownloaderWrapper) IncrementRefCount() {
	mrdWrapper.mu.Lock()
	defer mrdWrapper.mu.Unlock()

	mrdWrapper.refCount++
	if mrdWrapper.cancelCleanup != nil {
		mrdWrapper.cancelCleanup()
		mrdWrapper.cancelCleanup = nil
	}
}

// DecrementRefCount decrements the refcount. In case refcount reaches 0, cleanup the MRD.
// Returns error on invalid usage.
// This method should be called exactly once per user of this wrapper
// when MultiRangeDownloader is no longer needed & can be cleaned up.
func (mrdWrapper *MultiRangeDownloaderWrapper) DecrementRefCount() (err error) {
	mrdWrapper.mu.Lock()
	defer mrdWrapper.mu.Unlock()

	if mrdWrapper.refCount <= 0 {
		err = fmt.Errorf("MultiRangeDownloaderWrapper DecrementRefCount: Refcount cannot be negative")
		return
	}

	mrdWrapper.refCount--
	if mrdWrapper.refCount == 0 && mrdWrapper.mrdPool != nil {
		// Cancel any ongoing background creation of MRDs.
		if mrdWrapper.mrdPool.cancelCreation != nil {
			mrdWrapper.mrdPool.cancelCreation()
		}
		mrdWrapper.mrdPool.creationWg.Wait()

		for _, entry := range mrdWrapper.mrdPool.entries {
			entry.mu.Lock()
			if entry.mrd != nil {
				// Wait for all pending requests to complete.
				entry.wg.Wait()
				if mrdWrapper.handle == nil {
					mrdWrapper.handle = entry.mrd.GetHandle()
				}
				entry.mrd.Close()
				entry.mrd = nil
			}
			entry.mu.Unlock()
		}
		mrdWrapper.mrdPool = nil
	}
	return
}

// Ensures that MultiRangeDownloader exists, creating it if it does not exist.
// LOCK_REQUIRED(mrdWrapper.mu.RLock)
func (mrdWrapper *MultiRangeDownloaderWrapper) ensureMultiRangeDownloader(forceRecreateMRD bool) (err error) {
	if mrdWrapper.object == nil || mrdWrapper.bucket == nil {
		return fmt.Errorf("ensureMultiRangeDownloader error: Missing minObject or bucket")
	}

	// Create the MRD if it does not exist.
	if mrdWrapper.mrdPool == nil {
		// The calling function holds a read lock. To create a new downloader, we need to
		// upgrade to a write lock. This is done by releasing the read lock, acquiring
		// the write lock, and then using a deferred function to downgrade back to a
		// read lock before this function returns.
		mrdWrapper.mu.RUnlock()
		mrdWrapper.mu.Lock()
		// Downgrade the lock back to read lock before returning.
		defer func() {
			mrdWrapper.mu.Unlock()
			mrdWrapper.mu.RLock()
		}()

		// Checking if the mrdWrapper state is same after taking the lock.
		if mrdWrapper.mrdPool == nil {
			poolSize := 4
			// if mrdWrapper.config.Read.MultiRangeDownloaderPoolSize > 0 {
			// 	poolSize = mrdWrapper.config.Read.MultiRangeDownloaderPoolSize
			// }

			pool := &mrdPool{
				entries: make([]*mrdEntry, poolSize),
				size:    poolSize,
			}
			for i := range pool.entries {
				pool.entries[i] = &mrdEntry{}
			}

			// Create the first MRD synchronously.
			mrd, err := mrdWrapper.bucket.NewMultiRangeDownloader(context.Background(), &gcs.MultiRangeDownloaderRequest{
				Name:           mrdWrapper.object.Name,
				Generation:     mrdWrapper.object.Generation,
				ReadCompressed: mrdWrapper.object.HasContentEncodingGzip(),
				ReadHandle:     mrdWrapper.handle,
			})
			if err != nil {
				return err
			}
			pool.entries[0].mrd = mrd

			// Create the rest of the MRDs asynchronously.
			if poolSize > 1 {
				handle := mrd.GetHandle()
				pool.ctx, pool.cancelCreation = context.WithCancel(context.Background())
				pool.creationWg.Add(1)
				go func() {
					defer pool.creationWg.Done()
					mrdWrapper.createRemainingMRDs(pool, handle)
				}()
			}
			mrdWrapper.mrdPool = pool
		}
	}
	return
}

func (mrdWrapper *MultiRangeDownloaderWrapper) createRemainingMRDs(pool *mrdPool, handle []byte) {
	for i := 1; i < pool.size; i++ {
		if pool.ctx.Err() != nil {
			return
		}
		mrd, err := mrdWrapper.bucket.NewMultiRangeDownloader(context.Background(), &gcs.MultiRangeDownloaderRequest{
			Name:           mrdWrapper.object.Name,
			Generation:     mrdWrapper.object.Generation,
			ReadCompressed: mrdWrapper.object.HasContentEncodingGzip(),
			ReadHandle:     handle,
		})
		if err == nil {
			pool.entries[i].mu.Lock()
			pool.entries[i].mrd = mrd
			pool.entries[i].mu.Unlock()
		}
	}
}

func (mrdWrapper *MultiRangeDownloaderWrapper) recreateMRD(entry *mrdEntry) {
	entry.mu.Lock()
	defer entry.mu.Unlock()

	var handle []byte
	if entry.mrd != nil {
		handle = entry.mrd.GetHandle()
		entry.mrd.Close()
	} else {
		handle = mrdWrapper.handle
	}

	mrd, err := mrdWrapper.bucket.NewMultiRangeDownloader(context.Background(), &gcs.MultiRangeDownloaderRequest{
		Name:           mrdWrapper.object.Name,
		Generation:     mrdWrapper.object.Generation,
		ReadCompressed: mrdWrapper.object.HasContentEncodingGzip(),
		ReadHandle:     handle,
	})

	if err == nil {
		entry.mrd = mrd
	}
}

// Reads the data using MultiRangeDownloader.
func (mrdWrapper *MultiRangeDownloaderWrapper) Read(ctx context.Context, buf []byte, startOffset int64, endOffset int64, metricHandle metrics.MetricHandle, forceCreateMRD bool) (bytesRead int, err error) {
	// Bidi Api with 0 as read_limit means no limit whereas we do not want to read anything with empty buffer.
	// Hence, handling it separately.
	if len(buf) == 0 {
		return 0, nil
	}

	mrdWrapper.mu.RLock()
	err = mrdWrapper.ensureMultiRangeDownloader(forceCreateMRD)
	if err != nil {
		err = fmt.Errorf("MultiRangeDownloaderWrapper::Read: Error in creating MultiRangeDownloader:  %v", err)
		mrdWrapper.mu.RUnlock()
		return
	}

	// We will only read what is requested by the client. Hence, capping end to the requested value.
	if endOffset > startOffset+int64(len(buf)) {
		endOffset = startOffset + int64(len(buf))
	}

	// Get the next MRD from the pool.
	idx := atomic.AddUint64(&mrdWrapper.mrdPool.current, 1) % uint64(mrdWrapper.mrdPool.size)
	entry := mrdWrapper.mrdPool.entries[idx]

	entry.mu.RLock()
	// If the MRD is nil or unusable, recreate it.
	if entry.mrd == nil || entry.mrd.Error() != nil || forceCreateMRD {
		entry.mu.RUnlock()
		mrdWrapper.recreateMRD(entry)
		entry.mu.RLock()
	}

	buffer := bytes.NewBuffer(buf)
	buffer.Reset()
	done := make(chan readResult, 1)

	mu := sync.Mutex{}
	defer func() {
		mu.Lock()
		close(done)
		done = nil
		mu.Unlock()
	}()

	start := time.Now()
	if entry.mrd == nil {
		mrdWrapper.mu.RUnlock()
		entry.mu.RUnlock()
		return 0, fmt.Errorf("MultiRangeDownloaderWrapper::Read: Failed to create MultiRangeDownloader")
	}
	entry.wg.Add(1)
	entry.mrd.Add(buffer, startOffset, endOffset-startOffset, func(offsetAddCallback int64, bytesReadAddCallback int64, e error) {
		defer func() {
			mu.Lock()
			if done != nil {
				done <- readResult{bytesRead: int(bytesReadAddCallback), err: e}
			}
			mu.Unlock()
		}()
		defer entry.wg.Done()

		if e != nil && e != io.EOF {
			e = fmt.Errorf("error in Add call: %w", e)
		}
	})
	entry.mu.RUnlock()
	mrdWrapper.mu.RUnlock()

	if !mrdWrapper.config.FileSystem.IgnoreInterrupts {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case res := <-done:
			bytesRead = res.bytesRead
			err = res.err
		}
	} else {
		res := <-done
		bytesRead = res.bytesRead
		err = res.err
	}
	if err != nil {
		err = fmt.Errorf("MultiRangeDownloaderWrapper::Read: %w", err)
		// In case of short read or error, recreate the MRD instance.
		if bytesRead < int(endOffset-startOffset) {
			mrdWrapper.recreateMRD(entry)
		}
		logger.Error(err.Error())
	}
	monitor.CaptureMultiRangeDownloaderMetrics(ctx, metricHandle, "MultiRangeDownloader::Add", start)

	return
}
