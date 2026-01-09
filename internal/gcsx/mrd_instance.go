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

package gcsx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
)

// MrdInstance manages a pool of Multi-Range Downloader (MRD) instances for a
// single file inode. It handles the lifecycle of the MRD pool, including
// creation, destruction, and caching.
type MrdInstance struct {
	// mrdPool holds the pool of MultiRangeDownloader instances.
	mrdPool *MRDPool
	// inodeId is the ID of the file inode associated with this instance.
	inodeId fuseops.InodeID
	// object is the GCS object for which the downloaders are created.
	object atomic.Pointer[gcs.MinObject]
	// bucket is the GCS bucket containing the object.
	bucket gcs.Bucket
	// refCount tracks the number of active users of this instance.
	refCount int64
	// refCountMu protects access to refCount.
	refCountMu sync.Mutex
	// poolMu protects access to mrdPool.
	poolMu sync.RWMutex
	// mrdCache is a shared cache for inactive MrdInstance objects.
	mrdCache *lru.Cache
	// mrdConfig holds configuration for the MRD pool.
	mrdConfig cfg.MrdConfig
}

// NewMrdInstance creates a new MrdInstance for a given GCS object.
func NewMrdInstance(obj *gcs.MinObject, bucket gcs.Bucket, cache *lru.Cache, inodeId fuseops.InodeID, cfg cfg.MrdConfig) *MrdInstance {
	mrdInstance := MrdInstance{
		bucket:    bucket,
		mrdCache:  cache,
		inodeId:   inodeId,
		mrdConfig: cfg,
	}
	mrdInstance.object.Store(obj)
	return &mrdInstance
}

func (mi *MrdInstance) SetMinObject(minObj *gcs.MinObject) error {
	if minObj == nil {
		return fmt.Errorf("MrdInstance::SetMinObject: Missing MinObject")
	}
	mi.object.Store(minObj)
	return nil
}

func (mi *MrdInstance) GetMinObject() *gcs.MinObject {
	return mi.object.Load()
}

// getMRDEntry returns a valid MRDEntry from the pool.
// It ensures the pool is initialized and the returned entry is in a usable state,
// recreating it if necessary.
func (mi *MrdInstance) getMRDEntry() (*MRDEntry, error) {
	// Ensure the pool is initialized.
	if err := mi.ensureMRDPool(); err != nil {
		return nil, err
	}

	mi.poolMu.RLock()
	defer mi.poolMu.RUnlock()
	if mi.mrdPool == nil {
		return nil, fmt.Errorf("MrdInstance::getMRDEntry: MRDPool is nil")
	}

	entry := mi.mrdPool.Next()

	// Check if the entry is valid.
	entry.mu.RLock()
	isValid := entry.mrd != nil && entry.mrd.Error() == nil
	entry.mu.RUnlock()

	// Recreate the entry if it is not valid.
	if !isValid {
		if err := mi.mrdPool.RecreateMRD(entry, nil); err != nil {
			return nil, fmt.Errorf("MrdInstance::getMRDEntry: failed to recreate MRD: %w", err)
		}
	}
	return entry, nil
}

// Read downloads data from the GCS object into the provided buffer starting at the offset.
// It handles the details of selecting a valid MRD entry, locking it, and waiting for the async download to complete.
func (mi *MrdInstance) Read(ctx context.Context, p []byte, offset int64, metrics metrics.MetricHandle) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	entry, err := mi.getMRDEntry()
	if err != nil {
		return 0, err
	}

	// Prepare the buffer for the read operation.
	// bytes.NewBuffer creates a buffer using p as the initial content.
	// Reset() sets the length to 0 but keeps the capacity, allowing writes to fill p.
	buffer := bytes.NewBuffer(p)
	buffer.Reset()
	done := make(chan readResult, 1)

	// Take Read lock before using MRD for reading.
	entry.mu.RLock()
	if entry.mrd == nil {
		entry.mu.RUnlock()
		return 0, fmt.Errorf("MrdInstance::Read: mrd is nil")
	}

	start := time.Now()
	entry.mrd.Add(buffer, offset, int64(len(p)), func(offsetAddCallback int64, bytesReadAddCallback int64, e error) {
		done <- readResult{bytesRead: int(bytesReadAddCallback), err: e}
	})
	entry.mu.RUnlock()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case res := <-done:
		monitor.CaptureMultiRangeDownloaderMetrics(ctx, metrics, "MultiRangeDownloader::Add", start)
		if res.err != nil && res.err != io.EOF {
			return res.bytesRead, fmt.Errorf("Error in Add call: %w", res.err)
		}
		return res.bytesRead, res.err
	}
}

// ensureMRDPool ensures that the MRD pool is initialized. If the pool
// already exists, this function is a no-op.
func (mi *MrdInstance) ensureMRDPool() (err error) {
	// Return early if pool exists.
	mi.poolMu.RLock()
	if mi.mrdPool != nil {
		mi.poolMu.RUnlock()
		return
	}
	mi.poolMu.RUnlock()

	mi.poolMu.Lock()
	defer mi.poolMu.Unlock()

	// Re-check under write lock to handle race condition.
	if mi.mrdPool != nil {
		return
	}

	// Creating a new pool. Not reusing any handle while creating a new pool.
	mi.mrdPool, err = NewMRDPool(&MRDPoolConfig{PoolSize: int(mi.mrdConfig.PoolSize), object: mi.object.Load(), bucket: mi.bucket}, nil)
	if err != nil {
		err = fmt.Errorf("MrdInstance::ensureMRDPool Error in creating MRDPool: %w", err)
	}
	return
}

