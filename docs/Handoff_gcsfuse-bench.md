# Handoff: Transforming GCSFuse into a GCS Rapid Storage Benchmarking Tool

**Date**: March 28, 2026
**Purpose**: Comprehensive handoff document for the coding agent who will implement all changes.
**Repo**: `gcsfuse-bench` (fork of `github.com/GoogleCloudPlatform/gcsfuse`)

---

## 1. Executive Summary

The goal is to extend this GCSFuse driver fork into a dedicated performance benchmarking tool for Google Cloud Rapid Storage (Zonal/Hyperdisk-backed buckets). The tool must:

1. Accept YAML-defined workload specifications (read/write mix, concurrency, object sizes, access patterns)
2. Inject synthetic I/O directly at the GCS bucket abstraction layer (bypassing FUSE/kernel overhead when desired)
3. Collect high-precision per-request latency data via HDR histograms
4. Export results as structured CSV/JSON with percentile summaries

The codebase already has several primitives that align with this goal — they need extending, not building from scratch.

---

## 2. Codebase Architecture (Verified)

### Module & Language
- **Go** module: `github.com/googlecloudplatform/gcsfuse/v3`
- **Go version**: 1.26.1
- **Build**: `go build .` produces a single `gcsfuse` binary
- **Config system**: Viper + Cobra (YAML config file + CLI flags)
- **Metrics**: OpenTelemetry (auto-generated from `metrics/metrics.yaml`)
- **Config code generation**: `cfg/params.yaml` → templates → `cfg/config.go`

### Verified Data Path (Read)

```
Linux Kernel FUSE → internal/fs/wrappers/{tracing,monitoring,error_mapping}
  → internal/fs/fs.go (fileSystem.ReadFile)
    → internal/fs/handle/file.go (FileHandle.ReadWithReadManager)
      → internal/gcsx/read_manager/read_manager.go (ReadManager.ReadAt)
        → [Reader chain: FileCacheReader → BufferedReader → GCSReader]
          → internal/storage/bucket_handle.go (bucketHandle.NewReaderWithReadHandle)
            → cloud.google.com/go/storage (obj.NewRangeReader — actual network call)
```

### Verified Data Path (Write)

```
Linux Kernel FUSE → internal/fs/wrappers/{tracing,monitoring,error_mapping}
  → internal/fs/fs.go (fileSystem.WriteFile)
    → internal/fs/inode/file.go (FileInode via BufferedWriteHandler)
      → internal/bufferedwrites/buffered_write_handler.go (Write/Flush/Sync)
        → internal/bufferedwrites/upload_handler.go
          → internal/storage/bucket_handle.go (bucketHandle.CreateObject / CreateObjectChunkWriter)
            → cloud.google.com/go/storage (obj.NewWriter — actual network call)
```

---

## 3. Existing Infrastructure to Leverage

### 3.1 DummyIO Bucket (CRITICAL — Already Exists)

**The original analysis document missed this entirely.** A synthetic I/O wrapper already exists:

- **File**: `internal/storage/dummy_io_bucket.go`
- **Config struct**: `cfg.DummyIoConfig` (`cfg/config.go`)
  ```go
  type DummyIoConfig struct {
      Enable        bool          `yaml:"enable"`
      PerMbLatency  time.Duration `yaml:"per-mb-latency"`
      ReaderLatency time.Duration `yaml:"reader-latency"`
  }
  ```
- **CLI flags**: `--enable-dummy-io`, `--dummy-io-per-mb-latency`, `--dummy-io-reader-latency`
- **Wiring**: In `internal/gcsx/bucket_manager.go` lines ~195-202, the real `gcs.Bucket` is wrapped with `storage.NewDummyIOBucket()` if `DummyIOCfg.Enable` is true.
- **Pattern**: Implements `gcs.Bucket` interface; wraps another `gcs.Bucket`. `NewReaderWithReadHandle()` returns a `dummyReader` that generates zeros with configurable latency.

**For benchmarking**: This pattern is the template. The benchmark bucket wrapper should follow the same wrapping pattern but add instrumentation instead of (or in addition to) latency injection.

