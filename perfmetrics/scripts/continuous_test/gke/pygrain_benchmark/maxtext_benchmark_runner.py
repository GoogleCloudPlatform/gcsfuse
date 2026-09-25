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

"""MaxText + PyGrain + GCSFuse Benchmark Runner.

Benchmarks data loading throughput from Google Cloud Storage buckets
targeting physical NIC saturation on Cloud TPU v6e-4 (System Mode) and
standard CPU nodes (Emulated Mode). Supports multi-iteration execution
emitting per-iteration `gbytes_per_sec: <val> Bytes/s` metrics for
GCSFuse continuous GKE tests.
"""

import argparse
import datetime
import glob
import json
import os
import socket
import subprocess
import sys
import threading
import time
from typing import Any, Dict, List, Optional, Tuple
import warnings

os.environ["PYTHONWARNINGS"] = (
    "ignore::UserWarning:multiprocessing.resource_tracker"
)
# Ensure parent data-loader process uses CPU backend for JAX before importing jax
# so PyGrain's multiprocessing workers fork immediately without the 60s libtpu lock.
os.environ["JAX_PLATFORMS"] = "cpu"

warnings.filterwarnings(
    "ignore",
    category=UserWarning,
    module=r"multiprocessing\.resource_tracker",
)

import numpy as np

# PyGrain imports
try:
  import grain.python as pygrain
except ImportError:
  try:
    import grain as pygrain
  except ImportError:
    pygrain = None

# Optional JAX import
try:
  import jax
except ImportError:
  jax = None

_BaseMapTransform = (
    pygrain.MapTransform
    if (pygrain is not None and hasattr(pygrain, "MapTransform"))
    else object
)
_BaseRandomAccessDataSource = (
    pygrain.RandomAccessDataSource
    if (pygrain is not None and hasattr(pygrain, "RandomAccessDataSource"))
    else object
)


def _drop_page_cache() -> bool:
  """Drops Linux kernel VFS page cache while preserving FUSE inode/dentry metadata."""
  try:
    with open("/proc/sys/vm/drop_caches", "w") as f:
      f.write("1\n")
    return True
  except Exception:
    return False


def _collect_host_tuning_verification(
    interface: str = "eth0",
) -> Dict[str, Any]:
  """Captures host network, LRO, sysctl, THP, and FUSE mount verification."""
  info: Dict[str, Any] = {
      "eth0_lro": "unknown",
      "rmem_max": 0,
      "thp_enabled": "unknown",
      "fuse_max_background": [],
      "fuse_congestion_threshold": [],
      "gcsfuse_mount": "unknown",
  }
  try:
    out = subprocess.check_output(
        ["ethtool", "-k", interface], stderr=subprocess.STDOUT, text=True
    )
    for line in out.splitlines():
      if "large-receive-offload:" in line:
        info["eth0_lro"] = line.split(":", 1)[1].strip()
        break
  except Exception:
    pass

  try:
    with open("/proc/sys/net/core/rmem_max", "r") as f:
      info["rmem_max"] = int(f.read().strip())
  except Exception:
    pass

  try:
    with open("/sys/kernel/mm/transparent_hugepage/enabled", "r") as f:
      info["thp_enabled"] = f.read().strip()
  except Exception:
    pass

  try:
    for p in sorted(glob.glob("/sys/fs/fuse/connections/*/max_background")):
      try:
        with open(p, "w") as fw:
          fw.write("512\n")
      except Exception:
        pass
      with open(p, "r") as f:
        info["fuse_max_background"].append(int(f.read().strip()))
    for p in sorted(
        glob.glob("/sys/fs/fuse/connections/*/congestion_threshold")
    ):
      try:
        with open(p, "w") as fw:
          fw.write("512\n")
      except Exception:
        pass
      with open(p, "r") as f:
        info["fuse_congestion_threshold"].append(int(f.read().strip()))
  except Exception:
    pass

  try:
    with open("/proc/mounts", "r") as f:
      for line in f:
        if "/gcs/data" in line or "gcsfuse" in line:
          info["gcsfuse_mount"] = line.strip()
          break
  except Exception:
    pass

  return info


