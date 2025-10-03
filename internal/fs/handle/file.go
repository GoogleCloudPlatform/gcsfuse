// Copyright 2015 Google LLC
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

package handle

import (
	"errors"
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx/read_manager"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/syncutil"
	"golang.org/x/net/context"
	"golang.org/x/sync/semaphore"
)

type FileHandle struct {
	inode *inode.FileInode

	mu syncutil.InvariantMutex

	// A random reader configured to some (potentially previous) generation of
	// the object backing the inode, or nil.
	//
	// INVARIANT: If reader != nil, reader.CheckInvariants() doesn't panic.
	//
	// GUARDED_BY(mu)
	reader gcsx.RandomReader

	// A readManager configured to some (potentially previous) generation of
	// the object backing the inode, or nil.
	//
	// INVARIANT: If readManager != nil, readManager.CheckInvariants() doesn't panic.
	//
	// GUARDED_BY(mu)
	readManager gcsx.ReadManager

	// fileCacheHandler is used to get file cache handle and read happens using that.
	// This will be nil if the file cache is disabled.
	fileCacheHandler *file.CacheHandler

	// cacheFileForRangeRead is also valid for cache workflow, if true, object content
	// will be downloaded for random reads as well too.
	cacheFileForRangeRead bool
	metricHandle          metrics.MetricHandle
	// openMode is used to store the mode in which the file is opened.
	openMode util.OpenMode

	// Mount configuration.
	config *cfg.Config

	// bufferedReadWorkerPool is used to execute download tasks for buffered reads.
	bufferedReadWorkerPool workerpool.WorkerPool

	// globalMaxReadBlocksSem is a semaphore that limits the total number of blocks
	// that can be allocated for buffered read across all files in the file system.
	globalMaxReadBlocksSem *semaphore.Weighted
}

// LOCKS_REQUIRED(fh.inode.mu)
func NewFileHandle(inode *inode.FileInode, fileCacheHandler *file.CacheHandler, cacheFileForRangeRead bool, metricHandle metrics.MetricHandle, openMode util.OpenMode, c *cfg.Config, bufferedReadWorkerPool workerpool.WorkerPool, globalMaxReadBlocksSem *semaphore.Weighted) (fh *FileHandle) {
	fh = &FileHandle{
		inode:                  inode,
		fileCacheHandler:       fileCacheHandler,
		cacheFileForRangeRead:  cacheFileForRangeRead,
		metricHandle:           metricHandle,
		openMode:               openMode,
		config:                 c,
		bufferedReadWorkerPool: bufferedReadWorkerPool,
		globalMaxReadBlocksSem: globalMaxReadBlocksSem,
	}

	fh.inode.RegisterFileHandle(fh.openMode == util.Read)
	fh.mu = syncutil.NewInvariantMutex(fh.checkInvariants)

	return
}

// Destroy any resources associated with the handle, which must not be used
// again.
// LOCKS_REQUIRED(fh.mu)
// LOCK_FUNCTION(fh.inode.mu)
// UNLOCK_FUNCTION(fh.inode.mu)
func (fh *FileHandle) Destroy() {
	// Deregister the fileHandle with the inode.
	fh.inode.Lock()
	fh.inode.DeRegisterFileHandle(fh.openMode == util.Read)
	fh.inode.Unlock()
	if fh.reader != nil {
		fh.reader.Destroy()
	}
	if fh.readManager != nil {
		fh.readManager.Destroy()
	}
}

// Inode returns the inode backing this handle.
func (fh *FileHandle) Inode() *inode.FileInode {
	return fh.inode
}

func (fh *FileHandle) Lock() {
	fh.mu.Lock()
}

func (fh *FileHandle) Unlock() {
	fh.mu.Unlock()
}

// lockHandleAndRelockInode is a helper function which locks fh.mu maintaing the locking
// order, i.e. it first unlocks inode lock, then locks fh.mu (RLock or RWlock) and then
// relocks inode lock.
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) lockHandleAndRelockInode(rLock bool) {
	if rLock {
		fh.inode.Unlock()
		fh.mu.RLock()
		fh.inode.Lock()
	} else {
		fh.inode.Unlock()
		fh.mu.Lock()
		fh.inode.Lock()
	}
}

