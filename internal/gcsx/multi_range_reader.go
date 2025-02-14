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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
)

type MultiRangeReaderStrategy struct {
	wrapper *MultiRangeDownloaderWrapper
	// boolean variable to determine if MRD is being used or not.
	isMRDInUse     bool
	metricHandle   common.MetricHandle
	totalReadBytes uint64
}

func (m *MultiRangeReaderStrategy) Read(ctx context.Context, p []byte, offset, end int64, timeout time.Duration) (int, error) {
	if m.wrapper == nil {
		return 0, fmt.Errorf("readFromMultiRangeReader: Invalid MultiRangeDownloaderWrapper")
	}

	if !m.isMRDInUse {
		m.isMRDInUse = true
		m.wrapper.IncrementRefCount()
	}

	bytesRead, err := m.wrapper.Read(ctx, p, offset, end, timeout, m.metricHandle)
	m.totalReadBytes += uint64(bytesRead)
	return bytesRead, err
}

func (m *MultiRangeReaderStrategy) Destroy() {
	if m.isMRDInUse {
		err := m.wrapper.DecrementRefCount()
		if err != nil {
			logger.Errorf("randomReader::Destroy:%v", err)
		}
		m.isMRDInUse = false
	}
}

// closeReader fetches the readHandle before closing the reader instance.
func (m *MultiRangeReaderStrategy) closeReader() {
}
