# ADM fio job v3: vanilla vLLM v0.16 loading Qwen3-VL-235B through GCSFuse

Reproduces the vanilla vLLM (TP=8, default safetensors loader) weight-load read pattern with fio, so GCSFuse
can be tested on a CPU VM instead of 8× B200.

## Files

| File | What |
|---|---|
| `qwen3vl_vllm_tp8.fio` | fio job for ONE model file: 8 jobs = 8 TP ranks, each replaying its own iolog |
| `gen_vllm_iologs.py` | builds the 8 per-rank iologs for a model file from its safetensors header + vLLM's sharding rules |
| `run_qwen3vl_fio.sh` | runs the load file by file, in order, against a GCSFuse mount (drops cache once, sets read-ahead, snapshots metrics) |
| `headers/` | safetensors headers of files 1, 2 (layout of files 2-94), 95, 96 |
| `fio-mmap-iolog.patch` | fio patch required by v3 (see below) |

## What it reproduces

- **Page-fault reads.** `ioengine=mmap`: each job memcpy()s from a shared mapping of the whole file, so reads are
  page faults, the kernel applies the same fault read-around as for vLLM, and the 8 ranks share one page cache.
- **One file handle per rank per file**, kept open while the rank walks the whole file (vLLM: one safe_open +
  mmap per rank per file). v2 opened a new handle per tensor section, which resets GCSFuse's per-handle state.
- **Each rank's exact pages, in vLLM's copy order:**
  - files in order; tensors in on-disk order (= key order, no backward jumps);
  - down_proj: copied through a transposed view, so per expert the rank faults the first page of each of its
    192 rows, then the second pages ("even pages, then odd" in the GPU trace);
  - gate_up: w1 (gate half) for all experts, then w3 (up half); 384 B per 6 KiB row per rank, so rank 7 needs
    2 of every 3 pages and ranks 0-6 queue on the third, as on the GPUs;
  - attention: q_proj 8 MiB per rank, k/v 1 MiB shared by rank pairs, o_proj 2 KiB of every 16 KiB row;
  - embed_tokens / lm_head: 148 MiB contiguous per rank; norms and router replicated;
  - vision tower (file 1) TP-sharded: qkv split into q/k/v shards, row-parallel proj/fc2, column-parallel fc1,
    replicated norms, patch embed and pos embed. Column biases are read in full by every rank (see gaps).
- **No barriers between tensors**: each rank runs through the file on its own.

## Validation

**Against the real GPU trace** (files 1, 2, 50, 95, 96; read_ahead≈0 trace):
- Every page the trace fetched is in the model's page set of some rank (union covers 100%).
- Pages the trace shows a rank fetching that the model says that rank never touches: 3 out of ~4.5M fetches.
- Pages the model says only one rank touches were never fetched by another rank in the trace (0 violations).
- Order: inside down_proj, gate_up, o_proj and vision row-parallel weights, 100% of each rank's consecutive trace
  fetches move forward in the model's order (vision row-parallel: ≥99.7%). Plain ascending offsets (v2's order)
  give 99.74% in down_proj, because of the even/odd sweep.

**Local fio run** (sparse files of the real sizes, local disk):
- fio exits 0, each job's IO count equals its iolog line count.
- Exactly 8 opens of the model file, from 8 distinct processes (one handle per rank).
- Every page the model needs is resident afterwards.

**Run against GCSFuse** (CPU VM n2-standard-16, us-east7-b → bucket us-east1, same region pair as the GPU run;
GCSFuse 3.11.2; read_ahead_kb=0; 2026-11-23; results `(validation results)`, analysis
`compare_fio_vs_gpu.py` → `compare.json`):