### 3.2 WorkloadInsight / VisualReadManager (Already Exists)

- **Config struct**: `cfg.WorkloadInsightConfig` (`cfg/config.go`)
  ```go
  type WorkloadInsightConfig struct {
      ForwardMergeThresholdMb int64  `yaml:"forward-merge-threshold-mb"`
      OutputFile              string `yaml:"output-file"`
      Visualize               bool   `yaml:"visualize"`
  }
  ```
- **CLI flags** (all hidden): `--visualize-workload-insight`, `--workload-insight-forward-merge-threshold-mb`, `--workload-insight-output-file`
- **Renderer**: `internal/workloadinsight/io_renderer.go` — generates ASCII I/O pattern visualizations
- **VisualReadManager**: `internal/gcsx/read_manager/visual_read_manager.go` — wraps `ReadManager`, captures all read `(offset, size)` tuples, renders on `Destroy()`

**For benchmarking**: The VisualReadManager wrapper pattern is directly reusable. We need a `BenchmarkReadManager` that captures timing data instead of ASCII art.

### 3.3 Existing Metrics System

- **Definition file**: `metrics/metrics.yaml` — defines all OTel metrics (counters, histograms)
- **Generated code**: `metrics/otel_metrics.go` (auto-generated via `go:generate` in `main.go`)
- **MetricHandle interface**: `metrics/metric_handle.go`
- **Already tracked**: `gcs/read_latency` (histogram, µs), `gcs/read_bytes_count`, `fs/ops_count`, `fs/op_latencies`, `buffered_read/read_latency`
- **FUSE op monitoring**: `internal/fs/wrappers/monitoring.go` — wraps every FUSE op with `time.Since(start)` and sends to MetricHandle

**For benchmarking**: The OTel histograms have limited precision. HDR histograms must be added separately for per-request recording. But the existing metrics can serve as cross-validation.

### 3.4 Profiling Signal Handlers

- **Package**: `internal/perf/`
- **CPU**: `HandleCPUProfileSignals()` — listens SIGUSR1, captures 10s CPU profile to `/tmp/cpu-{timestamp}.pprof`
- **Memory**: `HandleMemoryProfileSignals()` — listens SIGUSR2, captures heap profile to `/tmp/mem-{timestamp}.pprof`
- **Wired in**: `main.go` → `go perf.HandleCPUProfileSignals()` / `go perf.HandleMemoryProfileSignals()`

**For benchmarking**: Add a third signal handler (or use a different trigger) for benchmark result flushing.

---

## 4. Corrections to Original Analysis Document

The prior analysis (`Creating_Test_tool_for_Rapid-Storage.md`) contained several inaccuracies:

| Claim in Original Doc | Actual State |
|----------------------|--------------|
| "Inject into `internal/gcsx/bucket.go`" | **No such file.** The bucket interface is `internal/storage/gcs/bucket.go`. The `internal/gcsx/` directory contains readers, syncers, and the bucket manager — not the bucket itself. |
| "SyncerBucket struct" as I/O target | `SyncerBucket` is a composite of `gcs.Bucket` + `Syncer` (for write-back). It is NOT the right injection point for reads. Read injection goes through the `gcs.Bucket` interface. |
| "`FileInode.ReadAt` and `FileInode.WriteAt`" | **These methods do not exist.** Reads go through `FileHandle.ReadWithReadManager()`. Writes go through `BufferedWriteHandler.Write()`. The FileInode has no direct `ReadAt`/`WriteAt`. |
| "`internal/storage/gcs/gcs_client.go`" | **No such file.** The concrete implementation is `internal/storage/bucket_handle.go` (`bucketHandle` struct). Transport creation is in `internal/storage/storage_handle.go`. |
| "`SourceGenerationIsAuthoritative()` to always return false" | **No such method found.** Content cache bypass is done by leaving `--cache-dir` empty and setting `--file-cache-max-size-mb 0`. |
| "`--stat-cache-capacity 0`" | **DEPRECATED.** Use `--stat-cache-max-size-mb 0` instead. |
| "`--type-cache-ttl 0`" | **DEPRECATED.** Use `--metadata-cache-ttl-secs 0` instead. |
| "`--stackdriver-export-interval=0`" | **DEPRECATED.** Use `--cloud-metrics-export-interval-secs 0` instead. |
| "`serverCfg.LocalFileCache` to false" | Config field is `file-cache.max-size-mb: 0` (YAML) or `--file-cache-max-size-mb 0` (CLI). |
| "Use `gopkg.in/yaml.v3`" | The project uses **Viper** + **Cobra** for config. YAML parsing is handled by Viper, not direct yaml.v3. |
| "Create `internal/gcsx/perf_engine.go`" | While the idea is right, the injection should follow the established `dummyIOBucket` wrapper pattern, not a standalone engine file. |
| No mention of DummyIO Bucket | **Critical miss.** `internal/storage/dummy_io_bucket.go` already provides synthetic I/O injection. |
| No mention of WorkloadInsight | **Critical miss.** `internal/workloadinsight/` and `visual_read_manager.go` already provide I/O pattern capturing. |
| No mention of BufferedWriteHandler | **Critical miss.** Write path goes through `internal/bufferedwrites/`, not inode-level methods. |

