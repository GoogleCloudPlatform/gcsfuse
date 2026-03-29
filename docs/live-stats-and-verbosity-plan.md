# Plan: Live Statistics, Verbosity, RAPID Mode, and Pre-flight Checks

**Scope:** gcs-bench bench subcommand

| Item | Status |
|------|--------|
| Part 1: Live statistics during benchmark runs | **PENDING** — not yet implemented |
| Part 2: Verbosity levels (`-v`/`-vv`/`-vvv`) | **IMPLEMENTED** (2026-03-29) |
| Part 3: RAPID Storage mode (`--rapid-mode`) | **IMPLEMENTED** (2026-03-29) |
| Part 4: Pre-flight connectivity checks | **IMPLEMENTED** (2026-03-29) |

---

## Implementation Summary (2026-03-29)

The following features were implemented and are available in the current binary:

### Part 2 — Verbosity (IMPLEMENTED)

**Files changed:**
- `cmd/benchmark.go` — Added `CountVarP(&verbosity, "verbose", "v", ...)` flag;
  `InitLogFile(cfg.LoggingConfig{Severity: cfg.LogSeverity(...), Format: "text"}, "gcs-bench")`
  called immediately after dry-run exits, mapping `-v`→INFO, `-vv`→DEBUG, `-vvv`→TRACE.
- `internal/benchmark/engine.go` — All `fmt.Printf` calls converted to `logger.Infof`
  (phase transitions, prepare progress) or `logger.Warnf` (errors). Default level is WARN
  so these are silent unless at least `-v` is passed.
- `internal/storage/instrumented_bucket.go` — `logger.Tracef()` calls added in
  `NewReaderWithReadHandle`, `CreateObject`, `StatObject`, and `ListObjects`.
  TRACE lines show op type, object name/prefix, elapsed time, and byte count.
  `logger.Debugf()` calls added for per-object errors on each op.

**Behaviour:**

| Flag | Level | What you see |
|------|-------|--------------|
| _(none)_ | WARN | Only errors; first 3 per track printed inline |
| `-v` | INFO | Phase transitions, RAPID mode selection, prepare progress |
| `-vv` | DEBUG | Per-object errors on read/write/stat/list |
| `-vvv` | TRACE | Every GCS call — op type, name, elapsed, bytes |

Output is always plain text (never JSON).

### Part 3 — RAPID Storage mode (IMPLEMENTED)

**Root cause discovered:**
GCS RAPID (zonal) buckets require bidi-streaming gRPC. Using the default HTTP/2
client results in 100% I/O failures with no error messages at the GCS layer.

**Files changed:**
- `internal/storage/storageutil/client.go` — Added `ForceZonal bool` field to
  `StorageClientConfig`.
- `internal/storage/storage_handle.go` — `lookupBucketType()` short-circuits to
  `&gcs.BucketType{Zonal: true}` when `ForceZonal == true`, skipping the
  `GetStorageLayout` RPC.
- `cfg/benchmark_config.go` — Added `RapidMode string yaml:"rapid-mode"` field.
- `cmd/benchmark.go` — Resolves `rapid-mode` (config + CLI flag), then:
  - `auto`: sets `EnableHNS=true` so `storageControlClient` is created;
    `GetStorageLayout` auto-detects RAPID vs standard.
  - `on`: sets `ForceZonal=true`; skips detection entirely.
  - `off`: plain HTTP/2, no detection (default).
  `--rapid-mode` CLI flag registered; informational `logger.Infof` printed for
  each mode (visible at `-v`).
- `examples/benchmark-configs/unet3d-like.yaml` and
  `examples/benchmark-configs/unet3d-like-prepare.yaml` — `rapid-mode: auto`
  added as a commented example.

### Part 4 — Pre-flight checks (IMPLEMENTED)

**Files added:**
- `internal/benchmark/preflight.go` — New `RunPreflight(ctx, bucket, bucketName, prefix, mode)`
  function. Uses the **raw** (un-instrumented) bucket handle so pre-flight I/O
  does not pollute benchmark histograms.

**Files changed:**
- `cmd/benchmark.go` — `benchmark.RunPreflight(...)` called after
  `BucketHandle()` is obtained but before `NewInstrumentedBucket()` wraps it.
  Skipped automatically in `--dry-run` (dry-run exits before the storage client
  is created).

**Check matrix:**

| Check | All modes | Prepare only | Result on failure |
|-------|-----------|--------------|-------------------|
| LIST prefix | ✓ | ✓ | **Fatal** (bad credentials / wrong bucket) |
| Empty prefix | ✓ (warn) | n/a (expected) | Warning, continues |
| PUT sentinel | | ✓ | **Fatal** (no write permission) |
| LIST sentinel | | ✓ | Warning (eventual consistency) |
| GET sentinel | | ✓ | Warning (round-trip verify) |
| DELETE sentinel | | ✓ | Warning (delete may not be granted) |

