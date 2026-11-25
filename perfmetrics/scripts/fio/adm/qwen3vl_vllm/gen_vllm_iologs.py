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

"""Generate per-TP-rank fio iologs that replay vanilla vLLM v0.16's weight-load page touches.

ADM fio job v3 for Qwen3-VL-235B-A22B-Instruct, vanilla vLLM, TP=8, safetensors lazy load (mmap).

Why iologs (v3) instead of fio sections (v2):
  * One file handle per rank per model file, kept open across all tensors of the file, as in vLLM
    (safe_open + one mmap per rank per file). v2 opened a new handle per section, which resets GCSFuse's
    per-handle reader state (read-type classifier, prefetch) between tensors.
  * The real copy ORDER inside each tensor, not just the right bytes:
      - fused experts are copied through a transposed view, so the copy walks the source column by column:
        down_proj faults the first page of every row in a rank's slice, then the second pages, ...
        ("even pages, then odd" in the GPU trace); gate_up faults the page holding each row's piece
        for all 4096 rows of an expert, then the pages where pieces straddle a page boundary.
      - gate_up: w1 (gate half) for all experts, then w3 (up half) for all experts.
  * The vision tower (file 1) is TP-sharded like vLLM does it (v2 modelled it as a replicated sweep).
  * No barriers between tensors: each rank runs through the file on its own, as vLLM workers do.

Each iolog line touches one page (or a contiguous run) at the first byte the rank's copy touches there.
fio's mmap engine memcpy()s from a shared mapping of the whole file, so each line faults exactly the pages
it covers, and the kernel applies the same read-around and page-cache sharing as for vLLM.

Sources (vLLM 0.16.0 wheel): models/qwen3_vl_moe.py L157-181, L208-217, L264-291 (fused experts: transpose,
chunk, w1 then w3, per-expert loop); layers/fused_moe/layer.py L1018-1068, L1198 (narrow on the sharded dim);
layers/linear.py (QKVParallelLinear fused qkv split into q/k/v, num_kv_head_replicas; Row/ColumnParallel);
layers/vocab_parallel_embedding.py L460; models/qwen3_vl.py (vision: QKV/Row/ColumnParallelLinear with
mm_encoder_tp_mode="weights", Conv3d and pos_embed replicated).

Usage:
  gen_vllm_iologs.py --header <hdr.json | safetensors file> --path <file path as fio will open it>
                     --out <dir> [--tp 8]
Writes <out>/rank{0..7}.iolog and <out>/summary.json (bytes/pages/lines per rank and per tensor kind).
"""
import argparse
import json
import os
import struct
import sys

import numpy as np

PAGE = 4096
DT = {"BF16": 2, "F16": 2, "F32": 4}
CHUNK = 1 << 20     # contiguous runs are emitted in <=1 MiB lines (memcpy faults them front to back anyway)


def read_header(src):
    """Header from a saved JSON (bytes of the safetensors header) or straight from a .safetensors file."""
    with open(src, "rb") as f:
        head = f.read(8)
        if src.endswith(".safetensors"):
            n = struct.unpack("<Q", head)[0]
            raw = f.read(n)
        else:
            raw = head + f.read()
    h = json.loads(raw)
    h.pop("__metadata__", None)
    base = 8 + len(raw)
    ts = []
    for k, v in h.items():
        s, e = v["data_offsets"]
        ts.append((base + s, base + e, k, v["dtype"], v["shape"]))
    ts.sort()
    return base, ts


# ----------------------------------------------------------------------------------------------- touch orders
def contiguous(start, length):
    """Front-to-back copy of [start, start+length)."""
    out = []
    o, end = start, start + length
    while o < end:
        n = min(CHUNK, end - o)
        out.append((o, n))
        o += n
    return out


def rows_piece(t0, n_rows, row_bytes, piece_off, piece_len):
    """Row-by-row copy of the same piece of every row (row-parallel weight: dest inner dim contiguous)."""
    starts = t0 + np.arange(n_rows, dtype=np.int64) * row_bytes + piece_off
    if piece_off == 0 and piece_len == row_bytes:
        return contiguous(t0, n_rows * row_bytes)
    return list(zip(starts.tolist(), [piece_len] * n_rows))


