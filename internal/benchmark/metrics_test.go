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

// Package benchmark — metrics collection unit tests.
//
// These tests cover three properties that must hold after any engine.Run():
//   1. AvgOpSizeBytes is populated and accurate (totalBytes / successfulOps).
//   2. TotalLatency.P50 > 0  (every op type must record its wall-clock time).
//   3. TTFB.P50 > 0 for read operations.
//
// All tests use the in-process mockBucket defined in engine_test.go so that no
// network calls are made.  The mockBucket injects a small delay
// (writeDelay / readDelay) to guarantee that measured latency is non-zero even
// on very fast machines.

package benchmark

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
)

// ── helpers ────────────────────────────────────────────────────────────────

const mockObjectSize = uint64(8192) // bytes the mock bucket reports per write

// makePrepareConfig returns a minimal prepare-mode BenchmarkConfig that writes
// objectCount flat objects with a fixed size equal to objectSizeMin/Max.
func makePrepareConfig(objectCount int) cfg.BenchmarkConfig {
	return cfg.BenchmarkConfig{
		Mode:             "prepare",
		TotalConcurrency: 2,
		OutputFormat:     "yaml",
		Histograms:       cfg.DefaultHistogramConfig(),
		Tracks: []cfg.BenchmarkTrack{
			{
				Name:          "test-prepare",
				OpType:        "write",
				Weight:        1,
				ObjectSizeMin: int64(mockObjectSize),
				ObjectSizeMax: int64(mockObjectSize),
				ObjectCount:   objectCount,
				Concurrency:   2,
			},
		},
	}
}

// makeWriteBenchConfig returns a time-bounded write benchmark config.
func makeWriteBenchConfig(duration time.Duration, concurrency int) cfg.BenchmarkConfig {
	return cfg.BenchmarkConfig{
		Duration:         duration,
		WarmupDuration:   0,
		TotalConcurrency: concurrency,
		OutputFormat:     "yaml",
		Histograms:       cfg.DefaultHistogramConfig(),
		Tracks: []cfg.BenchmarkTrack{
			{
				Name:          "write-track",
				OpType:        "write",
				Weight:        1,
				ObjectCount:   50,
				ObjectSizeMin: int64(mockObjectSize),
				ObjectSizeMax: int64(mockObjectSize),
				Concurrency:   concurrency,
			},
		},
	}
}

// runPrepare is a convenience wrapper.
func runPrepare(t *testing.T, mb *mockBucket, objectCount int) TrackStats {
	t.Helper()
	engine, err := NewEngine(mb, makePrepareConfig(objectCount))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	summary, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run (prepare): %v", err)
	}
	if len(summary.Tracks) != 1 {
		t.Fatalf("expected 1 track in summary, got %d", len(summary.Tracks))
	}
	return summary.Tracks[0]
}

// runBenchRead is a convenience wrapper for a short read benchmark.
func runBenchRead(t *testing.T, mb *mockBucket, duration time.Duration, concurrency int) TrackStats {
	t.Helper()
	engine, err := NewEngine(mb, makeReadOnlyConfig(duration, 0, concurrency))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	summary, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run (benchmark read): %v", err)
	}
	if len(summary.Tracks) != 1 {
		t.Fatalf("expected 1 track in summary, got %d", len(summary.Tracks))
	}
	return summary.Tracks[0]
}

// runBenchWrite is a convenience wrapper for a short write benchmark.
func runBenchWrite(t *testing.T, mb *mockBucket, duration time.Duration, concurrency int) TrackStats {
	t.Helper()
	engine, err := NewEngine(mb, makeWriteBenchConfig(duration, concurrency))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	summary, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run (benchmark write): %v", err)
	}
	if len(summary.Tracks) != 1 {
		t.Fatalf("expected 1 track in summary, got %d", len(summary.Tracks))
	}
	return summary.Tracks[0]
}

// ── AvgOpSizeBytes tests ───────────────────────────────────────────────────

// TestPrepareSummaryAvgOpSizeIsNonZero verifies that AvgOpSizeBytes is > 0
// after a prepare run that successfully writes objects.
//
// Fails before fix: AvgOpSizeBytes is always 0 (field not computed).
func TestPrepareSummaryAvgOpSizeIsNonZero(t *testing.T) {
	mb := &mockBucket{objectSize: mockObjectSize}
	ts := runPrepare(t, mb, 10)

	if ts.AvgOpSizeBytes <= 0 {
		t.Errorf("expected AvgOpSizeBytes > 0, got %.1f", ts.AvgOpSizeBytes)
	}
}

