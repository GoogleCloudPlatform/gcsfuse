#!/usr/bin/env python3
"""
Independent, self-contained FIO benchmark comparison script for GCSFuse.
Compares sequential write performance between:
  - master:        go run ./ --implicit-dirs vipin-us-west4 ~/mnt
  - vipin-pcu-poc: go run ./ --implicit-dirs --write-max-blocks-per-file=1 --client-protocol=grpc vipin-us-west4 ~/mnt

Runs 3 iterations per configuration, averages metrics, and writes results to CSV.

Requirements:
  - python3 (standard library only: no pip dependencies)
  - fio (sudo apt-get install -y fio)
  - fusermount (standard on Linux)
  - git, go
"""

import argparse
import csv
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
from typing import Any, Dict, List

# ==============================================================================
# CONFIGURATION DEFAULTS (can be overridden via CLI flags)
# ==============================================================================
DEFAULT_GCSFUSE_DIR = os.path.expanduser("~/gcsfuse")
DEFAULT_MOUNT_DIR = os.path.expanduser("~/mnt")
DEFAULT_BUCKET = "vipin-us-west4"
DEFAULT_ITERATIONS = 3
DEFAULT_SUMMARY_CSV = "benchmark_summary.csv"
DEFAULT_RAW_CSV = "benchmark_raw_iterations.csv"

# Configurations to test (matching docs/benchmarks.md)
BENCHMARK_CONFIGS = [
    {
        "id": "256k",
        "desc_size": "256 KiB",
        "desc_bs": "16 KiB",
        "filesize": "256k",
        "bs": "16k",
        "numjobs": 96,
        "nrfiles": 30,
    },
    {
        "id": "1m",
        "desc_size": "1 MiB",
        "desc_bs": "1 MiB",
        "filesize": "1m",
        "bs": "1m",
        "numjobs": 96,
        "nrfiles": 30,
    },
    {
        "id": "100m",
        "desc_size": "100 MiB",
        "desc_bs": "1 MiB",
        "filesize": "100m",
        "bs": "1m",
        "numjobs": 96,
        "nrfiles": 20,
    },
    {
        "id": "1g",
        "desc_size": "1 GiB",
        "desc_bs": "1 MiB",
        "filesize": "1g",
        "bs": "1m",
        "numjobs": 96,
        "nrfiles": 10,
    },
]

BRANCHES = [
    {
        "name": "master",
        "mount_args": [
            "--implicit-dirs",
            "--write-global-max-blocks=-1",
        ],
    },
    {
        "name": "vipin-pcu-poc",
        "mount_args": [
            "--implicit-dirs",
            "--write-global-max-blocks=-1",
            "--write-max-blocks-per-file=4",
            "--client-protocol=grpc",
        ],
    },
]

FIO_TEMPLATE = """[global]
ioengine=libaio
direct=1
fadvise_hint=0
iodepth=64
verify=0
invalidate=1
file_append=0
create_on_open=1
end_fsync=1
thread=1
openfiles=1
group_reporting=1
allrandrepeat=1
filename_format=$jobname.$jobnum.$filenum.size-{filesize}
rw=write

[write_seq]
directory={directory}
filesize={filesize}
bs={bs}
numjobs={numjobs}
nrfiles={nrfiles}
"""


def log(msg: str) -> None:
    now = time.strftime("%Y-%m-%d %H:%M:%S")
    print(f"[{now}] {msg}", flush=True)


def is_mounted(mount_dir: str) -> bool:
    res = subprocess.run(["mountpoint", "-q", mount_dir])
    return res.returncode == 0


def unmount(mount_dir: str) -> None:
    if not is_mounted(mount_dir):
        return
    log(f"Unmounting {mount_dir}...")
    subprocess.run(["fusermount", "-u", mount_dir], check=False)
    time.sleep(2)
    if is_mounted(mount_dir):
        log(f"Normal unmount timed out, using lazy unmount (-uz)...")
        subprocess.run(["fusermount", "-uz", mount_dir], check=False)
        time.sleep(2)
    if not is_mounted(mount_dir):
        log(f"Unmounted {mount_dir} successfully.")
    else:
        log(f"WARNING: {mount_dir} could not be unmounted.")


def mount(gcsfuse_dir: str, bucket: str, mount_dir: str, extra_flags: List[str], timeout: int = 45) -> None:
    unmount(mount_dir)
    os.makedirs(mount_dir, exist_ok=True)

    mount_cmd = ["go", "run", "./"] + extra_flags + [bucket, mount_dir]
    log(f"Mounting: cd {gcsfuse_dir} && {' '.join(mount_cmd)}")
    subprocess.run(mount_cmd, cwd=gcsfuse_dir, check=True)

    log(f"Waiting for {mount_dir} to mount...")
    start = time.time()
    while time.time() - start < timeout:
        if is_mounted(mount_dir):
            log(f"Filesystem mounted successfully on {mount_dir}.")
            time.sleep(1)
            return
        time.sleep(0.5)

    raise RuntimeError(f"Mount timed out after {timeout} seconds on {mount_dir}.")


