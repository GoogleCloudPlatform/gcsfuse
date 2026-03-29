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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
)

// TestNewTrackHistogramsDefaults verifies that newly created histograms are
// zeroed and accept valid values without error.
func TestNewTrackHistogramsDefaults(t *testing.T) {
	hcfg := cfg.DefaultHistogramConfig()
	h := NewTrackHistograms(hcfg)

	// Freshly created histograms should return zero for all percentiles.
	ttfb, total := h.Snapshot()
	if ttfb.P50 != 0 || ttfb.Max != 0 {
		t.Errorf("expected zero TTFB histogram after creation, got p50=%v max=%v", ttfb.P50, ttfb.Max)
	}
	if total.P50 != 0 || total.Max != 0 {
		t.Errorf("expected zero total latency histogram after creation, got p50=%v max=%v", total.P50, total.Max)
	}
}

// TestRecordTTFB verifies that a single recorded value appears in the histogram.
func TestRecordTTFB(t *testing.T) {
	hcfg := cfg.DefaultHistogramConfig()
	h := NewTrackHistograms(hcfg)

	// Record 5000 µs (5 ms).
	h.RecordTTFB(5000)

	ttfb, _ := h.Snapshot()
	// With a single value, p50 must equal that value (within HDR precision).
	if ttfb.P50 < 4900 || ttfb.P50 > 5100 {
		t.Errorf("p50 expected ~5000 µs, got %.0f", ttfb.P50)
	}
	if ttfb.Max < 4900 {
		t.Errorf("max expected >= 5000 µs, got %.0f", ttfb.Max)
	}
}

// TestRecordTotal verifies total latency recording.
func TestRecordTotal(t *testing.T) {
	hcfg := cfg.DefaultHistogramConfig()
	h := NewTrackHistograms(hcfg)

	h.RecordTotal(10000)

	_, total := h.Snapshot()
	if total.P50 < 9900 {
		t.Errorf("total p50 expected ~10000 µs, got %.0f", total.P50)
	}
}

// TestResetClearsHistograms verifies that Reset() zeroes the histograms.
func TestResetClearsHistograms(t *testing.T) {
	hcfg := cfg.DefaultHistogramConfig()
	h := NewTrackHistograms(hcfg)

	h.RecordTTFB(5000)
	h.RecordTotal(10000)
	h.Reset()

	ttfb, total := h.Snapshot()
	if ttfb.P50 != 0 || total.P50 != 0 {
		t.Errorf("after Reset() expected p50=0, got ttfb=%.0f total=%.0f", ttfb.P50, total.P50)
	}
}

// TestClampBelowOne verifies values < 1 µs are clamped rather than crashing.
func TestClampBelowOne(t *testing.T) {
	hcfg := cfg.DefaultHistogramConfig()
	h := NewTrackHistograms(hcfg)
	h.RecordTTFB(0)  // should clamp to 1
	h.RecordTTFB(-5) // should clamp to 1
	h.RecordTotal(0)

	ttfb, total := h.Snapshot()
	if ttfb.P50 < 1 {
		t.Errorf("expected clamped value >= 1, got %.0f", ttfb.P50)
	}
	if total.P50 < 1 {
		t.Errorf("expected clamped value >= 1, got %.0f", total.P50)
	}
}

// TestPercentilesOrder verifies that p99 >= p95 >= p90 >= p50 for a
// monotonically increasing set of values.
func TestPercentilesOrder(t *testing.T) {
	hcfg := cfg.DefaultHistogramConfig()
	h := NewTrackHistograms(hcfg)

	// Record 1000 values from 1 µs to 1000 µs.
	for i := int64(1); i <= 1000; i++ {
		h.RecordTTFB(i)
	}

	ttfb, _ := h.Snapshot()
	if ttfb.P50 > ttfb.P90 {
		t.Errorf("p50 (%.0f) > p90 (%.0f)", ttfb.P50, ttfb.P90)
	}
	if ttfb.P90 > ttfb.P95 {
		t.Errorf("p90 (%.0f) > p95 (%.0f)", ttfb.P90, ttfb.P95)
	}
	if ttfb.P95 > ttfb.P99 {
		t.Errorf("p95 (%.0f) > p99 (%.0f)", ttfb.P95, ttfb.P99)
	}
	if ttfb.P99 > ttfb.P999 {
		t.Errorf("p99 (%.0f) > p999 (%.0f)", ttfb.P99, ttfb.P999)
	}
	if ttfb.P999 > ttfb.Max {
		t.Errorf("p999 (%.0f) > max (%.0f)", ttfb.P999, ttfb.Max)
	}
}

