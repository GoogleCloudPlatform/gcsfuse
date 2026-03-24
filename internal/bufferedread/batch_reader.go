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

package bufferedread

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
)

// BatchReader implements the batch read logic with the central registry
type BatchReader struct {
	registry *BatchReadRegistry
	bucket   gcs.Bucket
	object   *gcs.MinObject

	// fallbackReader is used for direct reads that don't benefit from batching
	fallbackReader gcsx.Reader

	// mrdInstance for parallel batch reads (optional, can be nil)
	mrdInstance *gcsx.MrdInstance

	metricHandle metrics.MetricHandle
	traceHandle  tracing.TraceHandle
}

// BatchReaderOptions holds the dependencies for creating a BatchReader
type BatchReaderOptions struct {
	Registry       *BatchReadRegistry
	Bucket         gcs.Bucket
	Object         *gcs.MinObject
	FallbackReader gcsx.Reader
	MrdInstance    *gcsx.MrdInstance
	MetricHandle   metrics.MetricHandle
	TraceHandle    tracing.TraceHandle
}

// NewBatchReader creates a new batch reader
func NewBatchReader(opts *BatchReaderOptions) (*BatchReader, error) {
	if opts.Registry == nil {
		return nil, fmt.Errorf("registry cannot be nil")
	}
	if opts.Bucket == nil {
		return nil, fmt.Errorf("bucket cannot be nil")
	}
	if opts.Object == nil {
		return nil, fmt.Errorf("object cannot be nil")
	}
	if opts.FallbackReader == nil {
		return nil, fmt.Errorf("fallback reader cannot be nil")
	}

	return &BatchReader{
		registry:       opts.Registry,
		bucket:         opts.Bucket,
		object:         opts.Object,
		fallbackReader: opts.FallbackReader,
		mrdInstance:    opts.MrdInstance,
		metricHandle:   opts.MetricHandle,
		traceHandle:    opts.TraceHandle,
	}, nil
}

// CheckInvariants performs internal consistency checks
func (br *BatchReader) CheckInvariants() {
	// No invariants to check currently
}

// ReaderName returns the name of this reader
func (br *BatchReader) ReaderName() string {
	return "BatchReader"
}

// Destroy releases resources held by the reader
func (br *BatchReader) Destroy() {
	if br.registry != nil {
		br.registry.Stop()
	}
	if br.fallbackReader != nil {
		br.fallbackReader.Destroy()
	}
}

// ReadAt implements the batch read algorithm
func (br *BatchReader) ReadAt(ctx context.Context, req *gcsx.ReadRequest) (gcsx.ReadResponse, error) {
	if req == nil || len(req.Buffer) == 0 {
		return gcsx.ReadResponse{}, nil
	}

	// Step 1: Check if read size is below the batch threshold (e.g., < 1MB)
	maxReadSize := br.registry.config.MaxReadSizeMb * 1024 * 1024
	if int64(len(req.Buffer)) < maxReadSize {
		logger.Tracef("BatchReader: Read size %d is below threshold %d, using direct read",
			len(req.Buffer), maxReadSize)
		return br.directRead(ctx, req)
	}

	// Step 2: Calculate block ID
	blockID := br.registry.CalculateBlockID(req.Offset)
	logger.Tracef("BatchReader: Read at offset %d, size %d, blockID %d",
		req.Offset, len(req.Buffer), blockID)

	// Step 3: Register intention to read
	intent, registeredBlockID := br.registry.RegisterReadIntent(req.Offset, len(req.Buffer), req.Buffer)
	if registeredBlockID != blockID {
		logger.Warnf("BatchReader: Unexpected blockID mismatch: calculated %d, registered %d",
			blockID, registeredBlockID)
		blockID = registeredBlockID
	}

	// Step 4: Wait for batching opportunity (1ms by default)
	waitDuration := time.Duration(br.registry.config.WaitTimeMs) * time.Millisecond
	time.Sleep(waitDuration)

	// Step 5: Query registry for existing batch
	if existingRequest, exists := br.registry.QueryBatch(blockID); exists {
		existingRequest.mu.RLock()
		state := existingRequest.state
		existingRequest.mu.RUnlock()

		if state == SCHEDULED || state == READING {
			logger.Tracef("BatchReader: Joining existing batch in state %s for block %d",
				state.String(), blockID)
			// Join existing batch
			return br.joinBatch(ctx, existingRequest, intent, req)
		}
	}

	// Step 6: Check for adjacent reads
	adjacentReads := br.registry.GetAdjacentReads(blockID)
	if len(adjacentReads) > 1 {
		logger.Tracef("BatchReader: Found %d adjacent reads, creating batch for block %d",
			len(adjacentReads), blockID)
		// Create batch for outstanding reads
		return br.createAndExecuteBatch(ctx, blockID, adjacentReads, intent, req)
	}

	// Step 7: No batch opportunity, issue direct read
	logger.Tracef("BatchReader: No batch opportunity for block %d, using direct read", blockID)
	return br.directRead(ctx, req)
}

