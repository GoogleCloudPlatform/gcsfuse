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
	getFolderLatencies    [ArraySize]atomic.Int64
	createFolderLatencies [ArraySize]atomic.Int64
	deleteFolderLatencies [ArraySize]atomic.Int64
	renameFolderLatencies [ArraySize]atomic.Int64
	lookUpInodeLatencies  [ArraySize]atomic.Int64
	statObjectLatencies   [ArraySize]atomic.Int64
	shutdownCalled        atomic.Bool
)

func record(elapsed time.Duration, arr *[ArraySize]atomic.Int64) {
	sec := elapsed.Seconds()
	val := int(math.Ceil(sec))
	if val < 0 {
		val = 0
	}
	if val >= ArraySize {
		val = ArraySize - 1
	}
	arr[val].Add(1)
}

// RecordGetFolderLatency records a duration into the GetFolder latency array.
func RecordGetFolderLatency(elapsed time.Duration) {
	record(elapsed, &getFolderLatencies)
}

// RecordCreateFolderLatency records a duration into the CreateFolder latency array.
func RecordCreateFolderLatency(elapsed time.Duration) {
	record(elapsed, &createFolderLatencies)
}

// RecordDeleteFolderLatency records a duration into the DeleteFolder latency array.
func RecordDeleteFolderLatency(elapsed time.Duration) {
	record(elapsed, &deleteFolderLatencies)
}

// RecordRenameFolderLatency records a duration into the RenameFolder latency array.
func RecordRenameFolderLatency(elapsed time.Duration) {
	record(elapsed, &renameFolderLatencies)
}

// RecordLookUpInodeLatency records a duration into the LookUpInode latency array.
func RecordLookUpInodeLatency(elapsed time.Duration) {
	record(elapsed, &lookUpInodeLatencies)
}

// RecordStatObjectLatency records a duration into the StatObject latency array.
func RecordStatObjectLatency(elapsed time.Duration) {
	record(elapsed, &statObjectLatencies)
}

// Shutdown prints the final arrays. It is thread-safe and runs exactly once.
func Shutdown() {
	if shutdownCalled.CompareAndSwap(false, true) {
		logArray("GetFolder latency array (final):", &getFolderLatencies)
		logArray("CreateFolder latency array (final):", &createFolderLatencies)
		logArray("DeleteFolder latency array (final):", &deleteFolderLatencies)
		logArray("RenameFolder latency array (final):", &renameFolderLatencies)
		logArray("LookUpInode latency array (final):", &lookUpInodeLatencies)
		logArray("StatObject latency array (final):", &statObjectLatencies)
	}
}

func logArray(prefix string, arr *[ArraySize]atomic.Int64) {
	var sb strings.Builder
	sb.WriteString(" [")
	for i := 0; i < ArraySize; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.FormatInt(arr[i].Load(), 10))
	}
	sb.WriteByte(']')
	logger.Infof("%s %s", prefix, sb.String())
}

// ResetForTest resets the latency tracker state. Used for unit testing.
func ResetForTest() {
	shutdownCalled.Store(false)
	resetArray(&getFolderLatencies)
	resetArray(&createFolderLatencies)
	resetArray(&deleteFolderLatencies)
	resetArray(&renameFolderLatencies)
	resetArray(&lookUpInodeLatencies)
	resetArray(&statObjectLatencies)
}

func resetArray(arr *[ArraySize]atomic.Int64) {
	for i := 0; i < ArraySize; i++ {
		arr[i].Store(0)
	}
}

// GetLatenciesForTest returns snapshots of current latencies. Used for unit testing.
func GetLatenciesForTest() (
	getFolder [ArraySize]int64,
	createFolder [ArraySize]int64,
	deleteFolder [ArraySize]int64,
	renameFolder [ArraySize]int64,
	lookUpInode [ArraySize]int64,
	statObject [ArraySize]int64,
) {
	getFolder = snapArray(&getFolderLatencies)
	createFolder = snapArray(&createFolderLatencies)
	deleteFolder = snapArray(&deleteFolderLatencies)
	renameFolder = snapArray(&renameFolderLatencies)
	lookUpInode = snapArray(&lookUpInodeLatencies)
	statObject = snapArray(&statObjectLatencies)
	return
}

func snapArray(arr *[ArraySize]atomic.Int64) [ArraySize]int64 {
	var snap [ArraySize]int64
	for i := 0; i < ArraySize; i++ {
		snap[i] = arr[i].Load()
	}
	return snap
}
