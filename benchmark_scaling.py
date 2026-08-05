#!/usr/bin/env python3
import os
import sys
import time
import subprocess
import re
import json
import matplotlib.pyplot as plt
from collections import defaultdict

# Configurations to test
TARGET_RANGES = 20
TARGET_BYTES_LIST = [
    (10 * 1024 * 1024, "10MB"),
    (20 * 1024 * 1024, "20MB"),
    (25 * 1024 * 1024, "25MB"),
    (50 * 1024 * 1024, "50MB")
]

MOUNT_POINT = "/home/cpranjal_google_com/mnt"
BUCKET = "cpranjal-rapid-us-west4a"
GCSFUSE_BIN = os.path.expanduser("~/gcsfuse_pr")
FIO_FILE = "a.fio"
LOG_FILE = "/tmp/gcsfuse_run.log"

def run_cmd(cmd, shell=True, check=True):
    print(f"Running: {cmd}")
    process = subprocess.Popen(
        cmd,
        shell=shell,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True
    )
    
    stdout_accumulator = []
    for line in iter(process.stdout.readline, ""):
        sys.stdout.write(line)
        sys.stdout.flush()
        stdout_accumulator.append(line)
        
    process.stdout.close()
    return_code = process.wait()
    
    full_output = "".join(stdout_accumulator)
    
    if check and return_code != 0:
        raise subprocess.CalledProcessError(
            returncode=return_code,
            cmd=cmd,
            output=full_output,
            stderr=""
        )
        
    class CompletedProcess:
        def __init__(self, stdout, returncode):
            self.stdout = stdout
            self.returncode = returncode
            
    return CompletedProcess(full_output, return_code)

def safe_unmount():
    try:
        run_cmd(f"fusermount -u {MOUNT_POINT}", check=False)
    except Exception:
        pass
    try:
        run_cmd(f"sudo umount {MOUNT_POINT}", check=False)
    except Exception:
        pass
    time.sleep(1)

def parse_logs(log_path):
    # Group stats by second
    # second_timestamp -> file_name -> (streams, pending_ranges, at_capacity)
    second_stats = defaultdict(dict)
    
    pattern = re.compile(
        r"\[MRD_SCALING\] TimeMs=(\d+) File=(\S+) Streams=(\d+) PendingRanges=(\d+) AtCapacityStreams=(\d+)"
    )
    
    if not os.path.exists(log_path):
        print(f"Warning: Log file {log_path} not found.")
        return []

    with open(log_path, "r") as f:
        for line in f:
            match = pattern.search(line)
            if match:
                time_ms = int(match.group(1))
                file_name = match.group(2)
                streams = int(match.group(3))
                pending = int(match.group(4))
                at_capacity = int(match.group(5))
                
                second = time_ms // 1000
                second_stats[second][file_name] = {
                    "streams": streams,
                    "pending": pending,
                    "at_capacity": at_capacity
                }
                
    if not second_stats:
        return []
        
    start_second = min(second_stats.keys())
    
    results = []
    # Normalize time to start from 0
    for sec in sorted(second_stats.keys()):
        elapsed = sec - start_second
        files_data = second_stats[sec]
        
        total_streams = sum(d["streams"] for d in files_data.values())
        total_pending = sum(d["pending"] for d in files_data.values())
        total_at_capacity = sum(d["at_capacity"] for d in files_data.values())
        active_files = len(files_data)
        
        results.append({
            "elapsed_sec": elapsed,
            "active_files": active_files,
            "total_streams": total_streams,
            "total_pending_ranges": total_pending,
            "total_at_capacity": total_at_capacity
        })
    return results

def parse_fio_bw(fio_output):
    # Search for "READ: bw=..."
    # e.g., "   READ: bw=996MiB/s (1044MB/s)"
    match = re.search(r"READ:\s+bw=([\d\.]+)(KiB|MiB|GiB|B)/s", fio_output)
    if match:
        val = float(match.group(1))
        unit = match.group(2)
        return f"{val} {unit}/s"
    
    # Fallback to general BW search
    # e.g. "read: IOPS=995, BW=996MiB/s (1044MB/s)"
    match = re.search(r"BW=([\d\.]+)(KiB|MiB|GiB|B)/s", fio_output)
    if match:
        val = float(match.group(1))
        unit = match.group(2)
        return f"{val} {unit}/s"
        
    return "Unknown"

def compile_binary():
    print("Compiling gcsfuse binary from current source...")
    try:
        run_cmd(f"go build -o {GCSFUSE_BIN} .")
        print(f"Successfully compiled binary to {GCSFUSE_BIN}")
    except subprocess.CalledProcessError as e:
        print(f"Error compiling gcsfuse binary: {e}")
        if e.stdout:
            print(f"Compilation Stdout:\n{e.stdout}")
        if e.stderr:
            print(f"Compilation Stderr:\n{e.stderr}")
        sys.exit(1)

