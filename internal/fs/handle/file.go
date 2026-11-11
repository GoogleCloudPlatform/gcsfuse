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

	"context"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx/kernel_readers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx/read_manager"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workloadinsight"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"golang.org/x/sync/semaphore"
)

type FileHandle struct {
	inode *inode.FileInode

	mu locker.RWLocker

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

	// kernelReader is a reader that uses a kernel-optimized approach to read data
	// from a GCS object based on the bucket type (MRD for Zonal, kernelRangeReader for Regional).
	kernelReader gcsx.Reader
	// fileCacheHandler is used to get file cache handle and read happens using that.
	// This will be nil if the file cache is disabled.
	// Exactly one of these should be set:
	fileCacheHandler        *file.CacheHandler
	SharedChunkCacheManager *file.SharedChunkCacheManager

	// cacheFileForRangeRead is also valid for cache workflow, if true, object content
	// will be downloaded for random reads as well too.
	cacheFileForRangeRead bool
	metricHandle          metrics.MetricHandle
	traceHandle           tracing.TraceHandle

	// openMode is used to store the mode in which the file is opened.
	openMode util.OpenMode

	// Mount configuration.
	config *cfg.Config

	// bufferedReadWorkerPool is used to execute download tasks for buffered reads.
	bufferedReadWorkerPool workerpool.WorkerPool

	// globalMaxReadBlocksSem is a semaphore that limits the total number of blocks
	// that can be allocated for buffered read across all files in the file system.
	globalMaxReadBlocksSem *semaphore.Weighted

	// HandleID is an opaque 64-bit number used to create this File Handle, used for logging.
	handleID fuseops.HandleID
}