// TestPrepareSummaryAvgOpSizeMatchesMockObjectSize verifies that AvgOpSizeBytes
// matches the size that the mock bucket reports for every CreateObject call.
//
// Fails before fix: AvgOpSizeBytes is 0 instead of mockObjectSize.
func TestPrepareSummaryAvgOpSizeMatchesMockObjectSize(t *testing.T) {
	mb := &mockBucket{objectSize: mockObjectSize}
	ts := runPrepare(t, mb, 20)

	want := float64(mockObjectSize)
	if ts.AvgOpSizeBytes != want {
		t.Errorf("AvgOpSizeBytes: want %.0f, got %.1f", want, ts.AvgOpSizeBytes)
	}
}

// TestBenchmarkSummaryAvgOpSizeIsNonZero verifies AvgOpSizeBytes > 0 after a
// time-bounded read benchmark.
//
// Fails before fix: field is 0.
func TestBenchmarkSummaryReadAvgOpSizeIsNonZero(t *testing.T) {
	mb := &mockBucket{readBytes: 4096}
	ts := runBenchRead(t, mb, 200*time.Millisecond, 2)

	if ts.AvgOpSizeBytes <= 0 {
		t.Errorf("expected AvgOpSizeBytes > 0, got %.1f", ts.AvgOpSizeBytes)
	}
}

// TestBenchmarkSummaryWriteAvgOpSizeMatchesMockObjectSize verifies that an
// in-process write benchmark reports the correct average op size.
//
// Fails before fix: field is 0.
func TestBenchmarkSummaryWriteAvgOpSizeMatchesMockObjectSize(t *testing.T) {
	mb := &mockBucket{objectSize: mockObjectSize}
	ts := runBenchWrite(t, mb, 200*time.Millisecond, 2)

	want := float64(mockObjectSize)
	if ts.AvgOpSizeBytes != want {
		t.Errorf("AvgOpSizeBytes: want %.0f, got %.1f", want, ts.AvgOpSizeBytes)
	}
}

// TestAvgOpSizeIsZeroWhenAllOpsFail verifies that AvgOpSizeBytes is 0 (not
// NaN or Inf) when every CreateObject call fails and no bytes are transferred.
//
// This tests the zero-division guard: successfulOps == 0 → AvgOpSizeBytes = 0.
func TestAvgOpSizeIsZeroWhenAllOpsFail(t *testing.T) {
	simErr := fmt.Errorf("simulated CreateObject failure")

	mb := &mockBucket{createErr: simErr}
	// Only 5 objects so the prepare loop exits quickly.
	engine, err := NewEngine(mb, makePrepareConfig(5))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	summary, err := engine.Run(context.Background())
	// prepare returns non-nil err only on context cancellation; errors from
	// individual writes are counted but not fatal at the Run level.
	if err != nil {
		t.Fatalf("engine.Run: unexpected fatal error: %v", err)
	}
	if len(summary.Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(summary.Tracks))
	}
	ts := summary.Tracks[0]
	if ts.AvgOpSizeBytes != 0 {
		t.Errorf("expected AvgOpSizeBytes=0 when all ops fail, got %.1f", ts.AvgOpSizeBytes)
	}
}

// ── Latency tests ─────────────────────────────────────────────────────────

// TestPrepareSummaryWriteLatencyRecorded verifies that TotalLatency.P50 > 0
// after a prepare run.  The mock introduces a 1 ms write delay so that the
// measured latency is well above the 1 µs histogram floor.
//
// Fails before fix: doWrite never records to ts.hists → P50 == 0.
func TestPrepareSummaryWriteLatencyRecorded(t *testing.T) {
	mb := &mockBucket{
		objectSize: mockObjectSize,
		writeDelay: time.Millisecond,
	}
	ts := runPrepare(t, mb, 5)

	if ts.TotalLatency.P50 <= 0 {
		t.Errorf("expected TotalLatency.P50 > 0, got %.1f µs", ts.TotalLatency.P50)
	}
}

// TestPrepareSummaryWriteLatencyWithinExpectedRange verifies the measured write
// latency is in a plausible range: the mock sleeps for 1 ms, so P50 must be
// at least 500 µs (generous lower bound) and at most 10 s (sanity upper bound).
//
// Fails before fix: P50 == 0.
func TestPrepareSummaryWriteLatencyWithinExpectedRange(t *testing.T) {
	mb := &mockBucket{
		objectSize: mockObjectSize,
		writeDelay: time.Millisecond,
	}
	ts := runPrepare(t, mb, 5)

	const minUs = 500.0 // 0.5 ms — conservative lower bound
	const maxUs = 1e7   // 10 s  — sanity upper bound
	if ts.TotalLatency.P50 < minUs || ts.TotalLatency.P50 > maxUs {
		t.Errorf("TotalLatency.P50 out of expected range [%.0f, %.0f] µs, got %.1f µs",
			minUs, maxUs, ts.TotalLatency.P50)
	}
}

