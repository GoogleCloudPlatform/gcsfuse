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
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"golang.org/x/net/context"
)

func NewMultiRangeDownloaderWrapper(bucket gcs.Bucket, object *gcs.MinObject, config *cfg.Config, mrdCache *lru.Cache) (*MultiRangeDownloaderWrapper, error) {
	if object == nil {
		return nil, fmt.Errorf("NewMultiRangeDownloaderWrapper: Missing MinObject")
	}
	// In case of a local inode, MRDWrapper would be created with an empty minObject (i.e. with a minObject without any information)
	// and when the object is actually created, MRDWrapper would be updated using SetMinObject method.
	wrapper := MultiRangeDownloaderWrapper{
		bucket:   bucket,
		object:   object,
		config:   config,
		mrdCache: mrdCache,
	}
	return &wrapper, nil
}

type readResult struct {
	bytesRead int
	err       error
}

type MultiRangeDownloaderWrapper struct {
	lru.ValueType // For LRU cache compatibility

	// Holds the object implementing MultiRangeDownloader interface.
	Wrapped gcs.MultiRangeDownloader

	// Bucket and object details for MultiRangeDownloader.
	// Object should not be nil.
	object *gcs.MinObject
	bucket gcs.Bucket

	// Refcount is used to determine when to close the MultiRangeDownloader.
	refCount int
	// Mutex is used to synchronize access over refCount.
	mu sync.RWMutex
	// GCSFuse mount config.
	config *cfg.Config
	// MRD Read handle. Would be updated when MRD is being closed so that it can be used
	// next time during MRD recreation.
	handle []byte

	// MRD cache for LRU-based eviction of inactive MRD instances.
	mrdCache *lru.Cache
}

// SetMinObject sets the gcs.MinObject stored in the wrapper to passed value, only if it's non nil.
func (mrdWrapper *MultiRangeDownloaderWrapper) SetMinObject(minObj *gcs.MinObject) error {
	if minObj == nil {
		return fmt.Errorf("MultiRangeDownloaderWrapper::SetMinObject: Missing MinObject")
	}
	mrdWrapper.mu.Lock()
	defer mrdWrapper.mu.Unlock()
	mrdWrapper.object = minObj
	return nil
}

// wrapperKey generates a unique key for the given MultiRangeDownloaderWrapper.
// Uses the pointer address as the unique identifier, would be safe as long as
// wrapper is uniquely associated with the lifecycle of the FileInode instance.
func wrapperKey(wrapper *MultiRangeDownloaderWrapper) string {
	return fmt.Sprintf("%p", wrapper)
}

// GetMinObject returns the minObject stored in MultiRangeDownloaderWrapper. Used only for unit testing.
func (mrdWrapper *MultiRangeDownloaderWrapper) GetMinObject() *gcs.MinObject {
	mrdWrapper.mu.RLock()
	defer mrdWrapper.mu.RUnlock()
	return mrdWrapper.object
}

// GetRefCount returns current refcount.
func (mrdWrapper *MultiRangeDownloaderWrapper) GetRefCount() int {
	mrdWrapper.mu.RLock()
	defer mrdWrapper.mu.RUnlock()
	return mrdWrapper.refCount
}

// IncrementRefCount increments the refcount.
// This method should be called exactly once per user of this wrapper.
// It has to be called before using the MultiRangeDownloader.
// If lru cache is enabled and refCount was 0, the wrapper is removed from the cache.
func (mrdWrapper *MultiRangeDownloaderWrapper) IncrementRefCount() {
	mrdWrapper.mu.Lock()
	defer mrdWrapper.mu.Unlock()

	mrdWrapper.refCount++

	// If refCount was 0, remove from cache (file is being reopened)
	if mrdWrapper.refCount == 1 && mrdWrapper.mrdCache != nil {
		deletedEntry := mrdWrapper.mrdCache.Erase(wrapperKey(mrdWrapper))
		if deletedEntry != nil {
			logger.Tracef("MRDWrapper (%s) erased from cache", mrdWrapper.object.Name)
		}
	}
}

