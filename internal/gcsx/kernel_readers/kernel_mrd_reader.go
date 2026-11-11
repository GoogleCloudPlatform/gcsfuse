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

type inFlightRange struct {
	id    uint64
	start int64
	end   int64
}

type prefetchSpan struct {
	chunk    *gcsx.MrdPrefetchChunk
	relStart int
	relEnd   int
}

// KernelMRDReader is a reader that uses an MRD Instance to read data from a GCS object.
// This reader is kernel-optimized compared to the GCSReader as it doesn't have complex logic
// to switch between sequential and random read strategies.
type KernelMRDReader struct {
	mrdInstanceInUse   atomic.Bool
	mrdInstance        *gcsx.MrdInstance
	metrics            metrics.MetricHandle
	mu                 sync.Mutex
	chunks             []*gcsx.MrdPrefetchChunk
	inFlight           []inFlightRange
	nextInFlightID     uint64
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

func (kmr *KernelMRDReader) removeInFlightLocked(id uint64) {
	for i, r := range kmr.inFlight {
		if r.id == id {
			copy(kmr.inFlight[i:], kmr.inFlight[i+1:])
			kmr.inFlight = kmr.inFlight[:len(kmr.inFlight)-1]
			return
		}
	}
}

func (kmr *KernelMRDReader) contiguousInFlightEndLocked(base int64) int64 {
	curEnd := base
	for {
		advanced := false
		for _, r := range kmr.inFlight {
			if r.start <= curEnd && r.end > curEnd {
				curEnd = r.end
				advanced = true
			}
		}
		if !advanced {
			return curEnd
		}
	}
}

func (kmr *KernelMRDReader) connectsToSequentialLocked(reqStart int64) bool {
	if kmr.contiguousInFlightEndLocked(kmr.lastReadEnd) >= reqStart {
		return true
	}
	return kmr.contiguousInFlightEndLocked(0) >= reqStart
}

func (kmr *KernelMRDReader) maybeScheduleMorePrefetch() {
	kmr.mu.Lock()
	if len(kmr.chunks) < maxPrefetchChunks && kmr.seqReads > 0 {
		kmr.schedulePrefetchLocked()
	}
	kmr.mu.Unlock()
}

func (kmr *KernelMRDReader) acquirePrefetchSpans(offset int64, length int) []prefetchSpan {
	kmr.mu.Lock()
	defer kmr.mu.Unlock()

	if len(kmr.chunks) == 0 || length <= 0 {
		return nil
	}

	reqEnd := offset + int64(length)
	if kmr.mrdInstance != nil {
		objSize, gen := kmr.mrdInstance.ObjectSizeAndGeneration()
		if gen != kmr.prefetchGeneration || reqEnd > int64(objSize) || kmr.nextPrefetchOffset > int64(objSize) {
			kmr.clearPrefetchLocked()
			kmr.seqReads = 0
			return nil
		}
	}

	startIdx := -1
	for i, c := range kmr.chunks {
		cEnd := c.Offset + int64(c.Length)
		if offset >= c.Offset && offset < cEnd {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		// Non-sequential jump: discard stale lookahead.
		kmr.clearPrefetchLocked()
		kmr.seqReads = 0
		return nil
	}

	startChunk := kmr.chunks[startIdx]
	if reqEnd > startChunk.Offset+int64(startChunk.Length) && reqEnd > kmr.nextPrefetchOffset && length >= minPrefetchReadSize {
		kmr.schedulePrefetchLocked()
		if startIdx >= len(kmr.chunks) || kmr.chunks[startIdx] != startChunk {
			return nil
		}
	}

	endIdx := startIdx
	for endIdx < len(kmr.chunks) {
		cur := kmr.chunks[endIdx]
		curEnd := cur.Offset + int64(cur.Length)
		if curEnd >= reqEnd {
			break
		}
		nextIdx := endIdx + 1
		if nextIdx >= len(kmr.chunks) {
			endIdx = -1
			break
		}
		next := kmr.chunks[nextIdx]
		if next.Offset != curEnd {
			endIdx = -1
			break
		}
		endIdx = nextIdx
	}

	if endIdx == -1 {
		// Not completely covered by contiguous prefetched chunks: discard stale lookahead.
		kmr.clearPrefetchLocked()
		kmr.seqReads = 0
		return nil
	}

	spans := make([]prefetchSpan, 0, endIdx-startIdx+1)
	for idx := startIdx; idx <= endIdx; idx++ {
		c := kmr.chunks[idx]
		subStart := max(offset, c.Offset) - c.Offset
		subEnd := min(reqEnd, c.Offset+int64(c.Length)) - c.Offset
		subLen := int(subEnd - subStart)
		c.Retain()
		c.Consumed += subLen
		spans = append(spans, prefetchSpan{
			chunk:    c,
			relStart: int(subStart),
			relEnd:   int(subEnd),
		})
	}

	n := 0
	for idx, c := range kmr.chunks {
		evict := false
		if idx >= startIdx && idx <= endIdx {
			evict = c.Consumed >= c.Length
		} else if idx < startIdx {
			evict = c.Consumed >= c.Length || (c.Consumed > 0 && kmr.lastReadEnd >= c.Offset+int64(c.Length))
		}
		if evict {
			c.Release()
			continue
		}
		kmr.chunks[n] = c
		n++
	}
	for idx := n; idx < len(kmr.chunks); idx++ {
		kmr.chunks[idx] = nil
	}
	kmr.chunks = kmr.chunks[:n]

	if reqEnd > kmr.lastReadEnd {
		kmr.lastReadEnd = reqEnd
	}
	kmr.seqReads++
	if length >= minPrefetchReadSize {
		kmr.schedulePrefetchLocked()
	}
	return spans
}

func (kmr *KernelMRDReader) rollbackPrefetchSpans(spans []prefetchSpan) {
	kmr.mu.Lock()
	sameGen := true
	if kmr.mrdInstance != nil {
		_, gen := kmr.mrdInstance.ObjectSizeAndGeneration()
		sameGen = (gen == kmr.prefetchGeneration)
	}
	for _, sp := range spans {
		subLen := sp.relEnd - sp.relStart
		sp.chunk.Consumed = max(0, sp.chunk.Consumed-subLen)
		if sameGen && sp.chunk.Consumed < sp.chunk.Length {
			inQueue := false
			insertIdx := len(kmr.chunks)
			for i, c := range kmr.chunks {
				if c == sp.chunk {
					inQueue = true
					break
				}
				if c.Offset > sp.chunk.Offset && insertIdx == len(kmr.chunks) {
					insertIdx = i
				}
			}
			if !inQueue && len(kmr.chunks) < maxPrefetchChunks {
				sp.chunk.Retain()
				kmr.chunks = append(kmr.chunks, nil)
				copy(kmr.chunks[insertIdx+1:], kmr.chunks[insertIdx:])
				kmr.chunks[insertIdx] = sp.chunk
			}
		}
	}
	kmr.mu.Unlock()

	for _, sp := range spans {
		sp.chunk.Release()
	}
}

func (kmr *KernelMRDReader) acquirePrefetchChunk(offset int64, length int) (*gcsx.MrdPrefetchChunk, int, int) {
	spans := kmr.acquirePrefetchSpans(offset, length)
	if len(spans) == 1 {
		return spans[0].chunk, spans[0].relStart, spans[0].relEnd
	}
	if len(spans) > 1 {
		kmr.rollbackPrefetchSpans(spans)
	}
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

	if spans := kmr.acquirePrefetchSpans(req.Offset, len(req.Buffer)); len(spans) == 1 {
		sp := spans[0]
		wErr := sp.chunk.Wait(ctx, kmr.mrdInstance.IgnoreInterrupts())
		if wErr == nil {
			if len(req.Buffer) >= minPrefetchReadSize {
				kmr.maybeScheduleMorePrefetch()
			}
			if resp, ok := sp.chunk.Slice(sp.relStart, sp.relEnd); ok {
				bytesRead = resp.Size
				return resp, nil
			}
		}
		if errors.Is(wErr, context.Canceled) || errors.Is(wErr, context.DeadlineExceeded) {
			kmr.rollbackPrefetchSpans(spans)
			return gcsx.ReadResponse{}, wErr
		}
		sp.chunk.Release()
		kmr.clearPrefetch()
	} else if len(spans) > 1 {
		ignoreInterrupts := kmr.mrdInstance.IgnoreInterrupts()
		var wErr error
		for _, sp := range spans {
			if err := sp.chunk.Wait(ctx, ignoreInterrupts); err != nil {
				wErr = err
				break
			}
		}
		if wErr == nil {
			if len(req.Buffer) >= minPrefetchReadSize {
				kmr.maybeScheduleMorePrefetch()
			}
			var combinedData [][]byte
			var callbacks []func()
			totalSize := 0
			allOk := true
			for _, sp := range spans {
				resp, ok := sp.chunk.Slice(sp.relStart, sp.relEnd)
				if !ok {
					allOk = false
					break
				}
				if combinedData == nil {
					combinedData = make([][]byte, 0, len(resp.Data)*len(spans))
					callbacks = make([]func(), 0, len(spans))
				}
				combinedData = append(combinedData, resp.Data...)
				if resp.Callback != nil {
					callbacks = append(callbacks, resp.Callback)
				}
				totalSize += resp.Size
			}
			if allOk {
				bytesRead = totalSize
				return gcsx.ReadResponse{
					Data: combinedData,
					Size: totalSize,
					Callback: func() {
						for _, cb := range callbacks {
							cb()
						}
					},
				}, nil
			}
		}
		if errors.Is(wErr, context.Canceled) || errors.Is(wErr, context.DeadlineExceeded) {
			kmr.rollbackPrefetchSpans(spans)
			return gcsx.ReadResponse{}, wErr
		}
		for _, sp := range spans {
			sp.chunk.Release()
		}
		kmr.clearPrefetch()
	}

	var inFlightID uint64
	trackInFlight := len(req.Buffer) >= minPrefetchReadSize
	if trackInFlight {
		kmr.mu.Lock()
		kmr.nextInFlightID++
		inFlightID = kmr.nextInFlightID
		kmr.inFlight = append(kmr.inFlight, inFlightRange{
			id:    inFlightID,
			start: req.Offset,
			end:   req.Offset + int64(len(req.Buffer)),
		})
		kmr.mu.Unlock()
	}

	resp, err := kmr.mrdInstance.ReadVectored(ctx, req.Buffer, req.Offset, kmr.metrics)
	bytesRead = resp.Size
	if isShortRead(bytesRead, len(req.Buffer), err) {
		if trackInFlight {
			kmr.mu.Lock()
			kmr.removeInFlightLocked(inFlightID)
			kmr.mu.Unlock()
		}
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
		reqStart := req.Offset
		reqEnd := req.Offset + int64(bytesRead)
		kmr.mu.Lock()
		sequential := reqStart == 0 || reqStart <= kmr.lastReadEnd || kmr.connectsToSequentialLocked(reqStart)
		kmr.removeInFlightLocked(inFlightID)
		if sequential {
			kmr.seqReads++
		} else {
			kmr.clearPrefetchLocked()
			kmr.seqReads = 1
		}
		if reqEnd > kmr.lastReadEnd {
			kmr.lastReadEnd = reqEnd
		}
		n := 0
		for _, c := range kmr.chunks {
			ovStart := max(reqStart, c.Offset)
			ovEnd := min(reqEnd, c.Offset+int64(c.Length))
			if ovEnd > ovStart {
				c.Consumed += int(ovEnd - ovStart)
			}
			if c.Consumed >= c.Length {
				c.Release()
				continue
			}
			kmr.chunks[n] = c
			n++
		}
		for i := n; i < len(kmr.chunks); i++ {
			kmr.chunks[i] = nil
		}
		kmr.chunks = kmr.chunks[:n]

		contiguousEnd := kmr.contiguousInFlightEndLocked(kmr.lastReadEnd)
		if reqStart == 0 || kmr.seqReads >= 2 {
			if len(kmr.chunks) == 0 || kmr.nextPrefetchOffset < contiguousEnd {
				kmr.nextPrefetchOffset = max(kmr.nextPrefetchOffset, contiguousEnd)
			}
			kmr.schedulePrefetchLocked()
		}
		kmr.mu.Unlock()
	} else {
		if trackInFlight {
			kmr.mu.Lock()
			kmr.removeInFlightLocked(inFlightID)
			kmr.mu.Unlock()
		}
		if len(req.Buffer) < minPrefetchReadSize {
			kmr.mu.Lock()
			if req.Offset == kmr.lastReadEnd && err == nil {
				kmr.lastReadEnd = req.Offset + int64(bytesRead)
			} else if len(kmr.chunks) > 0 || kmr.seqReads > 0 {
				kmr.clearPrefetchLocked()
				kmr.seqReads = 0
			}
			kmr.mu.Unlock()
		}
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
