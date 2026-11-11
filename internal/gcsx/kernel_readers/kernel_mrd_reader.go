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

package kernel_readers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	minPrefetchReadSize      = 256 * 1024
	initialPrefetchChunkSize = 2 * 1024 * 1024
	prefetchChunkSize        = 4 * 1024 * 1024
	maxPrefetchChunks        = 6
)

// KernelMRDReader is a reader that uses an MRD Instance to read data from a GCS object.
// This reader is kernel-optimized compared to the GCSReader as it doesn't have complex logic
// to switch between sequential and random read strategies.
type KernelMRDReader struct {
	mrdInstanceInUse   atomic.Bool
	mrdInstance        *gcsx.MrdInstance
	metrics            metrics.MetricHandle
	mu                 sync.Mutex
	chunks             []*gcsx.MrdPrefetchChunk
	nextPrefetchOffset int64
	lastReadEnd        int64
	seqReads           int
	prefetchGeneration int64
}

// NewKernelMRDReader creates a new KernelMRDReader that uses the provided
// MrdInstance to manage MRD connections.
func NewKernelMRDReader(mrdInstance *gcsx.MrdInstance, metricsHandle metrics.MetricHandle) *KernelMRDReader {
	return &KernelMRDReader{
		mrdInstance: mrdInstance,
		metrics:     metricsHandle,
		chunks:      make([]*gcsx.MrdPrefetchChunk, 0, maxPrefetchChunks),
	}
}

func (kmr *KernelMRDReader) clearPrefetchLocked() {
	for i, c := range kmr.chunks {
		c.Release()
		kmr.chunks[i] = nil
	}
	kmr.chunks = kmr.chunks[:0]
}

func (kmr *KernelMRDReader) clearPrefetch() {
	kmr.mu.Lock()
	kmr.clearPrefetchLocked()
	kmr.seqReads = 0
	kmr.mu.Unlock()
}

func (kmr *KernelMRDReader) schedulePrefetchLocked() {
	if kmr.mrdInstance == nil {
		return
	}
	uObjSize, gen := kmr.mrdInstance.ObjectSizeAndGeneration()
	if len(kmr.chunks) > 0 && gen != kmr.prefetchGeneration {
		kmr.clearPrefetchLocked()
		kmr.nextPrefetchOffset = kmr.lastReadEnd
	}
	kmr.prefetchGeneration = gen
	objSize := int64(uObjSize)
	ready := max(1, kmr.mrdInstance.ReadySize())
	limit := min(maxPrefetchChunks, ready+1)
	if ready > 1 {
		limit = min(maxPrefetchChunks, ready+2)
	}
	for len(kmr.chunks) < limit && kmr.nextPrefetchOffset < objSize {
		rem := objSize - kmr.nextPrefetchOffset
		targetSize := int64(prefetchChunkSize)
		if kmr.nextPrefetchOffset <= 1024*1024 && ready == 1 {
			targetSize = int64(initialPrefetchChunkSize)
		}
		chunkLen := int(min(targetSize, rem))
		chunk, err := kmr.mrdInstance.StartPrefetchChunk(kmr.nextPrefetchOffset, chunkLen, kmr.metrics)
		if err != nil {
			break
		}
		kmr.chunks = append(kmr.chunks, chunk)
		kmr.nextPrefetchOffset += int64(chunkLen)
	}
}

func (kmr *KernelMRDReader) maybeScheduleMorePrefetch() {
	kmr.mu.Lock()
	if len(kmr.chunks) < maxPrefetchChunks && kmr.seqReads > 0 {
		kmr.schedulePrefetchLocked()
	}
	kmr.mu.Unlock()
}

func (kmr *KernelMRDReader) acquirePrefetchChunk(offset int64, length int) (*gcsx.MrdPrefetchChunk, int, int) {
	kmr.mu.Lock()
	defer kmr.mu.Unlock()

	if len(kmr.chunks) == 0 {
		return nil, 0, 0
	}

	reqEnd := offset + int64(length)
	if kmr.mrdInstance != nil {
		objSize, gen := kmr.mrdInstance.ObjectSizeAndGeneration()
		if gen != kmr.prefetchGeneration || reqEnd > int64(objSize) || kmr.nextPrefetchOffset > int64(objSize) {
			kmr.clearPrefetchLocked()
			kmr.seqReads = 0
			return nil, 0, 0
		}
	}

	for i, c := range kmr.chunks {
		cEnd := c.Offset + int64(c.Length)
		if offset >= c.Offset && reqEnd <= cEnd {
			relStart := int(offset - c.Offset)
			relEnd := relStart + length
			c.Retain()
			c.Consumed += length
			if c.Consumed >= c.Length {
				c.Release()
				copy(kmr.chunks[i:], kmr.chunks[i+1:])
				kmr.chunks[len(kmr.chunks)-1] = nil
				kmr.chunks = kmr.chunks[:len(kmr.chunks)-1]
			}
			if reqEnd > kmr.lastReadEnd {
				kmr.lastReadEnd = reqEnd
			}
			kmr.seqReads++
			if length >= minPrefetchReadSize {
				kmr.schedulePrefetchLocked()
			}
			return c, relStart, relEnd
		}
	}

	// Non-sequential jump or chunk boundary straddle: discard stale lookahead.
	kmr.clearPrefetchLocked()
	kmr.seqReads = 0
	return nil, 0, 0
}

