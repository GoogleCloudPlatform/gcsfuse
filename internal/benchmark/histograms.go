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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
)

// TrackHistograms holds paired HDR histograms (TTFB + total latency) for one
// benchmark track. All values are stored in microseconds.
type TrackHistograms struct {
	mu sync.Mutex

	// ttfb is time-to-first-byte latency in µs.
	ttfb *hdrhistogram.Histogram
	// totalLat is total-operation latency in µs.
	totalLat *hdrhistogram.Histogram
}

// NewTrackHistograms allocates two HDR histograms using the precision from cfg.
func NewTrackHistograms(hcfg cfg.HistogramConfig) *TrackHistograms {
	return &TrackHistograms{
		ttfb:     hdrhistogram.New(hcfg.MinValueMicros, hcfg.MaxValueMicros, hcfg.SignificantDigits),
		totalLat: hdrhistogram.New(hcfg.MinValueMicros, hcfg.MaxValueMicros, hcfg.SignificantDigits),
	}
}

// RecordTTFB records a time-to-first-byte value in microseconds.
// Values below 1 µs are clamped to 1 µs to stay within histogram range.
func (h *TrackHistograms) RecordTTFB(microseconds int64) {
	if microseconds < 1 {
		microseconds = 1
	}
	h.mu.Lock()
	_ = h.ttfb.RecordValue(microseconds)
	h.mu.Unlock()
}

// RecordTotal records a total-latency value in microseconds.
// Values below 1 µs are clamped to 1 µs to stay within histogram range.
func (h *TrackHistograms) RecordTotal(microseconds int64) {
	if microseconds < 1 {
		microseconds = 1
	}
	h.mu.Lock()
	_ = h.totalLat.RecordValue(microseconds)
	h.mu.Unlock()
}

// Snapshot returns computed LatencyPercentiles for both histograms while
// holding the lock so callers see a consistent view.
func (h *TrackHistograms) Snapshot() (ttfb, total LatencyPercentiles) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ttfb = percentilesFrom(h.ttfb)
	total = percentilesFrom(h.totalLat)
	return
}

// Reset clears both histograms (used between warm-up and measurement phases).
func (h *TrackHistograms) Reset() {
	h.mu.Lock()
	h.ttfb.Reset()
	h.totalLat.Reset()
	h.mu.Unlock()
}

// percentilesFrom extracts the standard percentile set from an HDR histogram.
// IMPORTANT: We read each percentile directly from the histogram - no averaging
// or approximation between different percentile levels is performed.
func percentilesFrom(h *hdrhistogram.Histogram) LatencyPercentiles {
	return LatencyPercentiles{
		P50:  float64(h.ValueAtQuantile(50.0)),
		P90:  float64(h.ValueAtQuantile(90.0)),
		P95:  float64(h.ValueAtQuantile(95.0)),
		P99:  float64(h.ValueAtQuantile(99.0)),
		P999: float64(h.ValueAtQuantile(99.9)),
		Max:  float64(h.Max()),
		Mean: h.Mean(),
	}
}

// ExportBase64 serializes the TTFB and total-latency histograms to
// base64-encoded JSON snapshots, suitable for embedding in YAML output files
// that will later be merged with 'gcs-bench merge-results'.
// ExportBase64 acquires mu internally; callers must hold no external lock.
func (h *TrackHistograms) ExportBase64() (ttfbB64, totalB64 string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ttfbB64, err = encodeSnapshot(h.ttfb)
	if err != nil {
		return "", "", fmt.Errorf("encoding TTFB histogram: %w", err)
	}
	totalB64, err = encodeSnapshot(h.totalLat)
	if err != nil {
		return "", "", fmt.Errorf("encoding total histogram: %w", err)
	}
	return ttfbB64, totalB64, nil
}

// MergeFromBase64 decodes two base64-encoded histogram snapshots (as produced
// by ExportBase64) and merges them into this TrackHistograms.
// MergeFromBase64 acquires mu internally; callers must hold no external lock.
func (h *TrackHistograms) MergeFromBase64(ttfbB64, totalB64 string) error {
	ttfbOther, err := decodeSnapshot(ttfbB64)
	if err != nil {
		return fmt.Errorf("decoding TTFB histogram: %w", err)
	}
	totalOther, err := decodeSnapshot(totalB64)
	if err != nil {
		return fmt.Errorf("decoding total histogram: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.ttfb.Merge(ttfbOther)
	h.totalLat.Merge(totalOther)
	return nil
}

// encodeSnapshot serializes an HDR histogram to a base64-encoded JSON string
// via the library's Snapshot (Export/Import) mechanism.
func encodeSnapshot(h *hdrhistogram.Histogram) (string, error) {
	snap := h.Export()
	b, err := json.Marshal(snap)
	if err != nil {
		return "", fmt.Errorf("json marshal snapshot: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// decodeSnapshot reconstructs an HDR histogram from a base64-encoded JSON
// string produced by encodeSnapshot.
func decodeSnapshot(b64 string) (*hdrhistogram.Histogram, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	var snap hdrhistogram.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("json unmarshal snapshot: %w", err)
	}
	return hdrhistogram.Import(&snap), nil
}

// WritePercentileDistribution writes the full HDR percentile distribution for
// both the TTFB and total-latency histograms to the provided writers, using
// the standard HdrHistogram text format (compatible with hgrplot and similar
// tools at https://hdrhistogram.github.io/HdrHistogram/plotFiles.html).
// All values are reported in milliseconds (scaled from µs internal storage).
// ticksPerHalfDistance=5 is the standard resolution.
func (h *TrackHistograms) WritePercentileDistribution(ttfbW, totalW io.Writer) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, err := h.ttfb.PercentilesPrint(ttfbW, 5, 1000.0); err != nil {
		return fmt.Errorf("TTFB percentile distribution: %w", err)
	}
	if _, err := h.totalLat.PercentilesPrint(totalW, 5, 1000.0); err != nil {
		return fmt.Errorf("total latency percentile distribution: %w", err)
	}
	return nil
}
