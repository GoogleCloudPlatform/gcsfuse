#!/usr/bin/env python3
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Generate per-slice fio iologs for vLLM v0.16 + RunAI Model Streamer 0.15.6 (TP=8, defaults).

Models RunAI's default non-distributed streaming load of Qwen3-VL-235B-A22B-Instruct (96 safetensors files):
  1. Packs tensors in file order into batches of up to MEMORY_LIMIT bytes (40,000,000,000 B default), matching
     runai_model_streamer/file_streamer/requests_iterator.py (FilesRequestsIterator.next_request).
  2. Splits each batch's byte stream evenly into CONCURRENCY (16 default) contiguous slices; a slice may cross
     a file boundary (finishing file k, then opening file k+1 at offset 0).
  3. Within each slice, emits 2 MiB (CHUNK_BYTESIZE default) sequential reads (plus any final partial block).

When replayed by qwen3vl_runai_tp8.fio (16 slice jobs x numjobs=8 = 128 workers, ioengine=sync, direct=0,
invalidate=0), all 8 TP processes walk the same 16 slices concurrently through the shared kernel page cache,
each worker opening its own fd per file and issuing lseek + 2 MiB read(). Running one fio invocation per batch
enforces the 40 GB batch barrier. Works with stock fio >= 3.36 (no fio patch needed).

Usage:
  gen_runai_iologs.py --model-dir /mnt/gcs/Qwen3-VL-235B --out <dir>
                      [--headers-dir ./headers] [--batches 1,2 | all]
                      [--memory-limit 40000000000] [--concurrency 16] [--block-size 2097152]
"""
import argparse
import json
import os
import struct
import sys


def load_tensor_layout(hdr_path):
    """Returns (data_start, file_end, [(abs_start, abs_end), ...]) sorted by offset."""
    with open(hdr_path, "rb") as f:
        head = f.read(8)
        if hdr_path.endswith(".safetensors"):
            n = struct.unpack("<Q", head)[0]
            raw = f.read(n)
        else:
            raw = head + f.read()
    h = json.loads(raw)
    h.pop("__metadata__", None)
    base = 8 + len(raw)
    spans = sorted((base + v["data_offsets"][0], base + v["data_offsets"][1]) for v in h.values())
    return base, spans[-1][1], spans


def build_batches(headers_dir, n_files, memory_limit):
    """Packs tensors across files 1..n_files into <= memory_limit batches (FilesRequestsIterator logic).
    Returns list of batches; each batch is a list of (file_no, start_off, end_off) contiguous file segments."""
    cache = {}
    for k in (1, 2, 95, 96):
        p = os.path.join(headers_dir, f"hdr_{k:05d}.json")
        if os.path.exists(p):
            cache[k] = load_tensor_layout(p)

    # Queue of (file_no, tensor_start, tensor_end)
    tensors = []
    for fno in range(1, n_files + 1):
        key = fno if fno in cache else (2 if 2 <= fno <= 94 else fno)
        _, _, spans = cache[key]
        for s, e in spans:
            tensors.append((fno, s, e))

    batches = []
    cur = []
    cur_bytes = 0
    for fno, s, e in tensors:
        sz = e - s
        if cur_bytes + sz > memory_limit and cur:
            batches.append(cur)
            cur = []
            cur_bytes = 0
        # Merge adjacent tensor spans within the same file
        if cur and cur[-1][0] == fno and cur[-1][2] == s:
            cur[-1] = (fno, cur[-1][1], e)
        else:
            cur.append((fno, s, e))
        cur_bytes += sz
    if cur:
        batches.append(cur)
    return batches


def split_batch(segments, concurrency):
    """Splits a batch's [(file_no, start, end)] byte stream into `concurrency` equal contiguous slices."""
    total = sum(e - s for _, s, e in segments)
    slices = [[] for _ in range(concurrency)]
    seg_i = 0
    seg_off = segments[0][1]
    for tid in range(concurrency):
        lo = (tid * total) // concurrency
        hi = ((tid + 1) * total) // concurrency
        need = hi - lo
        while need > 0 and seg_i < len(segments):
            fno, _, e = segments[seg_i]
            avail = e - seg_off
            take = min(need, avail)
            slices[tid].append((fno, seg_off, seg_off + take))
            seg_off += take
            need -= take
            if seg_off == e:
                seg_i += 1
                if seg_i < len(segments):
                    seg_off = segments[seg_i][1]
    return slices


def write_slice_iolog(path, model_dir, segs, block_size, n_files):
    lines = 0
    nbytes = 0
    with open(path, "w") as f:
        w = f.write
        w("fio version 2 iolog\n")
        for fno, s, e in segs:
            fp = os.path.join(model_dir, f"model-{fno:05d}-of-{n_files:05d}.safetensors")
            w(f"{fp} add\n{fp} open\n")
            off = s
            while off < e:
                ln = min(block_size, e - off)
                w(f"{fp} read {off} {ln}\n")
                off += ln
                lines += 1
                nbytes += ln
            w(f"{fp} close\n")
    return lines, nbytes


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--model-dir", required=True, help="path prefix as fio will open it (no spaces)")
    ap.add_argument("--out", required=True)
    ap.add_argument("--headers-dir", default=os.path.join(os.path.dirname(os.path.abspath(__file__)), "headers"))
    ap.add_argument("--n-files", type=int, default=96)
    ap.add_argument("--memory-limit", type=int, default=40_000_000_000)
    ap.add_argument("--concurrency", type=int, default=16)
    ap.add_argument("--block-size", type=int, default=2 * 1024 * 1024)
    ap.add_argument("--batches", default="all", help="comma-separated 1-based batch numbers, or 'all'")
    a = ap.parse_args()

    if any(c.isspace() for c in a.model_dir):
        sys.exit("--model-dir must not contain whitespace: fio splits iolog lines on spaces")

    batches = build_batches(a.headers_dir, a.n_files, a.memory_limit)
    want = set(range(1, len(batches) + 1)) if a.batches == "all" else {int(x) for x in a.batches.split(",")}
    os.makedirs(a.out, exist_ok=True)

    plan = {"n_batches": len(batches), "concurrency": a.concurrency, "block_size": a.block_size,
            "memory_limit": a.memory_limit, "batches": []}
    for bi, segs in enumerate(batches, start=1):
        bbytes = sum(e - s for _, s, e in segs)
        files = f"{segs[0][0]}-{segs[-1][0]}"
        entry = {"batch": bi, "GB": round(bbytes / 1e9, 2), "files": files, "slices": []}
        if bi in want:
            bdir = os.path.join(a.out, f"batch_{bi:02d}")
            os.makedirs(bdir, exist_ok=True)
            slices = split_batch(segs, a.concurrency)
            for si, s_segs in enumerate(slices):
                lines, sb = write_slice_iolog(os.path.join(bdir, f"slice_{si:02d}.iolog"),
                                              a.model_dir, s_segs, a.block_size, a.n_files)
                entry["slices"].append({"slice": si, "lines": lines, "GB": round(sb / 1e9, 3),
                                        "files": [x[0] for x in s_segs]})
        plan["batches"].append(entry)
    json.dump(plan, open(os.path.join(a.out, "batches_plan.json"), "w"), indent=1)
    for b in plan["batches"]:
        mark = "*" if b["batch"] in want else " "
        print(f"{mark} batch {b['batch']:2d}: {b['GB']:5.2f} GB  files {b['files']}")


if __name__ == "__main__":
    main()
