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

package storage

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

var (
	retryLatencyTrackers = map[string]*latencyTracker{
		"GetStorageLayout":     {},
		"DeleteFolder":         {},
		"GetFolder":            {},
		"RenameFolder":         {},
		"CreateFolder":         {},
		"PollRenameFolder":     {},
		"CompleteRenameFolder": {},
	}
	retryShutdownCalled atomic.Bool
)

// RecordRetryLatency records a duration into the corresponding retry latency tracker.
func RecordRetryLatency(apiName string, duration time.Duration) {
	sec := int(duration / time.Second)
	if sec < 0 {
		sec = 0
	}
	if sec >= 300 {
		sec = 299
	}
	if tracker, ok := retryLatencyTrackers[apiName]; ok {
		tracker.latencies[sec].Add(1)
	}
}



// LogRetryLatencyStats logs the final accumulated retry latency arrays to GCSFuse logs.
func LogRetryLatencyStats() {
	if retryShutdownCalled.CompareAndSwap(false, true) {
		for apiName, tracker := range retryLatencyTrackers {
			var sb strings.Builder
			sb.WriteString(" [")
			for i := 0; i < len(tracker.latencies); i++ {
				if i > 0 {
					sb.WriteByte(',')
				}
				sb.WriteString(strconv.FormatInt(tracker.latencies[i].Load(), 10))
			}
			sb.WriteByte(']')
			logger.Infof("Retry latency stats for API %s (final): %s", apiName, sb.String())
		}

	}
}