---

## 5. Implementation Plan

### Phase 1: Benchmark Configuration Schema

#### 5.1.1 New Config Struct

**File to modify**: `cfg/config.go`

Add a new top-level config section to the `Config` struct:

```go
// Add to Config struct
Benchmark BenchmarkConfig `yaml:"benchmark"`
```

```go
// New struct definitions
type BenchmarkConfig struct {
    Enabled           bool                 `yaml:"enabled"`
    Duration          time.Duration        `yaml:"duration"`
    WarmupDuration    time.Duration        `yaml:"warmup-duration"`
    OutputPath        string               `yaml:"output-path"`
    OutputFormat      string               `yaml:"output-format"` // "csv", "json", "both"
    TotalConcurrency  int                  `yaml:"total-concurrency"`
    ObjectPrefix      string               `yaml:"object-prefix"`
    BucketName        string               `yaml:"bucket-name"`
    Tracks            []BenchmarkTrack     `yaml:"tracks"`
    Histograms        HistogramConfig      `yaml:"histograms"`
}

type BenchmarkTrack struct {
    Name            string  `yaml:"name"`
    Weight          float64 `yaml:"weight"`          // 0.0 - 1.0
    Type            string  `yaml:"type"`            // "read", "write", "mixed", "list", "stat"
    ObjectSizeMin   string  `yaml:"object-size-min"` // "4KB", "1MB", "1GB"
    ObjectSizeMax   string  `yaml:"object-size-max"`
    AccessPattern   string  `yaml:"access-pattern"`  // "random", "sequential", "strided"
    ReadRangeSize   string  `yaml:"read-range-size"` // For partial reads
    Concurrency     int     `yaml:"concurrency"`     // Per-track override (0 = use global)
}

type HistogramConfig struct {
    MinValueMicros    int64 `yaml:"min-value-micros"`     // Default: 1
    MaxValueMicros    int64 `yaml:"max-value-micros"`     // Default: 3600000000 (1 hour)
    SignificantDigits int   `yaml:"significant-digits"`   // Default: 3
    CorrectForCoordinatedOmission bool `yaml:"correct-coordinated-omission"`
}
```

#### 5.1.2 New CLI Flags

**File to modify**: `cfg/config.go` (in `BuildFlagSet()`)

Add flags:
- `--benchmark-enabled` (bool, default false)
- `--benchmark-duration` (duration, default "300s")
- `--benchmark-warmup` (duration, default "30s")
- `--benchmark-output` (string, default "benchmark_results")
- `--benchmark-output-format` (string, default "csv")
- `--benchmark-concurrency` (int, default 64)
- `--benchmark-config` (string) — path to separate benchmark YAML file for complex track definitions

**Alternative approach**: If the code-gen pipeline (`cfg/params.yaml` → templates → `cfg/config.go`) is required, add entries to `cfg/params.yaml` instead of hand-editing the generated file. Check the `go:generate` directives in `main.go`.