// LOCKS_REQUIRED(fh.inode.mu)
func NewFileHandle(
	inode *inode.FileInode,
	fileCacheHandler *file.CacheHandler,
	sharedChunkCacheManager *file.SharedChunkCacheManager,
	cacheFileForRangeRead bool,
	metricHandle metrics.MetricHandle,
	traceHandle tracing.TraceHandle,
	openMode util.OpenMode,
	c *cfg.Config,
	bufferedReadWorkerPool workerpool.WorkerPool,
	globalMaxReadBlocksSem *semaphore.Weighted,
	handleID fuseops.HandleID,
) (fh *FileHandle) {
	fh = &FileHandle{
		inode:                   inode,
		fileCacheHandler:        fileCacheHandler,
		SharedChunkCacheManager: sharedChunkCacheManager,
		cacheFileForRangeRead:   cacheFileForRangeRead,
		metricHandle:            metricHandle,
		traceHandle:             traceHandle,
		openMode:                openMode,
		config:                  c,
		bufferedReadWorkerPool:  bufferedReadWorkerPool,
		globalMaxReadBlocksSem:  globalMaxReadBlocksSem,
		handleID:                handleID,
	}

	if c.FileSystem.EnableKernelReader {
		fh.kernelReader = kernel_readers.NewKernelReader(inode.Bucket(), inode.GetKernelRangeReaderInstance(), inode.GetMRDInstance(), metricHandle)
	}

	fh.inode.RegisterFileHandle(fh.openMode.AccessMode() == util.ReadOnly)
	fh.mu = locker.NewRW("FileHandle", fh.checkInvariants)

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
	fh.inode.DeRegisterFileHandle(fh.openMode.AccessMode() == util.ReadOnly)
	fh.inode.Unlock()
	if fh.reader != nil {
		fh.reader.Destroy()
	}
	if fh.readManager != nil {
		fh.readManager.Destroy()
	}
	if fh.kernelReader != nil {
		fh.kernelReader.Destroy()
		fh.kernelReader = nil
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

// TryFastReadWithReadManager executes the steady-state read path when readManager
// is already initialized and valid, acquiring fh.mu.RLock() before fh.inode.Lock()
// to avoid the double inode lock/unlock round-trip per FUSE ReadFile op.
//
// LOCKS_EXCLUDED(fh.mu)
// LOCKS_EXCLUDED(fh.inode.mu)
func (fh *FileHandle) TryFastReadWithReadManager(ctx context.Context, req *gcsx.ReadRequest) (gcsx.ReadResponse, bool, error) {
	fh.mu.RLock()
	if fh.readManager == nil {
		fh.mu.RUnlock()
		return gcsx.ReadResponse{}, false, nil
	}
	fh.inode.Lock()
	if fh.inode.IsUsingStreamingWriteHandler() || !fh.inode.SourceGenerationIsAuthoritative() || !fh.isValidReadManagerRLocked() {
		fh.inode.Unlock()
		fh.mu.RUnlock()
		return gcsx.ReadResponse{}, false, nil
	}
	if err := fh.inode.CacheEnsureContent(ctx); err != nil {
		fh.inode.Unlock()
		fh.mu.RUnlock()
		return gcsx.ReadResponse{}, true, fmt.Errorf("failed to ensure inode content: %w", err)
	}
	if !fh.inode.SourceGenerationIsAuthoritative() || !fh.isValidReadManagerRLocked() {
		fh.inode.Unlock()
		fh.mu.RUnlock()
		return gcsx.ReadResponse{}, false, nil
	}
	fh.inode.Unlock()
	defer fh.mu.RUnlock()

	req.SkipSizeChecks = fh.shouldSkipSizeChecks(req)

	readResponse, err := fh.readManager.ReadAt(ctx, req)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf("fh.readManager.ReadAt: %w", err)
	}
	readResponse, err = normalizeReadResponse(readResponse, err)
	return readResponse, true, err
}

// ReadWithReadManager reads data at the given offset using the read manager if available,
// falling back to inode.Read otherwise. It may be more efficient than directly calling inode.Read.
//
// LOCKS_REQUIRED(fh.inode.mu)
// UNLOCK_FUNCTION(fh.inode.mu)
func (fh *FileHandle) ReadWithReadManager(ctx context.Context, req *gcsx.ReadRequest, sequentialReadSizeMb int32) (gcsx.ReadResponse, error) {
	// If content cache enabled, CacheEnsureContent forces the file handler to fall through to the inode
	// and fh.inode.SourceGenerationIsAuthoritative() will return false
	if err := fh.inode.CacheEnsureContent(ctx); err != nil {
		fh.inode.Unlock()
		return gcsx.ReadResponse{}, fmt.Errorf("failed to ensure inode content: %w", err)
	}

	if err := fh.ensureStreamingWritesFlushedLocked(ctx); err != nil {
		fh.inode.Unlock()
		return gcsx.ReadResponse{}, err
	}

	if !fh.inode.SourceGenerationIsAuthoritative() {
		// Read from inode if source generation is not authoratative
		defer fh.inode.Unlock()
		return normalizeReadResponse(fh.inode.Read(ctx, req))
	}

	fh.lockHandleAndRelockInode(true)

	// If the inode is dirty, there's nothing we can do. Throw away our readManager if
	// we have one & create a new readManager
	if !fh.hasPendingStreamingWritesLocked() && fh.inode.SourceGenerationIsAuthoritative() && fh.isValidReadManagerRLocked() {
		fh.inode.Unlock()
	} else {
		// Acquire a RWLock on file handle as we will update readManager
		fh.unlockHandleAndInode(true)
		fh.mu.Lock()
		fh.inode.Lock()

		if err := fh.ensureStreamingWritesFlushedLocked(ctx); err != nil {
			fh.mu.Unlock()
			fh.inode.Unlock()
			return gcsx.ReadResponse{}, err
		}

		if !fh.inode.SourceGenerationIsAuthoritative() {
			defer fh.mu.Unlock()
			defer fh.inode.Unlock()
			return normalizeReadResponse(fh.inode.Read(ctx, req))
		}

		if !fh.isValidReadManager() {
			minObj := fh.inode.Source()
			bucket := fh.inode.Bucket()
			mrdWrapper := fh.inode.MRDWrapper

			fh.destroyReadManager()
			// Create a new read manager for the current inode state.
			fh.readManager = read_manager.NewReadManager(minObj, bucket, &read_manager.ReadManagerConfig{
				SequentialReadSizeMB:    sequentialReadSizeMb,
				FileCacheHandler:        fh.fileCacheHandler,
				SharedChunkCacheManager: fh.SharedChunkCacheManager,
				CacheFileForRangeRead:   fh.cacheFileForRangeRead,
				MetricHandle:            fh.metricHandle,
				TraceHandle:             fh.traceHandle,
				MrdWrapper:              mrdWrapper,
				Config:                  fh.config,
				GlobalMaxBlocksSem:      fh.globalMaxReadBlocksSem,
				WorkerPool:              fh.bufferedReadWorkerPool,
				HandleID:                fh.handleID,
				InitialOffset:           req.Offset,
			})

			// Override the read-manager with visual-read-manager (a wrapper over read_manager with visualizer) if configured.
			if fh.config.WorkloadInsight.Visualize {
				if renderer, err := workloadinsight.NewRenderer(); err == nil {
					fh.readManager = read_manager.NewVisualReadManager(fh.readManager, renderer, fh.config.WorkloadInsight)
				} else {
					logger.Warnf("Failed to construct workload insight visualizer: %v", err)
				}
			}
		}

		// Release inode lock, release RWLock, and take RLock on file handle again.
		fh.inode.Unlock()
		fh.mu.Unlock()
		fh.mu.RLock()
	}
	defer fh.mu.RUnlock()

	// For unfinalized objects, we may need to read past the known size.
	// Update the request to skip size checks if required.
	req.SkipSizeChecks = fh.shouldSkipSizeChecks(req)

	// Use the readManager to read data.
	var readResponse gcsx.ReadResponse
	readResponse, err := fh.readManager.ReadAt(ctx, req)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf("fh.readManager.ReadAt: %w", err)
	}
	return normalizeReadResponse(readResponse, err)
}

