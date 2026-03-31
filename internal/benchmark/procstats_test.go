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

package benchmark

import (
	"testing"
)

// TestReadPeakRSSKiB verifies that the process's peak RSS is positive. On any
// running Linux process this must be > 0.
func TestReadPeakRSSKiB(t *testing.T) {
	rss := readPeakRSSKiB()
	if rss <= 0 {
		t.Errorf("expected positive peak RSS, got %d KiB (check /proc/self/status)", rss)
	}
}

// TestReadProcCPU verifies that we can parse /proc/self/stat successfully and
// that the running process has accumulated at least some CPU time.
func TestReadProcCPU(t *testing.T) {
	stats, err := readProcCPU()
	if err != nil {
		t.Fatalf("readProcCPU: unexpected error: %v", err)
	}
	total := stats.userTicks + stats.sysTicks
	if total == 0 {
		t.Errorf("expected non-zero CPU ticks for running process, got user=%d sys=%d",
			stats.userTicks, stats.sysTicks)
	}
}

// TestReadSystemCPU verifies that we can parse /proc/stat and that the values
// are internally consistent (idle ≤ total).
func TestReadSystemCPU(t *testing.T) {
	info, err := readSystemCPU()
	if err != nil {
		t.Fatalf("readSystemCPU: unexpected error: %v", err)
	}
	if info.total() == 0 {
		t.Fatal("expected non-zero total CPU jiffies from /proc/stat")
	}
	if info.idleTotal() > info.total() {
		t.Errorf("idle (%d) > total (%d) — unexpected /proc/stat parsing",
			info.idleTotal(), info.total())
	}
}

// TestSystemCPUPercentRange verifies that systemCPUPercent returns a
// value in the range [0, 100 × NumCPU]. We take two snapshots around a CPU
// burn loop to ensure the ticks advance.
func TestSystemCPUPercentRange(t *testing.T) {
	before, err := readSystemCPU()
	if err != nil {
		t.Fatalf("readSystemCPU before: %v", err)
	}
	// Burn a small amount of CPU so ticks advance.
	sum := 0
	for i := 0; i < 2_000_000; i++ {
		sum += i
	}
	_ = sum
	after, err := readSystemCPU()
	if err != nil {
		t.Fatalf("readSystemCPU after: %v", err)
	}
	pct := systemCPUPercent(before, after)
	// On a quiet machine two successive reads may show 0 ticks difference —
	// that is still correct (0% utilisation). Upper bound is generous to
	// accommodate many-core machines.
	if pct < 0 || pct > 100.0*1024 {
		t.Errorf("systemCPUPercent = %.2f%% — outside plausible range", pct)
	}
}

// TestTicksToMs verifies the jiffy-to-millisecond conversion constant.
// At CLK_TCK = 100 Hz each tick is 10 ms, so 100 ticks → 1000 ms.
func TestTicksToMs(t *testing.T) {
	cases := []struct {
		ticks  uint64
		wantMs int64
	}{
		{0, 0},
		{1, 10},
		{100, 1000},
		{1000, 10000},
	}
	for _, tc := range cases {
		got := ticksToMs(tc.ticks)
		if got != tc.wantMs {
			t.Errorf("ticksToMs(%d) = %d ms, want %d ms", tc.ticks, got, tc.wantMs)
		}
	}
}

// TestSystemCPUPercentZeroTicks verifies that zero tick delta returns exactly 0.
func TestSystemCPUPercentZeroTicks(t *testing.T) {
	info := systemCPUInfo{user: 100, idle: 200}
	pct := systemCPUPercent(info, info) // same snapshot → delta = 0
	if pct != 0 {
		t.Errorf("systemCPUPercent with identical snapshots = %.2f, want 0", pct)
	}
}
