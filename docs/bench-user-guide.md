# gcs-bench bench — User Guide

`gcs-bench bench` is a standalone GCS I/O benchmark that measures real network
latency distributions without requiring a FUSE mount. It issues reads, writes,
stats, and list operations directly against GCS via the Go storage client and
records full end-to-end operation latency — as well as time-to-first-byte (TTFB)
for read operations — in [HDR histograms](http://hdrhistogram.org/), never
averaging percentiles. The primary metric for most use cases is the **Total**
latency, which is equivalent to **TTLB (time to last byte)**: the complete
wall-clock time from issuing the request until the last byte is received and the
object is fully available to the caller.

---

## Table of Contents

1. [Quick start](#1-quick-start)
2. [How it works](#2-how-it-works)
3. [CLI flags](#3-cli-flags)
4. [YAML configuration reference](#4-yaml-configuration-reference)
5. [Example configs](#5-example-configs)
6. [Understanding the results](#6-understanding-the-results)
7. [Standalone (single-host) usage](#7-standalone-single-host-usage)
8. [Distributed multi-host usage](#8-distributed-multi-host-usage)
   - [Prepare phase](#81-prepare-phase)
   - [1 controller + 1 worker](#82-1-controller--1-worker)
   - [1 controller + 2 workers](#83-1-controller--2-workers)
   - [1 controller + 4 workers](#84-1-controller--4-workers)
   - [Merging results](#85-merging-results)
   - [Coordinator script](#86-coordinator-script)
9. [Authentication](#9-authentication)
10. [Pre-flight connectivity checks](#10-pre-flight-connectivity-checks)
11. [RAPID Storage mode](#11-rapid-storage-mode)
12. [Verbosity and diagnostic logging](#12-verbosity-and-diagnostic-logging)

---

## 1. Quick start

```bash
# 1. Build the binary (from the gcsfuse-bench repo root)
GOTOOLCHAIN=auto go build -o gcs-bench .

# 2. Validate your config without touching GCS
./gcs-bench bench --config examples/benchmark-configs/unet3d-like.yaml --dry-run

# 3. Run a 5-minute benchmark
./gcs-bench bench --config examples/benchmark-configs/unet3d-like.yaml
```

Output is written to `bench-YYYYMMDD-HHMMSS.yaml` (and optionally `.tsv`) in
the path specified by `output-path` (defaults to `./`).

---

## 2. How it works

```
gcs-bench bench
  │
  ├─ Pre-flight check  (skipped with --dry-run)
  │    ├─ LIST bucket prefix          → verifies connectivity + credentials
  │    └─ [prepare mode only]
  │         PUT / LIST / GET / DELETE a sentinel object
  │         → verifies full read/write/delete permissions before commit
  │
  ├─ [optionally] sleep until --start-at  (synchronized multi-host start)
  │
  ├─ Warm-up phase  (warmup-duration, default 5s)
  │    └─ I/O goroutines run; histogram data is DISCARDED
  │
  └─ Measurement phase  (duration, default 30s)
       └─ I/O goroutines run; every op is recorded in HDR histograms
            ├─ TTFB latency: time from issuing the request until first byte
            └─ Total latency: wall-clock from request start to completion
```

Each **track** runs concurrently with its own goroutine pool, histogram pair,
and summary statistics. Tracks can represent different op types, object sizes,
or access patterns — all measured simultaneously against the same bucket.

In **prepare mode** (`mode: prepare`) the engine writes every object path
exactly once (no time limit, no warmup) then exits. This populates the bucket
before a read benchmark.

---

## 3. CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config <file>` | _(required)_ | Path to YAML config file |
| `--duration <dur>` | `30s` | Length of the measurement phase |
| `--warmup <dur>` | `5s` | Warm-up period (stats discarded) |
| `--concurrency <n>` | `8` | Total I/O goroutines across all tracks |
| `--object-prefix <s>` | `""` | Prefix prepended to all object paths |
| `--output-path <dir>` | cwd | Directory for result files |
| `--output-format <fmt>` | `yaml` | `yaml`, `tsv`, or `both` |
| `--mode <mode>` | `""` | Override mode: `benchmark` or `prepare` |
| `--worker-id <n>` | `0` | 0-based worker index (distributed runs) |
| `--num-workers <n>` | `1` | Total workers (activates sharding + raw histogram export) |
| `--start-at <unix>` | `0` | Unix epoch; sleep until this time before starting |
| `--key-file <path>` | _(ADC)_ | Service account key JSON (optional) |
| `--endpoint <url>` | _(GCS)_ | Custom endpoint (e.g. a proxy for testing) |
| `--dry-run` | `false` | Validate config and print plan; no GCS connection |
| `--rapid-mode <mode>` | `""` | RAPID/zonal bucket handling: `auto`, `on`, or `off` (see [§11](#11-rapid-storage-mode)) |
| `-v` / `--verbose` | _(WARN)_ | Increase log verbosity; repeat for more: `-v`=INFO, `-vv`=DEBUG, `-vvv`=TRACE |

CLI flags override the corresponding fields in the YAML config file.

### Invoking the subcommand

Both spellings are accepted. Global flags such as `-v` may appear **before or
after** the subcommand name — both of the following are equivalent:

```bash
gcs-bench bench      --config ...
gcs-bench benchmark  --config ...
gcs-bench -v bench   --config ...   # flags before the subcommand are fine
gcs-bench bench  -v  --config ...   # flags after the subcommand are fine too
```

---

## 4. YAML configuration reference

All parameters live under a top-level `benchmark:` key.

```yaml
benchmark:
  # ── Identity ──────────────────────────────────────────────────────────────
  bucket: "my-bucket"         # GCS bucket name (required)
  object-prefix: "bench/"     # Prefix added to every object path

  # ── Execution mode ────────────────────────────────────────────────────────
  mode: benchmark             # "benchmark" (default) | "prepare"

  # ── Timing ────────────────────────────────────────────────────────────────
  duration: 5m                # Measurement phase length (Go duration string)
  warmup-duration: 30s        # Warm-up period (stats discarded)

  # ── Concurrency ───────────────────────────────────────────────────────────
  total-concurrency: 32       # Total goroutines; divided by track weights

  # ── Output ────────────────────────────────────────────────────────────────
  output-path: "./results"
  output-format: both         # "yaml" | "tsv" | "both"

  # ── HDR histogram precision ───────────────────────────────────────────────
  histograms:
    min-value-micros: 1         # Smallest latency tracked (µs)
    max-value-micros: 60000000  # Largest latency tracked (60 s)
    significant-digits: 3       # Precision: 3 → ±0.1% error

  # ── RAPID / zonal bucket support (optional) ────────────────────────────────
  rapid-mode: auto      # "auto" (detect) | "on" (force bidi-gRPC) | "off" (HTTP/2 only)

  # ── Distributed run (optional) ────────────────────────────────────────────
  worker-id: 0          # This host's 0-based index
  num-workers: 1        # Total hosts (>1 enables sharding + raw histogram export)
  start-at: 0           # Unix epoch; 0 = start immediately

  # ── Tracks (one or more) ──────────────────────────────────────────────────
  tracks:
    - name: my-track          # Label used in output
      op-type: read           # "read" | "write" | "stat" | "list"
      weight: 1               # Relative goroutine share
      access-pattern: random  # "random" | "sequential"

      # Object selection — choose ONE of:

      # Option A: flat object list
      object-count: 1000             # Number of distinct objects
      object-size-min: 4194304       # Bytes (fixed when min == max)
      object-size-max: 4194304

      # Option B: directory tree
      directory-structure:
        width: 28              # Subdirs per level
        depth: 2               # Tree depth  →  28² = 784 leaf dirs
        files-per-dir: 64      # Objects per leaf  →  50,176 total
        dir-pattern: "dir-%04d"
        file-pattern: "obj-%06d"

      # Read granularity (reads only)
      read-size: 4194304       # Bytes per read call; 0 = read full object

      # Per-track concurrency override (optional)
      concurrency: 12          # Ignores weight when set

      # Size distribution for writes (optional; overrides object-size-min/max)
      size-spec:
        type: lognormal        # "fixed" | "uniform" | "lognormal"
        mean: 7235174          # Real-space mean (bytes)
        std-dev: 5788139       # Real-space std-dev (bytes)
        min: 1048576           # Floor (bytes)
        max: 33554432          # Ceiling (bytes)
```

### Goroutine distribution

When `concurrency` is not set on a track, goroutines are allocated
proportionally by `weight`:

```
goroutines_for_track = total-concurrency × (track.weight / sum_of_all_weights)
```

A track with weight 3 and another with weight 1 split 32 goroutines as 24 / 8.

### Directory-tree object names

With `object-prefix: "bench/"`, `dir-pattern: "dir-%04d"`, and
`file-pattern: "obj-%06d"`, the generated paths look like:

```
bench/dir-0000/dir-0000/obj-000000
bench/dir-0000/dir-0000/obj-000001
...
bench/dir-0027/dir-0027/obj-000063
```

The prepare config and the benchmark config **must use identical
directory-structure parameters** so the read workload knows which names to use
without listing the bucket.

---

## 5. Example configs

All examples live in `examples/benchmark-configs/`.

### 5.1 UNet3D-like random reads

Mirrors the MLPerf Storage UNet3D pattern: 50,176 medical-imaging files in a
28×28 tree, read in random order with lognormal size distribution.

```yaml
# examples/benchmark-configs/unet3d-like.yaml
benchmark:
  bucket: "my-bucket-name"
  object-prefix: "unet3d/"
  duration: 5m
  warmup-duration: 30s
  total-concurrency: 32
  output-path: "./results"
  output-format: both

  histograms:
    min-value-micros: 1
    max-value-micros: 60000000
    significant-digits: 3

  tracks:
    - name: unet3d-read
      op-type: read
      weight: 1
      access-pattern: random
      directory-structure:
        width: 28
        depth: 2
        files-per-dir: 64
        dir-pattern: "dir-%04d"
        file-pattern: "obj-%06d"
```

### 5.2 UNet3D-like prepare (write all objects first)

```yaml
# examples/benchmark-configs/unet3d-like-prepare.yaml
benchmark:
  mode: prepare
  bucket: "my-bucket-name"
  object-prefix: "unet3d/"
  total-concurrency: 64

  tracks:
    - name: unet3d-prepare
      op-type: write
      directory-structure:
        width: 28
        depth: 2
        files-per-dir: 64
        dir-pattern: "dir-%04d"
        file-pattern: "obj-%06d"
      size-spec:
        type: lognormal
        mean: 7235174
        std-dev: 5788139
        min: 1048576
        max: 33554432
```

### 5.3 Small random reads (checkpoint shards)

4 MiB shards read at full-object granularity, simulating model checkpoint load.

```yaml
# examples/benchmark-configs/small-random-reads.yaml
benchmark:
  bucket: "my-bucket-name"
  duration: 60s
  warmup-duration: 10s
  total-concurrency: 32
  object-prefix: "checkpoints/run-001/"
  output-path: "./results"
  output-format: both

  histograms:
    min-value-micros: 1
    max-value-micros: 60000000
    significant-digits: 3

  tracks:
    - name: random-read-4mb
      op-type: read
      weight: 1
      object-size-min: 4194304
      object-size-max: 4194304
      read-size: 4194304
      access-pattern: random
      object-count: 1000
```

### 5.4 Streaming large reads

512 MiB training shards streamed sequentially in 4 MiB chunks, with a
lightweight random-read track for index files.

```yaml
# examples/benchmark-configs/streaming-reads.yaml
benchmark:
  bucket: "my-zonal-bucket-name"
  duration: 120s
  warmup-duration: 15s
  total-concurrency: 64
  object-prefix: "dataset/shards/"
  output-path: "./results"
  output-format: both

  histograms:
    min-value-micros: 1
    max-value-micros: 60000000
    significant-digits: 3

  tracks:
    - name: sequential-read-512mb
      op-type: read
      weight: 3
      object-size-min: 536870912
      object-size-max: 536870912
      read-size: 4194304
      access-pattern: sequential
      object-count: 50

    - name: random-read-1mb
      op-type: read
      weight: 1
      object-size-min: 1048576
      object-size-max: 1048576
      read-size: 1048576
      access-pattern: random
      object-count: 500
```

### 5.5 Mixed read/write (checkpoint save + load)

Concurrent writers saving 100 MiB checkpoint shards and readers loading the
previous checkpoint.

```yaml
# examples/benchmark-configs/mixed-read-write.yaml
benchmark:
  bucket: "my-bucket-name"
  duration: 60s
  warmup-duration: 10s
  total-concurrency: 16
  object-prefix: "ckpt/step-10000/"
  output-path: "./results"
  output-format: yaml

  histograms:
    min-value-micros: 1
    max-value-micros: 60000000
    significant-digits: 3

  tracks:
    - name: write-checkpoint-100mb
      op-type: write
      weight: 1
      object-size-min: 104857600
      object-size-max: 104857600
      access-pattern: sequential
      object-count: 20
      concurrency: 4

    - name: read-checkpoint-100mb
      op-type: read
      weight: 3
      object-size-min: 104857600
      object-size-max: 104857600
      read-size: 4194304
      access-pattern: sequential
      object-count: 20
      concurrency: 12
```

### 5.6 Stat and list operations

Benchmark metadata-only operations (no data transfer).

```yaml
benchmark:
  bucket: "my-bucket-name"
  duration: 60s
  warmup-duration: 10s
  total-concurrency: 16
  object-prefix: "dataset/"
  output-path: "./results"
  output-format: yaml

  histograms:
    min-value-micros: 1
    max-value-micros: 60000000
    significant-digits: 3

  tracks:
    - name: stat-objects
      op-type: stat
      weight: 1
      access-pattern: random
      object-count: 10000

    - name: list-directories
      op-type: list
      weight: 1
      access-pattern: random
      directory-structure:
        width: 10
        depth: 2
        files-per-dir: 100
        dir-pattern: "shard-%03d"
        file-pattern: "file-%05d"
```

---

## 6. Understanding the results

### Console output

When the measurement phase completes you see a table like this:

```
=== Benchmark Results (2026-03-28T10:00:00Z, 300.2s) ===

Track: unet3d-read
  Ops/s:       147.3    Errors: 0 / 44197
  Throughput:  1033.51 MB/s
  TTFB (us)   p50=8241  p90=15832  p95=20114  p99=38290  p999=91443  max=312041  mean=10287.4
  Total (us)  p50=46823 p90=89441  p95=110322 p99=198442 p999=511230 max=891234  mean=55918.2
```

| Field | Meaning |
|-------|---------|
| `Ops/s` | Operations per second (successes + failures) over the measurement window |
| `Throughput MB/s` | Bytes transferred ÷ elapsed seconds |
| `Errors` | Failed operations (not included in latency histograms) |
| `TTFB p50` | Median time-to-first-byte: from issuing the request until the first byte is received |
| `TTFB p99` | 99th-percentile TTFB — the tail that 1% of requests exceeded |
| `TTFB p999` | 99.9th-percentile TTFB — the extreme tail |
| `Total p50` | **Median total operation latency — equivalent to TTLB (time to last byte).** This is the wall-clock time from issuing the request until the object is fully received and `Close()` returns. For most benchmarking purposes this is the metric that matters: you cannot use the data until you have all of it. |
| `Total p99` | 99th-percentile TTLB — the tail that 1% of operations exceeded |
| `Total max` | Single slowest complete operation observed |

All latency values are in **microseconds (µs)**. Divide by 1000 for milliseconds,
by 1,000,000 for seconds.

> **TTFB vs TTLB:** TTFB (time-to-first-byte) measures how quickly the storage
> service starts responding. It is useful for diagnosing server-side latency or
> connection overhead, and is reported here for completeness. However, **Total /
> TTLB is the more operationally meaningful metric** — for small to medium objects
> you cannot act on the data until the full transfer is complete, and for large
> objects being streamed the total transfer time determines throughput. When in
> doubt, focus on the `Total` row.

### YAML output file

```yaml
start_time: 2026-03-28T10:00:00Z
measurement_duration: 5m0.218s
worker-id: 0          # present only in distributed runs; -1 = merged summary
tracks:
  - TrackName: unet3d-read
    worker-id: 0      # present only in distributed runs
    TotalOps: 44197
    Errors: 0
    ThroughputBytesPerSec: 1.08402e+09
    OpsPerSec: 147.3
    TTFB:
      p50_us: 8241
      p90_us: 15832
      p95_us: 20114
      p99_us: 38290
      p999_us: 91443
      max_us: 312041
      mean_us: 10287.4
    TotalLatency:
      p50_us: 46823
      p90_us: 89441
      p95_us: 110322
      p99_us: 198442
      p999_us: 511230
      max_us: 891234
      mean_us: 55918.2
    # In distributed runs (--num-workers > 1) two additional fields appear:
    raw-ttfb-histogram: "eyJMb3dlc3RUcmFja2FibGVWYWx1..."   # base64 HDR snapshot
    raw-total-histogram: "eyJMb3dlc3RUcmFja2FibGVWYWx1..."  # base64 HDR snapshot
```

The `raw-*-histogram` fields contain the full HDR histogram serialized as
base64-encoded JSON. They are used by `gcs-bench merge-results` to compute
statistically correct combined percentiles across workers — no precision is lost
in the merge.

### Prepare mode output file

Prepare runs also produce a result file (YAML or TSV) — written after all
objects are uploaded. The file uses the same schema as benchmark results but
latency histograms are absent (no I/O latency is measured in prepare mode):

```yaml
start_time: 2026-03-29T14:00:00Z
measurement_duration: 1m33.2s    # total elapsed time of the prepare run
tracks:
  - TrackName: unet3d-prepare
    TotalOps: 50175              # objects successfully written
    Errors: 1
    ThroughputBytesPerSec: 3.71e+08   # ~354 MB/s
    OpsPerSec: 539.1
    TTFB: {}                     # empty — latency not measured in prepare mode
    TotalLatency: {}             # empty
```

This file is useful for comparing prepare throughput across runs and for
audit trails (confirming how many objects were written and how many failed).

### TSV output file

One row per track:

```
track   ops_total  errors  ops_per_sec  throughput_mb_s  ttfb_p50_us  ttfb_p90_us  ...
unet3d-read  44197  0  147.3  1033.5  8241.0  15832.0  20114.0  38290.0  91443.0  312041.0  10287.4  46823.0  ...
```

---

## 7. Standalone (single-host) usage

This is the simplest case — one machine, one bucket, no coordination needed.

### Step 1 — Populate the bucket (prepare mode)

If the objects do not yet exist, run the prepare phase first:

```bash
./gcs-bench bench --config examples/benchmark-configs/unet3d-like-prepare.yaml
```

Progress is printed every 5 seconds:

```
[prepare] Track "unet3d-prepare": writing 50176 objects with 64 goroutines...
  [prepare] 3200/50176 (6%)  640 obj/s  0 errors
  [prepare] 6720/50176 (13%)  672 obj/s  0 errors
  ...
[prepare] Track "unet3d-prepare": complete — 50176/50176 written in 1m18s (643 obj/s, 0 errors)
```

After the run finishes a YAML result file is written to `output-path`
(default: `./`), named `bench-YYYYMMDD-HHMMSS.yaml`. It records `TotalOps`,
`Errors`, `OpsPerSec`, and `ThroughputBytesPerSec` for each track — useful
for auditing what was written and comparing prepare throughput across runs.
Latency fields are empty (no latency is measured during prepare).

### Step 2 — Run the benchmark

```bash
./gcs-bench bench --config examples/benchmark-configs/unet3d-like.yaml
```

You can override parameters from the command line:

```bash
# 2-minute run with 64 goroutines, saving results to /tmp/bench-out
./gcs-bench bench \
    --config examples/benchmark-configs/unet3d-like.yaml \
    --duration 2m \
    --concurrency 64 \
    --output-path /tmp/bench-out \
    --output-format both
```

### Step 3 — Validate without connecting (dry-run)

Use `--dry-run` any time to verify what the benchmark will do:

```bash
./gcs-bench bench --config examples/benchmark-configs/unet3d-like.yaml --dry-run
```

```
=== DRY RUN — no GCS connection will be made ===

  Mode:             benchmark
  Bucket:           gs://my-bucket-name
  Object prefix:    unet3d/
  Warmup:           30s (stats discarded)
  Measurement:      5m0s
  Total goroutines: 32
  Output path:      ./results
  Output format:    both
  Histogram range:  1 µs – 60000000 µs (3 significant digits)

  Tracks (1):

  [1] unet3d-read
      op-type:        read
      access-pattern: random
      object-size:    lognormal (mean=7 MiB, σ=6 MiB, [1 MiB – 32 MiB])
      read-size:      full object
      goroutines:     32 (weight 1 / 1)
      directory-tree: width=28, depth=2, files-per-dir=64 → 50176 total objects

Config is valid. Re-run without --dry-run to execute.
```

---

## 8. Distributed multi-host usage

Running `gcs-bench bench` on multiple hosts simultaneously lets you drive more
aggregate I/O than a single machine allows, while still producing a single
merged summary with statistically correct percentiles across all workers.

The design is intentionally simple — workers are **independent processes** that
do not communicate with each other during the benchmark. Coordination is:

1. **Prepare**: each host writes a disjoint shard of objects (no overlap, no
   locking).
2. **Benchmark start**: all hosts sleep until a pre-computed future timestamp
   (`--start-at`), then start their warm-up simultaneously.
3. **Merge**: after all workers finish, `gcs-bench merge-results` combines the
   per-worker YAML files using HDR histogram merging (identical to sai3-bench's
   approach — no accuracy loss).

### Clock synchronization requirement

Workers use wall-clock time for the synchronized start. Host clocks must be
synchronized to within ~1 second (NTP or equivalent). Most cloud VMs satisfy
this automatically.

### Shared storage for result files

After each worker finishes it writes a YAML result file. For `gcs-bench
merge-results` to combine them, either:

- Workers write to a **shared NFS / Cloud Storage FUSE mount** (recommended),
  or
- You **scp / gsutil cp** the result files to one host before merging.

---

### 8.1 Prepare phase

Before the first benchmark run, populate the bucket. Each worker writes only
the objects assigned to it (modulo partition).

**Worker 0 of 4:**
```bash
./gcs-bench bench \
  --config unet3d-like-prepare.yaml \
  --worker-id 0 \
  --num-workers 4
```

**Worker 1 of 4 (on a different host, simultaneously):**
```bash
./gcs-bench bench \
  --config unet3d-like-prepare.yaml \
  --worker-id 1 \
  --num-workers 4
```

...and so on for workers 2 and 3. Each host writes ≈ 12,544 of the 50,176
objects (50,176 / 4). Writes do not overlap.

---

### 8.2 1 controller + 1 worker

Two hosts, one compute a synchronized start time, both run concurrently.

```bash
# Calculate a start time ~60 seconds in the future (run on any host)
START=$(date -d "+60 seconds" +%s)
echo "START_AT=${START}"

# On host 0 (acts as the "controller" — it will also merge results):
./gcs-bench bench \
  --config unet3d-like.yaml \
  --worker-id 0 \
  --num-workers 2 \
  --start-at ${START} \
  --output-path ./results/worker-0

# On host 1 (simultaneously):
./gcs-bench bench \
  --config unet3d-like.yaml \
  --worker-id 1 \
  --num-workers 2 \
  --start-at ${START} \
  --output-path ./results/worker-1
```

When both finish, merge on host 0:

```bash
./gcs-bench merge-results \
  ./results/worker-0/bench-*.yaml \
  ./results/worker-1/bench-*.yaml \
  --output-path ./results/merged \
  --output-format both
```

---

### 8.3 1 controller + 2 workers

```bash
START=$(date -d "+60 seconds" +%s)

# Host 0
./gcs-bench bench --config unet3d-like.yaml \
  --worker-id 0 --num-workers 3 --start-at ${START} \
  --output-path ./results/worker-0

# Host 1
./gcs-bench bench --config unet3d-like.yaml \
  --worker-id 1 --num-workers 3 --start-at ${START} \
  --output-path ./results/worker-1

# Host 2
./gcs-bench bench --config unet3d-like.yaml \
  --worker-id 2 --num-workers 3 --start-at ${START} \
  --output-path ./results/worker-2

# Merge (on host 0, after all workers complete)
./gcs-bench merge-results \
  ./results/worker-{0,1,2}/bench-*.yaml \
  --output-path ./results/merged --output-format both
```

---

### 8.4 1 controller + 4 workers

```bash
START=$(date -d "+90 seconds" +%s)    # more slack time for 5 hosts to launch

for WORKER_ID in 0 1 2 3 4; do
  # On host ${WORKER_ID}:
  ./gcs-bench bench --config unet3d-like.yaml \
    --worker-id ${WORKER_ID} \
    --num-workers 5 \
    --start-at ${START} \
    --output-path ./results/worker-${WORKER_ID} &
done

wait   # if running all on one machine for testing

# Merge
./gcs-bench merge-results \
  ./results/worker-{0,1,2,3,4}/bench-*.yaml \
  --output-path ./results/merged \
  --output-format both
```

> **Note:** In a real multi-host deployment each host runs its own
> `gcs-bench bench` command independently. The `for` loop above is a
> local simulation. See the [coordinator script](#86-coordinator-script) for
> a production-ready approach.

---

### 8.5 Merging results

`gcs-bench merge-results` takes two or more per-worker YAML result files and
produces a single merged summary:

```bash
./gcs-bench merge-results \
  worker-0/bench-20260328-100000.yaml \
  worker-1/bench-20260328-100000.yaml \
  worker-2/bench-20260328-100000.yaml \
  worker-3/bench-20260328-100000.yaml \
  --output-path ./merged \
  --output-format both
```

Console output shows the merged results followed by the per-worker comparison
table:

```
=== Merged Results (4 workers) ===

=== Benchmark Results (2026-03-28T10:00:00Z, 300.2s) ===

Track: unet3d-read
  Ops/s:       589.1    Errors: 0 / 176788
  Throughput:  4134.04 MB/s
  TTFB (us)   p50=8103  p90=15601  p95=19874  p99=37982  p999=90112  max=318294  mean=10102.3
  Total (us)  p50=46201 p90=88234  p95=108944 p99=196003 p999=508812 max=902341  mean=55104.5


=== Per-Worker Summary ===

Track: unet3d-read
  Worker file                      ops/s   throughput_mb  ttfb_p50   ttfb_p99  total_p50  total_p99
  ------------------------------  --------  ----------  ----------  ----------  ----------  ----------
  worker-0/bench-20260328-...yaml    147.3    1033.51    8241.0    38290.0    46823.0   198442.0
  worker-1/bench-20260328-...yaml    148.1    1038.70    8198.0    37841.0    46012.0   194231.0
  worker-2/bench-20260328-...yaml    146.9    1030.82    8312.0    38501.0    47201.0   199910.0
  worker-3/bench-20260328-...yaml    146.8    1030.11    8292.0    38411.0    47043.0   198876.0

Worker start skew: 312ms  (earliest: 2026-03-28T10:00:00Z)
```

> **Why HDR histogram merging matters:**
> Naive averaging of percentiles (e.g. mean of four p99 values) is statistically
> incorrect — in the worst case it can underestimate the true combined p99 by an
> order of magnitude. `gcs-bench merge-results` uses `hist.Merge()` from the HDR
> histogram library (the same method used by sai3-bench) which combines the full
> count distributions before computing percentiles. The result is identical to
> what you would get if all operations from all workers were fed into a single
> histogram.

**Requirements for merging:**
- Result files must have been produced with `--num-workers > 1`. Single-host
  runs do not embed raw histogram data to keep YAML output compact.
- All workers must have used the same track names (same `name:` fields in the
  YAML config) or the merge will warn about missing tracks.

---

### 8.6 Coordinator script

`examples/scripts/run-distributed.sh` automates the full workflow for a
shared-filesystem deployment (e.g. results written to NFS or Cloud Storage
FUSE):

```bash
# On each host, set WORKER_ID and run the same script:

# Host 0:
WORKER_ID=0 NUM_WORKERS=4 OUTPUT_DIR=/mnt/results \
  ./examples/scripts/run-distributed.sh \
  --config unet3d-like.yaml --duration 5m

# Hosts 1–3 (simultaneously):
WORKER_ID=1 NUM_WORKERS=4 OUTPUT_DIR=/mnt/results \
  ./examples/scripts/run-distributed.sh \
  --config unet3d-like.yaml --duration 5m

WORKER_ID=2 NUM_WORKERS=4 OUTPUT_DIR=/mnt/results \
  ./examples/scripts/run-distributed.sh \
  --config unet3d-like.yaml --duration 5m

WORKER_ID=3 NUM_WORKERS=4 OUTPUT_DIR=/mnt/results \
  ./examples/scripts/run-distributed.sh \
  --config unet3d-like.yaml --duration 5m
```

The script automatically:

1. Computes `START_AT = now + START_DELAY_SECS` (default 60 s).
2. Runs `gcs-bench bench` with the correct `--worker-id`, `--num-workers`, and
   `--start-at` flags.
3. On worker 0 only: polls for all worker result files, then calls
   `gcs-bench merge-results` and writes the combined summary to
   `${OUTPUT_DIR}/merged/`.

**Environment variables:**

| Variable | Default | Description |
|----------|---------|-------------|
| `WORKER_ID` | `0` | This host's 0-based worker index |
| `NUM_WORKERS` | `1` | Total number of hosts |
| `OUTPUT_DIR` | `./results` | Parent directory for per-worker and merged output |
| `GCS_BENCH` | `gcs-bench` | Path to the binary |
| `START_DELAY_SECS` | `60` | Seconds to wait before all workers start simultaneously |

---

## 9. Authentication

`gcs-bench bench` uses the same Google Cloud credentials as the Go storage
client:

1. **Application Default Credentials (ADC)** — automatically used when running
   on GCE/GKE, or after `gcloud auth application-default login`. This is the
   recommended default.

2. **Service account key file** — pass `--key-file /path/to/key.json` or set
   `key-file:` in the YAML config. Useful in environments without ADC.

```bash
./gcs-bench bench \
  --config unet3d-like.yaml \
  --key-file /etc/gcs-bench/service-account.json
```

The service account (or the ADC identity) must have at minimum:
- `roles/storage.objectViewer` for read-only benchmarks
- `roles/storage.objectUser` for write / prepare runs
- `roles/storage.legacyBucketReader` for list operations

---

## 10. Pre-flight connectivity checks

Before every benchmark or prepare run (but **not** in `--dry-run` mode),
`gcs-bench bench` performs a quick connectivity and permission check. The
results are always printed to stdout regardless of the `-v` verbosity level,
because "will this work?" is always useful feedback.

### What the check does

**All modes (benchmark and prepare):** issues a `LIST` of the configured object
prefix to verify that:
- Network connectivity to GCS is functional
- Credentials are valid and have at least list permission
- The bucket name and object prefix are correct
- Reports how many objects currently exist at the prefix

**Prepare mode additionally:** writes a small sentinel object
(`<prefix>_gcsbench_preflight_`), lists it to confirm visibility, reads it back
and verifies the round-tripped content, then deletes it. This validates full
read/write permissions before committing to writing potentially thousands of
objects.

### Sample output — benchmark mode

```
=== Pre-flight check ===

  [1/1] LIST gs://my-bucket/unet3d/ ... OK [42ms] — 50176+ object(s) found

  Pre-flight: PASSED — benchmark should work.
```

### Sample output — prepare mode (empty prefix)

```
=== Pre-flight check ===

  [1/5] LIST gs://my-bucket/unet3d/ ... OK [38ms] — prefix is EMPTY (ready for prepare)
  [2/5] PUT  unet3d/_gcsbench_preflight_ ... OK [28ms]
  [3/5] LIST unet3d/_gcsbench_preflight_ ... OK [14ms] — object visible
  [4/5] GET  unet3d/_gcsbench_preflight_ ... OK [19ms] — 36 bytes, content verified
  [5/5] DELETE unet3d/_gcsbench_preflight_ ... OK [11ms]

  Pre-flight: PASSED — prepare should work.
```

### Failure cases

| Condition | Behaviour |
|-----------|----------|
| LIST fails (bad credentials, wrong bucket) | **Fatal** — run aborts with error message |
| PUT fails in prepare mode | **Fatal** — run aborts before writing any real objects |
| Prefix is empty in benchmark mode | **Warning** — run continues (reads will fail at I/O time) |
| DELETE fails (permission denied) | **Warning** — run continues; sentinel name printed for manual cleanup |

The sentinel object does not count toward benchmark histograms — pre-flight
uses the raw bucket handle, not the instrumented wrapper.

---

## 11. RAPID Storage mode

GCS [RAPID storage](https://cloud.google.com/storage/docs/rapid-storage)
(also called "zonal storage") is a high-throughput bucket type designed for
AI/ML workloads. RAPID buckets use a different network protocol: instead of
standard HTTP/2 requests, they require **bidi-streaming gRPC**. Connecting to
a RAPID bucket without enabling this protocol results in 100% I/O failures —
all reads and writes fail silently at the network layer.

### Modes

| Value | Behaviour |
|-------|-----------|
| `auto` | Enable the storage control client; call `GetStorageLayout` to detect whether the bucket is RAPID/zonal. If it is, bidi-gRPC is selected automatically. Safe to use with any bucket — no-op for standard buckets. **Recommended default for RAPID buckets.** |
| `on` | Force bidi-gRPC unconditionally, skipping detection. Use when `auto` fails (e.g., missing `storage.googleapis.com/projects.locations.buckets.get` IAM permission). |
| `off` | Standard HTTP/2 only. No detection. Used for all non-RAPID buckets. This is the default when `rapid-mode` is not specified. |

### CLI

```bash
# Recommended for RAPID buckets: auto-detect
./gcs-bench bench --config rapid-bench.yaml --rapid-mode auto

# Force bidi-gRPC if auto-detection fails
./gcs-bench bench --config rapid-bench.yaml --rapid-mode on

# Explicit default for standard buckets (same as omitting the flag)
./gcs-bench bench --config standard-bench.yaml --rapid-mode off
```

### YAML config

```yaml
benchmark:
  bucket: "my-rapid-bucket"
  rapid-mode: auto    # auto | on | off
  object-prefix: "dataset/"
  ...
```

### IAM requirements for `auto` mode

The `auto` mode calls `GetStorageLayout` via the GCS storage control API. The
calling identity needs:

- `storage.googleapis.com/projects.locations.buckets.get` (included in
  `roles/storage.admin` and `roles/storage.insightsCollectorService`)

If this permission is not available, use `rapid-mode: on` to skip detection
and force bidi-gRPC directly.

### Diagnosing RAPID connection issues

Combine `--rapid-mode` with `-v` to see which protocol was selected:

```
$ ./gcs-bench bench --config rapid-bench.yaml --rapid-mode auto -v
RAPID mode: auto (detecting bucket type via GetStorageLayout)

=== Pre-flight check ===

  [1/1] LIST gs://my-rapid-bucket/dataset/ ... OK [31ms] — 50176+ object(s) found

  Pre-flight: PASSED — benchmark should work.

Waiting 0s for synchronized start...
Measuring for 5m0s...
```

If the LIST step fails in pre-flight, the protocol negotiation failed and
the RAPID bucket rejected the connection. Switch from `auto` to `on`.

---

## 12. Verbosity and diagnostic logging

By default, `gcs-bench bench` produces minimal output — only errors are shown.
Phase transition messages ("Warming up...", "Measuring...") and prepare
progress are suppressed to keep output clean when redirecting to log files.

Use the `-v` flag (repeat for more detail) to increase verbosity:

| Flag | Level | What you see |
|------|-------|--------------|
| _(none)_ | WARN | Only errors; first 3 errors per track are shown inline |
| `-v` | INFO | Phase transitions, RAPID mode selection, prepare progress (`N/M (X%)`) |
| `-vv` | DEBUG | Per-object errors on read/write/stat/list calls |
| `-vvv` | TRACE | Every individual GCS call — op type, object name, elapsed time, bytes |

All output is in plain-text format (never JSON) regardless of verbosity level.

### Usage examples

```bash
# See phase transitions and prepare progress:
./gcs-bench bench --config bench.yaml -v

# See per-object error detail:
./gcs-bench bench --config bench.yaml -vv

# See every GCS call (best with low concurrency or piped to a file):
./gcs-bench bench --config bench.yaml --concurrency 1 -vvv
./gcs-bench bench --config bench.yaml -vvv 2>&1 | tee gcs-trace.log
```

### Sample `-v` output

```
RAPID mode: auto (detecting bucket type via GetStorageLayout)

=== Pre-flight check ===

  [1/1] LIST gs://my-bucket/unet3d/ ... OK [42ms] — 50176+ object(s) found

  Pre-flight: PASSED — benchmark should work.

Warming up for 30s...
Measuring for 5m0s...
```

### Sample `-vvv` TRACE output

```
[gcs] READ start: unet3d/dir-0014/dir-0022/obj-000031
[gcs] READ start: unet3d/dir-0007/dir-0011/obj-000018
[gcs] READ done:  unet3d/dir-0014/dir-0022/obj-000031 size=8388608 elapsed=47ms
[gcs] LIST start: prefix="unet3d/dir-0003/"
[gcs] LIST done:  prefix="unet3d/dir-0003/" count=64 elapsed=32ms
```

> **Warning:** `-vvv` TRACE at high concurrency (32 goroutines × ~150 ops/s)
> produces roughly 4800 lines per second. Use it only for debugging single-
> goroutine runs or when piping output to a file for offline analysis.
