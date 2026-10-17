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

"""Synthetic Multimodal ArrayRecord Data Generator for MaxText / PyGrain Benchmark.

Generates representative Multimodal LLM (VLM) training data shards containing
high-resolution vision features/patches and text tokens to evaluate GCSFuse
bucket I/O throughput and measure NIC line-rate saturation.
"""

import argparse
from concurrent.futures import ProcessPoolExecutor, as_completed
import glob
import os
import shutil
import time
import numpy as np

try:
  from array_record.python.array_record_module import ArrayRecordWriter
except ImportError:
  try:
    from array_record import ArrayRecordWriter
  except ImportError:
    ArrayRecordWriter = None


def existing_shard_count(
    output_dir: str, shard_prefix: str = "train", num_shards: int = 128
) -> int:
  """Counts valid non-empty ArrayRecord shards present in output_dir."""
  if not os.path.isdir(output_dir):
    return 0
  pattern = os.path.join(
      output_dir, f"{shard_prefix}-*-of-{num_shards:05d}.arrayrecord"
  )
  matched = glob.glob(pattern)
  if not matched:
    matched = glob.glob(os.path.join(output_dir, f"{shard_prefix}-*.arrayrecord"))
  valid = [p for p in matched if os.path.isfile(p) and os.path.getsize(p) > 0]
  return len(valid)


def create_synthetic_vlm_record(
    seq_length: int = 2048,
    image_bytes_size: int = 2 * 1024 * 1024,
    preallocated_image_buf: bytes = b"",
) -> bytes:
  """Generates a serialized synthetic multimodal training sample.

  Args:
    seq_length: Number of token IDs in sequence.
    image_bytes_size: Size in bytes of high-resolution image patches.
    preallocated_image_buf: Optional pre-allocated random image patch bytes.

  Returns:
    Serialized binary payload representing the multimodal sample.
  """
  # Text tokens: int32 array (4 bytes per token)
  tokens = np.random.randint(
      1, 32000, size=seq_length, dtype=np.int32
  ).tobytes()
  # Positions: int32 array
  positions = np.arange(seq_length, dtype=np.int32).tobytes()
  # Image masks: boolean array (1 byte per token)
  image_masks = np.zeros(seq_length, dtype=np.bool_).tobytes()
  image_masks_arr = np.frombuffer(image_masks, dtype=np.bool_).copy()
  image_masks_arr[: min(256, seq_length)] = True
  image_masks = image_masks_arr.tobytes()

  # Vision feature payload (e.g. 2 MB raw image patches or ViT activations)
  if preallocated_image_buf and len(preallocated_image_buf) == image_bytes_size:
    image_payload = preallocated_image_buf
  else:
    image_payload = os.urandom(image_bytes_size)

  # Pack with length headers for zero-copy deserialization in PyGrain
  # Header: [tokens_len (4B), positions_len (4B), masks_len (4B), image_len (4B)]
  header = np.array(
      [len(tokens), len(positions), len(image_masks), len(image_payload)],
      dtype=np.int32,
  ).tobytes()

  return header + tokens + positions + image_masks + image_payload