// unlockHandleAndInode is a helper function which unlocks fh.inode.mu & fh.mu in order.
// LOCKS_REQUIRED(fh.mu)
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) unlockHandleAndInode(rLock bool) {
	if rLock {
		fh.inode.Unlock()
		fh.mu.RUnlock()
	} else {
		fh.inode.Unlock()
		fh.mu.Unlock()
	}
}

// ReadWithReadManager reads data at the given offset using the read manager if available,
// falling back to inode.Read otherwise. It may be more efficient than directly calling inode.Read.
//
// LOCKS_REQUIRED(fh.inode.mu)
// UNLOCK_FUNCTION(fh.inode.mu)
func (fh *FileHandle) ReadWithReadManager(ctx context.Context, dst []byte, offset int64, sequentialReadSizeMb int32) ([]byte, int, error) {
	// If content cache enabled, CacheEnsureContent forces the file handler to fall through to the inode
	// and fh.inode.SourceGenerationIsAuthoritative() will return false
	if err := fh.inode.CacheEnsureContent(ctx); err != nil {
		return nil, 0, fmt.Errorf("failed to ensure inode content: %w", err)
	}

	if !fh.inode.SourceGenerationIsAuthoritative() {
		// Read from inode if source generation is not authoratative
		defer fh.inode.Unlock()
		n, err := fh.inode.Read(ctx, dst, offset)
		return dst, n, err
	}

	fh.lockHandleAndRelockInode(true)
	defer fh.mu.RUnlock()

	// If the inode is dirty, there's nothing we can do. Throw away our readManager if
	// we have one & create a new readManager
	if fh.isValidReadManager() {
		fh.inode.Unlock()
	} else {
		minObj := fh.inode.Source()
		bucket := fh.inode.Bucket()
		mrdWrapper := &fh.inode.MRDWrapper

		// Acquire a RWLock on file handle as we will update readManager
		fh.unlockHandleAndInode(true)
		fh.mu.Lock()

		fh.destroyReadManager()
		// Create a new read manager for the current inode state.
		fh.readManager = read_manager.NewReadManager(minObj, bucket, &read_manager.ReadManagerConfig{
			SequentialReadSizeMB:  sequentialReadSizeMb,
			FileCacheHandler:      fh.fileCacheHandler,
			CacheFileForRangeRead: fh.cacheFileForRangeRead,
			MetricHandle:          fh.metricHandle,
			MrdWrapper:            mrdWrapper,
			Config:                fh.config,
			GlobalMaxBlocksSem:    fh.globalMaxReadBlocksSem,
			WorkerPool:            fh.bufferedReadWorkerPool,
		})

		// Release RWLock and take RLock on file handle again. Inode lock is not needed now.
		fh.mu.Unlock()
		fh.mu.RLock()
	}

	// Use the readManager to read data.
	var readerResponse gcsx.ReaderResponse
	var err error
	readerResponse, err = fh.readManager.ReadAt(ctx, dst, offset)
	switch {
	case errors.Is(err, io.EOF):
		if err != io.EOF {
			logger.Warnf("Unexpected EOF error encountered while reading, err: %v type: %T ", err, err)
		}
		return nil, 0, io.EOF

	case err != nil:
		return nil, 0, fmt.Errorf("fh.readManager.ReadAt: %w", err)
	}

	return readerResponse.DataBuf, readerResponse.Size, nil
}