// ReadWithKernelReader reads data at the given offset using the kernel reader.
//
// LOCKS_REQUIRED(fh.inode.mu)
// UNLOCK_FUNCTION(fh.inode.mu)
func (fh *FileHandle) ReadWithKernelReader(ctx context.Context, req *gcsx.ReadRequest) (gcsx.ReadResponse, error) {
	if !fh.inode.SourceGenerationIsAuthoritative() {
		// Read from inode if source generation is not authoritative.
		defer fh.inode.Unlock()
		return normalizeReadResponse(fh.inode.Read(ctx, req))
	}
	fh.inode.Unlock()

	fh.mu.RLock()
	defer fh.mu.RUnlock()

	if fh.kernelReader == nil {
		return gcsx.ReadResponse{}, errors.New("kernelReader is not initialized")
	}

	return normalizeReadResponse(fh.kernelReader.ReadAt(ctx, req))
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
		fh.inode.Unlock()
		return nil, 0, fmt.Errorf("failed to ensure inode content: %w", err)
	}

	if err = fh.ensureStreamingWritesFlushedLocked(ctx); err != nil {
		fh.inode.Unlock()
		return nil, 0, err
	}

	// If the inode is dirty, there's nothing we can do. Throw away our reader if
	// we have one.
	if !fh.inode.SourceGenerationIsAuthoritative() {
		defer fh.inode.Unlock()
		res, err := fh.inode.Read(ctx, &gcsx.ReadRequest{Buffer: dst, Offset: offset})
		return dst, res.Size, err
	}

	fh.lockHandleAndRelockInode(true)

	if !fh.hasPendingStreamingWritesLocked() && fh.inode.SourceGenerationIsAuthoritative() && fh.isValidReaderRLocked() {
		fh.inode.Unlock()
	} else {
		// Acquire a RWLock on file handle as we will update reader
		fh.unlockHandleAndInode(true)
		fh.mu.Lock()
		fh.inode.Lock()

		if err = fh.ensureStreamingWritesFlushedLocked(ctx); err != nil {
			fh.mu.Unlock()
			fh.inode.Unlock()
			return nil, 0, err
		}

		if !fh.inode.SourceGenerationIsAuthoritative() {
			defer fh.mu.Unlock()
			defer fh.inode.Unlock()
			res, err := fh.inode.Read(ctx, &gcsx.ReadRequest{Buffer: dst, Offset: offset})
			return dst, res.Size, err
		}

		if !fh.isValidReader() {
			minObj := fh.inode.Source()
			bucket := fh.inode.Bucket()
			mrdWrapper := fh.inode.MRDWrapper

			fh.destroyReader()
			fh.reader = gcsx.NewRandomReader(minObj, bucket, sequentialReadSizeMb, fh.fileCacheHandler, fh.cacheFileForRangeRead, fh.metricHandle, fh.traceHandle, mrdWrapper, fh.config, fh.handleID)
		}

		// Release inode lock, release RWLock, and take RLock on file handle again
		fh.inode.Unlock()
		fh.mu.Unlock()
		fh.mu.RLock()
	}
	defer fh.mu.RUnlock()

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

// isValidReadManagerRLocked validates & returns whether the current readManager
// is valid and its object size matches the inode source generation size without mutating state.
// LOCKS_REQUIRED(fh.mu.RLock)
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) isValidReadManagerRLocked() bool {
	return fh.readManager != nil &&
		fh.readManager.Object().Generation == fh.inode.SourceGeneration().Object &&
		fh.readManager.Object().Size == fh.inode.SourceGeneration().Size
}