def generate_shard(
    shard_path: str,
    target_shard_size_mb: int,
    record_size_mb: float,
    seq_length: int,
    seed: int = 42,
) -> int:
  """Generates a single ArrayRecord shard file.

  Args:
    shard_path: Destination path for the shard.
    target_shard_size_mb: Target file size in MiB (e.g., 512 MB to 1024 MB).
    record_size_mb: Target record size in MiB (e.g., 2.0 MB).
    seq_length: Number of tokens per record.
    seed: Random seed for worker process.

  Returns:
    Total bytes written.
  """
  np.random.seed(seed)
  image_bytes = int((record_size_mb * 1024 * 1024) - (seq_length * 9 + 16))
  if image_bytes < 1024:
    image_bytes = 1024

  target_bytes = int(target_shard_size_mb * 1024 * 1024)
  written_bytes = 0
  records_written = 0

  # Pre-allocate a ring buffer of 8 distinct random vision patch payloads per worker
  # so generation is not bottlenecked by single-threaded PRNG calls
  patch_ring = [os.urandom(image_bytes) for _ in range(8)]

  if ArrayRecordWriter is None:
    print(
        "[Warning] ArrayRecordWriter not found. Writing raw indexed binary"
        f" container: {shard_path}",
        flush=True,
    )
    with open(shard_path, "wb") as f:
      while written_bytes < target_bytes:
        rec = create_synthetic_vlm_record(
            seq_length,
            image_bytes,
            preallocated_image_buf=patch_ring[
                records_written % len(patch_ring)
            ],
        )
        rec_len = len(rec)
        f.write(np.array([rec_len], dtype=np.int32).tobytes())
        f.write(rec)
        written_bytes += 4 + rec_len
        records_written += 1
  else:
    # Use native ArrayRecordWriter with uncompressed group_size:1 for fast I/O
    try:
      writer = ArrayRecordWriter(shard_path, "group_size:1,uncompressed")
    except Exception:
      writer = ArrayRecordWriter(shard_path, "group_size:10")
    while written_bytes < target_bytes:
      rec = create_synthetic_vlm_record(
          seq_length,
          image_bytes,
          preallocated_image_buf=patch_ring[records_written % len(patch_ring)],
      )
      writer.write(rec)
      written_bytes += len(rec)
      records_written += 1
    writer.close()

  actual_size = (
      os.path.getsize(shard_path)
      if os.path.exists(shard_path)
      else written_bytes
  )
  print(
      f"Generated {shard_path}: {records_written} records, "
      f"payload={written_bytes / (1024 * 1024):.2f} MiB, "
      f"file={actual_size / (1024 * 1024):.2f} MiB",
      flush=True,
  )
  return actual_size


def _copy_shard_task(src_path: str, dst_path: str) -> int:
  """Copies a generated ArrayRecord shard to a target shard path."""
  shutil.copyfile(src_path, dst_path)
  size_bytes = os.path.getsize(dst_path)
  print(
      f"Populated {dst_path}: {size_bytes / (1024 * 1024):.2f} MiB",
      flush=True,
  )
  return size_bytes