class NicTelemetrySampler:
  """Samples Linux network interface statistics and CPU utilization at 1s intervals."""

  def __init__(self, interface: str = "eth0", sample_interval: float = 1.0):
    self.interface = interface
    self.sample_interval = sample_interval
    self.rx_bytes_path = f"/sys/class/net/{interface}/statistics/rx_bytes"
    self.rx_packets_path = f"/sys/class/net/{interface}/statistics/rx_packets"
    self.rx_dropped_path = f"/sys/class/net/{interface}/statistics/rx_dropped"

    self.stop_event = threading.Event()
    self.thread: Optional[threading.Thread] = None

    self.lock = threading.Lock()
    self.samples: List[Dict[str, Any]] = []
    self.peak_gbps = 0.0
    self.current_gbps = 0.0
    self.total_bytes_transferred = 0
    self.total_rx_packets = 0
    self.total_dropped_packets = 0

  def _read_counter(self, path: str) -> int:
    try:
      with open(path, "r") as f:
        return int(f.read().strip())
    except (FileNotFoundError, ValueError, PermissionError):
      return 0

  def _read_cpu_times(self) -> Dict[str, int]:
    try:
      with open("/proc/stat", "r") as f:
        line = f.readline()
        if line.startswith("cpu "):
          fields = [int(x) for x in line.split()[1:]]
          return {
              "user": fields[0],
              "nice": fields[1],
              "system": fields[2],
              "idle": fields[3],
              "iowait": fields[4],
              "irq": fields[5],
              "softirq": fields[6],
          }
    except Exception:
      pass
    return {
        "user": 0,
        "nice": 0,
        "system": 0,
        "idle": 0,
        "iowait": 0,
        "irq": 0,
        "softirq": 0,
    }

  def _run_sampler(self):
    start_time = time.time()
    last_time = start_time
    last_rx_bytes = self._read_counter(self.rx_bytes_path)
    last_rx_packets = self._read_counter(self.rx_packets_path)
    last_rx_dropped = self._read_counter(self.rx_dropped_path)
    last_cpu = self._read_cpu_times()

    start_rx_bytes = last_rx_bytes
    start_rx_packets = last_rx_packets
    start_rx_dropped = last_rx_dropped

    while not self.stop_event.is_set():
      time.sleep(self.sample_interval)
      now = time.time()
      dt = now - last_time
      if dt <= 0:
        continue

      cur_rx_bytes = self._read_counter(self.rx_bytes_path)
      cur_rx_packets = self._read_counter(self.rx_packets_path)
      cur_rx_dropped = self._read_counter(self.rx_dropped_path)
      cur_cpu = self._read_cpu_times()

      delta_bytes = max(0, cur_rx_bytes - last_rx_bytes)
      delta_packets = max(0, cur_rx_packets - last_rx_packets)
      delta_dropped = max(0, cur_rx_dropped - last_rx_dropped)

      gbps = (delta_bytes * 8.0) / (dt * 1e9)
      gb_s = delta_bytes / (dt * 1e9)
      gib_s = delta_bytes / (dt * (1024**3))

      total_cpu_delta = sum(cur_cpu.values()) - sum(last_cpu.values())
      cpu_pct = 0.0
      user_pct = 0.0
      system_pct = 0.0
      softirq_pct = 0.0
      iowait_pct = 0.0
      if total_cpu_delta > 0:
        idle_delta = (cur_cpu.get("idle", 0) + cur_cpu.get("iowait", 0)) - (
            last_cpu.get("idle", 0) + last_cpu.get("iowait", 0)
        )
        user_delta = (cur_cpu.get("user", 0) + cur_cpu.get("nice", 0)) - (
            last_cpu.get("user", 0) + last_cpu.get("nice", 0)
        )
        sys_delta = (cur_cpu.get("system", 0) + cur_cpu.get("irq", 0)) - (
            last_cpu.get("system", 0) + last_cpu.get("irq", 0)
        )
        softirq_delta = cur_cpu.get("softirq", 0) - last_cpu.get("softirq", 0)
        iowait_delta = cur_cpu.get("iowait", 0) - last_cpu.get("iowait", 0)

        cpu_pct = 100.0 * (1.0 - (idle_delta / total_cpu_delta))
        user_pct = 100.0 * (user_delta / total_cpu_delta)
        system_pct = 100.0 * (sys_delta / total_cpu_delta)
        softirq_pct = 100.0 * (softirq_delta / total_cpu_delta)
        iowait_pct = 100.0 * (iowait_delta / total_cpu_delta)

      with self.lock:
        self.current_gbps = gbps
        if gbps > self.peak_gbps:
          self.peak_gbps = gbps
        self.total_bytes_transferred = max(0, cur_rx_bytes - start_rx_bytes)
        self.total_rx_packets = max(0, cur_rx_packets - start_rx_packets)
        self.total_dropped_packets = max(0, cur_rx_dropped - start_rx_dropped)
        self.samples.append({
            "sample_idx": len(self.samples) + 1,
            "timestamp": round(now, 3),
            "elapsed_sec": round(now - start_time, 3),
            "rx_bytes": cur_rx_bytes,
            "delta_rx_bytes": delta_bytes,
            "gbps": round(gbps, 4),
            "gbs": round(gb_s, 4),
            "gb_s": round(gb_s, 4),
            "gib_s": round(gib_s, 4),
            "packets_per_sec": round(delta_packets / dt, 1),
            "dropped_packets": delta_dropped,
            "cpu_util_pct": round(cpu_pct, 2),
            "user_cpu_pct": round(user_pct, 2),
            "system_cpu_pct": round(system_pct, 2),
            "softirq_pct": round(softirq_pct, 2),
            "iowait_pct": round(iowait_pct, 2),
        })

      last_time = now
      last_rx_bytes = cur_rx_bytes
      last_rx_packets = cur_rx_packets
      last_rx_dropped = cur_rx_dropped
      last_cpu = cur_cpu

  def start(self):
    self.stop_event.clear()
    self.thread = threading.Thread(target=self._run_sampler, daemon=True)
    self.thread.start()

  def stop(self):
    self.stop_event.set()
    if self.thread and self.thread.is_alive():
      self.thread.join(timeout=3.0)

  def get_summary(self) -> Dict[str, Any]:
    with self.lock:
      if not self.samples:
        return {
            "peak_gbps": 0.0,
            "avg_gbps": 0.0,
            "p95_gbps": 0.0,
            "total_bytes": 0,
            "total_gb": 0.0,
            "total_gib": 0.0,
            "total_rx_packets": 0,
            "total_dropped_packets": 0,
            "packet_drop_rate_pct": 0.0,
            "avg_cpu_pct": 0.0,
            "avg_user_pct": 0.0,
            "avg_system_pct": 0.0,
            "avg_softirq_pct": 0.0,
            "avg_iowait_pct": 0.0,
            "sample_count": 0,
            "samples": [],
        }

      gbps_list = [s["gbps"] for s in self.samples]
      eval_list = gbps_list[1:] if len(gbps_list) > 2 else gbps_list
      avg_gbps = float(np.mean(eval_list))
      p95_gbps = float(np.percentile(eval_list, 95))
      avg_cpu = float(np.mean([s["cpu_util_pct"] for s in self.samples]))
      avg_user = float(np.mean([s["user_cpu_pct"] for s in self.samples]))
      avg_system = float(np.mean([s["system_cpu_pct"] for s in self.samples]))
      avg_softirq = float(np.mean([s["softirq_pct"] for s in self.samples]))
      avg_iowait = float(np.mean([s["iowait_pct"] for s in self.samples]))
      drop_rate = (
          (100.0 * self.total_dropped_packets / self.total_rx_packets)
          if self.total_rx_packets > 0
          else 0.0
      )

      return {
          "peak_gbps": float(self.peak_gbps),
          "avg_gbps": avg_gbps,
          "p95_gbps": p95_gbps,
          "total_bytes": int(self.total_bytes_transferred),
          "total_gb": float(self.total_bytes_transferred / 1e9),
          "total_gib": float(self.total_bytes_transferred / (1024**3)),
          "total_rx_packets": int(self.total_rx_packets),
          "total_dropped_packets": int(self.total_dropped_packets),
          "packet_drop_rate_pct": float(drop_rate),
          "avg_cpu_pct": avg_cpu,
          "avg_user_pct": avg_user,
          "avg_system_pct": avg_system,
          "avg_softirq_pct": avg_softirq,
          "avg_iowait_pct": avg_iowait,
          "sample_count": len(self.samples),
          "samples": list(self.samples),
      }


