# gcsfuse-bench

A standalone GCS I/O benchmarking tool built on top of Google's
[Cloud Storage FUSE](https://github.com/GoogleCloudPlatform/gcsfuse) storage
client. It measures real network latency distributions — reads, writes, stats,
and list operations — directly against GCS without requiring a FUSE mount.

> **Upstream base:** This repository is a fork of
> [GoogleCloudPlatform/gcsfuse](https://github.com/GoogleCloudPlatform/gcsfuse)
> (v3, snapshot `582a2201`, 2026-03-27).  
> The benchmark tool lives on the `gcs-bench-tool-v1` branch.  
> See [README-gcsfuse-upstream.md](README-gcsfuse-upstream.md) for the original
> upstream Cloud Storage FUSE documentation.

---

## Quick start

### 1. Build

```bash
# Clone the repo and switch to the benchmark branch
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git gcsfuse-bench
cd gcsfuse-bench
git checkout gcs-bench-tool-v1

# Build the gcs-bench binary (requires Go 1.26+)
make bench

# Verify
./gcs-bench --version
# gcsfuse version gcsfuse-v3-snap.582a2201+bench-v1.0 (Go version go1.26.1)
```

The `make bench` target injects a meaningful version string automatically.
To bump your revision: `make bench BENCH_VERSION=v1.1`.

### 2. Validate your config (no GCS traffic)

```bash
./gcs-bench bench --config docs/examples/unet3d-like.yaml --dry-run
```

### 3. Run a benchmark

```bash
./gcs-bench bench --config docs/examples/unet3d-like.yaml
```

Progress lines are printed every 10 seconds.  When the run finishes, results
are written to a timestamped directory:

```
results/bench-YYYYMMDD-HHMMSS/
  bench.txt                      human-readable summary
  bench.yaml                     machine-readable metrics
  bench.tsv                      TSV for spreadsheet import
  config.yaml                    exact config used for this run
  <track>-ttfb.hgrm              HDR histogram — time to first byte
  <track>-total-latency.hgrm     HDR histogram — end-to-end latency
  console.log                    full terminal output captured verbatim
```

---

## Make targets

| Target | Description |
|---|---|
| `make bench` | Build the `gcs-bench` binary with version info injected |
| `make bench BENCH_VERSION=vX.Y` | Build with a specific revision tag |
| `make bench-test` | Run vet + unit tests for the benchmark packages |
| `make bench-clean` | Remove the `gcs-bench` binary |

---

## Subcommands

| Subcommand | Description |
|---|---|
| `bench` | Run an I/O benchmark (reads, writes, stats, list) |
| `merge-results` | Merge per-worker YAML result files from a distributed run |
| `plot-hgrm` | Render an `.hgrm` HDR histogram file as an SVG chart |

---

## Documentation

| Document | Description |
|---|---|
| [docs/bench-user-guide.md](docs/bench-user-guide.md) | **Full user guide** — config reference, all flags, output format, distributed usage, authentication |
| [docs/Handoff_gcsfuse-bench.md](docs/Handoff_gcsfuse-bench.md) | Development history, architecture decisions, commit log |
| [docs/Creating_Test_tool_for_Rapid-Storage.md](docs/Creating_Test_tool_for_Rapid-Storage.md) | Background on RAPID storage testing motivation |
| [docs/performance.md](docs/performance.md) | Upstream gcsfuse performance notes |
| [docs/installing.md](docs/installing.md) | Upstream gcsfuse installation guide |

---

## Key features

- **No FUSE mount required** — talks directly to GCS via the Go storage client
- **HDR histograms** — accurate p50/p90/p95/p99/p99.9/max; never averaged
- **RAPID / zonal bucket support** — auto-detects bidi-gRPC transport (`--rapid-mode auto`)
- **Warmup phase** — goroutines run continuously; stats reset at the boundary so queues stay full
- **Distributed mode** — co-ordinate multiple hosts with `--worker-id` / `--num-workers`; merge with `merge-results`
- **Self-contained results** — every run directory includes the config and full console log

---

## Upstream Cloud Storage FUSE

This tool is built on the gcsfuse storage client library. For documentation on
the upstream project (FUSE mounting, CSI driver, v2/v3 feature descriptions,
supported platforms, pricing) see
[README-gcsfuse-upstream.md](README-gcsfuse-upstream.md).