def main():
  parser = argparse.ArgumentParser(
      description="Synthetic Multimodal ArrayRecord Generator"
  )
  parser.add_argument(
      "--output-dir",
      type=str,
      required=True,
      help="Output directory (local path or GCSFuse mount point)",
  )
  parser.add_argument(
      "--num-shards",
      type=int,
      default=128,
      help="Number of shard files to generate (default: 128)",
  )
  parser.add_argument(
      "--shard-size-mb",
      type=int,
      default=512,
      help="Size of each shard in MiB (default: 512 MB)",
  )
  parser.add_argument(
      "--record-size-mb",
      type=float,
      default=2.0,
      help=(
          "Target record size in MiB representing a multimodal sample (default:"
          " 2.0 MB)"
      ),
  )
  parser.add_argument(
      "--seq-length",
      type=int,
      default=2048,
      help="Sequence length in tokens (default: 2048)",
  )
  parser.add_argument(
      "--shard-prefix",
      type=str,
      default="train",
      help="Prefix for shard filenames (default: train)",
  )
  parser.add_argument(
      "--num-workers",
      type=int,
      default=min(32, os.cpu_count() or 8),
      help="Number of parallel worker processes (default: min(32, cpu_count))",
  )
  parser.add_argument(
      "--template-pool-size",
      type=int,
      default=8,
      help=(
          "Number of unique base ArrayRecord shards to synthesize before"
          " parallel fan-out (default: 8; set 0 to synthesize all from scratch)"
      ),
  )
  parser.add_argument(
      "--staging-dir",
      type=str,
      default="",
      help=(
          "Optional fast local staging directory (e.g., /dev/shm/staging) for"
          " template shards"
      ),
  )
  parser.add_argument(
      "--force-regenerate",
      action="store_true",
      default=False,
      help="Force regeneration even if all shards already exist in output-dir",
  )

  args = parser.parse_args()
  os.makedirs(args.output_dir, exist_ok=True)

  present_shards = existing_shard_count(
      args.output_dir, shard_prefix=args.shard_prefix, num_shards=args.num_shards
  )
  if present_shards >= args.num_shards and not args.force_regenerate:
    print(
        f"Dataset check: found {present_shards}/{args.num_shards} existing"
        f" ArrayRecord shards in {args.output_dir}. Skipping regeneration and"
        " reusing existing dataset.",
        flush=True,
    )
    return

  total_target_gib = (args.num_shards * args.shard_size_mb) / 1024.0
  print("=" * 70)
  print("Generating Synthetic Multimodal Dataset for MaxText Benchmark")
  print(f"Output Directory : {args.output_dir}")
  print(f"Existing Shards  : {present_shards}/{args.num_shards}")
  print(f"Number of Shards : {args.num_shards}")
  print(f"Shard Size       : {args.shard_size_mb} MiB")
  print(
      f"Record Size      : {args.record_size_mb:.2f} MiB (Multimodal Payload)"
  )
  print(f"Total Target Size: {total_target_gib:.2f} GiB")
  print(f"Parallel Workers : {args.num_workers}")
  print(
      "ArrayRecord Lib  :"
      f" {'ENABLED' if ArrayRecordWriter is not None else 'FALLBACK_BINARY'}"
  )
  print("=" * 70, flush=True)

  start_time = time.time()
  total_bytes = 0

  shard_paths = [
      os.path.join(
          args.output_dir,
          f"{args.shard_prefix}-{i:05d}-of-{args.num_shards:05d}.arrayrecord",
      )
      for i in range(args.num_shards)
  ]

  use_pool = (
      args.template_pool_size > 0 and args.num_shards > args.template_pool_size
  )

  if use_pool:
    pool_size = min(args.template_pool_size, args.num_shards)
    staging_dir = args.staging_dir if args.staging_dir else args.output_dir
    os.makedirs(staging_dir, exist_ok=True)

    if staging_dir != args.output_dir:
      base_paths = [
          os.path.join(staging_dir, f"template-{i:05d}.arrayrecord")
          for i in range(pool_size)
      ]
    else:
      base_paths = shard_paths[:pool_size]

    print(
        f"[Phase 1] Synthesizing {pool_size} unique {args.shard_size_mb} MiB"
        " ArrayRecord base shards...",
        flush=True,
    )
    with ProcessPoolExecutor(
        max_workers=min(pool_size, args.num_workers)
    ) as ex:
      futs = {
          ex.submit(
              generate_shard,
              base_paths[i],
              args.shard_size_mb,
              args.record_size_mb,
              args.seq_length,
              42 + i,
          ): i
          for i in range(pool_size)
      }
      for fut in as_completed(futs):
        sz = fut.result()
        if staging_dir == args.output_dir:
          total_bytes += sz

    if staging_dir != args.output_dir:
      target_indices = list(range(args.num_shards))
    else:
      target_indices = list(range(pool_size, args.num_shards))

    print(
        f"[Phase 2] Fanning out {len(target_indices)} ArrayRecord shards across"
        f" {args.num_workers} parallel workers...",
        flush=True,
    )
    with ProcessPoolExecutor(max_workers=args.num_workers) as ex:
      futs = [
          ex.submit(
              _copy_shard_task,
              base_paths[idx % pool_size],
              shard_paths[idx],
          )
          for idx in target_indices
      ]
      for fut in as_completed(futs):
        total_bytes += fut.result()

    if staging_dir != args.output_dir:
      for bp in base_paths:
        try:
          os.remove(bp)
        except OSError:
          pass
  else:
    with ProcessPoolExecutor(max_workers=args.num_workers) as ex:
      futs = [
          ex.submit(
              generate_shard,
              shard_paths[i],
              args.shard_size_mb,
              args.record_size_mb,
              args.seq_length,
              42 + i,
          )
          for i in range(args.num_shards)
      ]
      for fut in as_completed(futs):
        total_bytes += fut.result()

  elapsed = time.time() - start_time
  rate_mb_s = (total_bytes / (1024 * 1024)) / elapsed if elapsed > 0 else 0
  print("-" * 70)
  print(
      f"Generation Complete! Total {total_bytes / (1024**3):.2f} GiB "
      f"({total_bytes / 1e9:.2f} GB) across {args.num_shards} shards "
      f"in {elapsed:.1f}s ({rate_mb_s:.1f} MiB/s)",
      flush=True,
  )


if __name__ == "__main__":
  main()
