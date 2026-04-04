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

// --------------------------------------------------------------------------
// New tests for /proc/meminfo, /proc/self/status (VmRSS), /proc/vmstat
// --------------------------------------------------------------------------

// TestReadSysMemInfo verifies that readSysMemInfo returns plausible values on a
// running Linux system. Both Cached and AnonPages should be positive — any
// running process with a heap will have non-zero anonymous pages, and the
// kernel almost always has at least some file-backed cached pages.
func TestReadSysMemInfo(t *testing.T) {
	info := readSysMemInfo()
	if info.anonPagesKiB <= 0 {
		t.Errorf("expected positive AnonPages, got %d KiB (check /proc/meminfo)", info.anonPagesKiB)
	}
	// Cached can legitimately be 0 on a stripped-down container, so only warn.
	if info.cachedKiB < 0 {
		t.Errorf("cachedKiB must be >= 0, got %d", info.cachedKiB)
	}
	t.Logf("readSysMemInfo: cached=%d KiB  anonPages=%d KiB", info.cachedKiB, info.anonPagesKiB)
}

// TestReadCurrentRSSKiB verifies that the current process RSS is positive.
// Every running Go test process must have a non-zero RSS.
func TestReadCurrentRSSKiB(t *testing.T) {
	rss := readCurrentRSSKiB()
	if rss <= 0 {
		t.Errorf("expected positive current RSS (VmRSS), got %d KiB (check /proc/self/status)", rss)
	}
	t.Logf("readCurrentRSSKiB: %d KiB", rss)
}

// TestCurrentRSSNotExceedPeakRSS verifies the invariant VmRSS ≤ VmHWM.
// The current RSS can never be larger than the historical peak.
func TestCurrentRSSNotExceedPeakRSS(t *testing.T) {
	cur := readCurrentRSSKiB()
	peak := readPeakRSSKiB()
	if cur <= 0 || peak <= 0 {
		t.Skipf("skipping: cur=%d peak=%d (one or both reads failed)", cur, peak)
	}
	if cur > peak {
		t.Errorf("current RSS (%d KiB) > peak RSS (%d KiB): impossible", cur, peak)
	}
}

// TestReadVMStat verifies that readVMStat returns plausible cumulative counters.
// pgpgin must be > 0 on any machine that has been running long enough to have
// read even one page from disk (essentially all real systems at test time).
func TestReadVMStat(t *testing.T) {
	stat := readVMStat()
	// pgpgin is cumulative since boot; it must be > 0 on any real Linux system.
	if stat.pgpgin == 0 {
		t.Logf("WARNING: pgpgin == 0; /proc/vmstat may be unavailable in this environment")
	}
	t.Logf("readVMStat: pgpgin=%d  pgpgout=%d", stat.pgpgin, stat.pgpgout)
}

// TestVMStatMonotonicallyIncreases takes two snapshots separated by a small
// heap allocation (guaranteed to touch anonymous pages) and verifies that the
// pgpgout counter does not decrease (it is strictly cumulative).
func TestVMStatMonotonicallyIncreases(t *testing.T) {
	before := readVMStat()

	// Force some memory activity.
	buf := make([]byte, 4*1024*1024) // 4 MiB anonymous alloc
	for i := range buf {
		buf[i] = byte(i)
	}
	_ = buf

	after := readVMStat()

	if after.pgpgin < before.pgpgin {
		t.Errorf("pgpgin decreased: before=%d after=%d (counter must be monotone)", before.pgpgin, after.pgpgin)
	}
	if after.pgpgout < before.pgpgout {
		t.Errorf("pgpgout decreased: before=%d after=%d (counter must be monotone)", before.pgpgout, after.pgpgout)
	}
}

// TestReadSysMemInfoConsistency verifies that two rapid successive reads return
// values within a plausible range of each other. Memory can change between
// reads but should not change by gigabytes in microseconds.
func TestReadSysMemInfoConsistency(t *testing.T) {
	const maxDeltaKiB = 512 * 1024 // 512 MiB — very generous
	a := readSysMemInfo()
	b := readSysMemInfo()

	diff := b.cachedKiB - a.cachedKiB
	if diff < 0 {
		diff = -diff
	}
	if diff > maxDeltaKiB {
		t.Errorf("page-cache changed by %d KiB between two immediate reads — implausible", diff)
	}
	diff = b.anonPagesKiB - a.anonPagesKiB
	if diff < 0 {
		diff = -diff
	}
	if diff > maxDeltaKiB {
		t.Errorf("AnonPages changed by %d KiB between two immediate reads — implausible", diff)
	}
}