// Equivalent to locking fh.Inode() and calling fh.Inode().Read, but may be
// more efficient.
//
// LOCKS_REQUIRED(fh.inode.mu)
// UNLOCK_FUNCTION(fh.inode.mu)
func (fh *FileHandle) Read(ctx context.Context, dst []byte, offset int64, sequentialReadSizeMb int32) (output []byte, n int, err error) {
	// If content cache enabled, CacheEnsureContent forces the file handler to fall through to the inode
	// and fh.inode.SourceGenerationIsAuthoritative() will return false
	err = fh.inode.CacheEnsureContent(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to ensure inode content: %w", err)
	}

	// If the inode is dirty, there's nothing we can do. Throw away our reader if
	// we have one.
	if !fh.inode.SourceGenerationIsAuthoritative() {
		defer fh.inode.Unlock()
		n, err = fh.inode.Read(ctx, dst, offset)
		return dst, n, err
	}

	fh.lockHandleAndRelockInode(true)
	defer fh.mu.RUnlock()

	if fh.isValidReader() {
		fh.inode.Unlock()
	} else {
		minObj := fh.inode.Source()
		bucket := fh.inode.Bucket()
		mrdWrapper := &fh.inode.MRDWrapper

		// Acquire a RWLock on file handle as we will update reader
		fh.unlockHandleAndInode(true)
		fh.mu.Lock()

		fh.destroyReader()
		// Attempt to create an appropriate reader.
		fh.reader = gcsx.NewRandomReader(minObj, bucket, sequentialReadSizeMb, fh.fileCacheHandler, fh.cacheFileForRangeRead, fh.metricHandle, mrdWrapper, fh.config)

		// Release RWLock and take RLock on file handle again
		fh.mu.Unlock()
		fh.mu.RLock()
	}

	// Use the reader to read data.
	var objectData gcsx.ObjectData
	objectData, err = fh.reader.ReadAt(ctx, dst, offset)
	switch {
	case errors.Is(err, io.EOF):
		if err != io.EOF {
			logger.Warnf("Unexpected EOF error encountered while reading, err: %v type: %T ", err, err)
			err = io.EOF
		}
		return

	case err != nil:
		err = fmt.Errorf("fh.reader.ReadAt: %w", err)
		return
	}

	output = objectData.DataBuf
	n = objectData.Size
	return
}

// Adding the Write() method to fileHandle to be able to pass the fileOpenMode
// which is used for determining write path. For e.g. in case of append mode for
// unfinalized objects in zonal buckets, streaming writes is used.
// Note that the writes are still done at the inode level.
// LOCKS_REQUIRED(fh.inode)
func (fh *FileHandle) Write(ctx context.Context, data []byte, offset int64) (bool, error) {
	return fh.inode.Write(ctx, data, offset, fh.openMode)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// LOCKS_REQUIRED(fh.mu)
func (fh *FileHandle) checkInvariants() {
	// INVARIANT: If reader != nil, reader.CheckInvariants() doesn't panic.
	if fh.reader != nil {
		fh.reader.CheckInvariants()
	}

	// INVARIANT: If readManager != nil, readManager.CheckInvariants() doesn't panic.
	if fh.readManager != nil {
		fh.readManager.CheckInvariants()
	}
}

// destroyReadManager is a helper function to safely destroy the readManager & set it to nil.
// LOCKS_REQUIRED(fh.mu)
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) destroyReadManager() {
	if fh.readManager == nil {
		return
	}
	fh.readManager.Destroy()
	fh.readManager = nil
}

// isValidReadManager is a helper function which validates & returns whether the
// current readManager is valid or not.
// LOCKS_REQUIRED(fh.mu.RLock)
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) isValidReadManager() bool {
	// If we already have a readManager, and it's at the appropriate generation, we
	// can use it otherwise we must throw it away.
	if fh.readManager != nil && fh.readManager.Object().Generation == fh.inode.SourceGeneration().Object {
		// Update reader object size to source object size.
		fh.readManager.Object().Size = fh.inode.SourceGeneration().Size
		return true
	}
	return false
}

// destroyReader is a helper function to safely destroy the reader and set it to nil.
// LOCKS_REQUIRED(fh.mu)
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) destroyReader() {
	if fh.reader == nil {
		return
	}
	fh.reader.Destroy()
	fh.reader = nil
}

// isValidReader is a helper function which validates & returns whether the
// current reader is valid or not.
// LOCKS_REQUIRED(fh.mu.RLock)
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) isValidReader() bool {
	// If we already have a reader, and it's at the appropriate generation, we
	// can use it otherwise we must throw it away.
	if fh.reader != nil && fh.reader.Object().Generation == fh.inode.SourceGeneration().Object {
		// Update reader object size to source object size.
		fh.reader.Object().Size = fh.inode.SourceGeneration().Size
		return true
	}
	return false
}

func (fh *FileHandle) OpenMode() util.OpenMode {
	return fh.openMode
}