def column_sweep(line_starts, piece_len):
    """First-touch order of a copy whose OUTER loop walks the byte position inside each line's piece and whose
    INNER loop walks the lines (transposed fused-expert view). Returns [(offset, length)] in touch order, one
    entry per page, at the first byte touched in that page."""
    s = np.asarray(line_starts, dtype=np.int64)
    p0 = s // PAGE
    p1 = (s + piece_len - 1) // PAGE
    span = int((p1 - p0).max()) + 1
    pos, line, page = [], [], []
    for k in range(span):
        pg = p0 + k
        ok = pg <= p1
        if not ok.any():
            continue
        first_byte = np.maximum(pg[ok] * PAGE, s[ok])          # first byte of this line's piece in that page
        pos.append(first_byte - s[ok])                           # = byte position in the piece when touched
        line.append(np.nonzero(ok)[0])
        page.append(pg[ok])
    pos, line, page = (np.concatenate(a) for a in (pos, line, page))
    order = np.lexsort((line, pos))                              # outer: position, inner: line
    page = page[order]
    _, first = np.unique(page, return_index=True)                # keep each page's first touch
    keep = np.sort(first)
    o = order[keep]
    off = np.maximum(page[keep] * PAGE, s[line[o]])
    end = np.minimum((page[keep] + 1) * PAGE, s[line[o]] + piece_len)
    return list(zip(off.tolist(), (end - off).tolist()))