// directRead performs a single read using the fallback reader
func (br *BatchReader) directRead(ctx context.Context, req *gcsx.ReadRequest) (gcsx.ReadResponse, error) {
	logger.Tracef("BatchReader: Performing direct read at offset %d, size %d",
		req.Offset, len(req.Buffer))

	// Use the fallback reader for non-batched reads
	return br.fallbackReader.ReadAt(ctx, req)
}

// joinBatch waits for an existing batch to complete and serves the result
func (br *BatchReader) joinBatch(
	ctx context.Context,
	batchReq *BufferedReadRequest,
	intent *ReadIntent,
	req *gcsx.ReadRequest) (gcsx.ReadResponse, error) {

	logger.Tracef("BatchReader: Joining batch for blockID %d", batchReq.blockID)

	// Increment reference count since we're joining the batch
	br.registry.IncrementRefCount(batchReq.blockID)

	// Wait for batch completion with context timeout
	select {
	case <-ctx.Done():
		// Context cancelled, decrement refCount since we won't consume the result
		br.registry.DecrementRefCount(batchReq.blockID)
		return gcsx.ReadResponse{}, ctx.Err()
	case <-batchReq.completionChan:
		// Batch completed (channel closed), now read the result
		batchReq.mu.RLock()
		buffer := batchReq.buffer
		err := batchReq.err
		batchReq.mu.RUnlock()

		if err != nil {
			logger.Tracef("BatchReader: Batch %d completed with error: %v", batchReq.blockID, err)
			// Decrement refCount even on error
			br.registry.DecrementRefCount(batchReq.blockID)
			return gcsx.ReadResponse{}, err
		}

		// Extract the relevant portion from the batch buffer
		response, respErr := br.serveBatchResult(req.Offset, req.Buffer, buffer, batchReq.blockID)
		if respErr != nil {
			// Decrement refCount on error
			br.registry.DecrementRefCount(batchReq.blockID)
			return gcsx.ReadResponse{}, respErr
		}

		// Decrement reference count
		br.registry.DecrementRefCount(batchReq.blockID)

		logger.Tracef("BatchReader: Successfully served %d bytes from batch %d",
			response.Size, batchReq.blockID)
		return response, nil
	}
}

// createAndExecuteBatch creates a new batch and executes the read
func (br *BatchReader) createAndExecuteBatch(
	ctx context.Context,
	blockID int64,
	adjacentReads []*ReadIntent,
	currentIntent *ReadIntent,
	req *gcsx.ReadRequest) (gcsx.ReadResponse, error) {

	logger.Tracef("BatchReader: Creating batch for block %d with %d reads",
		blockID, len(adjacentReads))

	// Mark batch as scheduled (atomic - only one reader will succeed)
	if err := br.registry.MarkBatchScheduled(blockID); err != nil {
		// Another reader already scheduled this batch, join it instead
		logger.Tracef("BatchReader: Batch %d already scheduled by another reader, joining instead", blockID)
		batchReq, exists := br.registry.QueryBatch(blockID)
		if !exists {
			// Batch disappeared, fall back to direct read
			logger.Warnf("BatchReader: Batch %d not found after failed schedule, using direct read", blockID)
			return br.directRead(ctx, req)
		}
		return br.joinBatch(ctx, batchReq, currentIntent, req)
	}

	// We successfully scheduled the batch, now execute it
	// Execute the batch read in a goroutine
	go br.executeBatchRead(context.Background(), blockID)

	// Wait for the result like a joined read
	batchReq, exists := br.registry.QueryBatch(blockID)
	if !exists {
		return gcsx.ReadResponse{}, fmt.Errorf("batch %d disappeared after scheduling", blockID)
	}

	return br.joinBatch(ctx, batchReq, currentIntent, req)
}