// DecrementRefCount decrements the refcount. When refCount reaches 0, the wrapper
// with added to the cache for potential reuse. In case, cache is not enabled, MRD
// is closed immediately.
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
	// Do nothing if MRD is in use or cache is not enabled.
	if mrdWrapper.refCount > 0 || mrdWrapper.mrdCache == nil {
		return
	}

	// Cache with refCount 0: add the wrapper to cache and evict overflow wrappers.
	evictedValues, err := mrdWrapper.mrdCache.Insert(wrapperKey(mrdWrapper), mrdWrapper)
	if err != nil {
		logger.Errorf("failed to insert wrapper (%s) into cache: %v", mrdWrapper.object.Name, err)
		return
	}
	logger.Tracef("MRDWrapper (%s) added wrapper to cache", mrdWrapper.object.Name)

	// Do not proceed if no eviction happened.
	if evictedValues == nil {
		return nil
	}

	// Evict outside all locks to avoid deadlock.
	mrdWrapper.mu.Unlock()
	for _, wrapper := range evictedValues {
		mrdWrapper, ok := wrapper.(*MultiRangeDownloaderWrapper)
		if !ok {
			logger.Errorf("invalid value type, expected MultiRangeDownloaderWrapper, got %T", wrapper)
		} else {
			mrdWrapper.CloseMRDForEviction()
		}
	}
	// Reacquire the lock ensuring safe defer's Unlock.
	mrdWrapper.mu.Lock()

	return nil
}

// CloseMRDForEviction closes the MRD when evicted from cache.
// This method is called after wrapper was removed from cache for eviction.
// Race protection: wrapper could be reopened (refCount>0) or re-added to cache before eviction.
func (mrdWrapper *MultiRangeDownloaderWrapper) CloseMRDForEviction() {
	mrdWrapper.mu.Lock()
	defer mrdWrapper.mu.Unlock()

	// Check if wrapper was reopened (refCount>0) - must skip eviction.
	if mrdWrapper.refCount > 0 {
		return
	}

	// Check if wrapper was re-added to cache (refCount went 0→1→0 in between eviction and closure.)
	// Lock order: wrapper.mu -> cache.mu (consistent with Increment/DecrementRefCount)
	if mrdWrapper.mrdCache != nil && mrdWrapper.mrdCache.LookUpWithoutChangingOrder(wrapperKey(mrdWrapper)) != nil {
		return
	}
	mrdWrapper.closeLocked()
}

// Ensures that MultiRangeDownloader exists, creating it if it does not exist.
// LOCK_REQUIRED(mrdWrapper.mu.RLock)
func (mrdWrapper *MultiRangeDownloaderWrapper) ensureMultiRangeDownloader(ctx context.Context, forceRecreateMRD bool) (err error) {
	ctx = tracing.MaybePropagateTraceContext(context.Background(), ctx)
	if mrdWrapper.object == nil || mrdWrapper.bucket == nil {
		return fmt.Errorf("ensureMultiRangeDownloader error: Missing minObject or bucket")
	}

	// Create the MRD if it does not exist.
	// In case the existing MRD is unusable due to closed stream, recreate the MRD.
	if forceRecreateMRD || mrdWrapper.Wrapped == nil || mrdWrapper.Wrapped.Error() != nil {
		// The calling function holds a read lock. To create a new downloader, we need to
		// upgrade to a write lock. This is done by releasing the read lock, acquiring
		// the write lock, and then using a deferred function to downgrade back to a
		// read lock before this function returns.
		mrdWrapper.mu.RUnlock()
		mrdWrapper.mu.Lock()
		defer func() {
			mrdWrapper.mu.Unlock()
			mrdWrapper.mu.RLock()
		}()
		// Checking if the mrdWrapper state is same after taking the lock.
		if forceRecreateMRD || mrdWrapper.Wrapped == nil || mrdWrapper.Wrapped.Error() != nil {
			var mrd gcs.MultiRangeDownloader
			var handle []byte
			if !forceRecreateMRD {
				// Get read handle from MRD if it exists otherwise use the cached read handle
				if mrdWrapper.Wrapped != nil {
					handle = mrdWrapper.Wrapped.GetHandle()
				} else {
					handle = mrdWrapper.handle
				}
			}
			mrd, err = mrdWrapper.bucket.NewMultiRangeDownloader(ctx, &gcs.MultiRangeDownloaderRequest{
				Name:           mrdWrapper.object.Name,
				Generation:     mrdWrapper.object.Generation,
				ReadCompressed: mrdWrapper.object.HasContentEncodingGzip(),
				ReadHandle:     handle,
			})
			if err != nil {
				var notFoundError *gcs.NotFoundError
				if errors.As(err, &notFoundError) {
					return &gcsfuse_errors.FileClobberedError{
						Err:        fmt.Errorf("ensureMultiRangeDownloader: %w", err),
						ObjectName: mrdWrapper.object.Name,
					}
				}
				return err
			}
			// Updating mrdWrapper.Wrapped only when MRD creation was successful.
			mrdWrapper.Wrapped = mrd
		}
	}
	return
}