| File | GPU wall | fio wall | GPU down_proj | fio down_proj | GPU gate_up | fio gate_up | gate_up leader done / rest done (GPU) | (fio) |
|---|---|---|---|---|---|---|---|---|
| 1 (embed + vision) | 22.8 s | 22.4 s | | | | | | |
| 2 | 883 s | 673 s | 89 s | 75 s | 776 s | 584 s | r7 489 / 769 s | r7 432 / 579 s |
| 3 | 866 s | 658 s | 85 s | 72 s | 763 s | 573 s | r7 454 / 755 s | r7 437 / 567 s |
| 4 | 862 s | 632 s | 83 s | 68 s | 761 s | 550 s | r0 436 / 754 s | r7 409 / 545 s |
| 95 | 899 s | 847 s | 85 s | 68 s | 813 s | 779 s | no leader | no leader |
| 96 (lm_head) | 4.0 s | 4.2 s | | | | | | |

- Reproduced: file-level times for files 1 and 96 (v2 had file 1 at 276 s); down_proj split exactly evenly per
  rank (0.188 GiB each); in gate_up one rank runs ahead and fetches ~2 GiB while the rest queue; file 95 (different
  alignment) has no leader in both.
- fio is faster on layer files: gate_up ~25% faster, down_proj ~15% faster, file 95 ~6% faster. Likely causes: no
  copy work between faults in fio, different VM, network on a different day. Not separated.
- GCSFuse metrics during layer files (not available for the GPU run): ~11.7k GCS reads per file, all but one
  classified Random; 10.8-11.0 GB read from GCS per 4.98 GB file (~2.2x). So the per-handle Random classification
  measured for RunAI also shows up for the vanilla vLLM pattern, measured here on the replay.
- Per-read FUSE latency (fio): p50 0.06 ms, p90 ~1.6 ms, p99 ~16 ms on layer files.
- Leader: in the GPU trace gate_up's leader is rank 7 in 68 of 93 layer files, rank 0 in 24, rank 6 in 1. Rank 0
  also skips one of the three page positions (@6736), so it can lead too. fio had rank 7 lead in files 2-4.

## How to run

```bash
# fio 3.36 + patch
git clone https://github.com/axboe/fio && cd fio && git checkout fio-3.36
patch -p1 < fio-mmap-iolog.patch && ./configure && make -j
# mount like the GPU run, with a wire log (and metrics if wanted)
gcsfuse --wire-log=/out/wirelog.json --implicit-dirs --metadata-cache-ttl-secs -1 --file-cache-max-size-mb 0 \
        --prometheus-port 9091 <bucket> /mnt/gcs
# replay files 1-4, 95, 96 at read-ahead 0 (or "all" for the full ~22 h load)
sudo FIO=/path/to/fio PY=python3 METRICS_URL=http://localhost:9091/metrics \
     ./run_qwen3vl_fio.sh /mnt/gcs/Qwen3-VL-235B 0 /out 1 2 3 4 95 96
```
Needs python3 with numpy. Iologs are ~0.7 GB per layer file; the runner generates the next file's iologs while
the current one replays, and deletes them after use. Paths must not contain whitespace (fio splits iolog lines
on spaces).

## Known gaps

1. **A barrier between files.** Each file is one fio run; in vLLM the next file starts ~4 s before the previous
   one ends (~0.5% of a layer file's ~856 s).
2. **No compute between faults.** fio faults pages back to back; vLLM also copies data. v2 showed down_proj
   faster in fio (29 s vs ~84 s) for this reason. gate_up, which dominates, is bound by waiting on shared pages.
3. **Column biases in the vision tower** are modelled as read in full by every rank. The trace requires it
   (49 of 50 unexplained fetches in file 1 disappear), but the vLLM code path that touches them is not identified.
4. **Driver process** (config/metadata reads, 9th PID) is not modelled; it fetched 0 model pages in the trace.
5. **Files 2-94 are assumed to share file 2's layout**; the runner checks each file's size against it.
6. **GCSFuse version**: the GPU trace used v3.11.2 (inferred, not confirmed). Pin the version you compare.
