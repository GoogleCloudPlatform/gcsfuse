// Copyright 2025 Google LLC
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
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// MRDReader implements the Reader interface using a MultiRangeDownloaderWrapper.
// It provides efficient parallel downloads for reading GCS objects.
type MRDReader struct {
	object *gcs.MinObject

	// mrdWrapper points to the wrapper object within inode.
	mrdWrapper *MultiRangeDownloaderWrapper

	// boolean variable to determine if MRD is being used or not.
	isMRDInUse atomic.Bool

	metricHandle metrics.MetricHandle
	mu           sync.Mutex
}

// NewMRDReader creates a new MRDReader with the given object and MRD wrapper.
func NewMRDReader(object *gcs.MinObject, metricHandle metrics.MetricHandle, mrdWrapper *MultiRangeDownloaderWrapper) *MRDReader {
	return &MRDReader{
		object:       object,
		metricHandle: metricHandle,
		mrdWrapper:   mrdWrapper,
	}
}

// CheckInvariants performs internal consistency checks on the reader state.
func (mr *MRDReader) CheckInvariants() {
	// Verify object is not nil
	if mr.object == nil {
		panic("MRDReader: object is nil")
	}

	// Verify mrdWrapper is not nil
	if mr.mrdWrapper == nil {
		panic("MRDReader: mrdWrapper is nil")
	}
}

// readFromMultiRangeDownloader reads data from the underlying MultiRangeDownloaderWrapper.
//
// It increments the reference count of the mrdWrapper if it's not already in use.
// It then calls the Read method of the mrdWrapper with the provided parameters.
//
// Parameters:
//   - ctx: The context for the read operation. It can be used to cancel the operation or set a timeout.
//   - p: The byte slice to read data into.
//   - offset: The starting offset for the read operation.
//   - end: The ending offset for the read operation.
//   - forceCreateMRD: Determines whether to force the creation of a new MRD.
//
// Returns:
//   - int: The number of bytes read.
//   - error: An error if the read operation fails.
func (mr *MRDReader) readFromMultiRangeDownloader(ctx context.Context, p []byte, offset, end int64, forceCreateMRD bool) (int, error) {
	if mr.mrdWrapper == nil {
		return 0, fmt.Errorf("readFromMultiRangeDownloader: Invalid MultiRangeDownloaderWrapper")
	}

	if mr.isMRDInUse.CompareAndSwap(false, true) {
		mr.mrdWrapper.IncrementRefCount()
	}

	return mr.mrdWrapper.Read(ctx, p, offset, end, mr.metricHandle, forceCreateMRD)
}

// ReadAt attempts to read data from the object using the MRD wrapper.
// It implements the Reader interface's ReadAt method.
func (mr *MRDReader) ReadAt(ctx context.Context, req *ReadRequest) (ReadResponse, error) {
	var readResponse ReadResponse

	// Check if offset is beyond object size
	if req.Offset >= int64(mr.object.Size) {
		return readResponse, io.EOF
	}

	// Calculate the actual read length
	endOffset := req.Offset + int64(len(req.Buffer))
	if endOffset > int64(mr.object.Size) {
		endOffset = int64(mr.object.Size)
	}

	var err error
	readResponse.Size, err = mr.readFromMultiRangeDownloader(ctx, req.Buffer, req.Offset, endOffset, false)

	return readResponse, err
}

// Destroy releases any resources held by the reader.
func (mr *MRDReader) Destroy() {
	if mr.isMRDInUse.Load() {
		err := mr.mrdWrapper.DecrementRefCount()
		if err != nil {
			// Log error but don't panic during cleanup
			fmt.Printf("MRDReader::Destroy: %v\n", err)
		}
		mr.isMRDInUse.Store(false)
	}
}

// Object returns the underlying GCS object metadata associated with the reader.
func (mr *MRDReader) Object() *gcs.MinObject {
	return mr.object
}
