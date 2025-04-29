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

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// TODO (b/385826024): Revert timeout to an appropriate value
const TimeoutForMultiRangeRead = time.Hour

type MultiRangeReader struct {
	gcsx.GCSReader
	object *gcs.MinObject
	reader *gcs.StorageReader
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

func (mrd *MultiRangeReader) readFromMultiRangeReader(ctx context.Context, p []byte, offset, end int64, timeout time.Duration) (bytesRead int, err error) {
	if mrd.mrdWrapper == nil {
		return 0, fmt.Errorf("readFromMultiRangeReader: Invalid MultiRangeDownloaderWrapper")
	}

	if !mrd.isMRDInUse {
		mrd.isMRDInUse = true
		mrd.mrdWrapper.IncrementRefCount()
	}

	bytesRead, err = mrd.mrdWrapper.Read(ctx, p, offset, end, timeout, mrd.metricHandle)
	return
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