---

## Background

Currently the benchmark prints three lines total during a run:

```
Warming up for 30s...
Measuring for 5m0s...
Results written to ./results/bench-20260329-100000.yaml
```

There is complete silence during the warm-up and measurement phases. This makes
it impossible to know whether the benchmark is making progress, hitting errors,
or simply hanging.

The sai3-bench reference implementation (Rust) addresses this with two
complementary systems:

1. **`LiveStatsTracker`** — atomic counters plus mutex-protected HDR histograms
   that every worker goroutine updates in the hot path with near-zero overhead.
   A separate reporting goroutine samples these atomics on a ticker and formats
   terminal output.

2. **`indicatif` progress bars** — a spinner for the workload phase (time-bounded,
   no known total) and a real progress bar for the prepare phase (known total
   object count), updated from the same snapshot.

The equivalent Go design is described below.

---

## Part 1: Live Statistics During Benchmark Runs

### What data is already available

`trackState` in `internal/benchmark/engine.go` already maintains atomic counters
that are updated on every I/O completion:

```go
type trackState struct {
    totalOps   atomic.Int64   // successful + failed ops
    totalErrs  atomic.Int64
    totalBytes atomic.Int64
    hists      *TrackHistograms  // TTFB + total-latency HDR histograms
    ...
}
```

`TrackHistograms` in `internal/benchmark/histograms.go` already has a `Snapshot()`
method that reads current percentiles under a mutex lock. All the data needed
for live reporting already exists — nothing needs to be added to the hot path.

### Proposed approach: ticker goroutine inside `Run()`

The pattern from `runPrepare()` — a dedicated progress goroutine driven by
`time.NewTicker` — can be elevated to the measurement phase as well.

**Measurement phase change in `engine.go`:**

```go
// --- Measurement ---
fmt.Printf("Measuring for %s...  (live stats every %s)\n", e.bCfg.Duration, statsInterval)
start := time.Now()
measCtx, cancel := context.WithTimeout(ctx, e.bCfg.Duration)
defer cancel()

// Start live stats reporter
statsCtx, statsCancel := context.WithCancel(measCtx)
go e.runLiveStats(statsCtx, start, e.bCfg.Duration)

if err := e.runPhase(measCtx); err != nil && err != context.DeadlineExceeded {
    statsCancel()
    return RunSummary{}, fmt.Errorf("measurement: %w", err)
}
statsCancel()
```

**New `runLiveStats()` function in `engine.go`:**

```go
func (e *Engine) runLiveStats(ctx context.Context, start time.Time, total time.Duration) {
    ticker := time.NewTicker(statsInterval) // e.g. 5s; configurable
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            elapsed := time.Since(start)
            remaining := total - elapsed
            if remaining < 0 {
                remaining = 0
            }
            fmt.Printf("\n[%s elapsed, %s remaining]\n",
                elapsed.Round(time.Second), remaining.Round(time.Second))
            for _, ts := range e.trackState {
                ops  := ts.totalOps.Load()
                errs := ts.totalErrs.Load()
                byts := ts.totalBytes.Load()
                var opsRate, mbRate float64
                if elapsed.Seconds() > 0 {
                    opsRate = float64(ops) / elapsed.Seconds()
                    mbRate  = float64(byts) / elapsed.Seconds() / 1e6
                }
                _, total := ts.hists.Snapshot()
                fmt.Printf("  %-24s  %6.1f op/s  %6.1f MB/s  "+
                    "p50=%6.0fµs  p99=%6.0fµs  errs=%d\n",
                    ts.cfg.Name, opsRate, mbRate,
                    total.P50, total.P99, errs)
            }
        case <-ctx.Done():
            return
        }
    }
}
```

**Sample live-stats output during a 5-minute UNet3D run:**

```
Measuring for 5m0s...  (live stats every 5s)

[5s elapsed, 4m55s remaining]
  unet3d-read              142.3 op/s   998.4 MB/s  p50= 45231µs  p99=195442µs  errs=0

[10s elapsed, 4m50s remaining]
  unet3d-read              145.1 op/s  1018.2 MB/s  p50= 44821µs  p99=192314µs  errs=0

[15s elapsed, 4m45s remaining]
  unet3d-read              143.8 op/s  1008.7 MB/s  p50= 46012µs  p99=198210µs  errs=0
...
```

### Stats interval configuration

The interval should be configurable, not hardcoded. Two sensible options:

**Option A — YAML config field:**
```yaml
benchmark:
  stats-interval: 5s    # 0 = disable live stats
```