#### 5.1.3 Example Benchmark YAML

```yaml
benchmark:
  enabled: true
  duration: "300s"
  warmup-duration: "30s"
  output-path: "/tmp/gcsfuse_bench_results"
  output-format: "csv"
  total-concurrency: 128
  object-prefix: "bench_"
  
  histograms:
    min-value-micros: 1
    max-value-micros: 3600000000
    significant-digits: 3
    correct-coordinated-omission: true
  
  tracks:
    - name: "random-small-reads"
      weight: 0.5
      type: "read"
      object-size-min: "4KB"
      object-size-max: "64KB"
      access-pattern: "random"
    
    - name: "sequential-large-reads"
      weight: 0.3
      type: "read"
      object-size-min: "100MB"
      object-size-max: "1GB"
      access-pattern: "sequential"
    
    - name: "mixed-writes"
      weight: 0.2
      type: "write"
      object-size-min: "1MB"
      object-size-max: "100MB"
      access-pattern: "sequential"
```

---

### Phase 2: Benchmark Instrumentation Bucket (Wrapper)

#### 5.2.1 Instrumented Bucket Wrapper

**New file**: `internal/storage/instrumented_bucket.go`

Follow the same pattern as `internal/storage/dummy_io_bucket.go`:

```go
// Wraps gcs.Bucket to capture per-request latency
type instrumentedBucket struct {
    wrapped          gcs.Bucket
    readHistogram    *hdrhistogram.Histogram
    writeHistogram   *hdrhistogram.Histogram
    listHistogram    *hdrhistogram.Histogram
    statHistogram    *hdrhistogram.Histogram
    deleteHistogram  *hdrhistogram.Histogram
    mu               sync.Mutex
    events           []PerfEvent  // Raw event log
    eventsEnabled    bool
}

type PerfEvent struct {
    Timestamp        time.Time
    OpType           string  // "read", "write", "list", "stat", "delete"
    TrackName        string
    LatencyMicros    int64
    TTFBMicros       int64   // Time to first byte (reads only)
    PayloadBytes     int64
    IsError          bool
    ErrorMsg         string
}
```

**Key interface methods to implement** (all from `gcs.Bucket` in `internal/storage/gcs/bucket.go`):

| Method | Wrap With |
|--------|-----------|
| `NewReaderWithReadHandle(ctx, *ReadObjectRequest) (StorageReader, error)` | Timer around call + TTFB measurement on first `Read()` |
| `CreateObject(ctx, *CreateObjectRequest) (*Object, error)` | Timer around full write |
| `CreateObjectChunkWriter(ctx, *CreateObjectRequest, int, func(int64)) (Writer, error)` | Timer per chunk callback |
| `StatObject(ctx, *StatObjectRequest) (*MinObject, *ExtendedObjectAttributes, error)` | Timer |
| `ListObjects(ctx, *ListObjectsRequest) (*Listing, error)` | Timer |
| `DeleteObject(ctx, *DeleteObjectRequest) error` | Timer |
| `ComposeObjects(ctx, *ComposeObjectsRequest) (*Object, error)` | Timer |
| `Name() string` | Delegate |
| `BucketType() BucketType` | Delegate |

**TTFB Measurement** — For reads, wrap the returned `StorageReader` with a timer that records when the first `Read()` call returns data:

```go
type ttfbReader struct {
    wrapped     gcs.StorageReader
    startTime   time.Time
    firstRead   bool
    recorder    func(ttfb time.Duration)
}

func (r *ttfbReader) Read(p []byte) (int, error) {
    n, err := r.wrapped.Read(p)
    if !r.firstRead && n > 0 {
        r.firstRead = true
        r.recorder(time.Since(r.startTime))
    }
    return n, err
}
```

#### 5.2.2 Wiring the Instrumented Bucket

**File to modify**: `internal/gcsx/bucket_manager.go`

In `SetUpBucket()`, after the existing DummyIO wrapping (lines ~195-202), add:

