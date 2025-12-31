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

type MrdInstance struct {
	// This shoudl be the actual mrd instance or the mrd pool
	mrdPool *MRDPool
	// object associated with the mrd instance
	inodeId   fuseops.InodeID
	object    *gcs.MinObject
	bucket    gcs.Bucket
	refCount  int64
	mu        sync.RWMutex
	mrdCache  *lru.Cache
	mrdConfig cfg.MrdConfig
}

func NewMrdInstance(obj *gcs.MinObject, bucket gcs.Bucket, cache *lru.Cache, inodeId fuseops.InodeID, cfg cfg.MrdConfig) MrdInstance {
	return MrdInstance{
		object:    obj,
		bucket:    bucket,
		mrdCache:  cache,
		inodeId:   inodeId,
		mrdConfig: cfg,
	}
}

func (mi *MrdInstance) GetMRDEntry() *MRDEntry {
	mi.mu.RLock()
	defer mi.mu.RUnlock()
	if mi.mrdPool != nil {
		return mi.mrdPool.Next()
	}
	return nil
}

func (mi *MrdInstance) EnsureMrdInstance() {
	var handle []byte
	mi.mu.Lock()
	defer mi.mu.Unlock()
	if mi.mrdPool != nil {
		handle = mi.mrdPool.Close()
	}
	var err error
	mi.mrdPool, err = NewMRDPool(&MRDPoolConfig{PoolSize: int(mi.mrdConfig.PoolSize), object: mi.object, bucket: mi.bucket, Handle: handle}, handle)
	if err != nil {
		logger.Errorf("MrdInstance::EnsureMrdInstance Error in creating MRDPool")
	}
}

func (mi *MrdInstance) RecreateMRDEntry(entry *MRDEntry) {
	mi.mu.RLock()
	defer mi.mu.RUnlock()
	if mi.mrdPool != nil {
		mi.mrdPool.RecreateMRD(entry, nil)
	}
}

// This will be called by fileInode, when the minObject generation changes
func (mi *MrdInstance) RecreateMRD(object *gcs.MinObject) {
	mi.Destroy()
	mi.EnsureMrdInstance()
}

func (mi *MrdInstance) Destroy() {
	// delete the instance
	mi.mu.Lock()
	if mi.mrdPool != nil {
		mi.mrdPool.Close()
	}
	mi.mrdPool = nil
	mi.mu.Unlock()
}

func (mi *MrdInstance) IncrementRefCount() {
	// Add to cache
	mi.mu.Lock()
	defer mi.mu.Unlock()
	mi.refCount++

	if mi.refCount == 1 && mi.mrdCache != nil {
		mi.mrdCache.Erase(strconv.FormatUint(uint64(mi.inodeId), 10))
	}
}

func (mi *MrdInstance) DecRefCount() {
	// Delete from cache
	mi.mu.Lock()
	defer mi.mu.Unlock()
	mi.refCount--
	if mi.refCount < 0 {
		logger.Errorf("MrdInstance::DecRefCount: Invalid refcount")
		return
	}
	if mi.refCount > 0 || mi.mrdPool == nil {
		return
	}

	mi.mrdCache.Insert(strconv.FormatUint(uint64(mi.inodeId), 10), mi.mrdPool)
}