**Option B — CLI flag:**
```bash
gcs-bench bench --config bench.yaml --stats-interval 10s
```

Option B is simpler and avoids touching the config struct. Both can be supported.
Default recommendation: `5s` for runs ≥ 60s, `2s` for shorter runs.

### Files that need changes for live stats

| File | Change |
|------|--------|
| `internal/benchmark/engine.go` | Add `runLiveStats()` function; call it from `Run()` with a cancel context; add `statsInterval time.Duration` field to `Engine` or pass as parameter |
| `cfg/benchmark_config.go` | Add optional `StatsInterval Duration` field (if config-driven) |
| `cmd/benchmark.go` | Add `--stats-interval` flag; wire into engine |

**Estimated effort:** Small — 50–80 lines of new code, all in `engine.go` and
`cmd/benchmark.go`. No hot-path changes. No new dependencies.

### Comparison with sai3-bench approach

| Aspect | sai3-bench | Proposed gcs-bench |
|--------|-----------|-------------------|
| Hot-path overhead | Atomic counters + try_lock histogram | Same: atomic counters already in trackState; Snapshot() under mutex |
| Reporting goroutine | Separate thread reading LiveStatsTracker | Separate goroutine reading trackState atomics |
| Progress bar library | `indicatif` (Rust) | `github.com/schollz/progressbar/v3` or `github.com/vbauerster/mpb` (Go) — or plain `fmt.Printf` (simplest) |
| Prepare phase | ProgressBar with known total | Already implemented with plain ticker (done) |
| Workload phase | Spinner + throughput line | Ticker + multi-track table (proposed above) |

The plain `fmt.Printf` approach is recommended first — it has zero new
dependencies and is easy to read in log files. A progress-bar library can be
added later if desired.

---

## Part 2: Verbosity Levels (-v / -vv / -vvv)  ✅ IMPLEMENTED

> **Status:** Fully implemented. See implementation summary above for
> exact files changed and behaviour. The section below documents the
> original design, which was followed closely.

### Current logging situation

gcs-bench already has a full structured logging system inherited from gcsfuse:

- **Package:** `internal/logger` — wraps Go's standard `log/slog`
- **Levels defined** (in `internal/logger/slog_helper.go`):
  ```go
  LevelTrace = slog.Level(-8)   // -vvv
  LevelDebug = slog.LevelDebug  // -vv
  LevelInfo  = slog.LevelInfo   // -v
  LevelWarn  = slog.LevelWarn   // (default)
  LevelError = slog.LevelError
  LevelOff   = slog.Level(12)
  ```
- **Functions available:** `logger.Tracef()`, `logger.Debugf()`, `logger.Infof()`,
  `logger.Warnf()`, `logger.Errorf()`
- **Level control:** `logger.setLoggingLevel(string)` (package-private) — called
  by `InitLogFile()` using a `cfg.LoggingConfig.Severity` string value

The level infrastructure is **fully built**. The only missing piece is a
`-v/-vv/-vvv` flag in `cmd/benchmark.go` that maps to the existing level strings.

**Current problem:** `programLevel` is initialized at `init()` time to `INFO`
(the gcsfuse default). The benchmark subcommand never calls `InitLogFile()` or
`setLoggingLevel()`, so it always runs at INFO level — and no code currently
calls `logger.Infof()` or lower in the benchmark path anyway.

There is a secondary problem: the logger's default output format is **JSON**
(the gcsfuse structured-log format), which produces the ugly output you saw:
```
{"timestamp":{"seconds":...},"severity":"INFO","message":"2026/03/28..."}
```
This format is appropriate for production FUSE mount logs, not for an
interactive benchmark tool. The benchmark output should be plain text.

### Proposed implementation

**`cmd/benchmark.go` — add verbosity flag:**

```go
var verbosity int

// In newBenchmarkRootCmd():
rootCmd.Flags().CountVarP(&verbosity, "verbose", "v",
    "Increase log verbosity: -v=INFO, -vv=DEBUG, -vvv=TRACE")
```

`cobra.Command` supports count flags natively — `-v` sets it to 1, `-vv` to 2,
`-vvv` to 3. This is the idiomatic Go-CLI pattern (same as `kubectl -v=5`).

**Level mapping:**

```go
// In the RunE function, before creating the storage client:
logLevel := "warning"   // default: only errors/warnings shown
switch verbosity {
case 1:
    logLevel = "info"
case 2:
    logLevel = "debug"
case 3:
    logLevel = "trace"
}
// Initialize logger in plain-text mode at the requested level
err := logger.InitLogFile(cfg.LoggingConfig{
    Severity: cfg.LogSeverity(logLevel),
    Format:   "text",            // not "json"
    LogRotate: cfg.DefaultLogRotateConfig(),
}, "gcs-bench")
```

