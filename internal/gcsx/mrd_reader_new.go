package gcsx

import "sync"

type mrdReaderNew struct {
	mrdInstance MrdInstance
	mu          sync.RWMutex
}

func NewMrdReaderNew(instance MrdInstance) {}

func (mr *mrdReaderNew) ReadAt(p []byte, offset, end int64, objectSize int64) {
	if mr.mrdInstance.GetMRD == nil {
		mr.mu.Lock()
		mr.mrdInstance.EnsureMrdInstance()
		mr.mu.Unlock()
	}
	
	// first read, increment ref count

	mr.mu.RLock()
	// issue read request
	// Handle short read, recreate MRD if required
	mr.mu.RUnlock()
}

func (mr *mrdReaderNew) Close() {
	mr.mrdInstance.DecRefCount()
}
