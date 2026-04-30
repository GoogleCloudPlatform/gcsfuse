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

package metrics

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/stretchr/testify/require"
)

type mockMetricHandle struct {
	MetricHandle
	fsStreamingWriteFallbackCountCalled bool
	inc                                 int64
	openMode                            OpenMode
	writeFallbackReason                 WriteFallbackReason
}

func (m *mockMetricHandle) FsStreamingWriteFallbackCount(inc int64, openMode OpenMode, writeFallbackReason WriteFallbackReason) {
	m.fsStreamingWriteFallbackCountCalled = true
	m.inc = inc
	m.openMode = openMode
	m.writeFallbackReason = writeFallbackReason
}

func TestRecordStreamingWriteFallbackMetric(t *testing.T) {
	testCases := []struct {
		name                string
		accessMode          int
		fileFlags           int
		writeFallbackReason WriteFallbackReason
		expectedOpenMode    OpenMode
	}{
		{
			name:                "ReadWrite_NoAppend",
			accessMode:          util.ReadWrite,
			fileFlags:           0,
			writeFallbackReason: WriteFallbackReasonConcurrencyLimitBreachedAttr,
			expectedOpenMode:    OpenModeReadWriteAttr,
		},
		{
			name:                "ReadWrite_Append",
			accessMode:          util.ReadWrite,
			fileFlags:           util.O_APPEND,
			writeFallbackReason: WriteFallbackReasonExistingFileAttr,
			expectedOpenMode:    OpenModeReadWriteAppendAttr,
		},
		{
			name:                "WriteOnly_NoAppend",
			accessMode:          util.WriteOnly,
			fileFlags:           0,
			writeFallbackReason: WriteFallbackReasonOtherAttr,
			expectedOpenMode:    OpenModeWriteOnlyAttr,
		},
		{
			name:                "WriteOnly_Append",
			accessMode:          util.WriteOnly,
			fileFlags:           util.O_APPEND,
			writeFallbackReason: WriteFallbackReasonOutOfOrderAttr,
			expectedOpenMode:    OpenModeWriteOnlyAppendAttr,
		},
		{
			name:                "ReadOnly",
			accessMode:          util.ReadOnly,
			fileFlags:           0,
			writeFallbackReason: WriteFallbackReasonOtherAttr,
			expectedOpenMode:    OpenModeOtherAttr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockMetricHandle{
				MetricHandle: NewNoopMetrics(),
			}
			om := util.NewOpenMode(tc.accessMode, tc.fileFlags)

			RecordStreamingWriteFallbackMetric(mock, om, tc.writeFallbackReason)

			require.True(t, mock.fsStreamingWriteFallbackCountCalled)
			require.Equal(t, int64(1), mock.inc)
			require.Equal(t, tc.expectedOpenMode, mock.openMode)
			require.Equal(t, tc.writeFallbackReason, mock.writeFallbackReason)
		})
	}
}