# ----------------------------------------------------------------------------------------------- sharding rules
def rank_touches(name, dtype, shape, t0, t1, r, tp):
    """[(offset, length)] in copy order for rank r, and a short 'kind' label."""
    es = DT[dtype]
    n = name

    def rows_block(row0, nrows, row_bytes):
        return contiguous(t0 + row0 * row_bytes, nrows * row_bytes)

    # ---- fused MoE experts (transposed view, per-expert loop) ----
    if n.endswith("mlp.experts.down_proj"):
        E, I, H = shape                                   # [128, 1536, 4096], w2 sharded on I
        per = I // tp
        out = []
        for e in range(E):
            lines = t0 + (e * I + r * per + np.arange(per)) * (H * es)
            out += column_sweep(lines, H * es)
        return out, "experts.down_proj"
    if n.endswith("mlp.experts.gate_up_proj"):
        E, H, I2 = shape                                  # [128, 4096, 3072]; w1 = cols [0,I), w3 = [I,2I)
        I = I2 // 2
        per = I // tp
        row = I2 * es
        out = []
        for half in (0, 1):                               # w1 for all experts, then w3 for all experts
            for e in range(E):
                lines = t0 + (e * H + np.arange(H)) * row + (half * I + r * per) * es
                out += column_sweep(lines, per * es)
        return out, "experts.gate_up_proj"

    # ---- vocab-parallel ----
    if n.endswith("embed_tokens.weight") or n.endswith("lm_head.weight"):
        V, H = shape
        per = V // tp
        return rows_block(r * per, per, H * es), "vocab_parallel"

    # ---- language-model attention (64 q heads, 4 kv heads, head_dim 128) ----
    if n.endswith("self_attn.q_proj.weight"):
        O, H = shape
        per = O // tp
        return rows_block(r * per, per, H * es), "q_proj"
    if n.endswith("self_attn.k_proj.weight") or n.endswith("self_attn.v_proj.weight"):
        O, H = shape                                      # 4 kv heads x 128; replicas = tp // 4
        heads = O // 128
        rep = max(1, tp // heads)
        per = O // min(tp, heads)
        return rows_block((r // rep) * per, per, H * es), "kv_proj"
    if n.endswith("self_attn.o_proj.weight"):
        O, I = shape                                      # row-parallel on the input dim
        per = I // tp
        return rows_piece(t0, O, I * es, r * per * es, per * es), "o_proj"

    # ---- vision tower (TP-sharded, mm_encoder_tp_mode="weights") ----
    if n.endswith("linear_fc1.bias") or n.endswith("attn.qkv.bias"):
        # Column-parallel biases are sharded in vLLM, yet the GPU trace shows ranks fetching pages of these
        # biases outside their shard (49 of 50 out-of-model fetches in file 1 go away when every rank reads the
        # whole bias). Modelled as read in full by every rank; the code path that touches them is not identified.
        return contiguous(t0, t1 - t0), "vision.bias_full"
    if n.endswith("attn.qkv.weight"):
        O = shape[0]                                      # fused q|k|v, equal sizes (16 heads, no GQA)
        per_part = O // 3
        per = per_part // tp
        row = (shape[1] if len(shape) > 1 else 1) * es
        out = []
        for part in range(3):                             # q, then k, then v shard
            out += rows_block(part * per_part + r * per, per, row)
        return out, "vision.qkv"
    if n.endswith("linear_fc1.weight"):
        O = shape[0]                                      # column-parallel: rows / bias elements
        per = O // tp
        row = (shape[1] if len(shape) > 1 else 1) * es
        return rows_block(r * per, per, row), "vision.column"
    if n.endswith("attn.proj.weight") or n.endswith("linear_fc2.weight"):
        O, I = shape                                      # row-parallel on the input dim
        per = I // tp
        return rows_piece(t0, O, I * es, r * per * es, per * es), "vision.row"

    # ---- replicated: every rank copies the whole tensor ----
    replicated = ("norm.weight", "norm.bias", "layernorm.weight", "norm1.weight", "norm1.bias", "norm2.weight",
                  "norm2.bias", "q_norm.weight", "k_norm.weight", "mlp.gate.weight", "proj.bias", "linear_fc2.bias",
                  "patch_embed.proj.weight", "patch_embed.proj.bias", "pos_embed.weight")
    if n.endswith(replicated):
        return contiguous(t0, t1 - t0), "replicated"
    raise ValueError(f"no sharding rule for tensor {n} {dtype} {shape}")


# ----------------------------------------------------------------------------------------------- main
def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--header", required=True)
    ap.add_argument("--path", required=True, help="model file path as fio will open it")
    ap.add_argument("--out", required=True)
    ap.add_argument("--tp", type=int, default=8)
    a = ap.parse_args()
    if any(c.isspace() for c in a.path):
        sys.exit("--path must not contain whitespace: fio splits iolog lines on spaces")
    base, ts = read_header(a.header)
    os.makedirs(a.out, exist_ok=True)
    summary = {"path": a.path, "data_start": base, "file_end": max(t[1] for t in ts), "tensors": len(ts), "ranks": []}
    for r in range(a.tp):
        lines, nbytes, kinds = 0, 0, {}
        pages = set()
        fn = os.path.join(a.out, f"rank{r}.iolog")
        with open(fn, "w") as f:
            w = f.write
            w("fio version 2 iolog\n")
            w(f"{a.path} add\n{a.path} open\n")
            # safe_open reads the 8-byte length + JSON header first
            w(f"{a.path} read 0 {base}\n")
            lines += 1
            for (t0, t1, name, dtype, shape) in ts:
                touches, kind = rank_touches(name, dtype, shape, t0, t1, r, a.tp)
                k = kinds.setdefault(kind, [0, 0])
                for off, ln in touches:
                    w(f"{a.path} read {off} {ln}\n")
                    k[0] += ln
                    k[1] += 1
                    nbytes += ln
                    pages.update(range(off // PAGE, (off + ln - 1) // PAGE + 1))
                lines += len(touches)
            w(f"{a.path} close\n")
        summary["ranks"].append({"rank": r, "lines": lines, "bytes": nbytes, "pages": len(pages),
                                 "kinds": {k: {"bytes": v[0], "lines": v[1]} for k, v in kinds.items()}})
        print(f"rank {r}: {lines:,} lines, {nbytes / 2**20:,.1f} MiB touched, {len(pages):,} pages", flush=True)
    json.dump(summary, open(os.path.join(a.out, "summary.json"), "w"), indent=1)


if __name__ == "__main__":
    sys.exit(main())
