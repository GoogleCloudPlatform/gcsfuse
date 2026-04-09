# gcs-bench examples

This directory contains ready-to-use benchmark configurations and automation
scripts.  All configs use the same YAML format described in the
[full user guide](../docs/bench-user-guide.md).

---

## Directory layout

```
examples/
  benchmark-configs/       YAML config files (one workload per file)
  scripts/                 Shell helpers for sweeps and distributed runs
```

---

## Quick start

```bash
# 1. Build the binary (one-time)
cd ..
make bench

# 2. Set up credentials (Application Default Credentials)
gcloud auth application-default login

# 3. Dry-run any config — validates without touching GCS
./gcs-bench bench --config examples/benchmark-configs/unet3d-like.yaml --dry-run

# 4. Run — replace bucket name in the YAML, or pass it on the CLI
./gcs-bench bench --config examples/benchmark-configs/unet3d-like.yaml \
    --bucket my-gcs-bucket
```

> **Tip:** The `--bucket` flag overrides the `bucket:` field in any YAML
> config file. You never need to edit a config just to point at a different
> bucket or compare two buckets side-by-side.

---

## Benchmark configs

### UNet3D-like (MLPerf Storage reference)

Simulates the [MLPerf Storage UNet3D](https://mlcommons.org/working-groups/benchmarks/storage/)
training workload: 50,176 medical-imaging files, lognormal sizes (mean ≈ 6.9 MiB),
random access, 32 goroutines.

| Config | Transport | When to use |
|---|---|---|
| `unet3d-like.yaml` | HTTP/2 (default) or RAPID auto-detect | Baseline; non-RAPID or RAPID buckets |
| `unet3d-like-mrd.yaml` | `read-type: multirange` (bidi-gRPC) | RAPID/zonal buckets — best for repeated reads of the same objects |
| `unet3d-like-mrd-ranged.yaml` | MRD, 8 KiB range reads | Reproduce the D. Jones RAPID 8 KB MRD benchmark |
| `unet3d-like-prepare.yaml` | writes | One-time object population for the above three configs |

```bash
# Step 1 — populate objects (skip if already present)
./gcs-bench bench --config examples/benchmark-configs/unet3d-like-prepare.yaml \
    --bucket my-gcs-bucket

# Step 2 — run baseline benchmark
./gcs-bench bench --config examples/benchmark-configs/unet3d-like.yaml \
    --bucket my-gcs-bucket

# Step 3 — switch to MRD on a RAPID bucket (no re-prepare needed)
./gcs-bench bench --config examples/benchmark-configs/unet3d-like-mrd.yaml \
    --bucket my-rapid-bucket --rapid-mode on
```

---

### RAPID 8 KiB MRD — Google Engineering: D. Jones referencħ

Reproduces the reference MRD 8 KB benchmark (96 goroutines, `read-size: 8192`,
`read-type: multirange`).  Models the expected result on a Tier 1 VM
(n2-standard-96, 400 Gbps vNIC) in the same zone as the RAPID bucket:

```
Throughput:  ~54 MiB/s    Ops/sec:  ~6,900
P50: 1.8 ms  P90: 3.9 ms  P99: 15 ms
```

```bash
# Objects must already exist (use unet3d-like-prepare.yaml above).

# Dry-run validation
./gcs-bench bench --config examples/benchmark-configs/rapid-mrd-8k-example.yaml \
    --bucket my-rapid-bucket --rapid-mode on --dry-run

# Live run
./gcs-bench bench --config examples/benchmark-configs/rapid-mrd-8k-example.yaml \
    --bucket my-rapid-bucket --rapid-mode on
```

---

### ResNet50-like image classification

Simulates ResNet50 training I/O: 614,400 JPEG-sized objects (lognormal, mean
≈ 224 KiB), full-object reads, 64 goroutines.

```bash
# Step 1 — populate objects (~134 GiB per host; one-time)
./gcs-bench bench --config examples/benchmark-configs/resnet50-prepare.yaml \
    --bucket my-gcs-bucket

# Step 2 — benchmark
./gcs-bench bench --config examples/benchmark-configs/resnet50.yaml \
    --bucket my-gcs-bucket

# Step 3 — compare RAPID vs standard bucket (same YAML, different bucket)
./gcs-bench bench --config examples/benchmark-configs/resnet50.yaml \
    --bucket my-rapid-bucket --rapid-mode on

./gcs-bench bench --config examples/benchmark-configs/resnet50.yaml \
    --bucket my-standard-bucket --rapid-mode off
```

---

### Small random reads (checkpoint shards)

Simulates fast model loading: 1,000 fixed 4 MiB checkpoint shards, random
access, 32 goroutines.

```bash
./gcs-bench bench --config examples/benchmark-configs/small-random-reads.yaml \
    --bucket my-gcs-bucket
```

---

### Streaming reads (large training shards)

Mixed-track config: 50 × 512 MiB sequential shards blended with 500 × 1 MiB
random index reads.  Designed for RAPID/zonal buckets (`rapid-mode: on`).

```bash
./gcs-bench bench --config examples/benchmark-configs/streaming-reads.yaml \
    --bucket my-rapid-bucket --rapid-mode on
```

---

### Mixed read/write (checkpoint save + load)

Simulate concurrent checkpoint writes (new shards at 4 writers) while
simultaneously reading previous checkpoint shards for fault recovery (12
readers).

```bash
./gcs-bench bench --config examples/benchmark-configs/mixed-read-write.yaml \
    --bucket my-gcs-bucket
```

---

## Scripts

### `thread-curve.sh` — concurrency sweep

Runs `gcs-bench bench` at a sweep of concurrency levels and prints a
consolidated summary table.  Useful for finding the optimal thread count for
a given workload.

```bash
# Flags
#   --config  <file>       (required) YAML config
#   --bucket  <name>       Override bucket (no YAML edit needed)
#   --rapid-mode <m>       Override rapid-mode: auto | on | off
#   --sweep   <list>       Comma-separated concurrency levels
#   --duration <dur>       Measurement duration per level  (Go duration string)
#   --warmup   <dur>       Warmup duration per level
#   --output   <dir>       Root output directory
#   --dry-run              Print commands without running

# Basic sweep (uses defaults from the config)
./examples/scripts/thread-curve.sh \
    --config examples/benchmark-configs/unet3d-like.yaml \
    --bucket my-gcs-bucket

# RAPID bucket sweep for MRD — compare throughput from 8 to 256 goroutines
./examples/scripts/thread-curve.sh \
    --config examples/benchmark-configs/rapid-mrd-8k-example.yaml \
    --bucket my-rapid-bucket --rapid-mode on \
    --sweep 8,16,32,48,64,96,128,192,256 \
    --duration 3m --warmup 30s

# ResNet50 — known-latency-bound sweep (Little's Law: TTFB ≈ 20 ms)
./examples/scripts/thread-curve.sh \
    --config examples/benchmark-configs/resnet50.yaml \
    --bucket my-gcs-bucket \
    --sweep 128,256,384,512,768,1024,1536,2048

# Compare RAPID vs standard bucket without editing the YAML
./examples/scripts/thread-curve.sh \
    --config examples/benchmark-configs/resnet50.yaml \
    --bucket my-rapid-bucket --rapid-mode on \
    --sweep 64,128,256,512

./examples/scripts/thread-curve.sh \
    --config examples/benchmark-configs/resnet50.yaml \
    --bucket my-standard-bucket --rapid-mode off \
    --sweep 64,128,256,512
```

The script writes per-level result directories under the output root and
produces a consolidated `thread-curve.tsv` and a human-readable table:

```
Threads   Throughput (MiB/s)   Ops/sec   P50 (ms)   P99 (ms)
-------   ------------------   -------   --------   --------
     64              1,240       5,520        5.8       42.1
    128              2,380       10,600       6.1       48.3
    256              4,120       18,300       7.0       62.9
    512              5,890       26,200       9.8      105.4
```

---

### `run-distributed.sh` — multi-host benchmark

Coordinates a distributed run across multiple hosts.  Each host runs the
script with a different `WORKER_ID`.  Worker 0 merges results at the end.

```bash
# 4-host run — run this on each host with the correct WORKER_ID:

WORKER_ID=0 NUM_WORKERS=4 OUTPUT_DIR=/shared/results \
  ./examples/scripts/run-distributed.sh \
    --config examples/benchmark-configs/unet3d-like.yaml \
    --bucket my-gcs-bucket \
    --duration 300s

WORKER_ID=1 NUM_WORKERS=4 OUTPUT_DIR=/shared/results \
  ./examples/scripts/run-distributed.sh \
    --config examples/benchmark-configs/unet3d-like.yaml \
    --bucket my-gcs-bucket \
    --duration 300s

# (repeat for WORKER_ID=2 and 3)
# Worker 0 automatically runs gcs-bench merge-results when all workers finish.
```

> **Note:** `OUTPUT_DIR` must be a shared path visible from all hosts
> (e.g. NFS mount or GCS FUSE mount) so that worker 0 can find all result
> files when merging.

---

### `run-distributed-per-host.sh` — per-host data layout

Variant of `run-distributed.sh` that gives each host its own object namespace
(`resnet50/host-N/`) for linear scaling without cross-host contention.

```bash
WORKER_ID=0 NUM_WORKERS=4 LAYOUT=per-host OUTPUT_DIR=/shared/results \
  ./examples/scripts/run-distributed-per-host.sh \
    --config examples/benchmark-configs/resnet50.yaml \
    --bucket my-gcs-bucket
```

---

## Tips

**Validate before running** — always use `--dry-run` first to confirm the
config is correct without making any GCS API calls:

```bash
./gcs-bench bench --config examples/benchmark-configs/resnet50.yaml \
    --bucket my-bucket --dry-run
```

**Comparing RAPID vs standard** — pass `--bucket` and `--rapid-mode` on the
CLI so you can use the same YAML file for both tests:

```bash
./gcs-bench bench --config myconfig.yaml --bucket rapid-bucket  --rapid-mode on
./gcs-bench bench --config myconfig.yaml --bucket normal-bucket --rapid-mode off
```

**Finding optimal thread count** — use `thread-curve.sh`.  As a rough guide,
target threads ≈ target\_throughput\_MiB\_s / object\_size\_MiB / (1 / TTFB\_s).
For ResNet50 at 20 ms TTFB you need ~450 threads per GiB/s of target
throughput.

**Self-contained results** — every run directory includes `config.yaml` and
`console.log` so runs are fully reproducible without re-running.
