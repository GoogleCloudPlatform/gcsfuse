#!/usr/bin/env python3
"""
RCU (Rapid Cache Ultra) Micro-Benchmark Runner for GCSFuse.

Concurrency Rules (Identical numjobs for Write and Read):
- < 1 GB (128 KB, 100 MB): numjobs = 192
- >= 1 GB (1 GB, 10 GB):   numjobs = 144

Workloads (Direct 1:1 File Reuse):
1. 100 MB:
   - Sequential Write: 192 jobs x 10 files (BS=1M)
   - Sequential Read:  192 jobs x 10 files (BS=1M, reusing 100 MB files)
2. 10 GB:
   - Sequential Write: 144 jobs x 2 files  (BS=1M)
   - Sequential Read:  144 jobs x 2 files  (BS=1M, reusing 10 GB files)
3. 128 KB:
   - Sequential Write: 192 jobs x 30 files (BS=128k)
   - Sequential Read:  192 jobs x 30 files (BS=128k, reusing 128 KB files)
4. 1 GB:
   - Write:            144 jobs x 10 files (BS=1M)
   - Random Reads:     144 jobs x 10 files (BS: 4k, 64k, 1M, 4M, 16M; QD: 1, 4, reusing 1 GB files)
"""

import json
import os
import shutil
import subprocess
import sys
import time

# --- Target Configuration ---
BUCKET = "rcu-bench-uscentral1-a-fastbyte-avoidnull"
MOUNT_POINT = "/mnt/gcs"
ENDPOINT = "storage-preprod-test-grpc.googleusercontent.com:443"
BILLING_PROJECT = "gcs-fuse-test"

# --- Mount Flags (Rapid Path Always Enabled) ---
MOUNT_FLAGS = [
    "--experimental-enable-pirlo",
    "--enable-rapid-writes=true",
    "--client-protocol=grpc",
    f"--custom-endpoint={ENDPOINT}",
    f"--billing-project={BILLING_PROJECT}",
    "--implicit-dirs",
    "--metadata-cache-ttl-secs=-1",
    "--write-global-max-blocks=-1",
    "--log-file=/tmp/gcsfuse_bench.log",
]

# --- FIO Templates (Page Cache ON: direct=0, Uniform filename_format) ---
FIO_WRITE_TEMPLATE = """[global]
ioengine=libaio
direct=0
fadvise_hint=0
iodepth=1
verify=0
invalidate=1
file_append=0
create_on_open=1
end_fsync=1
thread=1
openfiles=1
group_reporting=1
allrandrepeat=1
filename_format=bench.$jobnum.$filenum.size-${FILESIZE}
rw=write

[write_job]
directory=${DIR}
filesize=${FILESIZE}
bs=${BS}
numjobs=${NUMJOBS}
nrfiles=${NRFILES}
"""

FIO_SEQ_READ_TEMPLATE = """[global]
ioengine=libaio
direct=0
fadvise_hint=0
iodepth=1
invalidate=1
thread=1
openfiles=1
group_reporting=1
create_serialize=0
allrandrepeat=0
file_service_type=random
rw=read
filename_format=bench.$jobnum.$filenum.size-${FILESIZE}

[seq_read_job]
directory=${DIR}
filesize=${FILESIZE}
bs=${BS}
numjobs=${NUMJOBS}
nrfiles=${NRFILES}
"""

FIO_RAND_READ_TEMPLATE = """[global]
ioengine=libaio
direct=0
fadvise_hint=0
iodepth=${IODEPTH}
invalidate=1
thread=1
openfiles=1
group_reporting=1
create_serialize=0
allrandrepeat=0
file_service_type=random
rw=randread
filename_format=bench.$jobnum.$filenum.size-${FILESIZE}

[rand_read_job]
directory=${DIR}
filesize=${FILESIZE}
bs=${BS}
numjobs=${NUMJOBS}
nrfiles=${NRFILES}
"""

# --- Helper Functions ---

