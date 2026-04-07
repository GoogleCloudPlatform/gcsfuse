# gcs-bench Changelog

This file tracks changes to the **gcs-bench benchmark tool** added on top of the
upstream [GoogleCloudPlatform/gcsfuse](https://github.com/GoogleCloudPlatform/gcsfuse)
library. Upstream changes are not recorded here — see the upstream repository's
own history for those.

The version string embedded in the binary is `gcsfuse-v3-snap.<upstream-sha>+bench-<BENCH_VERSION>`.
Use `./gcs-bench --version` to confirm.

---

## v1.2 — MultiRangeDownloader (MRD) read path

Integrates GCS's bidi-gRPC `MultiRangeDownloader` API as a second read strategy,
selectable per track via the new `read-type` config field.

### New features

- **`read-type: multirange`** — New track-level configuration field (default:
  `new-reader`). When set to `multirange`, reads use the GCS
  `NewMultiRangeDownloader` bidi-gRPC API instead of the standard
  `NewReader` path. MRD is only available on RAPID/zonal buckets with
  `rapid-mode: auto` or `rapid-mode: on`.

- **LRU connection cache** — MRD connections are cached per object key in a
  2048-entry LRU (`internal/cache/lru`). Repeated reads against the same objects
  reuse the open bidi-gRPC stream rather than creating a new connection each time.

- **Singleflight deduplication** — Concurrent goroutines racing to obtain an
  MRD connection for the same object key are collapsed to a single
  `NewMultiRangeDownloader` call via `golang.org/x/sync/singleflight`. All
  waiters share the result. This eliminates connection storms on cache misses.

- **Push-based drain via `io.Discard`** — The MRD API pushes data to the caller's
  `io.Writer`. The engine uses `io.Discard` as the drain writer (no allocation,
  no memory copy). Data is "received" for correctness but discarded immediately,
  consistent with the `new-reader` path.

- **Instrumented `MultiRangeDownloader`** — `instrumented_bucket.go` now wraps
  the MRD with the same per-op metrics as the standard reader path: `totalOps`,
  `totalBytes`, HDR histograms (TTFB + total latency), error counting, and
  TRACE-level logging.

- **Shared `TTFBWriter`** — A single `benchmark.TTFBWriter` type (new file
  `internal/benchmark/ttfb_writer.go`) is used by both the `new-reader` and
  `multirange` paths. Fires a TTFB callback once ≥ 256 KiB is received (or on
  `Finalize()` for sub-threshold objects).

- **New example configs** — Two ready-to-use MRD configs added to
  `examples/benchmark-configs/`:
  - `unet3d-like-mrd.yaml` — full-object MRD reads, 32 goroutines
  - `unet3d-like-mrd-ranged.yaml` — 8 KiB range MRD reads, 96 goroutines

### Source changes

| File | Change |
|------|--------|
| `cfg/benchmark_config.go` | Added `ReadType string \`yaml:"read-type"\`` to `BenchmarkTrack` |
| `internal/benchmark/ttfb_writer.go` | **New file** — shared `TTFBWriter` struct |
| `internal/benchmark/engine.go` | Added `mrdCache`, `mrdGroup`, `getOrCreateMRD()`, `doReadMultiRange()`, dispatch in `doRead()` |
| `internal/storage/instrumented_bucket.go` | `NewMultiRangeDownloader` now returns an instrumented wrapper; new `instrumentedMultiRangeDownloader` struct |
| `examples/benchmark-configs/unet3d-like-mrd.yaml` | **New file** — full-object MRD example |
| `examples/benchmark-configs/unet3d-like-mrd-ranged.yaml` | **New file** — range MRD example |

---

## v1.1 — `/proc` memory monitoring

Adds per-tick RSS and page-cache tracking to the live progress output,
making it easy to observe memory growth and kernel page-cache activity
during a benchmark run.

### New features

- **RSS and page-cache metrics** — Each 10-second progress tick now includes
  a `[memory]` line alongside the throughput tick:

  ```
  [bench] track="unet3d-read"  interval-ops=15690  interval-throughput=10.62 GiB/s  total-ops=15690
  [memory] rss=1423 MiB  page-cache=8192 MiB  pgpgin-delta=131072 pages/s
  ```

- **Start/end RSS in result files** — `bench.yaml` and `bench.txt` include
  `start_rss_kib` and `end_rss_kib` fields for the measurement phase.

- **`/proc`-based implementation** — Reads `/proc/self/status` (RSS),
  `/proc/meminfo` (Cached + Buffers), and `/proc/vmstat` (`pgpgin`) directly.
  No external dependencies.

### Source changes

| File | Change |
|------|--------|
| `internal/benchmark/procstats.go` | **New file** — `/proc` reader functions |
| `internal/benchmark/types.go` | Added `StartRSSKiB`, `EndRSSKiB` to result structs |
| `internal/benchmark/engine.go` | Capture RSS at phase start/end; emit `[memory]` ticks |
| `internal/benchmark/exporter.go` | Write RSS fields to YAML and `.txt` output |

---

## v1.0 — Initial gcs-bench tool

The initial standalone GCS I/O benchmark tool, built as an overlay on the
upstream `gcsfuse` v3 storage client library.

### Features at initial release

- Direct GCS reads, writes, stats, and list operations — no FUSE mount required
- HDR histogram latency recording (TTFB + end-to-end / TTLB) — never averaged
- RAPID/zonal bucket support via bidi-gRPC (`--rapid-mode auto|on|off`)
- Warmup phase with continuous goroutines; stats reset at the measurement boundary
- Distributed multi-host mode (`--worker-id` / `--num-workers` / `--start-at`)
- `merge-results` subcommand — statistically correct HDR merge across workers
- `plot-hgrm` subcommand — built-in SVG frequency-distribution renderer
- Write pool (`ChanPool`) — pre-fills random data before measurement; zero consumer stall
- lognormal size distribution for writes (`size-spec: type: lognormal`)
- Directory-tree object naming (`directory-structure` config block)
- Self-contained result directories: `bench.txt`, `bench.yaml`, `bench.tsv`,
  per-track `.hgrm` files, `config.yaml` copy, `console.log` capture
