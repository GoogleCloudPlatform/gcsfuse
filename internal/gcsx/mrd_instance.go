package gcsx

import (
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

type MrdInstance struct {
	// This shoudl be the actual mrd instance or the mrd pool
	wrapper MultiRangeDownloaderWrapper
	// object associated with the mrd instance
	object   *gcs.MinObject
	refCount int64
	mu       sync.RWMutex
	mrdCache *lru.Cache
}

func NewMrdInstance() MrdInstance {
	return MrdInstance{}
}

func (mi *MrdInstance) GetMRD() {
	// return the connection
}

func (mi *MrdInstance) EnsureMrdInstance() {
	mi.mu.Lock()
	// Create the instance
	mi.mu.Unlock()
}

// This will be called by fileInode, when the minObject generation changes
func (mi *MrdInstance) RecreateMRD(object *gcs.MinObject) {
	mi.Destroy()
	mi.EnsureMrdInstance()
}

func (mi *MrdInstance) Destroy() {
	// delete the instance
	//mi.wrapper = nil
}

func (mi *MrdInstance) IncrementRefCount() {
	// Add to cache
}

func (mi *MrdInstance) DecRefCount() {
	// Delete from cache
}