def drop_system_caches():
    """Drop kernel page cache to ensure storage reads are measured."""
    try:
        subprocess.run(["sudo", "sh", "-c", "echo 3 > /proc/sys/vm/drop_caches"], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    except Exception as e:
        print(f"Warning dropping caches: {e}", flush=True)

def unmount_gcsfuse():
    print("Unmounting GCSFuse...", flush=True)
    subprocess.run(["sudo", "killall", "-9", "fio"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    subprocess.run(["sudo", "killall", "-9", "gcsfuse"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    time.sleep(1)
    subprocess.run(["sudo", "fusermount", "-z", "-u", MOUNT_POINT], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    time.sleep(1)

def mount_gcsfuse(flags):
    unmount_gcsfuse()
    os.makedirs(MOUNT_POINT, exist_ok=True)
    cmd = ["gcsfuse"] + flags + [BUCKET, MOUNT_POINT]
    print(f"Mounting: {' '.join(cmd)}", flush=True)
    res = subprocess.run(cmd)
    if res.returncode != 0:
        print(f"Mount failed with code {res.returncode}", flush=True)
        sys.exit(1)
    for _ in range(10):
        mount_check = subprocess.run(["mountpoint", "-q", MOUNT_POINT])
        if mount_check.returncode == 0:
            print("Mount verified successfully.", flush=True)
            return
        time.sleep(1)
    print("Warning: mountpoint check timed out.", flush=True)

def cleanup_dir(dir_path):
    print(f"Cleaning up {dir_path}...", flush=True)
    subprocess.run(["rm", "-rf", dir_path], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    time.sleep(1)

def run_fio(template_str, env_vars, jobfile_name="job.fio"):
    with open(jobfile_name, "w") as f:
        f.write(template_str)
    
    env = os.environ.copy()
    env.update(env_vars)
    
    cmd = ["fio", "--output-format=json", jobfile_name]
    print(f"Running FIO: {' '.join(f'{k}={v}' for k, v in env_vars.items())} fio {jobfile_name}", flush=True)
    start_t = time.time()
    res = subprocess.run(cmd, env=env, capture_output=True, text=True)
    duration = time.time() - start_t
    print(f"FIO finished in {duration:.2f}s (returncode: {res.returncode})", flush=True)
    
    if os.path.exists(jobfile_name):
        os.remove(jobfile_name)

    if res.returncode != 0:
        print(f"FIO error: {res.stderr[:500]}", flush=True)
        return None
    
    try:
        data = json.loads(res.stdout)
        job = data["jobs"][0]
        return job
    except Exception as e:
        print(f"Failed to parse FIO JSON: {e}\nRaw output:\n{res.stdout[:500]}", flush=True)
        return None

def extract_metrics(job, rw_type):
    if not job:
        return {"bw_gb": 0.0, "iops_k": 0.0, "lat_ms": 0.0}
    section = job.get(rw_type, {})
    bw_bytes = section.get("bw_bytes", 0)
    bw_gb = bw_bytes / 1e9  # GB/s
    iops = section.get("iops", 0) / 1000.0  # K IOPS
    
    lat_mean_ns = section.get("lat_ns", {}).get("mean", 0)
    if not lat_mean_ns:
        lat_mean_ns = section.get("clat_ns", {}).get("mean", 0)
    lat_ms = lat_mean_ns / 1e6  # msec
    
    return {
        "bw_gb": round(bw_gb, 4),
        "iops_k": round(iops, 4),
        "lat_ms": round(lat_ms, 4),
    }

def save_results(results):
    results_path = "/tmp/rcu_benchmark_results.json"
    with open(results_path, "w") as f:
        json.dump(results, f, indent=2)

def main():
    results = {
        "sequential_writes_rapid": [],
        "sequential_reads_rapid": [],
        "random_reads_rapid": [],
    }

    print("=" * 60, flush=True)
    print("STARTING RCU MICRO-BENCHMARKS (Unified Concurrency & 1:1 Reuse)", flush=True)
    print("Rules: < 1 GB = 192 jobs | >= 1 GB = 144 jobs for both Write & Read", flush=True)
    print(f"Bucket: {BUCKET} | Page Cache: ON (direct=0)", flush=True)
    print("=" * 60, flush=True)

    mount_gcsfuse(MOUNT_FLAGS)

    # Dedicated directories for 1:1 reuse
    dir_100m = os.path.join(MOUNT_POINT, "data_100M")
    dir_10g  = os.path.join(MOUNT_POINT, "data_10G")
    dir_128k = os.path.join(MOUNT_POINT, "data_128k")
    dir_1g   = os.path.join(MOUNT_POINT, "data_1G")

    # =========================================================================
    # 1. 100 MB WORKLOAD (192 jobs x 10 files)
    # =========================================================================
    print("\n" + "=" * 50, flush=True)
    print("100 MB WORKLOAD (192 jobs x 10 files)", flush=True)
    print("=" * 50, flush=True)
    cleanup_dir(dir_100m)
    os.makedirs(dir_100m, exist_ok=True)

    # Write
    env_vars_100m = {
        "DIR": dir_100m, "FILESIZE": "100M", "BS": "1M", "NUMJOBS": "192", "NRFILES": "10",
    }
    job_100m_w = run_fio(FIO_WRITE_TEMPLATE, env_vars_100m, "write_100m.fio")
    m_100m_w = extract_metrics(job_100m_w, "write")
    print(f"[Rapid Write] 100 MB (192 jobs x 10 files): Bandwidth={m_100m_w['bw_gb']} GB/s, Latency={m_100m_w['lat_ms']} ms", flush=True)
    results["sequential_writes_rapid"].append({
        "label": "Sequential Write 100 MB (BS=1M, 192x10)",
        "filesize": "100M", "bs": "1M", "numjobs": 192, "nrfiles": 10, **m_100m_w,
    })
    save_results(results)

    # Read (Directly reuses the 192x10 files)
    drop_system_caches()
    job_100m_r = run_fio(FIO_SEQ_READ_TEMPLATE, env_vars_100m, "read_100m.fio")
    m_100m_r = extract_metrics(job_100m_r, "read")
    print(f"[Rapid Read] 100 MB (192 jobs x 10 files): Bandwidth={m_100m_r['bw_gb']} GB/s, Latency={m_100m_r['lat_ms']} ms", flush=True)
    results["sequential_reads_rapid"].append({
        "label": "Sequential Read 100 MB (BS=1M, 192x10)",
        "filesize": "100M", "bs": "1M", "numjobs": 192, "nrfiles": 10, **m_100m_r,
    })
    save_results(results)
    cleanup_dir(dir_100m)

    # =========================================================================
    # 2. 10 GB WORKLOAD (144 jobs x 2 files)
    # =========================================================================
    print("\n" + "=" * 50, flush=True)
    print("10 GB WORKLOAD (144 jobs x 2 files)", flush=True)
    print("=" * 50, flush=True)
    cleanup_dir(dir_10g)
    os.makedirs(dir_10g, exist_ok=True)

    # Write
    env_vars_10g = {
        "DIR": dir_10g, "FILESIZE": "10G", "BS": "1M", "NUMJOBS": "144", "NRFILES": "2",
    }
    job_10g_w = run_fio(FIO_WRITE_TEMPLATE, env_vars_10g, "write_10g.fio")
    m_10g_w = extract_metrics(job_10g_w, "write")
    print(f"[Rapid Write] 10 GB (144 jobs x 2 files): Bandwidth={m_10g_w['bw_gb']} GB/s, Latency={m_10g_w['lat_ms']} ms", flush=True)
    results["sequential_writes_rapid"].append({
        "label": "Sequential Write 10 GB (BS=1M, 144x2)",
        "filesize": "10G", "bs": "1M", "numjobs": 144, "nrfiles": 2, **m_10g_w,
    })
    save_results(results)

    # Read (Directly reuses the 144x2 files)
    drop_system_caches()
    job_10g_r = run_fio(FIO_SEQ_READ_TEMPLATE, env_vars_10g, "read_10g.fio")
    m_10g_r = extract_metrics(job_10g_r, "read")
    print(f"[Rapid Read] 10 GB (144 jobs x 2 files): Bandwidth={m_10g_r['bw_gb']} GB/s, Latency={m_10g_r['lat_ms']} ms", flush=True)
    results["sequential_reads_rapid"].append({
        "label": "Sequential Read 10 GB (BS=1M, 144x2)",
        "filesize": "10G", "bs": "1M", "numjobs": 144, "nrfiles": 2, **m_10g_r,
    })
    save_results(results)
    cleanup_dir(dir_10g)

    # =========================================================================
    # 3. 128 KB WORKLOAD (192 jobs x 30 files)
    # =========================================================================
    print("\n" + "=" * 50, flush=True)
    print("128 KB WORKLOAD (192 jobs x 30 files)", flush=True)
    print("=" * 50, flush=True)
    cleanup_dir(dir_128k)
    os.makedirs(dir_128k, exist_ok=True)

    # Write
    env_vars_128k = {
        "DIR": dir_128k, "FILESIZE": "128k", "BS": "128k", "NUMJOBS": "192", "NRFILES": "30",
    }
    job_128k_w = run_fio(FIO_WRITE_TEMPLATE, env_vars_128k, "write_128k.fio")
    m_128k_w = extract_metrics(job_128k_w, "write")
    print(f"[Rapid Write] 128 KB (192 jobs x 30 files): Bandwidth={m_128k_w['bw_gb']} GB/s, Latency={m_128k_w['lat_ms']} ms", flush=True)
    results["sequential_writes_rapid"].append({
        "label": "Sequential Write 128 KB (BS=128K, 192x30)",
        "filesize": "128k", "bs": "128k", "numjobs": 192, "nrfiles": 30, **m_128k_w,
    })
    save_results(results)

    # Read (Directly reuses the 192x30 files)
    drop_system_caches()
    job_128k_r = run_fio(FIO_SEQ_READ_TEMPLATE, env_vars_128k, "read_128k.fio")
    m_128k_r = extract_metrics(job_128k_r, "read")
    print(f"[Rapid Read] 128 KB (192 jobs x 30 files): Bandwidth={m_128k_r['bw_gb']} GB/s, Latency={m_128k_r['lat_ms']} ms", flush=True)
    results["sequential_reads_rapid"].append({
        "label": "Sequential Read 128 KB (BS=128K, 192x30)",
        "filesize": "128k", "bs": "128k", "numjobs": 192, "nrfiles": 30, **m_128k_r,
    })
    save_results(results)
    cleanup_dir(dir_128k)

    # =========================================================================
    # 4. 1 GB RANDOM READ WORKLOAD (144 jobs x 10 files)
    # =========================================================================
    print("\n" + "=" * 50, flush=True)
    print("1 GB WORKLOAD (144 jobs x 10 files)", flush=True)
    print("=" * 50, flush=True)
    cleanup_dir(dir_1g)
    os.makedirs(dir_1g, exist_ok=True)

    # Write 1GB dataset once (144 jobs x 10 files = 1.44 TB)
    env_vars_1g = {
        "DIR": dir_1g, "FILESIZE": "1G", "BS": "1M", "NUMJOBS": "144", "NRFILES": "10",
    }
    job_1g_w = run_fio(FIO_WRITE_TEMPLATE, env_vars_1g, "write_1g.fio")
    m_1g_w = extract_metrics(job_1g_w, "write")
    print(f"[Rapid Write] 1 GB (144 jobs x 10 files): Bandwidth={m_1g_w['bw_gb']} GB/s, Latency={m_1g_w['lat_ms']} ms", flush=True)
    results["sequential_writes_rapid"].append({
        "label": "Sequential Write 1 GB (BS=1M, 144x10)",
        "filesize": "1G", "bs": "1M", "numjobs": 144, "nrfiles": 10, **m_1g_w,
    })
    save_results(results)

    # Random Reads (Directly reuses the 144x10 files across all BS & QD)
    rand_read_workloads = [
        {"bs": "4k",  "iodepth": 1, "label": "Random Read 1 GB (BS=4K, QD=1, 144x10)"},
        {"bs": "4k",  "iodepth": 4, "label": "Random Read 1 GB (BS=4K, QD=4, 144x10)"},
        {"bs": "64k", "iodepth": 1, "label": "Random Read 1 GB (BS=64K, QD=1, 144x10)"},
        {"bs": "64k", "iodepth": 4, "label": "Random Read 1 GB (BS=64K, QD=4, 144x10)"},
        {"bs": "1M",  "iodepth": 1, "label": "Random Read 1 GB (BS=1M, QD=1, 144x10)"},
        {"bs": "1M",  "iodepth": 4, "label": "Random Read 1 GB (BS=1M, QD=4, 144x10)"},
        {"bs": "4M",  "iodepth": 1, "label": "Random Read 1 GB (BS=4M, QD=1, 144x10)"},
        {"bs": "4M",  "iodepth": 4, "label": "Random Read 1 GB (BS=4M, QD=4, 144x10)"},
        {"bs": "16M", "iodepth": 1, "label": "Random Read 1 GB (BS=16M, QD=1, 144x10)"},
        {"bs": "16M", "iodepth": 4, "label": "Random Read 1 GB (BS=16M, QD=4, 144x10)"},
    ]

    for item in rand_read_workloads:
        drop_system_caches()
        env_vars_rand = {
            "DIR": dir_1g,
            "FILESIZE": "1G",
            "BS": item["bs"],
            "NUMJOBS": "144",
            "NRFILES": "10",
            "IODEPTH": str(item["iodepth"]),
        }
        job = run_fio(FIO_RAND_READ_TEMPLATE, env_vars_rand, "rand_read.fio")
        metrics = extract_metrics(job, "read")
        print(f"[Rapid Rand Read] {item['label']}: Bandwidth={metrics['bw_gb']} GB/s, IOPS={metrics['iops_k']} K, Latency={metrics['lat_ms']} ms", flush=True)
        
        results["random_reads_rapid"].append({
            "label": item["label"],
            "filesize": "1G",
            "bs": item["bs"],
            "iodepth": item["iodepth"],
            "numjobs": 144,
            "nrfiles": 10,
            **metrics,
        })
        save_results(results)

    cleanup_dir(dir_1g)

    print("\n" + "=" * 60, flush=True)
    print("ALL RAPID BENCHMARK TESTS COMPLETED SUCCESSFULLY!", flush=True)
    print("=" * 60, flush=True)
    print(json.dumps(results, indent=2), flush=True)

if __name__ == "__main__":
    main()
