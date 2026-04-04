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

// RuntimeStats captures process-level memory and CPU resource usage over the
// measurement phase. All fields are written to the YAML output automatically
// via struct tags; the human-readable report renders them in a dedicated
// "Runtime Statistics" section.
type RuntimeStats struct {
	// Go runtime heap statistics (snapshot at end of measurement).
	GoHeapAllocBytes  uint64 `yaml:"go_heap_alloc_bytes"`  // live heap objects
	GoHeapSysBytes    uint64 `yaml:"go_heap_sys_bytes"`    // heap virtual memory reserved
	GoTotalAllocBytes uint64 `yaml:"go_total_alloc_bytes"` // cumulative, includes freed objects

	// GC activity during the measurement window (delta between start and end).
	GCCycles       uint32 `yaml:"gc_cycles"`         // number of completed GC cycles
	GCPauseTotalNs uint64 `yaml:"gc_pause_total_ns"` // cumulative GC STW pause time (ns)

	// OS-level peak resident-set-size (from /proc/self/status VmHWM).
	PeakRSSKiB int64 `yaml:"peak_rss_kib"`

	// Process RSS at start and end of the measurement window (VmRSS from
	// /proc/self/status). Both values together show whether this process
	// accumulated memory during the run.
	StartRSSKiB int64 `yaml:"start_rss_kib,omitempty"`
	EndRSSKiB   int64 `yaml:"end_rss_kib,omitempty"`

	// System-wide memory counters at start and end of the measurement window
	// (from /proc/meminfo, all in KiB).
	//
	// Cached    = Linux page cache (file-backed pages). Growth here means data
	//             was read from disk and cached — impossible for pure socket I/O.
	// AnonPages = anonymous mapped pages (Go heap, stacks, socket recv buffers).
	//             Growth here can indicate memory accumulation or larger socket buffers.
	StartCachedKiB    int64 `yaml:"start_cached_kib,omitempty"`
	EndCachedKiB      int64 `yaml:"end_cached_kib,omitempty"`
	StartAnonPagesKiB int64 `yaml:"start_anon_pages_kib,omitempty"`
	EndAnonPagesKiB   int64 `yaml:"end_anon_pages_kib,omitempty"`

	// Disk page-in/out deltas during the measurement window (from /proc/vmstat,
	// pages). A non-zero PgpginDelta is the definitive indicator that the kernel
	// read data from disk — if zero, page cache was not the data source.
	PgpginDelta  uint64 `yaml:"pgpgin_delta,omitempty"`
	PgpgoutDelta uint64 `yaml:"pgpgout_delta,omitempty"`

	// Per-process absolute CPU time during the measurement window (from /proc/self/stat).
	ProcessUserCPUMs int64 `yaml:"process_user_cpu_ms"`
	ProcessSysCPUMs  int64 `yaml:"process_sys_cpu_ms"`

	// Per-process CPU as a percentage of TOTAL system CPU capacity.
	// Computed as (process_ticks / system_total_ticks) × 100.
	// Normalised across all cores so values are bounded by 100 % regardless
	// of core count.  Both values together show this process's share of the system.
	ProcessUserCPUPct float64 `yaml:"process_user_cpu_pct"`
	ProcessSysCPUPct  float64 `yaml:"process_sys_cpu_pct"`

	// System-wide CPU utilisation breakdown during the measurement window (from
	// /proc/stat).  All values are as a percentage of total capacity across all
	// cores (sum across user+sys+iowait+idle+... = 100 %).
	SystemUserPct    float64 `yaml:"system_user_pct,omitempty"`    // user + nice
	SystemSysPct     float64 `yaml:"system_sys_pct,omitempty"`     // kernel
	SystemIOWaitPct  float64 `yaml:"system_iowait_pct,omitempty"`  // I/O wait
	SystemCPUPercent float64 `yaml:"system_cpu_percent,omitempty"` // total active (100 − idle)
}

// PipelineStats captures write-pool producer/consumer pipeline performance.
// Only populated when the DataPool is active (write tracks whose object size
// fits within the pool slot — ≤ 512 MiB by default).
//
// The pool runs one background producer goroutine that pre-fills buffers with
// Xoshiro256++ random data.  Consumer goroutines (the writers) acquire a
// filled slot, stream it to GCS, then release the slot back to the pool.
//
//	HeadroomRatio >> 1.0  →  producer comfortably ahead; data generation is
//	                          NOT the throughput bottleneck.
//	HeadroomRatio ≈ 1.0   →  producer barely keeping up; generation may be
//	                          limiting write throughput.
//
// Stall semantics:
//
//	ProducerStall — the single producer goroutine yielded waiting for an empty
//	                slot (consumers hadn't released one yet).  Expressed as
//	                wall-clock seconds and as a percentage of elapsed time.
//	ConsumerStall — cumulative goroutine·seconds across all writer goroutines
//	                spent waiting for a filled slot.  Divide by goroutine count
//	                and elapsed time to get an average per-writer stall fraction.
type PipelineStats struct {
	// ProducerRateGiBps is the average data-generation rate across all fill
	// goroutines combined (GiB/s).
	ProducerRateGiBps float64 `yaml:"producer_rate_gib_ps"`

	// ConsumerRateGiBps is the average GCS write throughput (GiB/s) — bytes
	// acknowledged by the service, not merely handed to the network stack.
	ConsumerRateGiBps float64 `yaml:"consumer_rate_gib_ps"`

	// HeadroomRatio is the slot-based headroom: producerSlotsPerSec /
	// consumerSlotsPerSec = numProducers × elapsedNs / totalFillNs.
	// A value of 2.0 means the producers can fill slots twice as fast as
	// consumers demand them.  Values below 1.0 indicate certain stalls.
	HeadroomRatio float64 `yaml:"headroom_ratio"`

	// NumProducers is the number of active fill goroutines at the end of the
	// measurement phase.
	NumProducers int `yaml:"num_producers"`

	// ProducerStallSec is the total wall-clock time producer goroutines spent
	// waiting for an empty pool slot (pool full — all slots in-flight or ready).
	ProducerStallSec float64 `yaml:"producer_stall_sec"`

	// ProducerStallPct is ProducerStallSec expressed as a percentage of the
	// total elapsed run time.
	ProducerStallPct float64 `yaml:"producer_stall_pct"`

	// ConsumerStallSec is the cumulative goroutine·seconds all writer goroutines
	// spent blocking on a filled pool slot (producer not yet ready).
	// This is the key metric: any value above zero means consumers were starved.
	ConsumerStallSec float64 `yaml:"consumer_stall_sec"`
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

	// Runtime holds process-level memory and CPU statistics captured during
	// the measurement phase.
	Runtime RuntimeStats `yaml:"runtime,omitempty"`

	// Pipeline holds write-pool producer/consumer pipeline statistics.
	// Nil when no write pool was active (read-only workloads, or objects too
	// large for the pool).
	Pipeline *PipelineStats `yaml:"pipeline,omitempty"`
}