```go
// After DummyIO wrapping
if bm.config.BenchmarkCfg.Enabled {
    b = storage.NewInstrumentedBucket(b, storage.InstrumentedBucketParams{
        HistogramConfig: bm.config.BenchmarkCfg.Histograms,
        EventsEnabled:   true,
    })
}
```

**Also modify**: `gcsx.BucketConfig` struct to include `BenchmarkCfg` field, and flow it from `cmd/mount.go`.

---

### Phase 3: Synthetic Workload Engine

#### 5.3.1 New Package

**New file**: `internal/benchmark/engine.go`

The engine bypasses the FUSE layer entirely, calling the `gcs.Bucket` interface directly:

```go
type Engine struct {
    bucket       gcs.Bucket
    config       BenchmarkConfig
    tracks       []*trackRunner
    wg           sync.WaitGroup
    results      chan PerfEvent
    startTime    time.Time
}

type trackRunner struct {
    track        BenchmarkTrack
    bucket       gcs.Bucket
    cdf          float64  // Cumulative weight for weighted selection
}
```

**Key methods**:

| Method | Purpose |
|--------|---------|
| `NewEngine(bucket gcs.Bucket, config BenchmarkConfig) *Engine` | Create engine, validate weights sum to 1.0 |
| `(e *Engine) Run(ctx context.Context) error` | Main loop: spawn goroutines, run for duration, collect results |
| `(e *Engine) PrepareTestData(ctx context.Context) error` | Pre-create objects needed by read tracks |
| `(e *Engine) PickTrack() *trackRunner` | Weighted random selection via CDF |
| `(e *Engine) executeRead(ctx, track, object) PerfEvent` | Single read operation with timing |
| `(e *Engine) executeWrite(ctx, track) PerfEvent` | Single write operation with timing |
| `(e *Engine) executeList(ctx, track) PerfEvent` | Single list operation with timing |
| `(e *Engine) executeStat(ctx, track) PerfEvent` | Single stat operation with timing |
| `(e *Engine) Cleanup(ctx context.Context) error` | Remove test objects |

**Object size generation**: Use `crypto/rand` for content (prevents backend compression optimization). Pre-allocate a single large junk buffer and `Read()` from it to avoid GC pressure.

#### 5.3.2 Coordinated Omission Correction

The engine must NOT wait for one request to finish before scheduling the next for a given goroutine. Instead:

```go
// Correct: Schedule at fixed intervals
ticker := time.NewTicker(expectedInterval)
for range ticker.C {
    go func() {
        intendedStart := time.Now() // Record when the request SHOULD have started
        result := e.executeRead(ctx, track, object)
        result.CorrectedLatency = time.Since(intendedStart)
        e.results <- result
    }()
}
```

Use `hdrhistogram.RecordCorrectedValue(value, expectedInterval)` for each recorded latency.

---

### Phase 4: HDR Histogram Integration

#### 5.4.1 New Package

**New file**: `internal/benchmark/histograms.go`

**Dependency to add**: `github.com/HdrHistogram/hdrhistogram-go` to `go.mod`

```go
type HistogramStore struct {
    histograms map[string]*hdrhistogram.Histogram  // keyed by "track_name:op_type"
    mu         sync.Mutex
    config     HistogramConfig
}

func NewHistogramStore(config HistogramConfig) *HistogramStore
func (hs *HistogramStore) Record(key string, valueMicros int64)
func (hs *HistogramStore) RecordCorrected(key string, valueMicros int64, expectedIntervalMicros int64)
func (hs *HistogramStore) Snapshot() map[string]HistogramSnapshot
func (hs *HistogramStore) Merge(other *HistogramStore)
```

```go
type HistogramSnapshot struct {
    TrackName  string
    OpType     string
    Count      int64
    Min        int64
    Max        int64
    Mean       float64
    StdDev     float64
    P50        int64
    P75        int64
    P90        int64
    P95        int64
    P99        int64
    P999       int64
    P9999      int64
    TotalBytes int64  // For throughput calculation
}
```

---

### Phase 5: Result Export

#### 5.5.1 New File

**New file**: `internal/benchmark/exporter.go`