// RecreateMRD recreates the entire MRD pool. This is typically called by the
// file inode when the backing GCS object's generation changes, invalidating
// all existing downloader instances.
func (mi *MrdInstance) RecreateMRD() error {
	// Create the new pool first to avoid a period where mrdPool is nil.
	newPool, err := NewMRDPool(&MRDPoolConfig{
		PoolSize: int(mi.mrdConfig.PoolSize),
		object:   mi.object.Load(),
		bucket:   mi.bucket,
	}, nil)
	if err != nil {
		return fmt.Errorf("MrdInstance::RecreateMRD Error in recreating MRD: %w", err)
	}

	mi.poolMu.Lock()
	oldPool := mi.mrdPool
	mi.mrdPool = newPool
	mi.poolMu.Unlock()

	// Close the old pool after swapping.
	if oldPool != nil {
		oldPool.Close()
	}
	return nil
}

// Destroy closes all MRD instances in the pool and releases associated resources.
func (mi *MrdInstance) Destroy() {
	if mi.mrdCache != nil {
		mi.mrdCache.Erase(getKey(mi.inodeId))
	}
	mi.poolMu.Lock()
	defer mi.poolMu.Unlock()
	if mi.mrdPool != nil {
		// Delete the instance.
		mi.mrdPool.Close()
		mi.mrdPool = nil
	}
}

// getKey generates a unique key for the MrdInstance based on its inode ID.
func getKey(id fuseops.InodeID) string {
	return strconv.FormatUint(uint64(id), 10)
}

// IncrementRefCount increases the reference count for the MrdInstance. When the
// instance is actively used (refCount > 0), it is removed from the inactive
// MRD cache to prevent eviction.
func (mi *MrdInstance) IncrementRefCount() {
	mi.refCountMu.Lock()
	defer mi.refCountMu.Unlock()
	mi.refCount++

	if mi.refCount == 1 && mi.mrdCache != nil {
		// Remove from cache
		deletedEntry := mi.mrdCache.Erase(getKey(mi.inodeId))
		if deletedEntry != nil {
			logger.Tracef("MrdInstance::IncrementRefCount: MrdInstance (%s) erased from cache", mi.object.Load().Name)
		}
	}
}

// destroyEvictedCacheEntries is a helper function to destroy evicted MrdInstance objects.
// It handles type assertion and ensures that only truly inactive instances are destroyed.
// This function should not be called when refCountMu is held.
func destroyEvictedCacheEntries(evictedValues []lru.ValueType) {
	for _, instance := range evictedValues {
		mrdInstance, ok := instance.(*MrdInstance)
		if !ok {
			logger.Errorf("destroyEvictedCacheEntries: Invalid value type, expected *MrdInstance, got %T", instance)
		} else {
			// Check if the instance was resurrected.
			mrdInstance.refCountMu.Lock()
			if mrdInstance.refCount > 0 {
				mrdInstance.refCountMu.Unlock()
				continue
			}
			// Safe to destroy. Hold refCountMu to prevent concurrent resurrection.
			mrdInstance.Destroy()
			mrdInstance.refCountMu.Unlock()
		}
	}
}

// DecrementRefCount decreases the reference count. When the count drops to zero, the
// instance is considered inactive and is added to the LRU cache for potential
// reuse. If the cache is full, this may trigger the eviction and closure of the
// least recently used MRD instances.
func (mi *MrdInstance) DecrementRefCount() {
	mi.refCountMu.Lock()
	defer mi.refCountMu.Unlock()

	if mi.refCount <= 0 {
		logger.Errorf("MrdInstance::DecrementRefCount: Refcount cannot be negative")
		return
	}

	mi.refCount--
	// Do nothing if MRDInstance is in use or cache is not enabled.
	if mi.refCount > 0 || mi.mrdCache == nil {
		return
	}

	// Add to cache.
	// Lock order: refCountMu -> cache.mu -> poolMu (via Size() inside Insert).
	// This is a safe order.
	evictedValues, err := mi.mrdCache.Insert(getKey(mi.inodeId), mi)
	if err != nil {
		logger.Errorf("MrdInstance::DecrementRefCount: Failed to insert MrdInstance for object (%s) into cache, destroying immediately: %v", mi.object.Load().Name, err)
		// The instance could not be inserted into the cache. Since the refCount is 0,
		// we must destroy it now to prevent it from being leaked.
		mi.Destroy()
		return
	}
	logger.Tracef("MrdInstance::DecrementRefCount: MrdInstance for object (%s) added to cache", mi.object.Load().Name)

	// Do not proceed if no eviction happened.
	if evictedValues == nil {
		return
	}

	// Evict outside all locks.
	mi.refCountMu.Unlock()
	destroyEvictedCacheEntries(evictedValues)
	// Reacquire the lock ensuring safe defer's Unlock.
	mi.refCountMu.Lock()
}

// Size returns the number of active MRDs.
func (mi *MrdInstance) Size() uint64 {
	mi.poolMu.RLock()
	defer mi.poolMu.RUnlock()
	if mi.mrdPool != nil {
		return mi.mrdPool.Size()
	}
	return 0
}