// TestDefaultBenchmarkConfig verifies defaults are populated.
func TestDefaultBenchmarkConfig(t *testing.T) {
	c := cfg.DefaultBenchmarkConfig()
	if c.Duration <= 0 {
		t.Errorf("expected positive duration, got %v", c.Duration)
	}
	if c.WarmupDuration < 0 {
		t.Errorf("expected non-negative warmup, got %v", c.WarmupDuration)
	}
	if c.TotalConcurrency <= 0 {
		t.Errorf("expected positive concurrency, got %d", c.TotalConcurrency)
	}
	if c.OutputFormat == "" {
		t.Error("expected non-empty output format")
	}
}

// TestDefaultHistogramConfig verifies histogram defaults make sense.
func TestDefaultHistogramConfig(t *testing.T) {
	hc := cfg.DefaultHistogramConfig()
	if hc.MinValueMicros <= 0 {
		t.Errorf("expected positive min, got %d", hc.MinValueMicros)
	}
	if hc.MaxValueMicros <= hc.MinValueMicros {
		t.Errorf("expected max > min, got max=%d min=%d", hc.MaxValueMicros, hc.MinValueMicros)
	}
	if hc.SignificantDigits <= 0 {
		t.Errorf("expected positive significant digits, got %d", hc.SignificantDigits)
	}
}

// TestRunSummaryRoundTrip verifies that RunSummary round-trips through its fields.
func TestRunSummaryRoundTrip(t *testing.T) {
	now := time.Now().UTC()
	s := RunSummary{
		StartTime:           now,
		MeasurementDuration: 30 * time.Second,
		WorkerID:            2,
		Tracks: []TrackStats{
			{
				TrackName:             "test",
				WorkerID:              2,
				TotalOps:              100,
				Errors:                2,
				ThroughputBytesPerSec: 1e9,
				OpsPerSec:             33.3,
				TTFB:                  LatencyPercentiles{P50: 1000, P99: 5000, Max: 10000, Mean: 1500},
				TotalLatency:          LatencyPercentiles{P50: 5000, P99: 20000, Max: 50000, Mean: 6000},
			},
		},
	}

	if s.WorkerID != 2 {
		t.Errorf("expected WorkerID=2, got %d", s.WorkerID)
	}
	if s.Tracks[0].TotalOps != 100 {
		t.Errorf("expected TotalOps=100, got %d", s.Tracks[0].TotalOps)
	}
	if s.Tracks[0].TTFB.P99 != 5000 {
		t.Errorf("expected TTFB.P99=5000, got %.0f", s.Tracks[0].TTFB.P99)
	}
}

// TestExportBase64RoundTrip verifies that histogram data survives a
// ExportBase64 → MergeFromBase64 round-trip and produces correct percentiles
// after merging two identical histograms.
func TestExportBase64RoundTrip(t *testing.T) {
	hcfg := cfg.DefaultHistogramConfig()
	h1 := NewTrackHistograms(hcfg)
	h2 := NewTrackHistograms(hcfg)

	// Record distinct values in h1 (TTFB: 1000 µs, total: 2000 µs).
	h1.RecordTTFB(1000)
	h1.RecordTotal(2000)

	// Record distinct values in h2 (TTFB: 3000 µs, total: 6000 µs).
	h2.RecordTTFB(3000)
	h2.RecordTotal(6000)

	// Serialize h2 and merge into h1.
	ttfbB64, totalB64, err := h2.ExportBase64()
	if err != nil {
		t.Fatalf("ExportBase64: %v", err)
	}
	if ttfbB64 == "" || totalB64 == "" {
		t.Fatal("ExportBase64 returned empty strings")
	}

	if err := h1.MergeFromBase64(ttfbB64, totalB64); err != nil {
		t.Fatalf("MergeFromBase64: %v", err)
	}

	// After merge h1 contains {1000, 3000} for TTFB and {2000, 6000} for total.
	// p50 should be ~1000 and p99 should be ~3000 for TTFB.
	ttfb, total := h1.Snapshot()
	if ttfb.P50 < 900 || ttfb.P50 > 1100 {
		t.Errorf("TTFB p50 expected ~1000 µs after merge, got %.0f", ttfb.P50)
	}
	if ttfb.Max < 2900 {
		t.Errorf("TTFB max expected >= 3000 µs after merge, got %.0f", ttfb.Max)
	}
	if total.Max < 5900 {
		t.Errorf("total max expected >= 6000 µs after merge, got %.0f", total.Max)
	}
}

// TestMergeFromBase64InvalidInput verifies that corrupt base64 returns an error.
func TestMergeFromBase64InvalidInput(t *testing.T) {
	hcfg := cfg.DefaultHistogramConfig()
	h := NewTrackHistograms(hcfg)

	err := h.MergeFromBase64("not-valid-base64!!!", "also-not-valid!!!")
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}
