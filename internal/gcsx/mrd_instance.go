package gcsx

import (
	"strconv"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
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
	object *gcs.MinObject
	// bucket is the GCS bucket containing the object.
	bucket gcs.Bucket
	// refCount tracks the number of active users of this instance.
	refCount int64
	// mu protects access to the mrdPool and refCount.
	mu sync.RWMutex
	// mrdCache is a shared cache for inactive MrdInstance objects.
	mrdCache *lru.Cache
	// mrdConfig holds configuration for the MRD pool.
	mrdConfig cfg.MrdConfig
}

// NewMrdInstance creates a new MrdInstance for a given GCS object.
func NewMrdInstance(obj *gcs.MinObject, bucket gcs.Bucket, cache *lru.Cache, inodeId fuseops.InodeID, cfg cfg.MrdConfig) MrdInstance {
	return MrdInstance{
		object:    obj,
		bucket:    bucket,
		mrdCache:  cache,
		inodeId:   inodeId,
		mrdConfig: cfg,
	}
}

// GetMRDEntry returns the next available MRDEntry from the pool using a
// round-robin strategy. It is thread-safe.
func (mi *MrdInstance) GetMRDEntry() *MRDEntry {
	mi.mu.RLock()
	defer mi.mu.RUnlock()
	if mi.mrdPool != nil {
		return mi.mrdPool.Next()
	}
	return nil
}

// EnsureMrdInstance ensures that the MRD pool is initialized. If the pool
// already exists, this function is a no-op.
func (mi *MrdInstance) EnsureMrdInstance() {
	mi.mu.Lock()
	defer mi.mu.Unlock()

	// Return early if pool exists.
	if mi.mrdPool != nil {
		return
	}

	// Creating a new pool. Not reusing any handle while creating a new pool.
	var err error
	mi.mrdPool, err = NewMRDPool(&MRDPoolConfig{PoolSize: int(mi.mrdConfig.PoolSize), object: mi.object, bucket: mi.bucket}, nil)
	if err != nil {
		logger.Errorf("MrdInstance::EnsureMrdInstance Error in creating MRDPool")
	}
}

// RecreateMRDEntry recreates a specific, potentially failed, entry in the MRD pool.
func (mi *MrdInstance) RecreateMRDEntry(entry *MRDEntry) {
	mi.mu.RLock()
	defer mi.mu.RUnlock()
	if mi.mrdPool != nil {
		mi.mrdPool.RecreateMRD(entry, nil)
	}
}

// RecreateMRD recreates the entire MRD pool. This is typically called by the
// file inode when the backing GCS object's generation changes, invalidating
// all existing downloader instances.
func (mi *MrdInstance) RecreateMRD(object *gcs.MinObject) {
	mi.Destroy()
	mi.EnsureMrdInstance()
}

// Destroy closes all MRD instances in the pool and releases associated resources.
func (mi *MrdInstance) Destroy() {
	// delete the instance
	mi.mu.Lock()
	defer mi.mu.Unlock()
	mi.destroyLocked()
}

// destroyLocked closes the MRD pool while holding the lock.
func (mi *MrdInstance) destroyLocked() {
	if mi.mrdPool != nil {
		mi.mrdPool.Close()
		mi.mrdPool = nil
	}
}

// IncrementRefCount increases the reference count for the MrdInstance. When the
// instance is actively used (refCount > 0), it is removed from the inactive
// MRD cache to prevent eviction.
func (mi *MrdInstance) IncrementRefCount() {
	mi.mu.Lock()
	mi.refCount++
	mi.mu.Unlock()

	mi.mu.RLock()
	defer mi.mu.RUnlock()
	if mi.refCount == 1 && mi.mrdCache != nil {
		// Remove from cache
		deletedEntry := mi.mrdCache.Erase(strconv.FormatUint(uint64(mi.inodeId), 10))
		if deletedEntry != nil {
			logger.Tracef("MrdInstance (%s) erased from cache", mi.object.Name)
		}
	}
}

// DecRefCount decreases the reference count. When the count drops to zero, the
// instance is considered inactive and is added to the LRU cache for potential
// reuse. If the cache is full, this may trigger the eviction and closure of the
// least recently used MRD instances.
func (mi *MrdInstance) DecRefCount() {
	mi.mu.Lock()
	defer mi.mu.Unlock()
	mi.refCount--
	if mi.refCount < 0 {
		logger.Errorf("MrdInstance::DecRefCount: Invalid refcount")
		return
	}
	if mi.refCount > 0 || mi.mrdCache == nil {
		return
	}

	// Add to cache & evict outside all locks to avoid deadlock.
	mi.mu.Unlock()
	evictedValues, err := mi.mrdCache.Insert(strconv.FormatUint(uint64(mi.inodeId), 10), mi)
	if err != nil {
		logger.Errorf("failed to insert MrdInstance for object (%s) into cache: %v", mi.object.Name, err)
		// Reacquire the lock ensuring safe defer's Unlock.
		mi.mu.Lock()
		return
	}
	logger.Tracef("MrdInstance for object (%s) added to cache", mi.object.Name)

	for _, instance := range evictedValues {
		mrdInstance, ok := instance.(*MrdInstance)
		if !ok {
			logger.Errorf("invalid value type, expected MrdInstance, got %T", mrdInstance)
		} else {
			// Check if the instance was resurrected while we were unlocked.
			mrdInstance.mu.Lock()
			if mrdInstance.refCount > 0 {
				mrdInstance.mu.Unlock()
				continue
			}
			mrdInstance.destroyLocked()
			mrdInstance.mu.Unlock()
		}
	}
	// Reacquire the lock ensuring safe defer's Unlock.
	mi.mu.Lock()
}

// Size returns the number of active MRDs.
func (mi *MrdInstance) Size() uint64 {
	mi.mu.RLock()
	defer mi.mu.RUnlock()
	if mi.mrdPool != nil {
		return mi.mrdPool.Size()
	}
	return 1
}