// TestBenchmarkSummaryReadTotalLatencyRecorded verifies that TotalLatency.P50
// is non-zero after a time-bounded read benchmark.  The mock injects a 1 ms
// read delay on the first Read() call.
//
// Fails before fix: doRead never records to ts.hists → P50 == 0.
func TestBenchmarkSummaryReadTotalLatencyRecorded(t *testing.T) {
	mb := &mockBucket{
		readBytes: 4096,
		readDelay: time.Millisecond,
	}
	ts := runBenchRead(t, mb, 200*time.Millisecond, 1)

	if ts.TotalLatency.P50 <= 0 {
		t.Errorf("expected TotalLatency.P50 > 0, got %.1f µs", ts.TotalLatency.P50)
	}
}

// TestBenchmarkSummaryReadTTFBRecorded verifies that TTFB.P50 > 0 after a
// read benchmark.  TTFB is the time from issuing the request until the first
// byte is available; the mock simulates this by sleeping in the first Read().
//
// Fails before fix: doRead does not record TTFB → TTFB.P50 == 0.
func TestBenchmarkSummaryReadTTFBRecorded(t *testing.T) {
	mb := &mockBucket{
		readBytes: 4096,
		readDelay: time.Millisecond,
	}
	ts := runBenchRead(t, mb, 200*time.Millisecond, 1)

	if ts.TTFB.P50 <= 0 {
		t.Errorf("expected TTFB.P50 > 0, got %.1f µs", ts.TTFB.P50)
	}
}

// TestBenchmarkSummaryWriteLatencyRecorded verifies TotalLatency.P50 > 0 for
// a time-bounded write benchmark.
//
// Fails before fix: doWrite never records to ts.hists → P50 == 0.
func TestBenchmarkSummaryWriteLatencyRecorded(t *testing.T) {
	mb := &mockBucket{
		objectSize: mockObjectSize,
		writeDelay: time.Millisecond,
	}
	ts := runBenchWrite(t, mb, 200*time.Millisecond, 1)

	if ts.TotalLatency.P50 <= 0 {
		t.Errorf("expected TotalLatency.P50 > 0, got %.1f µs", ts.TotalLatency.P50)
	}
}

// TestBenchmarkSummaryTTFBNotSetForWrites verifies that TTFB remains 0 for
// write-only benchmarks: TTFB is only meaningful for reads.
func TestBenchmarkSummaryTTFBNotSetForWrites(t *testing.T) {
	mb := &mockBucket{
		objectSize: mockObjectSize,
		writeDelay: time.Millisecond,
	}
	ts := runBenchWrite(t, mb, 200*time.Millisecond, 1)

	// TTFB is not recorded for writes; it should remain 0.
	if ts.TTFB.P50 != 0 {
		t.Errorf("expected TTFB.P50==0 for write-only benchmark, got %.1f µs", ts.TTFB.P50)
	}
}

// ── Throughput consistency tests ──────────────────────────────────────────

// TestPrepareSummaryThroughputConsistentWithAvgSize verifies that the reported
// throughput (bytes/s) is consistent with ops/s × avg size to within 1%.
//
// throughput = opsPerSec × avgOpSizeBytes (within floating-point rounding)
func TestPrepareSummaryThroughputConsistentWithAvgSize(t *testing.T) {
	mb := &mockBucket{objectSize: mockObjectSize}
	ts := runPrepare(t, mb, 50)

	if ts.OpsPerSec <= 0 || ts.AvgOpSizeBytes <= 0 {
		t.Skipf("skipping consistency check: OpsPerSec=%.1f AvgOpSizeBytes=%.1f", ts.OpsPerSec, ts.AvgOpSizeBytes)
	}
	derived := ts.OpsPerSec * ts.AvgOpSizeBytes
	ratio := ts.ThroughputBytesPerSec / derived
	if ratio < 0.99 || ratio > 1.01 {
		t.Errorf("throughput inconsistency: reported %.0f B/s, derived %.0f B/s (ratio %.4f)",
			ts.ThroughputBytesPerSec, derived, ratio)
	}
}

// TestBenchmarkSummaryThroughputConsistentWithAvgSize performs the same
// consistency check for a time-bounded read benchmark.
func TestBenchmarkSummaryThroughputConsistentWithAvgSize(t *testing.T) {
	mb := &mockBucket{readBytes: 4096}
	ts := runBenchRead(t, mb, 200*time.Millisecond, 2)

	if ts.OpsPerSec <= 0 || ts.AvgOpSizeBytes <= 0 {
		t.Skipf("no ops completed")
	}
	derived := ts.OpsPerSec * ts.AvgOpSizeBytes
	ratio := ts.ThroughputBytesPerSec / derived
	if ratio < 0.99 || ratio > 1.01 {
		t.Errorf("throughput inconsistency: reported %.0f B/s, derived %.0f B/s (ratio %.4f)",
			ts.ThroughputBytesPerSec, derived, ratio)
	}
}