def run_fio(mount_dir: str, branch: str, cfg: Dict[str, Any], iteration: int, cleanup: bool = True) -> Dict[str, float]:
    run_id = f"fio_test_{branch}_{cfg['id']}_iter{iteration}"
    job_dir = os.path.join(mount_dir, run_id)
    os.makedirs(job_dir, exist_ok=True)

    config_text = FIO_TEMPLATE.format(
        directory=job_dir,
        filesize=cfg["filesize"],
        bs=cfg["bs"],
        numjobs=cfg["numjobs"],
        nrfiles=cfg["nrfiles"],
    )

    with tempfile.NamedTemporaryFile("w", suffix=".fio", delete=False) as f:
        f.write(config_text)
        fio_file = f.name

    try:
        log(f"Running fio: branch={branch}, size={cfg['desc_size']}, bs={cfg['desc_bs']}, iter={iteration}...")
        start_ts = time.time()
        proc = subprocess.run(
            ["fio", fio_file, "--output-format=json"],
            capture_output=True,
            text=True,
            check=True,
        )
        elapsed = time.time() - start_ts
        log(f"fio finished in {elapsed:.1f}s.")

        data = json.loads(proc.stdout)
        write_stat = data["jobs"][0]["write"]

        bw_bytes = float(write_stat.get("bw_bytes", write_stat.get("bw", 0) * 1024))
        bw_gb_s = bw_bytes / 1e9

        iops = float(write_stat.get("iops", 0))
        iops_k = iops / 1000.0

        lat_ns = float(
            write_stat.get("lat_ns", {}).get(
                "mean",
                write_stat.get("clat_ns", {}).get(
                    "mean",
                    write_stat.get("lat_usec", {}).get("mean", 0.0) * 1000.0,
                ),
            )
        )
        lat_ms = lat_ns / 1e6

        log(f"Result: BW = {bw_gb_s:.2f} GB/s | IOPS = {iops_k:.2f} K | Latency = {lat_ms:.2f} ms")

        return {
            "bw_gb_s": bw_gb_s,
            "iops_k": iops_k,
            "lat_ms": lat_ms,
        }
    finally:
        if os.path.exists(fio_file):
            os.remove(fio_file)
        if cleanup and os.path.exists(job_dir):
            log(f"Cleaning up bucket directory {job_dir}...")
            shutil.rmtree(job_dir, ignore_errors=True)


def parse_arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Independent GCSFuse sequential write benchmark script comparing master and vipin-pcu-poc."
    )
    parser.add_argument("--gcsfuse-dir", default=DEFAULT_GCSFUSE_DIR, help=f"Path to gcsfuse repo (default: {DEFAULT_GCSFUSE_DIR})")
    parser.add_argument("--mount-dir", default=DEFAULT_MOUNT_DIR, help=f"Local mount path (default: {DEFAULT_MOUNT_DIR})")
    parser.add_argument("--bucket", default=DEFAULT_BUCKET, help=f"GCS bucket name (default: {DEFAULT_BUCKET})")
    parser.add_argument("--iterations", type=int, default=DEFAULT_ITERATIONS, help=f"Number of iterations per config (default: {DEFAULT_ITERATIONS})")
    parser.add_argument("--configs", nargs="*", default=["all"], help="Specific configs: 256k, 1m, 100m, 1g, or all")
    parser.add_argument("--summary-csv", default=DEFAULT_SUMMARY_CSV, help=f"Summary CSV output filename (default: {DEFAULT_SUMMARY_CSV})")
    parser.add_argument("--raw-csv", default=DEFAULT_RAW_CSV, help=f"Raw iterations CSV output filename (default: {DEFAULT_RAW_CSV})")
    parser.add_argument("--no-cleanup", action="store_true", help="Do not delete written objects from the bucket after testing")
    return parser.parse_args()


