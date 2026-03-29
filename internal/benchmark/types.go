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

// Package benchmark implements a standalone GCS I/O benchmark engine for the
// gcsfuse-bench fork. It instruments the gcs.Bucket interface to record
// per-operation latency distributions using HDR histograms (no averaging of
// percentiles), and exports results as YAML or TSV.
package benchmark

import "time"

// OpType classifies a GCS operation for histogram bucketing.
type OpType string

const (
	OpRead  OpType = "read"
	OpWrite OpType = "write"
	OpStat  OpType = "stat"
	OpList  OpType = "list"
)

// PerfEvent captures a single completed I/O operation's key metrics.
// Events are collected by the instrumentedBucket and fed into histograms.
type PerfEvent struct {
	// Op is the operation type.
	Op OpType

	// TTFB is the time-to-first-byte: from issuing the request until the first
	// byte was available to the caller. Zero for non-streaming operations.
	TTFB time.Duration

	// TotalLatency is the wall-clock duration from request start to completion.
	TotalLatency time.Duration

	// BytesTransferred is the number of bytes read or written.
	BytesTransferred int64

	// ObjectSize is the logical size of the GCS object involved.
	ObjectSize int64

	// Err is non-nil when the operation failed. Failed ops are counted separately
	// but excluded from latency histograms.
	Err error
}

// TrackStats holds aggregated statistics for a single BenchmarkTrack after
// the measurement phase completes.
type TrackStats struct {
	// TrackName matches BenchmarkTrack.Name.
	TrackName string

	// OpType is the operation executed by this track ("read", "write", "stat",
	// "list"). Set to "write" for prepare-mode tracks. Used by the exporter to
	// decide which metrics are applicable (e.g. TTFB is N/A for writes).
	OpType string `yaml:"op_type,omitempty"`

	// WorkerID identifies which worker (0-based) produced this track result.
	// Set only when gcs-bench is run with --worker-id; zero otherwise.
	WorkerID int `yaml:"worker-id,omitempty"`

	// Goroutines is the number of goroutines (threads) that issued I/O for
	// this track during the benchmark run.
	Goroutines int `yaml:"goroutines"`

	// TotalOps is the count of completed operations (successes + failures).
	TotalOps int64

	// Errors is the count of failed operations.
	Errors int64

	// ThroughputBytesPerSec is the average bytes/s over the measurement window.
	ThroughputBytesPerSec float64

	// OpsPerSec is the average operation rate over the measurement window.
	OpsPerSec float64

	// AvgOpSizeBytes is the mean bytes per successful operation.
	// For writes this is the GCS-confirmed object size; for reads it is the
	// number of bytes actually received from the service.
	AvgOpSizeBytes float64

	// TTFB contains latency percentiles for time-to-first-byte (µs).
	TTFB LatencyPercentiles

	// TotalLatency contains latency percentiles for total operation time (µs).
	TotalLatency LatencyPercentiles

	// RawTTFB is the base64-encoded JSON snapshot of the TTFB HDR histogram.
	// Written when --worker-id is set; consumed by 'gcs-bench merge-results'.
	RawTTFB string `yaml:"raw-ttfb-histogram,omitempty"`

	// RawTotal is the base64-encoded JSON snapshot of the total-latency HDR histogram.
	// Written when --worker-id is set; consumed by 'gcs-bench merge-results'.
	RawTotal string `yaml:"raw-total-histogram,omitempty"`
}

// LatencyPercentiles holds a set of pre-computed HDR percentile values.
// All values are in microseconds.
type LatencyPercentiles struct {
	P50  float64 `yaml:"p50_us"`
	P90  float64 `yaml:"p90_us"`
	P95  float64 `yaml:"p95_us"`
	P99  float64 `yaml:"p99_us"`
	P999 float64 `yaml:"p999_us"`
	Max  float64 `yaml:"max_us"`
	Mean float64 `yaml:"mean_us"`
}

// RunSummary is the top-level output structure written to YAML / TSV.
type RunSummary struct {
	// StartTime is when the measurement phase began (UTC).
	StartTime time.Time `yaml:"start_time"`

	// MeasurementDuration is how long the measurement phase ran.
	MeasurementDuration time.Duration `yaml:"measurement_duration"`

	// WorkerID identifies which worker produced this summary (0-based).
	// Set only when gcs-bench runs with --worker-id.
	// -1 indicates a merged summary produced by 'gcs-bench merge-results'.
	WorkerID int `yaml:"worker-id,omitempty"`

	// Tracks contains per-track statistics.
	Tracks []TrackStats `yaml:"tracks"`
}