**Existing `logger.setLoggingLevel()` is package-private** (lowercase `s`).
`InitLogFile()` calls it internally — so calling `InitLogFile()` with the right
`Severity` is the correct public API.

### What each level would show

| Flag | Level | What gets printed |
|------|-------|-------------------|
| _(none)_ | WARN | Only errors and unexpected conditions |
| `-v` | INFO | Phase transitions ("Warming up…", "Measuring…"), live-stats lines (from Item 1), prepare progress |
| `-vv` | DEBUG | Per-object errors, retry attempts, goroutine startup/shutdown |
| `-vvv` | TRACE | Every individual I/O call with start time and byte count |

> **Note on `-vvv` TRACE:** Per-call tracing at high concurrency will produce
> enormous output (32 goroutines × 150 ops/s = 4800 lines/sec). It should only
> be used for debugging a single-goroutine run or when redirected to a file.

### Files that need changes for verbosity

> **As-implemented note:** The changes below were all completed. The one
> deviation from the plan: `main.go` was not changed — the JSON log format
> issue was fully resolved by the `InitLogFile(Format: "text")` call in
> `cmd/benchmark.go`, which overrides gcsfuse's default JSON format for the
> bench subcommand path without touching `main.go`.

| File | Change | Status |
|------|--------|--------|
| `cmd/benchmark.go` | Add `-v` count flag; call `logger.InitLogFile()` with mapped level and `format: "text"` | ✅ Done |
| `main.go` | Fix JSON log output on startup errors for bench path | N/A — resolved via `InitLogFile` in `cmd/benchmark.go` |
| `internal/benchmark/engine.go` | Replace `fmt.Printf(...)` calls with `logger.Infof(...)` | ✅ Done (all ~12 sites) |
| `internal/storage/instrumented_bucket.go` | Add `logger.Tracef()` calls around each I/O operation | ✅ Done (read, write, stat, list) |

**Estimated effort:** Small-Medium — ~30 lines of new plumbing in
`cmd/benchmark.go`; search-and-replace of `fmt.Printf` → `logger.Infof` in
`engine.go` (~15 call sites); optional TRACE additions in `instrumented_bucket.go`.

---

## Summary: What Changes, and Where

### Item 1 — Live statistics  (PENDING)

```
internal/benchmark/engine.go      ← new runLiveStats() function + wire-up in Run()
cmd/benchmark.go                  ← new --stats-interval flag
cfg/benchmark_config.go           ← optional: new StatsInterval field
```

**Complexity:** Low. Pure addition, no hot-path changes, no new dependencies.

### Item 2 — Verbosity levels  (IMPLEMENTED ✅)

```
cmd/benchmark.go                  ← -v count flag + InitLogFile() call            ✅
internal/benchmark/engine.go      ← fmt.Printf → logger.Infof (all ~12 sites)    ✅
internal/storage/instrumented_bucket.go  ← logger.Tracef() per I/O               ✅
main.go                           ← no change needed (InitLogFile in cmd/ suffices)
```

**Complexity:** Low-Medium. The logger infrastructure is already built. The
work is wiring the flag in and converting `fmt.Printf` → `logger.Infof`.

### Recommended implementation order

1. **Fix JSON log format first** — the ugly JSON output on startup errors is the
   most jarring UX problem. One `InitLogFile()` call in `cmd/benchmark.go` fixes
   this and also sets the stage for verbosity control.

2. **Add live stats** — the biggest practical improvement; fully self-contained.

3. **Add `-v/-vv/-vvv`** — depends on step 1 being done (format fixed), and
   makes step 2 output respect the level setting.

### New external Go dependencies required

**None.** All building blocks are already present:

- Ticker-based reporting: `time.NewTicker` (stdlib)
- Log levels: `internal/logger` package (already in repo)
- Count flag: `cobra.Command.Flags().CountVarP` (cobra, already a dependency)

An optional progress-bar library (`github.com/schollz/progressbar/v3`) could be
added later for a more polished terminal experience matching sai3-bench's
`indicatif` bars, but it is not required for the initial implementation.

---

## Part 3: RAPID Storage Mode  ✅ IMPLEMENTED

> **Status:** Fully implemented. See implementation summary above for exact
> files changed. The feature was discovered as necessary when testing against
> GCS RAPID (zonal) buckets where 100% of I/O failed silently.

See `docs/bench-user-guide.md` §11 for full usage documentation.

---

## Part 4: Pre-flight Connectivity Checks  ✅ IMPLEMENTED

> **Status:** Fully implemented. See implementation summary above for exact
> files changed.

See `docs/bench-user-guide.md` §10 for full usage documentation.