// isValidReadManager is a helper function which validates & returns whether the
// current readManager is valid or not.
// LOCKS_REQUIRED(fh.mu)
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) isValidReadManager() bool {
	// If we already have a readManager, and it's at the appropriate generation, we
	// can use it otherwise we must throw it away.
	if fh.readManager != nil && fh.readManager.Object().Generation == fh.inode.SourceGeneration().Object {
		// Update reader object size to source object size.
		if fh.readManager.Object().Size != fh.inode.SourceGeneration().Size {
			fh.readManager.Object().Size = fh.inode.SourceGeneration().Size
		}
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

// isValidReaderRLocked validates & returns whether the current reader
// is valid and its object size matches the inode source generation size without mutating state.
// LOCKS_REQUIRED(fh.mu.RLock)
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) isValidReaderRLocked() bool {
	return fh.reader != nil &&
		fh.reader.Object().Generation == fh.inode.SourceGeneration().Object &&
		fh.reader.Object().Size == fh.inode.SourceGeneration().Size
}

// isValidReader is a helper function which validates & returns whether the
// current reader is valid or not.
// LOCKS_REQUIRED(fh.mu)
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) isValidReader() bool {
	// If we already have a reader, and it's at the appropriate generation, we
	// can use it otherwise we must throw it away.
	if fh.reader != nil && fh.reader.Object().Generation == fh.inode.SourceGeneration().Object {
		// Update reader object size to source object size.
		if fh.reader.Object().Size != fh.inode.SourceGeneration().Size {
			fh.reader.Object().Size = fh.inode.SourceGeneration().Size
		}
		return true
	}
	return false
}

func (fh *FileHandle) OpenMode() util.OpenMode {
	return fh.openMode
}

// shouldSkipSizeChecks determines if the read request should skip object size checks.
// This is true for direct I/O reads on unfinalized objects that extend beyond
// the known object size. This allows reading from an object that is being
// concurrently written to.
func (fh *FileHandle) shouldSkipSizeChecks(req *gcsx.ReadRequest) bool {
	if !fh.inode.Bucket().BucketType().IsRapid() {
		return false
	}
	if !fh.readManager.Object().IsUnfinalized() {
		return false
	}
	if !fh.OpenMode().IsDirect() {
		return false
	}
	if req.Offset < 0 {
		return false
	}
	if req.Offset+int64(len(req.Buffer)) <= int64(fh.readManager.Object().Size) {
		return false
	}

	return true
}

func normalizeReadResponse(readResponse gcsx.ReadResponse, err error) (gcsx.ReadResponse, error) {
	switch {
	case errors.Is(err, io.EOF):
		if err != io.EOF {
			logger.Warnf("Unexpected EOF error encountered while reading, err: %v type: %T ", err, err)
		}
		if readResponse.Size > 0 {
			return readResponse, io.EOF
		}
		if readResponse.Callback != nil {
			readResponse.Callback()
		}
		return gcsx.ReadResponse{}, io.EOF
	case err != nil:
		if readResponse.Callback != nil {
			readResponse.Callback()
		}
		return gcsx.ReadResponse{}, err
	default:
		return readResponse, nil
	}
}

// hasPendingStreamingWritesLocked reports whether fh.inode has active streaming writes
// that require flushing or syncing before reading under the current inode lock.
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) hasPendingStreamingWritesLocked() bool {
	if !fh.inode.IsUsingStreamingWriteHandler() {
		return false
	}
	if fh.inode.Bucket().BucketType().WriteStrategy() == gcs.WriteStrategyAppendable && fh.config != nil && fh.config.Write.FinalizeFileForRapid {
		return fh.inode.IsLocal() || fh.inode.Source().Size != fh.inode.SourceGeneration().Size
	}
	return true
}

// ensureStreamingWritesFlushedLocked flushes or syncs pending streaming writes
// initiated while fh.inode.mu was temporarily released during handle lock acquisition.
// LOCKS_REQUIRED(fh.inode.mu)
func (fh *FileHandle) ensureStreamingWritesFlushedLocked(ctx context.Context) error {
	if !fh.hasPendingStreamingWritesLocked() {
		return nil
	}
	if fh.inode.Bucket().BucketType().WriteStrategy() == gcs.WriteStrategyAppendable && fh.config != nil && fh.config.Write.FinalizeFileForRapid {
		_, err := fh.inode.SyncPendingStreamingWrites(ctx)
		return err
	}
	return fh.inode.Flush(ctx, &inode.WriteContext{
		Config:             fh.config,
		GlobalMaxBlocksSem: fh.globalMaxReadBlocksSem,
		TraceHandle:        fh.traceHandle,
		MetricHandle:       fh.metricHandle,
	})
}