**CSV output** (per-request raw events):
```
timestamp,track,op_type,latency_us,ttfb_us,payload_bytes,is_error,error_msg
2026-03-28T10:00:00.123Z,random-small-reads,read,850,320,65536,false,
2026-03-28T10:00:00.124Z,mixed-writes,write,2100,0,1048576,false,
```

**JSON summary output**:
```json
{
  "test_config": { "duration": "300s", "concurrency": 128 },
  "results": {
    "random-small-reads": {
      "read": { "count": 1000000, "p50_us": 850, "p99_us": 4500, ... }
    }
  },
  "throughput": {
    "total_read_bytes": 68719476736,
    "total_write_bytes": 10737418240,
    "effective_read_gbps": 1.83,
    "effective_write_gbps": 0.29
  }
}
```

#### 5.5.2 Signal-Based Flushing

**File to modify**: `main.go`

Add alongside existing signal handlers:
```go
go benchmark.HandleFlushSignal()  // SIGQUIT triggers immediate result flush
```

Or better: integrate with the benchmark engine's lifecycle (flush on duration complete, flush on graceful shutdown).

---

### Phase 6: Entry Point Integration

#### 5.6.1 Two Operating Modes

**Option A: Benchmark via FUSE mount** (measures full-stack including kernel FUSE)
- Mount normally, inject workloads through the mounted filesystem
- The instrumented bucket wrapper captures GCS-level timings
- External tools (fio, dd) drive the workload
- Use `--benchmark-enabled` to activate the instrumented bucket wrapper only

**Option B: Direct benchmark mode** (bypasses FUSE entirely)
- New subcommand: `gcsfuse benchmark --config bench.yaml my-bucket`
- Creates `StorageHandle` and `gcs.Bucket` directly (no FUSE mount)
- Runs the synthetic workload engine
- This is the **recommended mode** for measuring Rapid Storage performance

#### 5.6.2 New Cobra Subcommand

**File to modify**: `cmd/root.go`

Add a new subcommand alongside the existing mount command:

```go
var benchCmd = &cobra.Command{
    Use:   "benchmark [flags] bucket",
    Short: "Run direct GCS performance benchmark (no FUSE mount)",
    RunE:  runBenchmark,
}
```

**New file**: `cmd/benchmark.go`

```go
func runBenchmark(cmd *cobra.Command, args []string) error {
    // 1. Parse config (reuse existing Viper/cfg pipeline)
    // 2. Create StorageHandle (reuse internal/storage/storage_handle.go)
    // 3. Create BucketHandle (reuse internal/storage/bucket_handle.go)
    // 4. Wrap with InstrumentedBucket
    // 5. Create benchmark.Engine
    // 6. Run PrepareTestData → Run → export results → Cleanup
}
```

---

## 6. Key Data Structures Reference

### Existing Structs to Understand (Do Not Modify Unless Necessary)

| Struct | File | Purpose |
|--------|------|---------|
| `cfg.Config` | `cfg/config.go` | Top-level config, add `Benchmark BenchmarkConfig` field |
| `gcs.Bucket` | `internal/storage/gcs/bucket.go` | Core interface — all storage ops. Instrumented wrapper must implement this. |
| `gcs.ReadObjectRequest` | `internal/storage/gcs/request.go` | Read request with Name, Range, Generation, ReadHandle |
| `gcs.CreateObjectRequest` | `internal/storage/gcs/request.go` | Write request with Name, Contents, ContentType, etc. |
| `gcs.Object` / `gcs.MinObject` | `internal/storage/gcs/object.go` | Object metadata types |
| `gcs.StorageReader` | `internal/storage/gcs/bucket.go` | Reader interface returned by read ops |
| `gcs.Writer` | (same) | Writer interface for chunk writes |
| `gcs.ByteRange` | `internal/storage/gcs/request.go` | `{Start, Limit}` for range reads |
| `storage.bucketHandle` | `internal/storage/bucket_handle.go` | Concrete GCS client — network calls happen here |
| `storage.storageClient` | `internal/storage/storage_handle.go` | Holds HTTP + gRPC clients |
| `storage.dummyIOBucket` | `internal/storage/dummy_io_bucket.go` | Template for the instrumented bucket wrapper |
| `gcsx.BucketManager` | `internal/gcsx/bucket_manager.go` | Creates/configures buckets — wiring point |
| `gcsx.BucketConfig` | `internal/gcsx/bucket_manager.go` | Flows config to bucket setup |
| `gcsx.SyncerBucket` | `internal/gcsx/syncer_bucket.go` | Composite: `gcs.Bucket` + `Syncer` |
| `gcsx.ReadManager` | `internal/gcsx/read_manager/` | Orchestrates read strategies |
| `gcsx.RandomReader` | `internal/gcsx/random_reader.go` | Legacy direct reader |
| `handle.FileHandle` | `internal/fs/handle/file.go` | FUSE file handle — drives reads via ReadManager |
| `inode.FileInode` | `internal/fs/inode/file.go` | File inode — owns `SyncerBucket`, `BufferedWriteHandler` |
| `bufferedwrites.BufferedWriteHandler` | `internal/bufferedwrites/buffered_write_handler.go` | Streaming write pipeline |
| `metrics.MetricHandle` | `metrics/metric_handle.go` | OTel metrics recording interface |
| `workloadinsight.Renderer` | `internal/workloadinsight/io_renderer.go` | ASCII I/O pattern visualization |

