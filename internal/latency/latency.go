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

package latency

import (
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

const ArraySize = 3002

var (
	getFolderLatencies [ArraySize]atomic.Int64
	shutdownCalled     atomic.Bool
)

// RecordGetFolderLatency records a duration into the latency array.
func RecordGetFolderLatency(elapsed time.Duration) {
	sec := elapsed.Seconds()
	val := int(math.Ceil(sec))
	if val < 0 {
		val = 0
	}
	if val >= ArraySize {
		val = ArraySize - 1
	}
	getFolderLatencies[val].Add(1)
}

// Shutdown prints the final array. It is thread-safe and runs exactly once.
func Shutdown() {
	if shutdownCalled.CompareAndSwap(false, true) {
		logArray("GetFolder latency array (final):")
	}
}

func logArray(prefix string) {
	var sb strings.Builder
	sb.WriteString(" [")
	for i := 0; i < ArraySize; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.FormatInt(getFolderLatencies[i].Load(), 10))
	}
	sb.WriteByte(']')
	logger.Infof("%s %s", prefix, sb.String())
}

// ResetForTest resets the latency tracker state. Used for unit testing.
func ResetForTest() {
	shutdownCalled.Store(false)
	for i := 0; i < ArraySize; i++ {
		getFolderLatencies[i].Store(0)
	}
}

// GetLatenciesForTest returns a snapshot of current latencies. Used for unit testing.
func GetLatenciesForTest() [ArraySize]int64 {
	var snap [ArraySize]int64
	for i := 0; i < ArraySize; i++ {
		snap[i] = getFolderLatencies[i].Load()
	}
	return snap
}
