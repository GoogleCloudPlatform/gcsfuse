// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
)

// CaptureGCSReadMetrics is a helper function to encapsulate the logic for recording
// GCS read-related metrics.
func CaptureGCSReadMetrics(mh MetricHandle, readType ReadType, downloadBytes int64) {
	mh.GcsReadCount(1, readType)
	mh.GcsDownloadBytesCount(downloadBytes, readType)
}

func getMetricOpenMode(openMode util.OpenMode) OpenMode {
	isAppend := openMode.IsAppend()

	switch openMode.AccessMode() {
	case util.ReadWrite:
		if isAppend {
			return OpenModeReadWriteAppendAttr
		}
		return OpenModeReadWriteAttr
	case util.WriteOnly:
		if isAppend {
			return OpenModeWriteOnlyAppendAttr
		}
		return OpenModeWriteOnlyAttr
	default:
		return OpenModeOtherAttr
	}
}

func RecordStreamingWriteFallbackMetric(mh MetricHandle, openMode util.OpenMode, reason WriteFallbackReason) {
	var metricOpenMode OpenMode = getMetricOpenMode(openMode)
	mh.FsStreamingWriteFallbackCount(1, metricOpenMode, reason)
}