// executeBatchRead performs the actual batch read operation
func (br *BatchReader) executeBatchRead(ctx context.Context, blockID int64) {
	logger.Tracef("BatchReader: Executing batch read for block %d", blockID)

	// Mark as reading
	if err := br.registry.MarkBatchReading(blockID); err != nil {
		logger.Errorf("BatchReader: Failed to mark batch as reading: %v", err)
		br.registry.FailRead(blockID, err)
		return
	}

	// Determine read range: entire block
	blockSize := br.registry.config.BlockSizeMb * 1024 * 1024
	startOffset := blockID
	endOffset := blockID + blockSize

	// Don't read beyond object size
	if endOffset > int64(br.object.Size) {
		endOffset = int64(br.object.Size)
	}

	readSize := endOffset - startOffset
	if readSize <= 0 {
		err := fmt.Errorf("invalid read size: %d for block %d", readSize, blockID)
		logger.Errorf("BatchReader: %v", err)
		br.registry.FailRead(blockID, err)
		return
	}

	// Allocate buffer for the entire block
	buffer := make([]byte, readSize)

	logger.Tracef("BatchReader: Reading block %d from offset %d to %d (%d bytes)",
		blockID, startOffset, endOffset, readSize)

	// Perform the read using MRD if available, otherwise use bucket reader
	var bytesRead int
	var err error

	if br.mrdInstance != nil {
		// Use MRD for parallel reads
		bytesRead, err = br.mrdInstance.Read(ctx, buffer, startOffset, br.metricHandle)
		if err != nil && bytesRead > 0 {
			// Partial read is acceptable
			logger.Tracef("BatchReader: MRD read returned partial data: %d bytes with err: %v",
				bytesRead, err)
			buffer = buffer[:bytesRead]
			err = nil
		}
	} else {
		// Fallback to regular bucket read
		readReq := &gcs.ReadObjectRequest{
			Name:       br.object.Name,
			Generation: br.object.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(startOffset),
				Limit: uint64(endOffset),
			},
		}

		rc, err := br.bucket.NewReaderWithReadHandle(ctx, readReq)
		if err != nil {
			logger.Errorf("BatchReader: Failed to create reader for block %d: %v", blockID, err)
			br.registry.FailRead(blockID, err)
			return
		}
		defer rc.Close()

		bytesRead, err = io.ReadFull(rc, buffer)
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			// Partial read at end of file is acceptable
			buffer = buffer[:bytesRead]
			err = nil
		}
	}

	if err != nil {
		logger.Errorf("BatchReader: Failed to read block %d: %v", blockID, err)
		br.registry.FailRead(blockID, err)
		return
	}

	logger.Tracef("BatchReader: Successfully read %d bytes for block %d", bytesRead, blockID)

	// Complete the read and notify all waiting requests
	if err := br.registry.CompleteRead(blockID, buffer, nil); err != nil {
		logger.Errorf("BatchReader: Failed to complete read for block %d: %v", blockID, err)
	}
}

// serveBatchResult extracts the requested data from the batch buffer
func (br *BatchReader) serveBatchResult(
	requestOffset int64,
	destBuffer []byte,
	batchBuffer []byte,
	blockID int64) (gcsx.ReadResponse, error) {

	// Calculate the offset within the batch buffer
	offsetInBatch := requestOffset - blockID
	if offsetInBatch < 0 {
		return gcsx.ReadResponse{}, fmt.Errorf("invalid offset: request offset %d is before block start %d",
			requestOffset, blockID)
	}

	if offsetInBatch >= int64(len(batchBuffer)) {
		return gcsx.ReadResponse{}, fmt.Errorf("offset %d is beyond batch buffer size %d",
			offsetInBatch, len(batchBuffer))
	}

	// Calculate bytes to copy
	remainingInBatch := int64(len(batchBuffer)) - offsetInBatch
	bytesToCopy := int64(len(destBuffer))
	if bytesToCopy > remainingInBatch {
		bytesToCopy = remainingInBatch
	}

	// Copy data from batch buffer to destination
	copied := copy(destBuffer, batchBuffer[offsetInBatch:offsetInBatch+bytesToCopy])

	logger.Tracef("BatchReader: Copied %d bytes from batch buffer (offset in batch: %d)",
		copied, offsetInBatch)

	return gcsx.ReadResponse{
		Size: copied,
	}, nil
}