### New Structs to Create

| Struct | Proposed File | Purpose |
|--------|--------------|---------|
| `BenchmarkConfig` | `cfg/config.go` | YAML config for benchmark mode |
| `BenchmarkTrack` | `cfg/config.go` | Per-track workload definition |
| `HistogramConfig` | `cfg/config.go` | HDR histogram parameters |
| `instrumentedBucket` | `internal/storage/instrumented_bucket.go` | Wrapper implementing `gcs.Bucket` with per-op timing |
| `ttfbReader` | `internal/storage/instrumented_bucket.go` | Wrapper implementing `gcs.StorageReader` for TTFB |
| `PerfEvent` | `internal/benchmark/types.go` | Single request timing record |
| `Engine` | `internal/benchmark/engine.go` | Synthetic workload generator |
| `trackRunner` | `internal/benchmark/engine.go` | Per-track goroutine manager |
| `HistogramStore` | `internal/benchmark/histograms.go` | Thread-safe HDR histogram collection |
| `HistogramSnapshot` | `internal/benchmark/histograms.go` | Percentile summary for export |
| `Exporter` | `internal/benchmark/exporter.go` | CSV/JSON result writer |

---

## 7. Files to Create

| File | Purpose |
|------|---------|
| `internal/storage/instrumented_bucket.go` | `gcs.Bucket` wrapper with HDR histogram recording |
| `internal/storage/instrumented_bucket_test.go` | Tests for the wrapper |
| `internal/benchmark/engine.go` | Synthetic workload engine |
| `internal/benchmark/engine_test.go` | Engine tests |
| `internal/benchmark/histograms.go` | HDR histogram store |
| `internal/benchmark/histograms_test.go` | Histogram tests |
| `internal/benchmark/exporter.go` | CSV/JSON result export |
| `internal/benchmark/exporter_test.go` | Exporter tests |
| `internal/benchmark/types.go` | Shared types (PerfEvent, etc.) |
| `cmd/benchmark.go` | Cobra subcommand for direct benchmark mode |
| `cmd/benchmark_test.go` | Subcommand tests |
| `configs/benchmark_example.yaml` | Example benchmark configuration |

## 8. Files to Modify

| File | Change |
|------|--------|
| `cfg/config.go` | Add `BenchmarkConfig`, `BenchmarkTrack`, `HistogramConfig` structs. Add `Benchmark` field to `Config`. Add CLI flags. Add Viper bindings. |
| `cfg/validate.go` | Add validation for benchmark config (weights sum to 1.0, valid sizes, etc.) |
| `internal/gcsx/bucket_manager.go` | Add `BenchmarkCfg` to `BucketConfig`. Wire `instrumentedBucket` wrapper in `SetUpBucket()`. |
| `cmd/root.go` | Register `benchCmd` subcommand. |
| `cmd/mount.go` | Flow `BenchmarkConfig` into `BucketConfig`. |
| `go.mod` | Add `github.com/HdrHistogram/hdrhistogram-go` dependency. |
| `main.go` | Optionally add benchmark signal handler for result flushing. |

