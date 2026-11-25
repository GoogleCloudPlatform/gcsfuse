# ADM fio job: vLLM v0.16 + RunAI Model Streamer 0.15.6 loading Qwen3-VL-235B through GCSFuse

Reproduces the default (non-distributed) RunAI streamer read pattern with **stock fio** (no fio patch needed),
so GCSFuse can be tested on a CPU VM instead of 8× B200.

## Files

| File | What |
|---|---|
| `qwen3vl_runai_tp8.fio` | fio job for ONE 40 GB batch: 16 slices × `numjobs=8` = 128 concurrent workers |
| `gen_runai_iologs.py` | packs tensors into 40 GB batches (`FilesRequestsIterator` logic) and splits each batch into 16 contiguous 2 MiB-read slice iologs |
| `run_qwen3vl_runai_fio.sh` | runs batches in order against a GCSFuse mount (drops cache once, sets read-ahead, snapshots Prometheus metrics after every batch) |

Includes the safetensors headers in `headers/` (`hdr_00001/2/95/96.json`).

## What it reproduces

- **Syscall and block size:** `ioengine=sync` (`open`, `lseek`, 2 MiB buffered `read()`), `direct=0`, `fadvise_hint=0` — verified via `strace` to match `libstreamer.so` (no `pread`, no `O_DIRECT`).
- **Concurrency:** 8 TP processes × 16 reader threads = **128 concurrent workers**, each opening its own fd per file (1:1 thread:handle).
- **8× app-level reads, shared page cache (`invalidate=0`):** all 8 workers on a slice walk the same byte range concurrently, so GCSFuse sees each byte once while each handle fetches ~1/8 of its range in forward jumps — the exact trigger for GCSFuse's per-handle Random classification and ~5.7× GCS read amplification.
- **40 GB batches + batch barrier:** one fio invocation per batch. All 12 batches match the full GPU run exactly:
  - Batch 1: `38.98 GB` (files `1–9`)
  - Batches 2–11: `39.80 GB` each (files `9–17`, `17–25`, …, `81–89`)
  - Batch 12: `34.32 GB` (files `89–96`)

## Validation

- **Batch plan vs full RunAI run:** exact match on all 12 batch sizes (to 0.01 GB) and file ranges.
- **Local fio run (stock fio 3.36, Batch 1 on sparse files of the real sizes):** `rc=0`, `err=0`, 128 concurrent workers, `148,792` 2 MiB reads, `311.84 GB` app-level reads (`= 8 × 38.98 GB`).
- **Not yet run against GCSFuse:** ready to run on `a CPU VM` once the RunAI doc is reviewed.

## How to run

```bash
# Mount like the RunAI run (kernel reader off, infinite metadata TTL, no file cache), with wire log + metrics:
gcsfuse --wire-log=/out/wirelog.json --implicit-dirs --metadata-cache-ttl-secs -1 --file-cache-max-size-mb 0 \
        --prometheus-port 9091 <bucket> /mnt/gcs

# Replay Batch 1 (~39 GB, ~20 min cross-region) at default 128 KiB read-ahead (or "all" for all 12 batches):
sudo FIO=fio PY=python3 METRICS_URL=http://localhost:9091/metrics \
     ./run_qwen3vl_runai_fio.sh /mnt/gcs/Qwen3-VL-235B 128 /out 1
```

**What to check in GCSFuse metrics after Batch 1:**
- `gcs_read_count{read_type="Random"}` vs `Sequential` (expected: >99% Random)
- `gcs_read_bytes_count` / `38.98 GB` (expected: ~5.7× read amplification)
- FUSE wire log: 0% duplicate bytes, 4–128 KiB request sizes, ~22 reads in flight.
