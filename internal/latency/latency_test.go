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
	"sync"
	"testing"
	"time"
)

func TestRecordGetFolderLatency(t *testing.T) {
	ResetForTest()
	defer ResetForTest()

	tests := []struct {
		duration      time.Duration
		expectedIndex int
	}{
		{duration: 0 * time.Second, expectedIndex: 0},
		{duration: 500 * time.Millisecond, expectedIndex: 1}, // ceil(0.5) = 1
		{duration: 1 * time.Second, expectedIndex: 1},
		{duration: 1001 * time.Millisecond, expectedIndex: 2},     // ceil(1.001) = 2
		{duration: 1500 * time.Millisecond, expectedIndex: 2},     // ceil(1.5) = 2
		{duration: 299500 * time.Millisecond, expectedIndex: 300}, // ceil(299.5) = 300
		{duration: 300 * time.Second, expectedIndex: 300},
		{duration: 3000001 * time.Microsecond, expectedIndex: 4}, // ceil(3.000001) = 4
		{duration: 3000 * time.Second, expectedIndex: 3000},
		{duration: 3001 * time.Second, expectedIndex: 3001},
		{duration: 3600 * time.Second, expectedIndex: 3001}, // capped at 3001
		{duration: -1 * time.Second, expectedIndex: 0},      // capped at 0
	}

	for _, tc := range tests {
		ResetForTest()
		RecordGetFolderLatency(tc.duration)
		snap, _, _, _, _, _ := GetLatenciesForTest()
		if snap[tc.expectedIndex] != 1 {
			t.Errorf("For duration %v, expected count at index %d to be 1, got 0", tc.duration, tc.expectedIndex)
		}
		// Verify other indexes are 0
		for i, val := range snap {
			if i != tc.expectedIndex && val != 0 {
				t.Errorf("For duration %v, index %d should be 0, got %d", tc.duration, i, val)
			}
		}
	}
}

func TestConcurrentRecordLatency(t *testing.T) {
	ResetForTest()
	defer ResetForTest()

	var wg sync.WaitGroup
	workers := 100
	iterations := 1000

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Record a mixture of latencies
				RecordGetFolderLatency(time.Duration(workerID) * time.Second)
			}
		}(i)
	}

	wg.Wait()
	snap, _, _, _, _, _ := GetLatenciesForTest()

	// Verify total count matches
	var total int64
	for _, val := range snap {
		total += val
	}
	expectedTotal := int64(workers * iterations)
	if total != expectedTotal {
		t.Errorf("Expected total recorded latency count to be %d, got %d", expectedTotal, total)
	}
}