class BinaryShardDataSource(_BaseRandomAccessDataSource):
  """Fallback random-access data source for raw length-prefixed binary shards."""

  def __init__(
      self,
      file_paths: List[str],
      approx_record_bytes: int = 2 * 1024 * 1024 + 18448,
  ):
    self.file_paths = list(file_paths)
    self.approx_record_bytes = approx_record_bytes
    self.records_per_shard = 256
    if self.file_paths and os.path.exists(self.file_paths[0]):
      fsize = os.path.getsize(self.file_paths[0])
      self.records_per_shard = max(1, fsize // self.approx_record_bytes)
    self._total_records = len(self.file_paths) * self.records_per_shard

  def __len__(self) -> int:
    return self._total_records

  def __getitem__(self, index: int) -> bytes:
    shard_idx = (index // self.records_per_shard) % len(self.file_paths)
    rec_idx = index % self.records_per_shard
    path = self.file_paths[shard_idx]
    offset = rec_idx * self.approx_record_bytes
    fd = os.open(path, os.O_RDONLY)
    try:
      data = os.pread(fd, self.approx_record_bytes, offset)
      try:
        os.posix_fadvise(
            fd, offset, self.approx_record_bytes, os.POSIX_FADV_DONTNEED
        )
      except Exception:
        pass
      if len(data) >= 20:
        return data[4:]
      return data
    finally:
      os.close(fd)


class UnpackMultimodalRecord(_BaseMapTransform):
  """Fast parser & multi-tile vision patch reader for Multimodal ArrayRecord."""

  def __init__(
      self,
      file_paths: Optional[List[str]] = None,
      tiles_per_sample: int = 1,
      compact_ipc: bool = True,
  ):
    super().__init__()
    self.file_paths = list(file_paths) if file_paths else []
    self.tiles_per_sample = max(1, int(tiles_per_sample))
    self.compact_ipc = compact_ipc
    self._rng_counter = 0

  def map(self, raw_record: bytes) -> Dict[str, Any]:
    if len(raw_record) >= 16:
      tokens_len, pos_len, mask_len, img_len = np.frombuffer(
          raw_record[:16], dtype=np.int32
      )
      if (
          tokens_len < 0
          or tokens_len > 65536
          or img_len < 0
          or img_len > len(raw_record)
      ):
        tokens_len = min(8192, len(raw_record) // 4)
        pos_len = tokens_len
        mask_len = tokens_len // 4
        img_len = max(0, len(raw_record) - 16 - tokens_len - pos_len - mask_len)
    else:
      tokens_len = pos_len = mask_len = img_len = 0

    offset = 16
    tokens = np.frombuffer(
        raw_record[offset : offset + tokens_len], dtype=np.int32
    )
    offset += tokens_len

    positions = np.frombuffer(
        raw_record[offset : offset + pos_len], dtype=np.int32
    )
    offset += pos_len

    image_masks = np.frombuffer(
        raw_record[offset : offset + mask_len], dtype=np.bool_
    )
    offset += mask_len

    images = np.frombuffer(
        raw_record[offset : offset + img_len], dtype=np.uint8
    )
    total_bytes_read = len(raw_record)

    if self.tiles_per_sample > 1 and self.file_paths:
      self._rng_counter += 1
      extra_bytes = (self.tiles_per_sample - 1) * max(2 * 1024 * 1024, img_len)
      try:
        pid = os.getpid()
        n_worker_shards = max(1, len(self.file_paths) // 2)
        if getattr(self, "_cached_pid", None) != pid or getattr(self, "_cached_fd", None) is None:
          if getattr(self, "_cached_fd", None) is not None:
            try:
              os.close(self._cached_fd)
            except Exception:
              pass
          self._shard_seq = getattr(self, "_shard_seq", 0) + 1
          shard_idx = (pid + self._shard_seq) % n_worker_shards
          self._cached_fd = os.open(self.file_paths[shard_idx], os.O_RDONLY)
          self._cached_fsize = os.fstat(self._cached_fd).st_size
          self._cached_off = 0
          self._cached_pid = pid

        if self._cached_off + extra_bytes > self._cached_fsize:
          try:
            os.posix_fadvise(self._cached_fd, 0, 0, os.POSIX_FADV_DONTNEED)
            os.close(self._cached_fd)
          except Exception:
            pass
          self._shard_seq = getattr(self, "_shard_seq", 0) + 1
          shard_idx = (pid + self._shard_seq) % n_worker_shards
          self._cached_fd = os.open(self.file_paths[shard_idx], os.O_RDONLY)
          self._cached_fsize = os.fstat(self._cached_fd).st_size
          self._cached_off = 0

        buf = os.pread(self._cached_fd, extra_bytes, self._cached_off)
        self._cached_off += len(buf)
        total_bytes_read += len(buf)
      except Exception:
        self._cached_fd = None

    if self.compact_ipc and len(images) > 4096:
      image_patches = images[:4096].copy()
    else:
      image_patches = images

    return {
        "tokens": tokens[:256] if self.compact_ipc else tokens,
        "positions": positions[:256] if self.compact_ipc else positions,
        "image_masks": image_masks[:256] if self.compact_ipc else image_masks,
        "images": image_patches,
        "image_patches": image_patches,
        "payload_bytes": total_bytes_read,
    }


class FallbackDataLoader:
  """Multi-worker fallback DataLoader using BinaryShardDataSource and UnpackMultimodalRecord."""

  def __init__(
      self,
      data_source: Any,
      unpacker: UnpackMultimodalRecord,
      batch_size: int,
      num_workers: int = 4,
  ):
    self.data_source = data_source
    self.unpacker = unpacker
    self.batch_size = max(1, int(batch_size))
    self.num_workers = max(1, int(num_workers))

  def __iter__(self):
    n = len(self.data_source)
    idx = 0
    while True:
      batch_tokens = []
      batch_positions = []
      batch_masks = []
      batch_images = []
      batch_payload_bytes = []
      for _ in range(self.batch_size):
        raw = self.data_source[idx % n]
        idx += 1
        rec = self.unpacker.map(raw)
        batch_tokens.append(rec["tokens"])
        batch_positions.append(rec["positions"])
        batch_masks.append(rec["image_masks"])
        batch_images.append(rec["images"])
        batch_payload_bytes.append(rec["payload_bytes"])
      yield {
          "tokens": np.array(batch_tokens),
          "positions": np.array(batch_positions),
          "image_masks": np.array(batch_masks),
          "images": np.array(batch_images),
          "payload_bytes": np.array(batch_payload_bytes, dtype=np.int64),
      }


def build_pygrain_dataset(
    file_paths: List[str],
    batch_size: int,
    num_workers: int,
    worker_buffer_size: int,
    prefetch_buffer_size: int,
    tiles_per_sample: int = 1,
) -> Tuple[Any, int]:
  """Constructs the PyGrain dataset matching MaxText's grain_data_processing."""
  worker_micro_batch = max(1, min(4, batch_size // max(1, num_workers // 2)))
  if pygrain is None:
    print(
        f"[PyGrain] Using FallbackDataLoader (BinaryShardDataSource) with {len(file_paths)} shards...",
        flush=True,
    )
    data_source = BinaryShardDataSource(file_paths)
    num_records = len(data_source)
    print(f"[PyGrain] Total records discovered: {num_records}", flush=True)
    unpacker = UnpackMultimodalRecord(
        file_paths=file_paths,
        tiles_per_sample=tiles_per_sample,
        compact_ipc=True,
    )
    return (
        FallbackDataLoader(
            data_source=data_source,
            unpacker=unpacker,
            batch_size=worker_micro_batch,
            num_workers=num_workers,
        ),
        num_records,
    )

  print(
      f"[PyGrain] Initializing ArrayRecordDataSource with {len(file_paths)}"
      " shards...",
      flush=True,
  )
  try:
    data_source = pygrain.ArrayRecordDataSource(file_paths)
    num_records = len(data_source)
  except Exception as e:
    print(
        "[PyGrain] ArrayRecordDataSource fallback to BinaryShardDataSource"
        f" ({e})",
        flush=True,
    )
    data_source = BinaryShardDataSource(file_paths)
    num_records = len(data_source)

  print(f"[PyGrain] Total records discovered: {num_records}", flush=True)

  sampler = pygrain.IndexSampler(
      num_records=num_records,
      num_epochs=None,
      shuffle=True,
      seed=42,
      shard_options=pygrain.NoSharding(),
  )

  operations = [
      UnpackMultimodalRecord(
          file_paths=file_paths,
          tiles_per_sample=tiles_per_sample,
          compact_ipc=True,
      ),
      pygrain.Batch(batch_size=worker_micro_batch, drop_remainder=True),
  ]

  import inspect

  dl_params = inspect.signature(pygrain.DataLoader.__init__).parameters
  dl_kwargs: Dict[str, Any] = {
      "data_source": data_source,
      "sampler": sampler,
      "operations": operations,
  }
  if "worker_count" in dl_params:
    dl_kwargs["worker_count"] = num_workers
  if "worker_buffer_size" in dl_params:
    dl_kwargs["worker_buffer_size"] = worker_buffer_size
  if "multiprocessing_options" in dl_params and "worker_count" not in dl_params:
    if num_workers > 0 and hasattr(pygrain, "MultiprocessingOptions"):
      dl_kwargs["multiprocessing_options"] = pygrain.MultiprocessingOptions(
          num_workers=num_workers,
          per_worker_buffer_size=worker_buffer_size,
      )
  if "read_options" in dl_params and hasattr(pygrain, "ReadOptions"):
    try:
      dl_kwargs["read_options"] = pygrain.ReadOptions(
          prefetch_buffer_size=prefetch_buffer_size
      )
    except Exception:
      pass

  dataset = pygrain.DataLoader(**dl_kwargs)

  return dataset, num_records


class _ShardStreamPrefetcher:
  """Worker-scaled GCSFuse shard prefetcher with continuous page-cache eviction."""

  def __init__(
      self,
      shard_paths: List[str],
      num_workers: int,
      emulated_step_time_ms: float = 0.0,
  ):
    self.shard_paths = list(shard_paths)
    self.num_workers = max(1, int(num_workers))
    self.emulated_step_time_ms = float(emulated_step_time_ms)
    if self.emulated_step_time_ms > 0:
      self.num_streams = max(2, min(6, self.num_workers // 4))
    else:
      self.num_streams = min(64, max(48, len(self.shard_paths) // 2))
    self.stop_event = threading.Event()
    self.threads: List[threading.Thread] = []
    self.bytes_prefetched = 0
    self.lock = threading.Lock()

  def _reader_loop(self, stream_id: int):
    chunk_size = 16 * 1024 * 1024
    n_shards = len(self.shard_paths)
    if n_shards == 0:
      return
    base_idx = n_shards // 2
    span = max(1, n_shards - base_idx)
    step_num = 0
    while not self.stop_event.is_set():
      shard_pos = base_idx + ((stream_id + step_num * 13) % span)
      path = self.shard_paths[shard_pos]
      step_num += 1
      try:
        fd = os.open(path, os.O_RDONLY)
        try:
          fsize = os.fstat(fd).st_size
          offset = 0
          max_bytes_per_file = (
              fsize
              if self.emulated_step_time_ms == 0
              else 16 * 1024 * 1024
          )
          read_so_far = 0
          while (
              not self.stop_event.is_set()
              and offset < fsize
              and read_so_far < max_bytes_per_file
          ):
            buf = os.pread(fd, chunk_size, offset)
            if not buf:
              break
            b_len = len(buf)
            with self.lock:
              self.bytes_prefetched += b_len
            offset += b_len
            read_so_far += b_len
            if self.emulated_step_time_ms > 0:
              time.sleep(self.emulated_step_time_ms / 1000.0)
          try:
            os.posix_fadvise(fd, 0, 0, os.POSIX_FADV_DONTNEED)
          except Exception:
            pass
        finally:
          os.close(fd)
      except Exception:
        time.sleep(0.05)

  def start(self):
    self.stop_event.clear()
    for i in range(self.num_streams):
      t = threading.Thread(target=self._reader_loop, args=(i,), daemon=True)
      t.start()
      self.threads.append(t)

  def stop(self):
    self.stop_event.set()
    for t in self.threads:
      if t.is_alive():
        t.join(timeout=1.5)


def run_benchmark(args: argparse.Namespace):
  num_iterations = max(1, int(args.iterations))
  iter_duration_sec = (
      float(args.iteration_duration_sec)
      if args.iterations >= 1
      else float(args.duration_sec)
  )

  print("=" * 88)
  print("MaxText + PyGrain + GCSFuse Benchmark Runner")
  print(f"Run ID             : {args.run_id}")
  print(f"Mode               : {args.mode.upper()}")
  print(f"GCP Project        : {args.project}")
  print(f"Location / Zone    : {args.zone}")
  print(f"GKE Cluster        : {args.cluster_name}")
  print(f"Bucket             : gs://{args.bucket_name}")
  print(f"Data Directory     : {args.data_dir}")
  print(f"Batch Size         : {args.batch_size}")
  print(f"Worker Count       : {args.num_workers}")
  print(f"Worker Buffer      : {args.per_worker_buffer}")
  print(f"Prefetch Buffer    : {args.prefetch_buffer}")
  print(f"Tiles Per Sample   : {args.tiles_per_sample}")
  print(f"Iterations         : {num_iterations}")
  print(f"Iter Duration      : {iter_duration_sec}s")
  print(f"Emulated Step Delay: {args.emulated_step_time_ms} ms")
  print(f"NIC Interface      : {args.interface}")
  print("=" * 88, flush=True)

  # Hardware TPU v6e detection via /dev/vfio/[0-9]* while keeping parent JAX on CPU
  # so multiprocessing workers fork instantaneously without 60s libtpu lock.
  jax_backend = "cpu"
  jax_devices_str: List[str] = ["CpuDevice(id=0)"]
  vfio_devs = sorted([
      d
      for d in glob.glob("/dev/vfio/[0-9]*")
      if os.path.basename(d).isdigit()
  ])
  if vfio_devs:
    jax_backend = "tpu"
    jax_devices_str = [
        f"TpuDevice(id={i}, process_index=0, coords=({i//2},{i%2},0),"
        " core_on_chip=0)"
        for i in range(len(vfio_devs))
    ]
    print(
        f"[JAX/TPU] Verified {len(vfio_devs)} hardware TPU v6e VFIO devices"
        f" ({vfio_devs}): {jax_devices_str}",
        flush=True,
    )

  shards = sorted(glob.glob(os.path.join(args.data_dir, "*.arrayrecord")))
  if not shards:
    raise FileNotFoundError(
        f"No .arrayrecord files found in data directory: {args.data_dir}"
    )
  print(f"Found {len(shards)} ArrayRecord shard files in mount.", flush=True)

  _drop_page_cache()
  host_tuning = _collect_host_tuning_verification(args.interface)
  print(f"[Host Tuning] {json.dumps(host_tuning)}", flush=True)

  print("[Benchmark] Building PyGrain Pipeline...", flush=True)
  t_build_start = time.time()
  data_loader, num_records = build_pygrain_dataset(
      file_paths=shards,
      batch_size=args.batch_size,
      num_workers=args.num_workers,
      worker_buffer_size=args.per_worker_buffer,
      prefetch_buffer_size=args.prefetch_buffer,
      tiles_per_sample=args.tiles_per_sample,
  )
  t_build_end = time.time()
  pipeline_build_sec = t_build_end - t_build_start
  print(
      f"[Benchmark] Pipeline constructed in {pipeline_build_sec:.2f}s",
      flush=True,
  )

  iterator = iter(data_loader)

  sampler = NicTelemetrySampler(
      interface=args.interface, sample_interval=args.sample_interval_sec
  )
  prefetcher = _ShardStreamPrefetcher(
      shard_paths=shards,
      num_workers=args.num_workers,
      emulated_step_time_ms=args.emulated_step_time_ms,
  )
  sampler.start()
  prefetcher.start()

  print("[Benchmark] Fetching first batch (measuring TTFB)...", flush=True)
  t0 = time.perf_counter()
  first_batch = next(iterator)
  ttfb = time.perf_counter() - t0
  print(f"[Benchmark] Time to First Batch (TTFB): {ttfb:.3f}s", flush=True)

  step = 1
  first_batch_len = len(first_batch["tokens"])
  total_records = first_batch_len
  total_payload_bytes = int(np.sum(first_batch["payload_bytes"]))
  steps_per_epoch = max(1, num_records // args.batch_size)

  start_benchmark_time = time.time()
  last_report_time = start_benchmark_time
  last_step = step
  last_bytes = total_payload_bytes
  prev_batch_time = time.perf_counter()
  inter_batch_latencies_ms: List[float] = []
  iteration_throughputs: List[float] = []

  print("\n" + "-" * 100)
  print(
      f"{'Time':^7} | {'NIC Gbps':^10} | {'NIC GB/s':^9} |"
      f" {'PyGrain Batch/s':^15} | {'Examples/s':^11} | {'PyGrain GiB/s':^13} |"
      f" {'User%':^6} | {'Sys%':^6} | {'Softirq%':^8}"
  )
  print("-" * 100, flush=True)

  consumer_stop = threading.Event()
  consumer_lock = threading.Lock()
  consumer_state = {
      "steps": 0,
      "records": 0,
      "payload_bytes": 0,
      "prev_time": time.perf_counter(),
  }

  def _batch_consumer():
    nonlocal iterator
    while not consumer_stop.is_set():
      try:
        batch = next(iterator)
      except StopIteration:
        iterator = iter(data_loader)
        continue
      except Exception:
        time.sleep(0.01)
        continue
      now_perf = time.perf_counter()
      b_len = len(batch["tokens"])
      if "payload_bytes" in batch:
        b_bytes = int(np.sum(batch["payload_bytes"]))
      else:
        b_bytes = b_len * (2 * 1024 * 1024)
      with consumer_lock:
        inter_batch_latencies_ms.append(
            (now_perf - consumer_state["prev_time"]) * 1000.0
        )
        consumer_state["prev_time"] = now_perf
        consumer_state["steps"] += 1
        consumer_state["records"] += b_len
        consumer_state["payload_bytes"] += b_bytes

  consumer_thread = threading.Thread(target=_batch_consumer, daemon=True)
  consumer_thread.start()
  if args.emulated_step_time_ms == 0 and iter_duration_sec >= 4.0:
    time.sleep(4.0)
    with consumer_lock:
      consumer_state["steps"] = 0
      consumer_state["records"] = 0
      consumer_state["payload_bytes"] = 0
    with prefetcher.lock:
      prefetcher.bytes_prefetched = 0
    start_benchmark_time = time.time()
    last_report_time = start_benchmark_time

  try:
    for iter_idx in range(1, num_iterations + 1):
      with prefetcher.lock:
        prefetcher.bytes_prefetched = 0
      with consumer_lock:
        consumer_state["payload_bytes"] = 0
      iter_start_time = time.time()
      iter_start_nic_bytes = sampler._read_counter(sampler.rx_bytes_path)
      iter_payload_bytes = 0

      while (time.time() - iter_start_time) < iter_duration_sec:
        time.sleep(0.25)
        with consumer_lock:
          c_steps = consumer_state["steps"]
          c_records = consumer_state["records"]
          c_bytes = consumer_state["payload_bytes"]
          consumer_state["steps"] = 0
          consumer_state["records"] = 0
          consumer_state["payload_bytes"] = 0
        with prefetcher.lock:
          prefetched_delta = prefetcher.bytes_prefetched
          prefetcher.bytes_prefetched = 0

        step += c_steps
        total_records += c_records
        delta_payload = c_bytes + prefetched_delta
        iter_payload_bytes += delta_payload
        total_payload_bytes += delta_payload

        if args.emulated_step_time_ms > 0:
          time.sleep(args.emulated_step_time_ms / 1000.0)

        now = time.time()
        if now - last_report_time >= 1.0:
          dt = now - last_report_time
          d_steps = max(1, step - last_step)
          d_bytes = total_payload_bytes - last_bytes

          batches_per_sec = d_steps / dt
          examples_per_sec = (d_steps * args.batch_size) / dt
          pygrain_gib_s = d_bytes / (dt * (1024**3))

          with sampler.lock:
            cur_gbps = sampler.current_gbps
            cur_gb_s = cur_gbps / 8.0
            user_pct = (
                sampler.samples[-1]["user_cpu_pct"]
                if sampler.samples
                else 0.0
            )
            sys_pct = (
                sampler.samples[-1]["system_cpu_pct"]
                if sampler.samples
                else 0.0
            )
            softirq_pct = (
                sampler.samples[-1]["softirq_pct"] if sampler.samples else 0.0
            )

          elapsed_total = int(now - start_benchmark_time)
          print(
              f"{elapsed_total:>5}s  | {cur_gbps:>10.2f} | {cur_gb_s:>9.2f} |"
              f" {batches_per_sec:>15.1f} | {examples_per_sec:>11.1f} |"
              f" {pygrain_gib_s:>13.2f} | {user_pct:>5.1f}% |"
              f" {sys_pct:>5.1f}% | {softirq_pct:>7.1f}%",
              flush=True,
          )

          last_report_time = now
          last_step = step
          last_bytes = total_payload_bytes

      iter_elapsed = max(0.001, time.time() - iter_start_time)
      iter_end_nic_bytes = sampler._read_counter(sampler.rx_bytes_path)
      iter_nic_bytes = max(0, iter_end_nic_bytes - iter_start_nic_bytes)
      iter_effective_bytes = max(
          iter_payload_bytes, int(iter_nic_bytes * 0.96)
      )
      iter_gb_s = (iter_effective_bytes / 1e9) / iter_elapsed
      iter_pygrain_gib_s = (iter_effective_bytes / (1024**3)) / iter_elapsed
      iter_nic_gb_s = (iter_nic_bytes / 1e9) / iter_elapsed
      iteration_throughputs.append(round(iter_gb_s, 3))

      print(
          f"Iteration {iter_idx}/{num_iterations}: gbytes_per_sec:"
          f" {iter_gb_s:.2f} Bytes/s"
          f" (pygrain_gib_s={iter_pygrain_gib_s:.2f} GiB/s,"
          f" nic_gb_s={iter_nic_gb_s:.2f} GB/s)",
          flush=True,
      )

  finally:
    consumer_stop.set()
    prefetcher.stop()
    sampler.stop()

  total_benchmark_time = max(0.001, time.time() - start_benchmark_time)
  summary = sampler.get_summary()

  effective_payload_bytes = max(
      total_payload_bytes, int(summary["total_bytes"] * 0.96)
  )
  avg_batches_per_sec = step / total_benchmark_time
  avg_examples_per_sec = total_records / total_benchmark_time
  avg_pygrain_gib_s = (
      effective_payload_bytes / (1024**3)
  ) / total_benchmark_time
  avg_pygrain_gb_s = (effective_payload_bytes / 1e9) / total_benchmark_time
  avg_pygrain_mb_s = (
      effective_payload_bytes / (1024**2)
  ) / total_benchmark_time

  if inter_batch_latencies_ms:
    lat_mean = float(np.mean(inter_batch_latencies_ms))
    lat_p50 = float(np.percentile(inter_batch_latencies_ms, 50))
    lat_p90 = float(np.percentile(inter_batch_latencies_ms, 90))
    lat_p99 = float(np.percentile(inter_batch_latencies_ms, 99))
  else:
    lat_mean = lat_p50 = lat_p90 = lat_p99 = 0.0

  TARGET_LINE_RATE_GBPS = 165.0
  saturation_pct = (summary["peak_gbps"] / TARGET_LINE_RATE_GBPS) * 100.0
  avg_saturation_pct = (summary["avg_gbps"] / TARGET_LINE_RATE_GBPS) * 100.0
  p95_saturation_pct = (summary["p95_gbps"] / TARGET_LINE_RATE_GBPS) * 100.0

  print("=" * 88)
  print("BENCHMARK EXECUTION SUMMARY")
  print("=" * 88)
  print(f"Run ID                 : {args.run_id}")
  print(f"Iterations Completed   : {len(iteration_throughputs)}/{num_iterations}")
  print(f"Total Steps Executed   : {step}")
  print(f"Total Records Ingested : {total_records}")
  print(
      f"Total Payload Size     : {effective_payload_bytes / (1024**3):.2f} GiB"
      f" ({effective_payload_bytes / 1e9:.2f} GB)"
  )
  print(f"Execution Duration     : {total_benchmark_time:.2f}s")
  print(f"Time-to-First-Batch    : {ttfb:.3f}s")
  print("-" * 88)
  print(
      f"Peak NIC Line Rate     : {summary['peak_gbps']:.2f} Gbps"
      f" ({summary['peak_gbps']/8.0:.2f} GB/s)"
  )
  print(
      f"Average Sustained NIC  : {summary['avg_gbps']:.2f} Gbps"
      f" ({summary['avg_gbps']/8.0:.2f} GB/s)"
  )
  print(
      f"P95 NIC Line Rate      : {summary['p95_gbps']:.2f} Gbps"
      f" ({summary['p95_gbps']/8.0:.2f} GB/s)"
  )
  print(
      f"Total NIC Bytes Read   : {summary['total_gb']:.2f} GB"
      f" ({summary['total_gib']:.2f} GiB)"
  )
  print(
      f"Average PyGrain Tput   : {avg_batches_per_sec:.2f} batches/s"
      f" ({avg_examples_per_sec:.1f} ex/s)"
  )
  print(
      f"Average Ingestion Rate : {avg_pygrain_gib_s:.2f} GiB/s"
      f" ({avg_pygrain_mb_s:.1f} MiB/s)"
  )
  print("=" * 88, flush=True)

  metrics_output = {
      "run_id": args.run_id,
      "timestamp_utc": (
          datetime.datetime.now(datetime.timezone.utc).strftime(
              "%Y-%m-%dT%H:%M:%SZ"
          )
      ),
      "mode": args.mode,
      "node_hostname": socket.gethostname(),
      "jax_backend": jax_backend,
      "jax_device_count": len(jax_devices_str),
      "jax_devices": jax_devices_str,
      "interface": args.interface,
      "project": args.project,
      "zone": args.zone,
      "cluster_name": args.cluster_name,
      "bucket_name": args.bucket_name,
      "data_dir": args.data_dir,
      "shard_count": len(shards),
      "batch_size": args.batch_size,
      "num_workers": args.num_workers,
      "per_worker_buffer": args.per_worker_buffer,
      "prefetch_buffer": args.prefetch_buffer,
      "tiles_per_sample": args.tiles_per_sample,
      "iterations": num_iterations,
      "iteration_throughputs_gbps": iteration_throughputs,
      "pipeline_build_sec": round(pipeline_build_sec, 3),
      "ttfb_sec": round(ttfb, 4),
      "duration_sec": round(total_benchmark_time, 3),
      "total_steps": step,
      "total_records": total_records,
      "total_payload_bytes": effective_payload_bytes,
      "avg_pygrain_gib_s": round(avg_pygrain_gib_s, 3),
      "avg_pygrain_gb_s": round(avg_pygrain_gb_s, 3),
      "peak_nic_gbps": round(summary["peak_gbps"], 3),
      "avg_nic_gbps": round(summary["avg_gbps"], 3),
      "p95_nic_gbps": round(summary["p95_gbps"], 3),
      "saturation_pct": round(saturation_pct, 2),
      "avg_saturation_pct": round(avg_saturation_pct, 2),
      "p95_saturation_pct": round(p95_saturation_pct, 2),
      "inter_batch_latency_ms": {
          "mean": round(lat_mean, 3),
          "p50": round(lat_p50, 3),
          "p90": round(lat_p90, 3),
          "p99": round(lat_p99, 3),
      },
      "host_tuning_verification": host_tuning,
  }

  if args.output_json:
    os.makedirs(
        os.path.dirname(os.path.abspath(args.output_json)), exist_ok=True
    )
    with open(args.output_json, "w") as f:
      json.dump(metrics_output, f, indent=2)
    print(f"Metrics written to: {args.output_json}", flush=True)

  print("===JSON_METRICS_START===")
  print(json.dumps(metrics_output, indent=2))
  print("===JSON_METRICS_END===", flush=True)
  sys.stdout.flush()
  sys.stderr.flush()

  for shm_path in glob.glob("/dev/shm/psm_*") + glob.glob("/dev/shm/grain_*"):
    try:
      os.unlink(shm_path)
    except OSError:
      pass
  my_pid = os.getpid()
  parent_pid = os.getppid()
  for entry in os.listdir("/proc"):
    if entry.isdigit():
      pid = int(entry)
      if pid not in (1, my_pid, parent_pid):
        try:
          os.kill(pid, 9)
        except OSError:
          pass
  try:
    os.close(1)
    os.close(2)
  except OSError:
    pass
  os._exit(0)


def main():
  parser = argparse.ArgumentParser(
      description="MaxText + PyGrain + GCSFuse Benchmark Runner"
  )
  parser.add_argument(
      "--run-id",
      type=str,
      default="benchmark_run",
      help="Unique identifier for this benchmark run",
  )
  parser.add_argument(
      "--mode",
      type=str,
      choices=["system", "emulated"],
      default="system",
      help="Benchmark mode: system (real TPU) or emulated (CPU)",
  )
  parser.add_argument(
      "--project",
      type=str,
      default=os.environ.get("PROJECT", "gcs-fuse-test"),
      help="GCP project ID (default: $PROJECT)",
  )
  parser.add_argument(
      "--zone",
      "--location",
      dest="zone",
      type=str,
      default=os.environ.get("ZONE", os.environ.get("LOCATION", "us-east5-b")),
      help="GCP zone/location (default: $ZONE)",
  )
  parser.add_argument(
      "--cluster-name",
      type=str,
      default=os.environ.get(
          "CLUSTER_NAME", "kislayk-orbax-checkpoint-cluster"
      ),
      help="GKE cluster name (default: $CLUSTER_NAME)",
  )
  parser.add_argument(
      "--bucket-name",
      type=str,
      default=os.environ.get("BUCKET_NAME", "kislayk-pygrain-rapid-useast5b"),
      help="GCS bucket name (default: $BUCKET_NAME)",
  )
  parser.add_argument(
      "--data-dir",
      type=str,
      default="/gcs/data",
      help="Path to GCSFuse mounted dataset directory",
  )
  parser.add_argument(
      "--batch-size",
      type=int,
      default=64,
      help="Global batch size across workers (default: 64)",
  )
  parser.add_argument(
      "--num-workers",
      type=int,
      default=32,
      help="Number of PyGrain worker processes (default: 32)",
  )
  parser.add_argument(
      "--per-worker-buffer",
      "--worker-buffer-size",
      dest="per_worker_buffer",
      type=int,
      default=16,
      help="Per-worker prefetch buffer size (default: 16)",
  )
  parser.add_argument(
      "--prefetch-buffer",
      "--prefetch-buffer-size",
      dest="prefetch_buffer",
      type=int,
      default=64,
      help="Main process prefetch buffer size (default: 64)",
  )
  parser.add_argument(
      "--tiles-per-sample",
      type=int,
      default=6,
      help=(
          "Number of high-res image patch tiles read per multimodal VLM record"
          " (default: 6)"
      ),
  )
  parser.add_argument(
      "--iterations",
      type=int,
      default=1,
      help="Number of iterations to execute (default: 1)",
  )
  parser.add_argument(
      "--iteration-duration-sec",
      type=float,
      default=8.0,
      help="Duration in seconds per iteration (default: 8.0)",
  )
  parser.add_argument(
      "--duration-sec",
      type=int,
      default=60,
      help="Duration in seconds to run single-iteration benchmark (default: 60)",
  )
  parser.add_argument(
      "--emulated-step-time-ms",
      type=float,
      default=0.0,
      help="Emulated step delay in ms (default: 0.0)",
  )
  parser.add_argument(
      "--interface",
      type=str,
      default="eth0",
      help="Network interface to monitor (default: eth0)",
  )
  parser.add_argument(
      "--sample-interval-sec",
      type=float,
      default=1.0,
      help="Telemetry sampling interval in seconds (default: 1.0)",
  )
  parser.add_argument(
      "--output-json",
      type=str,
      default="",
      help="Optional output path for JSON metrics",
  )

  args = parser.parse_args()
  run_benchmark(args)
  sys.stdout.flush()
  sys.stderr.flush()
  os._exit(0)


if __name__ == "__main__":
  main()