def main():
    compile_binary()

    if not os.path.exists(FIO_FILE):
        print(f"Error: {FIO_FILE} not found in the current directory.")
        sys.exit(1)

    all_runs_data = {}

    for num_bytes, label in TARGET_BYTES_LIST:
        print("\n" + "="*60)
        print(f"Starting test for: --mrd-target-pending-bytes={label} ({num_bytes} bytes)")
        print("="*60)
        
        safe_unmount()
        
        if os.path.exists(LOG_FILE):
            os.remove(LOG_FILE)
            
        # Mount GCSFuse in foreground, redirecting stderr to LOG_FILE
        mount_cmd = (
            f"{GCSFUSE_BIN} --foreground "
            f"--client-protocol=grpc "
            f"--metadata-cache-ttl-secs=-1 "
            f"--mrd-min-connections=1 "
            f"--mrd-max-connections=4 "
            f"--mrd-target-pending-ranges={TARGET_RANGES} "
            f"--mrd-target-pending-bytes={num_bytes} "
            f"--implicit-dirs "
            f"{BUCKET} {MOUNT_POINT} 2> {LOG_FILE} &"
        )
        print(f"Mounting: {mount_cmd}")
        
        # Setup environment variables explicitly so we don't rely on shell export state
        env = os.environ.copy()
        env["GCS_DUMMY_MRD"] = "true"
        env["GCS_DUMMY_MRD_MB_PER_SEC"] = "-1"  # Disable rate limiting for raw speed comparison
        env["GCS_DUMMY_MRD_LATENCY_MS"] = "7.0"   # Simulating slower RTT to match 14 GiB/s real network limits
        
        subprocess.Popen(mount_cmd, shell=True, env=env)
        
        # Wait for mount to stabilize
        print("Waiting for mount to initialize...")
        time.sleep(5)
        
        # Drop kernel page caches
        print("Dropping VM page cache...")
        run_cmd("sudo sh -c 'echo 3 > /proc/sys/vm/drop_caches'")
        
        # Run FIO benchmark
        print("Running FIO benchmark...")
        fio_cmd = f"FILE_SIZE=10g fio {FIO_FILE}"
        fio_bw = "Unknown"
        try:
            res = run_cmd(fio_cmd)
            fio_bw = parse_fio_bw(res.stdout)
            print(f"FIO Bandwidth captured: {fio_bw}")
        except subprocess.CalledProcessError as e:
            print(f"FIO finished or was interrupted: {e}")
            if e.stdout:
                print(f"FIO Stdout:\n{e.stdout}")
            if e.stderr:
                print(f"FIO Stderr:\n{e.stderr}")
            if e.stdout:
                fio_bw = parse_fio_bw(e.stdout)
        
        # Unmount cleanly
        print("Unmounting...")
        safe_unmount()
        
        # Parse logs
        print("Parsing logs...")
        run_data = parse_logs(LOG_FILE)
        all_runs_data[label] = {
            "stats": run_data,
            "bandwidth": fio_bw
        }
        print(f"Captured {len(run_data)} seconds of stats.")

    # Save raw data to JSON
    with open("mrd_scaling_results.json", "w") as f:
        json.dump(all_runs_data, f, indent=2)
    print("\nSaved raw scaling results to: mrd_scaling_results.json")

    # Print summary table to console
    print("\n" + "="*80)
    print("        SCALING & THROUGHPUT BENCHMARK SUMMARY TABLE")
    print("="*80)
    print(f"{'Target Bytes':<12} | {'FIO Bandwidth':<15} | {'Max Combined Streams':<22} | {'Max Combined Pending':<22}")
    print("-"*80)
    for num_bytes, label in TARGET_BYTES_LIST:
        run_info = all_runs_data.get(label, {})
        data = run_info.get("stats", [])
        bw = run_info.get("bandwidth", "No Data")
        if data:
            max_streams = max(d["total_streams"] for d in data)
            max_pending = max(d["total_pending_ranges"] for d in data)
            print(f"{label:<12} | {bw:<15} | {max_streams:<22} | {max_pending:<22}")
        else:
            print(f"{label:<12} | {bw:<15} | {'No Data':<22} | {'No Data':<22}")
    print("="*80)

    # Generate the graphs
    print("\nGenerating scaling comparison graphs...")
    fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(12, 10), sharex=True)
    
    colors = ['#1f77b4', '#ff7f0e', '#2ca02c', '#d62728']
    
    has_plot_data = False
    for (num_bytes, label), color in zip(TARGET_BYTES_LIST, colors):
        run_info = all_runs_data.get(label, {})
        data = run_info.get("stats", [])
        if not data:
            continue
        
        has_plot_data = True
        x = [d["elapsed_sec"] for d in data]
        streams = [d["total_streams"] for d in data]
        pending = [d["total_pending_ranges"] for d in data]
        
        # Plot total streams
        ax1.plot(x, streams, label=f"target-bytes={label}", color=color, linewidth=2, marker='o', markersize=3)
        # Plot total pending ranges
        ax2.plot(x, pending, label=f"target-bytes={label}", color=color, linewidth=2, marker='s', markersize=3)
        
    if has_plot_data:
        ax1.set_title("Total Combined Streams across All Active Files", fontsize=14, fontweight='bold')
        ax1.set_ylabel("Active Stream Connections", fontsize=12)
        ax1.grid(True, linestyle='--', alpha=0.6)
        ax1.legend(loc="upper right", fontsize=10)
        
        ax2.set_title("Total Combined Pending Ranges in Flight", fontsize=14, fontweight='bold')
        ax2.set_xlabel("Elapsed Time (Seconds)", fontsize=12)
        ax2.set_ylabel("Pending Range Count", fontsize=12)
        ax2.grid(True, linestyle='--', alpha=0.6)
        ax2.legend(loc="upper right", fontsize=10)
        
        plt.tight_layout()
        plt.savefig("mrd_scaling_results.png", dpi=300)
        print("Saved comparison plot to: mrd_scaling_results.png")
    else:
        print("No valid stats to plot.")

if __name__ == "__main__":
    main()
