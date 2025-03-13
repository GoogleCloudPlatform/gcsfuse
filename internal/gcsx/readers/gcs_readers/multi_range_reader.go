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

package gcs_readers

import (
	"fmt"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"golang.org/x/net/context"
)

// TODO(b/385826024): Revert timeout to an appropriate value
const TimeoutForMultiRangeRead = time.Hour

type MultiRangeReader struct {
	End int64
	// mrdWrapper points to the wrapper object within inode.
	MrdWrapper *MultiRangeDownloaderWrapper

	// boolean variable to determine if MRD is being used or not.
	isMRDInUse bool

	MetricHandle common.MetricHandle

	TotalReadBytes uint64
}

func (mrd *MultiRangeReader) Object() *gcs.MinObject {
	return nil
}

func (mrd *MultiRangeReader) CheckInvariants() {
}

func (mrd *MultiRangeReader) readFromMultiRangeReader(ctx context.Context, p []byte, offset, end int64, timeout time.Duration) (bytesRead int, err error) {
	if mrd.MrdWrapper == nil {
		return 0, fmt.Errorf("readFromMultiRangeReader: Invalid MultiRangeDownloaderWrapper")
	}

	if !mrd.isMRDInUse {
		mrd.isMRDInUse = true
		mrd.MrdWrapper.IncrementRefCount()
	}

	bytesRead, err = mrd.MrdWrapper.Read(ctx, p, offset, end, timeout, mrd.MetricHandle)
	mrd.TotalReadBytes += uint64(bytesRead)
	return
}

func (mrd *MultiRangeReader) ReadAt(ctx context.Context, p []byte, offset int64) (readers.ObjectData, error) {
	o := readers.ObjectData{
		DataBuf:  p,
		CacheHit: false,
		Size:     0,
	}
	var err error

	o.Size, err = mrd.readFromMultiRangeReader(ctx, p, offset, mrd.End, TimeoutForMultiRangeRead)

	return o, err
}

func (mrd *MultiRangeReader) Destroy() {
	if mrd.isMRDInUse {
		err := mrd.MrdWrapper.DecrementRefCount()
		if err != nil {
			logger.Errorf("randomReader::Destroy:%v", err)
		}
		mrd.isMRDInUse = false
	}
}
