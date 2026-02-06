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
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MrdKernelReader is a reader that uses an MRD Instance to read data from a GCS object.
// This reader is kernel-optimized compared to the GCSReader as it doesn't have complex logic
// to switch between sequential and random read strategies.
type MrdKernelReader struct {
	mrdInstanceInUse atomic.Bool
	mrdInstance      *MrdInstance
	metrics          metrics.MetricHandle
}

// NewMrdKernelReader creates a new MrdKernelReader that uses the provided
// MrdInstance to manage MRD connections.
func NewMrdKernelReader(mrdInstance *MrdInstance, metricsHandle metrics.MetricHandle) *MrdKernelReader {
	return &MrdKernelReader{
		mrdInstance: mrdInstance,
		metrics:     metricsHandle,
	}
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

// ReadAt reads data into the provided request buffer starting at the specified
// offset. It retrieves an available MRD entry and uses it to download the
// requested byte range.
func (mkr *MrdKernelReader) ReadAt(ctx context.Context, req *ReadRequest) (ReadResponse, error) {
	// If the destination buffer is empty, there's nothing to read.
	if len(req.Buffer) == 0 {
		return ReadResponse{}, nil
	}

	// mrdInstance is set to nil in Destroy which will be called only after all active Read operations
	// have finished. Hence, not taking RLock to access it.
	if mkr.mrdInstance == nil {
		return ReadResponse{}, fmt.Errorf("MrdKernelReader: mrdInstance is nil")
	}

	if mkr.mrdInstanceInUse.CompareAndSwap(false, true) {
		mkr.mrdInstance.IncrementRefCount()
	}

	var bytesRead int
	defer func() {
		metrics.CaptureGCSReadMetrics(mkr.metrics, metrics.ReadTypeParallelAttr, int64(bytesRead))
		mkr.metrics.GcsReadBytesCount(int64(bytesRead))
	}()

	var err error
	bytesRead, err = mkr.mrdInstance.Read(ctx, req.Buffer, req.Offset, mkr.metrics)
	if isShortRead(bytesRead, len(req.Buffer), err) {
		logger.Tracef("Short read detected: read %d bytes out of %d requested. Retrying...", bytesRead, len(req.Buffer))
		if err = mkr.mrdInstance.RecreateMRD(); err != nil {
			logger.Warnf("Failed to recreate MRD for short read retry. Will retry with older MRD: %v", err)
		}
		retryOffset := req.Offset + int64(bytesRead)
		retryBuffer := req.Buffer[bytesRead:]
		var bytesReadOnRetry int
		bytesReadOnRetry, err = mkr.mrdInstance.Read(ctx, retryBuffer, retryOffset, mkr.metrics)
		bytesRead += bytesReadOnRetry
	}
	return ReadResponse{Size: bytesRead}, err
}

// Destroy cleans up the resources used by the reader, primarily by destroying
// the associated MrdInstance. This should be called when the reader is no
// longer needed.
func (mkr *MrdKernelReader) Destroy() {
	// No need to take lock as Destroy will only be called when file handle is being released
	// and there will be no read calls at that point.
	if mkr.mrdInstance != nil {
		if mkr.mrdInstanceInUse.CompareAndSwap(true, false) {
			mkr.mrdInstance.DecrementRefCount()
		}
		mkr.mrdInstance = nil
	}
}
