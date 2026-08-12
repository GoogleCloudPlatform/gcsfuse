#!/usr/bin/env python3
import os
import sys
import time
import argparse
import statistics
import concurrent.futures
import tempfile
import uuid
import shutil
import subprocess

DATA_4KB = os.urandom(4096)
DATA_64KB = os.urandom(65536)
DATA_1MB = os.urandom(1024 * 1024)

def run_cmd(cmd):
    subprocess.run(cmd, shell=True, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

def drop_caches():
    try:
        subprocess.run("sync", shell=True, check=True)
        subprocess.run("sudo sh -c 'echo 3 > /proc/sys/vm/drop_caches'", shell=True, check=True, stderr=subprocess.DEVNULL)
    except subprocess.CalledProcessError:
        print("Warning: Could not drop caches.", file=sys.stderr)

def get_stats(times):
    if not times:
        return 0, 0
    times.sort()
    median = statistics.median(times)
    p95 = times[int(len(times) * 0.95)] if len(times) >= 20 else max(times)
    return median, p95

def op_small_write(filename):
    with open(filename, 'wb') as f:
        f.write(DATA_4KB)

def op_small_read(filename):
    with open(filename, 'rb') as f:
        f.read()

def gcloud_upload_files(bucket_url, prefix, num_files, size_bytes):
    with tempfile.TemporaryDirectory() as tmpdir:
        sample = os.path.join(tmpdir, "sample.bin")
        with open(sample, "wb") as f:
            f.write(os.urandom(size_bytes))
        
        upload_dir = os.path.join(tmpdir, "upload")
        os.makedirs(upload_dir)
        for i in range(num_files):
            shutil.copy(sample, os.path.join(upload_dir, f"file_{i}_{uuid.uuid4().hex}.bin"))
            
        print(f"  [gcloud] Uploading {num_files} files of size {size_bytes}B to {bucket_url}/{prefix}/")
        run_cmd(f"gcloud storage cp -r {upload_dir}/* {bucket_url}/{prefix}/")
        return [f for f in os.listdir(upload_dir)]

def measure_small_ops(mnt_dir, bucket_url, test_id):
    print("--- Small Ops Latency (1 thread, n=100) ---")
    n = 100
    write_times = []
    
    test_dir = os.path.join(mnt_dir, test_id, "small_writes")
    os.makedirs(test_dir, exist_ok=True)
    files = [os.path.join(test_dir, f"small_{i}_{uuid.uuid4().hex}.bin") for i in range(n)]
    for f in files:
        start = time.perf_counter()
        op_small_write(f)
        write_times.append((time.perf_counter() - start) * 1000)
    med, p95 = get_stats(write_times)
    print(f"Small Write (4KB)     : Median {med:7.2f} ms | p95 {p95:7.2f} ms")
    
    prefix = f"{test_id}/small_reads"
    os.makedirs(os.path.join(mnt_dir, prefix), exist_ok=True)
    
    read_file_names = gcloud_upload_files(bucket_url, prefix, n, 4096)
    read_files = [os.path.join(mnt_dir, prefix, fname) for fname in read_file_names]
    
    read_times = []
    for f in read_files:
        start = time.perf_counter()
        op_small_read(f)
        read_times.append((time.perf_counter() - start) * 1000)
    med, p95 = get_stats(read_times)
    print(f"Small Read (4KB)      : Median {med:7.2f} ms | p95 {p95:7.2f} ms")

def measure_throughput_concurrency(mnt_dir, bucket_url, test_id):
    print("\n--- Small Ops Concurrency (1 vs 8 threads, n=100) ---")
    n = 100
    
    test_dir = os.path.join(mnt_dir, test_id, "concurrency")
    os.makedirs(test_dir, exist_ok=True)
    
    new_files_1t = [os.path.join(test_dir, f"sw_1t_{i}_{uuid.uuid4().hex}.bin") for i in range(n)]
    start = time.perf_counter()
    for f in new_files_1t:
        op_small_write(f)
    dur_1t_write = time.perf_counter() - start
    
    new_files_8t = [os.path.join(test_dir, f"sw_8t_{i}_{uuid.uuid4().hex}.bin") for i in range(n)]
    start = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as ex:
        list(ex.map(op_small_write, new_files_8t))
    dur_8t_write = time.perf_counter() - start

    prefix_1t = f"{test_id}/read_1t"
    os.makedirs(os.path.join(mnt_dir, prefix_1t), exist_ok=True)
    fnames_1t = gcloud_upload_files(bucket_url, prefix_1t, n, 4096)
    read_files_1t = [os.path.join(mnt_dir, prefix_1t, fname) for fname in fnames_1t]
    
    prefix_8t = f"{test_id}/read_8t"
    os.makedirs(os.path.join(mnt_dir, prefix_8t), exist_ok=True)
    fnames_8t = gcloud_upload_files(bucket_url, prefix_8t, n, 4096)
    read_files_8t = [os.path.join(mnt_dir, prefix_8t, fname) for fname in fnames_8t]

    start = time.perf_counter()
    for f in read_files_1t:
        op_small_read(f)
    dur_1t_read = time.perf_counter() - start
    
    start = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as ex:
        list(ex.map(op_small_read, read_files_8t))
    dur_8t_read = time.perf_counter() - start

    print(f"Write 1 thread  : {n/dur_1t_write:6.1f} files/s ({(n*4096/1024/1024)/dur_1t_write:5.3f} MB/s)")
    print(f"Write 8 threads : {n/dur_8t_write:6.1f} files/s ({(n*4096/1024/1024)/dur_8t_write:5.3f} MB/s) -> {dur_1t_write/dur_8t_write:.1f}x scaling")
    print(f"Read 1 thread   : {n/dur_1t_read:6.1f} files/s ({(n*4096/1024/1024)/dur_1t_read:5.3f} MB/s)")
    print(f"Read 8 threads  : {n/dur_8t_read:6.1f} files/s ({(n*4096/1024/1024)/dur_8t_read:5.3f} MB/s) -> {dur_1t_read/dur_8t_read:.1f}x scaling")

def measure_append_patterns(mnt_dir, test_id):
    print("\n--- Append Patterns ---")
    test_dir = os.path.join(mnt_dir, test_id, "appends")
    os.makedirs(test_dir, exist_ok=True)
    
    f_cpr = os.path.join(test_dir, f"append_cpr_{uuid.uuid4().hex}.bin")
    cpr_times = []
    open(f_cpr, 'wb').close() 
    for _ in range(100):
        start = time.perf_counter()
        with open(f_cpr, 'ab') as f:
            f.write(DATA_64KB)
        cpr_times.append((time.perf_counter() - start) * 1000)
    med, p95 = get_stats(cpr_times)
    print(f"Close-per-record (64KB) : Median {med:7.2f} ms | p95 {p95:7.2f} ms (n=100)")
    
    f_stream = os.path.join(test_dir, f"append_stream_{uuid.uuid4().hex}.bin")
    stream_times = []
    with open(f_stream, 'wb') as f:
        for _ in range(1000):
            start = time.perf_counter()
            f.write(DATA_64KB)
            stream_times.append((time.perf_counter() - start) * 1000)
        start_close = time.perf_counter()
    close_time = (time.perf_counter() - start_close) * 1000
    med, p95 = get_stats(stream_times)
    print(f"Streaming (64KB/write)  : Median {med:7.3f} ms | p95 {p95:7.3f} ms (n=1000) | Finalize(Close): {close_time:.1f} ms")

    f_1gb = os.path.join(test_dir, f"append_1gb_{uuid.uuid4().hex}.bin")
    print(f"Creating 1GB file for append test (this will take a moment)...")
    with open(f_1gb, 'wb') as f:
        for _ in range(1024):
            f.write(DATA_1MB)
            
    append_1gb_times = []
    for _ in range(5):
        start = time.perf_counter()
        with open(f_1gb, 'ab') as f:
            f.write(DATA_64KB)
        append_1gb_times.append((time.perf_counter() - start) * 1000)
    med, p95 = get_stats(append_1gb_times)
    print(f"Append to 1GB file      : Median {med:7.1f} ms | p95 {p95:7.1f} ms (n=5)")
    return f_1gb

def measure_large_seq(mnt_dir, bucket_url, test_id):
    print("\n--- Large Sequential Throughput ---")
    write_mbps = []
    test_dir = os.path.join(mnt_dir, test_id, "large_seq")
    os.makedirs(test_dir, exist_ok=True)
    
    for i in range(30):
        f_large = os.path.join(test_dir, f"large_write_{i}_{uuid.uuid4().hex}.bin")
        start = time.perf_counter()
        with open(f_large, 'wb') as f:
            for _ in range(256):
                f.write(DATA_1MB)
        dur = time.perf_counter() - start
        write_mbps.append(256 / dur)
        os.remove(f_large)
    print(f"Large Write (256 MB)    : Median {statistics.median(write_mbps):6.1f} MB/s (n=30)")
    
    prefix = f"{test_id}/large_read"
    os.makedirs(os.path.join(mnt_dir, prefix), exist_ok=True)
    print(f"  [gcloud] Creating 1GB file for Large Read Cold test...")
    with tempfile.TemporaryDirectory() as tmpdir:
        local_1gb = os.path.join(tmpdir, "large_1gb.bin")
        run_cmd(f"dd if=/dev/urandom of={local_1gb} bs=1M count=1024")
        run_cmd(f"gcloud storage cp {local_1gb} {bucket_url}/{prefix}/large_1gb.bin")
        
    f_1gb = os.path.join(mnt_dir, prefix, "large_1gb.bin")
    
    read_mbps = []
    for i in range(30):
        drop_caches()
        start = time.perf_counter()
        with open(f_1gb, 'rb') as f:
            while f.read(1024 * 1024):
                pass
        dur = time.perf_counter() - start
        read_mbps.append(1024 / dur)
    print(f"Large Read Cold (1 GB)  : Median {statistics.median(read_mbps):6.1f} MB/s (n=30)")

def main():
    parser = argparse.ArgumentParser(description="Reproduce GCSFuse Stage 3 benchmark")
    parser.add_argument("mnt_dir", help="Path to the mounted GCSFuse directory")
    parser.add_argument("bucket_url", help="GS URL of the bucket (e.g. gs://my-bucket)")
    args = parser.parse_args()
    
    test_id = f"repro_stage3_{uuid.uuid4().hex}"
    
    try:
        print(f"Running benchmarks in: {args.mnt_dir}/{test_id}\n")
        measure_small_ops(args.mnt_dir, args.bucket_url, test_id)
        measure_throughput_concurrency(args.mnt_dir, args.bucket_url, test_id)
        measure_append_patterns(args.mnt_dir, test_id)
        measure_large_seq(args.mnt_dir, args.bucket_url, test_id)
    finally:
        print(f"\nCleaning up {args.bucket_url}/{test_id} using gcloud...")
        run_cmd(f"gcloud storage rm -r {args.bucket_url}/{test_id} || true")
        print("Done.")

if __name__ == "__main__":
    main()