// isShortRead determines what constitutes a short read for retry purposes.
// It returns true if bytesRead < bufferSize and the error is a gRPC OutOfRange error.
func isShortRead(bytesRead int, bufferSize int, err error) bool {
	if bytesRead >= bufferSize {
		return false
	}

	// Even without O_DIRECT, OutOfRange errors can occur during appends from the same mount.
	// The kernel tracks the updated file size and allows reads, but the active MRD connection might
	// still reference the old object size. We update the local object in the MrdInstance without
	// recreating the MRD connection. Reads beyond the old size thus return OutOfRange, which
	// we handle as a short read to trigger MRD recreation and retry.

	// Check for gRPC OutOfRange error, handling wrapped errors.
	var se interface{ GRPCStatus() *status.Status }
	if errors.As(err, &se) {
		return se.GRPCStatus().Code() == codes.OutOfRange
	}

	return false
}

// ReaderName returns the name of the reader.
func (kmr *KernelMRDReader) ReaderName() string {
	return "KernelMRDReader"
}

// CheckInvariants performs internal consistency checks on the reader state.
func (kmr *KernelMRDReader) CheckInvariants() {
}

// ReadAt reads data into the provided request buffer starting at the specified
// offset. It retrieves an available MRD entry and uses it to download the
// requested byte range.
func (kmr *KernelMRDReader) ReadAt(ctx context.Context, req *gcsx.ReadRequest) (gcsx.ReadResponse, error) {
	// If the destination buffer is empty, there's nothing to read.
	if len(req.Buffer) == 0 {
		return gcsx.ReadResponse{}, nil
	}

	// mrdInstance is set to nil in Destroy which will be called only after all active Read operations
	// have finished. Hence, not taking RLock to access it.
	if kmr.mrdInstance == nil {
		return gcsx.ReadResponse{}, fmt.Errorf("KernelMRDReader: mrdInstance is nil")
	}

	if kmr.mrdInstanceInUse.CompareAndSwap(false, true) {
		kmr.mrdInstance.IncrementRefCount()
	}

	var bytesRead int
	defer func() {
		metrics.CaptureGCSReadMetrics(kmr.metrics, metrics.ReadTypeParallelAttr, int64(bytesRead))
		kmr.metrics.GcsReadBytesCount(int64(bytesRead))
	}()

	if chunk, relStart, relEnd := kmr.acquirePrefetchChunk(req.Offset, len(req.Buffer)); chunk != nil {
		wErr := chunk.Wait(ctx, kmr.mrdInstance.IgnoreInterrupts())
		if wErr == nil {
			if len(req.Buffer) >= minPrefetchReadSize {
				kmr.maybeScheduleMorePrefetch()
			}
			if resp, ok := chunk.Slice(relStart, relEnd); ok {
				bytesRead = resp.Size
				return resp, nil
			}
		}
		chunk.Release()
		if errors.Is(wErr, context.Canceled) || errors.Is(wErr, context.DeadlineExceeded) {
			return gcsx.ReadResponse{}, wErr
		}
		kmr.clearPrefetch()
	}

	resp, err := kmr.mrdInstance.ReadVectored(ctx, req.Buffer, req.Offset, kmr.metrics)
	bytesRead = resp.Size
	if isShortRead(bytesRead, len(req.Buffer), err) {
		kmr.clearPrefetch()
		if resp.Callback != nil {
			pos := 0
			for _, s := range resp.Data {
				pos += copy(req.Buffer[pos:], s)
			}
			resp.Callback()
			resp.Data = nil
			resp.Callback = nil
		}
		logger.Tracef("Short read detected: read %d bytes out of %d requested. Retrying...", bytesRead, len(req.Buffer))
		if err = kmr.mrdInstance.RecreateMRD(); err != nil {
			logger.Warnf("Failed to recreate MRD for short read retry. Will retry with older MRD: %v", err)
		}
		retryOffset := req.Offset + int64(bytesRead)
		retryBuffer := req.Buffer[bytesRead:]
		var bytesReadOnRetry int
		bytesReadOnRetry, err = kmr.mrdInstance.Read(ctx, retryBuffer, retryOffset, kmr.metrics)
		bytesRead += bytesReadOnRetry
		return gcsx.ReadResponse{Size: bytesRead}, err
	}

	if err == nil && bytesRead == len(req.Buffer) && len(req.Buffer) >= minPrefetchReadSize && resp.Callback != nil {
		kmr.mu.Lock()
		if req.Offset == 0 || req.Offset == kmr.lastReadEnd {
			kmr.seqReads++
		} else {
			kmr.clearPrefetchLocked()
			kmr.seqReads = 1
		}
		kmr.lastReadEnd = req.Offset + int64(bytesRead)
		if req.Offset == 0 || kmr.seqReads >= 2 {
			if len(kmr.chunks) == 0 {
				kmr.nextPrefetchOffset = kmr.lastReadEnd
			}
			kmr.schedulePrefetchLocked()
		}
		kmr.mu.Unlock()
	} else if len(req.Buffer) < minPrefetchReadSize {
		kmr.mu.Lock()
		if req.Offset == kmr.lastReadEnd && err == nil {
			kmr.lastReadEnd = req.Offset + int64(bytesRead)
		} else if len(kmr.chunks) > 0 || kmr.seqReads > 0 {
			kmr.clearPrefetchLocked()
			kmr.seqReads = 0
		}
		kmr.mu.Unlock()
	}

	return resp, err
}

// Destroy cleans up the resources used by the reader, primarily by destroying
// the associated MrdInstance. This should be called when the reader is no
// longer needed.
func (kmr *KernelMRDReader) Destroy() {
	kmr.clearPrefetch()
	// No need to take lock as Destroy will only be called when file handle is being released
	// and there will be no read calls at that point.
	if kmr.mrdInstance != nil {
		if kmr.mrdInstanceInUse.CompareAndSwap(true, false) {
			kmr.mrdInstance.DecrementRefCount()
		}
		kmr.mrdInstance = nil
	}
}