def main() -> None:
    args = parse_arguments()
    gcsfuse_dir = os.path.abspath(args.gcsfuse_dir)
    mount_dir = os.path.abspath(args.mount_dir)
    bucket = args.bucket
    num_iters = args.iterations
    cleanup = not args.no_cleanup

    selected_configs = BENCHMARK_CONFIGS
    if "all" not in args.configs:
        selected_configs = [c for c in BENCHMARK_CONFIGS if c["id"] in args.configs]
        if not selected_configs:
            print(f"Error: Invalid configs {args.configs}. Choose from: 256k, 1m, 100m, 1g", file=sys.stderr)
            sys.exit(1)

    # Check dependencies
    for tool in ["fio", "fusermount", "git", "go"]:
        if not shutil.which(tool):
            print(f"Error: Required tool '{tool}' is not installed or not in PATH.", file=sys.stderr)
            if tool == "fio":
                print("Run: sudo apt-get update && sudo apt-get install -y fio", file=sys.stderr)
            sys.exit(1)

    # Remember initial git branch
    initial_branch = subprocess.run(
        ["git", "branch", "--show-current"],
        cwd=gcsfuse_dir,
        capture_output=True,
        text=True,
        check=True,
    ).stdout.strip()
    log(f"Starting branch: {initial_branch}")

    # Data store: results[branch_name][config_id] = [ {iter1}, {iter2}, {iter3} ]
    results: Dict[str, Dict[str, List[Dict[str, float]]]] = {
        b["name"]: {c["id"]: [] for c in selected_configs} for b in BRANCHES
    }

    try:
        for b_info in BRANCHES:
            b_name = b_info["name"]
            log("=" * 70)
            log(f"SWITCHING TO BRANCH: {b_name}")
            log("=" * 70)

            # Checkout branch
            subprocess.run(["git", "checkout", b_name], cwd=gcsfuse_dir, check=True)

            # Mount gcsfuse
            mount(
                gcsfuse_dir=gcsfuse_dir,
                bucket=bucket,
                mount_dir=mount_dir,
                extra_flags=b_info["mount_args"],
            )

            try:
                for cfg in selected_configs:
                    log("-" * 60)
                    log(f"Testing Config: Size={cfg['desc_size']}, BS={cfg['desc_bs']}, NumJobs={cfg['numjobs']}, NRFiles={cfg['nrfiles']}")
                    log("-" * 60)
                    for iter_idx in range(1, num_iters + 1):
                        metrics = run_fio(
                            mount_dir=mount_dir,
                            branch=b_name,
                            cfg=cfg,
                            iteration=iter_idx,
                            cleanup=cleanup,
                        )
                        results[b_name][cfg["id"]].append(metrics)

                    cfg_list = results[b_name][cfg["id"]]
                    avg_lat = sum(x["lat_ms"] for x in cfg_list) / len(cfg_list)
                    avg_bw = sum(x["bw_gb_s"] for x in cfg_list) / len(cfg_list)
                    avg_iops = sum(x["iops_k"] for x in cfg_list) / len(cfg_list)
                    log(f"--> [{b_name} | {cfg['id']}] Average Latency: {avg_lat:.2f} ms | Avg BW: {avg_bw:.2f} GB/s | Avg IOPS: {avg_iops:.2f} K")
            finally:
                unmount(mount_dir)

        # ----------------------------------------------------------------------
        # Write Detailed Raw CSV
        # ----------------------------------------------------------------------
        raw_csv_path = os.path.abspath(args.raw_csv)
        with open(raw_csv_path, "w", newline="") as f:
            writer = csv.writer(f)
            writer.writerow([
                "Branch", "File Size", "Block Size", "NumJobs", "NRFiles",
                "Iteration", "Bandwidth (GB/s)", "IOPS (K)", "Latency (msec)"
            ])
            for b_info in BRANCHES:
                b_name = b_info["name"]
                for cfg in selected_configs:
                    for i, m in enumerate(results[b_name][cfg["id"]], start=1):
                        writer.writerow([
                            b_name, cfg["desc_size"], cfg["desc_bs"], cfg["numjobs"], cfg["nrfiles"],
                            i, f"{m['bw_gb_s']:.2f}", f"{m['iops_k']:.2f}", f"{m['lat_ms']:.2f}"
                        ])
        log(f"Saved raw iteration metrics to: {raw_csv_path}")

        # ----------------------------------------------------------------------
        # Calculate Averages & Write Summary CSV
        # ----------------------------------------------------------------------
        summary_rows = []
        for cfg in selected_configs:
            cid = cfg["id"]
            m_list = results["master"][cid]
            p_list = results["vipin-pcu-poc"][cid]

            m_bw = sum(x["bw_gb_s"] for x in m_list) / len(m_list)
            p_bw = sum(x["bw_gb_s"] for x in p_list) / len(p_list)
            bw_diff = ((p_bw - m_bw) / m_bw * 100.0) if m_bw > 0 else 0.0

            m_iops = sum(x["iops_k"] for x in m_list) / len(m_list)
            p_iops = sum(x["iops_k"] for x in p_list) / len(p_list)

            m_lat = sum(x["lat_ms"] for x in m_list) / len(m_list)
            p_lat = sum(x["lat_ms"] for x in p_list) / len(p_list)
            lat_diff = ((p_lat - m_lat) / m_lat * 100.0) if m_lat > 0 else 0.0

            summary_rows.append({
                "cfg": cfg,
                "m_bw": m_bw, "p_bw": p_bw, "bw_diff": bw_diff,
                "m_iops": m_iops, "p_iops": p_iops,
                "m_lat": m_lat, "p_lat": p_lat, "lat_diff": lat_diff,
            })

        summary_csv_path = os.path.abspath(args.summary_csv)
        with open(summary_csv_path, "w", newline="") as f:
            writer = csv.writer(f)
            writer.writerow([
                "Config", "File Size", "Block Size", "NumJobs", "NRFiles",
                "Master Avg Bandwidth (GB/s)", "PCU Avg Bandwidth (GB/s)", "Bandwidth Change (%)",
                "Master Avg Latency (msec)", "PCU Avg Latency (msec)", "Latency Change (%)",
                "Master Avg IOPS (K)", "PCU Avg IOPS (K)"
            ])
            for r in summary_rows:
                cfg = r["cfg"]
                writer.writerow([
                    cfg["id"], cfg["desc_size"], cfg["desc_bs"], cfg["numjobs"], cfg["nrfiles"],
                    f"{r['m_bw']:.2f}", f"{r['p_bw']:.2f}", f"{r['bw_diff']:+.1f}%",
                    f"{r['m_lat']:.2f}", f"{r['p_lat']:.2f}", f"{r['lat_diff']:+.1f}%",
                    f"{r['m_iops']:.2f}", f"{r['p_iops']:.2f}"
                ])
        log(f"Saved summary comparison to: {summary_csv_path}")

        # ----------------------------------------------------------------------
        # Print Markdown Comparison Table to Terminal
        # ----------------------------------------------------------------------
        print("\n" + "=" * 145)
        print(f"BENCHMARK COMPARISON SUMMARY (Average of {num_iters} runs)")
        print("=" * 145)
        print(
            f"| {'Config':<8} | {'File Size':<9} | {'BlockSize':<9} | {'NumJobs':>7} | {'NRFiles':>7} "
            f"| {'Master BW (GB/s)':>17} | {'PCU BW (GB/s)':>14} | {'BW Diff':>9} "
            f"| {'Master Lat(ms)':>14} | {'PCU Lat(ms)':>12} | {'Lat Diff':>9} "
            f"| {'Master IOPS':>11} | {'PCU IOPS':>10} |"
        )
        print(
            f"| {'-'*8} | {'-'*9} | {'-'*9} | {'-'*7} | {'-'*7} "
            f"| {'-'*17} | {'-'*14} | {'-'*9} "
            f"| {'-'*14} | {'-'*12} | {'-'*9} "
            f"| {'-'*11} | {'-'*10} |"
        )
        for r in summary_rows:
            cfg = r["cfg"]
            print(
                f"| {cfg['id']:<8} "
                f"| {cfg['desc_size']:<9} "
                f"| {cfg['desc_bs']:<9} "
                f"| {cfg['numjobs']:>7} "
                f"| {cfg['nrfiles']:>7} "
                f"| {r['m_bw']:>17.2f} "
                f"| {r['p_bw']:>14.2f} "
                f"| {r['bw_diff']:>+8.1f}% "
                f"| {r['m_lat']:>14.2f} "
                f"| {r['p_lat']:>12.2f} "
                f"| {r['lat_diff']:>+8.1f}% "
                f"| {r['m_iops']:>11.2f} "
                f"| {r['p_iops']:>10.2f} |"
            )
        print("=" * 145)

        print("\n" + "=" * 80)
        print(f"AVERAGE LATENCY PER FIO CONFIG ({num_iters} runs)")
        print("=" * 80)
        for r in summary_rows:
            cfg = r["cfg"]
            print(f"  • Config {cfg['id']} (Size={cfg['desc_size']}, BS={cfg['desc_bs']}, Jobs={cfg['numjobs']}, Files={cfg['nrfiles']}):")
            print(f"      Master:  {r['m_lat']:>8.2f} ms")
            print(f"      PCU:     {r['p_lat']:>8.2f} ms   (Change: {r['lat_diff']:>+6.1f}%)")
        print("=" * 80 + "\n")

    finally:
        unmount(mount_dir)
        # Restore git branch
        curr = subprocess.run(["git", "branch", "--show-current"], cwd=gcsfuse_dir, capture_output=True, text=True).stdout.strip()
        if curr != initial_branch:
            log(f"Restoring git branch back to: {initial_branch}")
            subprocess.run(["git", "checkout", initial_branch], cwd=gcsfuse_dir, check=False)


if __name__ == "__main__":
    main()
