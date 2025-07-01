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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// TimeoutForMultiRangeRead is the timeout value for multi-range read operations.
//
// TODO(b/385826024): Revert timeout to an appropriate value. This value is currently a placeholder and needs to be adjusted.
const TimeoutForMultiRangeRead = time.Hour

type MultiRangeReader struct {
	gcsx.GCSReader

	object *gcs.MinObject

	// mrdWrapper points to the wrapper object within inode.
	mrdWrapper *gcsx.MultiRangeDownloaderWrapper

	// boolean variable to determine if MRD is being used or not.
	isMRDInUse bool

	metricHandle common.MetricHandle
}

func NewMultiRangeReader(object *gcs.MinObject, metricHandle common.MetricHandle, mrdWrapper *gcsx.MultiRangeDownloaderWrapper) *MultiRangeReader {
	return &MultiRangeReader{
		object:       object,
		metricHandle: metricHandle,
		mrdWrapper:   mrdWrapper,
	}
}

// readFromMultiRangeReader reads data from the underlying MultiRangeDownloaderWrapper.
//
// It increments the reference count of the mrdWrapper if it's not already in use.
// It then calls the Read method of the mrdWrapper with the provided parameters.
//
// Parameters:
//   - ctx: The context for the read operation. It can be used to cancel the operation or set a timeout.
//   - p: The byte slice to read data into.
//   - offset: The starting offset for the read operation.
//   - end: The ending offset for the read operation.
//   - timeout: The maximum duration for the read operation.
//
// Returns:
//   - int: The number of bytes read.
//   - error: An error if the read operation fails.
func (mrd *MultiRangeReader) readFromMultiRangeReader(ctx context.Context, p []byte, offset, end int64, timeout time.Duration) (int, error) {
	if mrd.mrdWrapper == nil {
		return 0, fmt.Errorf("readFromMultiRangeReader: Invalid MultiRangeDownloaderWrapper")
	}

	if !mrd.isMRDInUse {
		mrd.isMRDInUse = true
		mrd.mrdWrapper.IncrementRefCount()
	}

	return mrd.mrdWrapper.Read(ctx, p, offset, end, timeout, mrd.metricHandle)
}

func (mrd *MultiRangeReader) ReadAt(ctx context.Context, req *gcsx.GCSReaderRequest) (gcsx.ReaderResponse, error) {
	readerResponse := gcsx.ReaderResponse{
		DataBuf: req.Buffer,
		Size:    0,
	}
	var err error

	if req.Offset >= int64(mrd.object.Size) {
		err = io.EOF
		return readerResponse, err
	}

	readerResponse.Size, err = mrd.readFromMultiRangeReader(ctx, req.Buffer, req.Offset, req.EndOffset, TimeoutForMultiRangeRead)

	return readerResponse, err
}

func (mrd *MultiRangeReader) destroy() {
	if mrd.isMRDInUse {
		err := mrd.mrdWrapper.DecrementRefCount()
		if err != nil {
			logger.Errorf("randomReader::Destroy:%v", err)
		}
		mrd.isMRDInUse = false
	}
}