---

## 9. Recommended Mount Flags for Backend-Only Testing

When running benchmarks against Rapid Storage, use these flags to minimize driver noise:

```bash
# Direct benchmark mode (recommended — bypasses FUSE entirely)
./gcsfuse benchmark \
  --client-protocol=grpc \
  --benchmark-config=configs/benchmark_example.yaml \
  my-zonal-bucket

# FUSE-mounted benchmark mode (measures full stack)
./gcsfuse \
  --client-protocol=grpc \
  --stat-cache-max-size-mb=0 \
  --metadata-cache-ttl-secs=0 \
  --file-cache-max-size-mb=0 \
  --benchmark-enabled \
  --benchmark-output=/tmp/bench_results \
  --cloud-metrics-export-interval-secs=0 \
  --write.enable-streaming-writes \
  my-zonal-bucket /mnt/test
```

**Flag reasoning**:
- `--client-protocol=grpc`: Use gRPC transport with DirectPath for lowest latency to GCS
- `--stat-cache-max-size-mb=0`: Disable metadata caching (every stat hits GCS)
- `--metadata-cache-ttl-secs=0`: Zero TTL for all metadata caches
- `--file-cache-max-size-mb=0`: Disable local file content cache
- `--cloud-metrics-export-interval-secs=0`: Disable OTel export (reduces overhead)
- `--write.enable-streaming-writes`: Use streaming uploads (default: true in current code)

---

## 10. Dependency: HDR Histogram Library

**Package**: `github.com/HdrHistogram/hdrhistogram-go`

**Key API**:
```go
hist := hdrhistogram.New(minValue, maxValue, sigFigs)
hist.RecordValue(latencyMicros)                    // Standard recording
hist.RecordCorrectedValue(value, expectedInterval) // Coordinated omission fix
hist.ValueAtQuantile(99.0)                         // P99
hist.Mean()                                        // Mean
hist.Max()                                         // Max
hist.TotalCount()                                  // Operation count
hist.Merge(other)                                  // Combine histograms
```

---

## 11. Testing Strategy

1. **Unit tests**: Each new package (`benchmark/`, `instrumented_bucket`) gets standard Go table-driven tests
2. **Integration test with fake bucket**: Use the existing `internal/storage/fake/` package (contains fake GCS bucket implementations) to test the engine without network calls
3. **Manual validation**: Run against a real GCS bucket and cross-check HDR results against the existing OTel `gcs/read_latency` metric
4. **Build validation**: `go build .` must succeed with zero warnings; `go vet ./...` must pass

---

## 12. Build & Run

```bash
cd gcsfuse-bench

# Build
go build .

# Run direct benchmark
./gcsfuse benchmark --config configs/benchmark_example.yaml my-bucket

# Run tests
go test ./internal/benchmark/... ./internal/storage/...
```

---

## 13. Open Design Questions for Implementor

1. **Config generation pipeline**: Should new `BenchmarkConfig` fields go through the `cfg/params.yaml` → code-gen pipeline, or be hand-coded in `cfg/config.go`? The code-gen approach is more consistent but adds complexity. Hand-coding is faster for a fork. **Recommendation**: Hand-code in the fork since this is not upstream Google code.

2. **Object lifecycle**: Should benchmark test objects persist between runs (to allow read-only benchmarks against existing data), or always be cleaned up? **Recommendation**: Support both via a `--benchmark-cleanup` flag.

3. **Multi-bucket support**: Should the engine support benchmarking multiple buckets simultaneously? The existing codebase supports multi-bucket mounts. **Recommendation**: Start with single-bucket, extend later.

4. **Progress reporting**: Should the engine report real-time throughput during the run? **Recommendation**: Yes, print summary every 10 seconds to stderr.

5. **Warm-up period**: Should warm-up period results be discarded from histograms? **Recommendation**: Yes, use separate histograms for warm-up vs measurement phases.