// Reads the data using MultiRangeDownloader.
func (mrdWrapper *MultiRangeDownloaderWrapper) Read(ctx context.Context, buf []byte, startOffset int64, endOffset int64, metricHandle metrics.MetricHandle, forceCreateMRD bool) (bytesRead int, err error) {
	// Bidi Api with 0 as read_limit means no limit whereas we do not want to read anything with empty buffer.
	// Hence, handling it separately.
	if len(buf) == 0 {
		return 0, nil
	}

	mrdWrapper.mu.RLock()
	err = mrdWrapper.ensureMultiRangeDownloader(ctx, forceCreateMRD)
	if err != nil {
		err = fmt.Errorf("MultiRangeDownloaderWrapper::Read: Error in creating MultiRangeDownloader:  %v", err)
		mrdWrapper.mu.RUnlock()
		return
	}

	// We will only read what is requested by the client. Hence, capping end to the requested value.
	if endOffset > startOffset+int64(len(buf)) {
		endOffset = startOffset + int64(len(buf))
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
	mrdWrapper.Wrapped.Add(buffer, startOffset, endOffset-startOffset, func(offsetAddCallback int64, bytesReadAddCallback int64, e error) {
		defer func() {
			mu.Lock()
			if done != nil {
				done <- readResult{bytesRead: int(bytesReadAddCallback), err: e}
			}
			mu.Unlock()
		}()

		if e != nil && e != io.EOF {
			e = fmt.Errorf("error in Add call: %w", e)
		}
	})
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
		logger.Error(err.Error())
	}
	monitor.CaptureMultiRangeDownloaderMetrics(ctx, metricHandle, "MultiRangeDownloader::Add", start)

	return
}

// closeLocked closes the MultiRangeDownloader.
// LOCK_REQUIRED(mrdWrapper.mu.Lock)
func (mrdWrapper *MultiRangeDownloaderWrapper) closeLocked() {
	if mrdWrapper.Wrapped == nil {
		return
	}

	// Save handle for potential recreation
	mrdWrapper.handle = mrdWrapper.Wrapped.GetHandle()

	// Close the MRD
	if err := mrdWrapper.Wrapped.Close(); err != nil {
		logger.Warnf("Error closing MRD (%s): %v", mrdWrapper.object.Name, err)
		return
	}
	logger.Tracef("MRDWrapper (%s) closed MRD", mrdWrapper.object.Name)
	mrdWrapper.Wrapped = nil
}

// Size returns the size of the wrapper for LRU cache accounting.
// Later, we can set to the number of MRD instances within the cache.
func (mrdWrapper *MultiRangeDownloaderWrapper) Size() uint64 {
	return 1
}
