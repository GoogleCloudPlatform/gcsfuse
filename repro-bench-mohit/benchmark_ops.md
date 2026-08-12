# Benchmark Operations (Stage 3 Reproduction - GCloud Setup)

This document describes the exact operations executed by the `stage3_repro.py` benchmark script to measure GCSFuse latency and throughput under various workloads.

*To ensure true "cold" reads that accurately test the GCSFuse backend architecture rather than the local Linux page cache, the updated script creates all target files via `gcloud storage cp` directly on the GCS bucket before initiating the read tests.*

## 1. Small Operations Latency
*   **Small Write:** `open(O_CREAT|O_WRONLY|O_TRUNC) -> write(4KB) -> close()`. Creates one brand new, uniquely named 4KB file per operation. Repeated 100 times serially.
*   **Small Read:** Uses `gcloud` to upload a directory of 100 newly generated 4KB files directly to GCS. It then performs `open(O_RDONLY) -> read(4KB) -> close()` sequentially on those 100 files via the GCSFuse mount.

## 2. Concurrency Scaling
*   **Small Write (8 threads):** The same 4KB file creation loop, but distributed across a thread pool of 8 concurrent workers. 
*   **Small Read (8 threads):** Uses `gcloud` to upload a disjoint set of 100 new 4KB files directly to GCS. It then reads these 100 files via 8 concurrent threads through GCSFuse. 

## 3. Append Patterns
*   **Close-per-record (Log pattern):** `open(O_APPEND) -> write(64KB) -> close()`. Performed 100 times sequentially on a single growing file. Represents the worst-case scenario where each `close()` triggers a full object rewrite on standard GCS.
*   **Streaming (Held-open handle):** `open(O_APPEND) -> write(64KB) x 1000 -> close()`. A single stream executing 1000 sequential 64KB writes before finally closing. The script measures the median per-write time (in memory/buffer) and the final blocking `close()` finalize latency.
*   **Append to 1GB Object:** Appends a 64KB chunk to an already existing 1GB object and closes it. Repeated 5 times. This measures the extreme tail latency of a full 1GB read-modify-write operation on GCS.

## 4. Large Sequential Throughput
*   **Large Write (256MB):** Sequentially writes out a 256MB file and closes it. Measures upload throughput (MB/s). Repeated 30 times.
*   **Large Read Cold (1GB):** Uses `gcloud` to upload a brand new 1GB file to GCS. Before each of the 30 read iterations, the script executes `sync; echo 3 > /proc/sys/vm/drop_caches` to explicitly drop the Linux kernel page cache, guaranteeing a true "cold" read. Repeated 30 times.
## 5. Directory and File Structure

For each benchmark run, a unique test ID (e.g., `repro_stage3_<uuid>`) is generated. The files and directories are organized within the GCS bucket (and reflected in the GCSFuse mount) as follows:

```text
<bucket_root>/repro_stage3_<uuid>/
├── small_writes/      # Target directory for 1-thread small write latency tests
│   └── small_*.bin    # 100 files of 4KB each (created via GCSFuse)
├── small_reads/       # Target directory for 1-thread small read latency tests
│   └── file_*.bin     # 100 files of 4KB each (uploaded via gcloud)
├── concurrency/       # Target directory for concurrency write tests
│   ├── sw_1t_*.bin    # 100 files of 4KB each (created via GCSFuse in 1-thread mode)
│   └── sw_8t_*.bin    # 100 files of 4KB each (created via GCSFuse in 8-thread mode)
├── read_1t/           # Target directory for 1-thread read concurrency tests
│   └── file_*.bin     # 100 files of 4KB each (uploaded via gcloud)
├── read_8t/           # Target directory for 8-thread read concurrency tests
│   └── file_*.bin     # 100 files of 4KB each (uploaded via gcloud)
├── appends/           # Target directory for append pattern tests
│   ├── append_cpr_*.bin    # Close-per-record file (growing)
│   ├── append_stream_*.bin # Streaming append file (growing)
│   └── append_1gb_*.bin    # 1GB file used for append-to-1GB test
├── large_seq/         # Target directory for large sequential writes
│   └── large_write_*.bin   # 256MB files (created and deleted per iteration)
└── large_read/        # Target directory for large sequential cold reads
    └── large_1gb.bin  # 1GB file (uploaded via gcloud)
```

*Note: All paths are prefixed with the unique test ID to ensure isolated runs. The entire `repro_stage3_<uuid>/` prefix is deleted from GCS at the end of the benchmark run.*
