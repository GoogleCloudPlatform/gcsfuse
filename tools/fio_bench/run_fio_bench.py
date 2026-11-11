#!/usr/bin/env python3
# Copyright 2025 Google LLC
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

import argparse
import json
import os
import shlex
import subprocess
import time
import threading

CLK_TCK = os.sysconf(os.sysconf_names["SC_CLK_TCK"])


def format_cpu_list(cpus):
    cpus = sorted(cpus)
    if not cpus:
        return ""
    ranges = []
    start = prev = cpus[0]
    for c in cpus[1:]:
        if c == prev + 1:
            prev = c
        else:
            ranges.append(f"{start}-{prev}" if start != prev else f"{start}")
            start = prev = c
    ranges.append(f"{start}-{prev}" if start != prev else f"{start}")
    return ",".join(ranges)


def get_proc_io(pid):
    try:
        syscr, syscw = 0, 0
        with open(f"/proc/{pid}/io", "r") as f:
            for line in f:
                if line.startswith("syscr:"):
                    syscr = int(line.split()[1])
                elif line.startswith("syscw:"):
                    syscw = int(line.split()[1])
        return syscr, syscw
    except Exception:
        return 0, 0


def get_proc_ticks(pid):
    try:
        with open(f"/proc/{pid}/stat", "r") as f:
            content = f.read()
        rparen = content.rfind(")")
        fields = content[rparen + 2 :].split()
        utime = int(fields[11])
        stime = int(fields[12])
        rss_pages = int(fields[21])
        return utime, stime, rss_pages * 4096
    except Exception:
        return 0, 0, 0


def get_system_cpu_ticks():
    with open("/proc/stat", "r") as f:
        for line in f:
            if line.startswith("cpu "):
                parts = [int(x) for x in line.split()[1:]]
                busy = (
                    parts[0]
                    + parts[1]
                    + parts[2]
                    + parts[5]
                    + parts[6]
                    + (parts[7] if len(parts) > 7 else 0)
                )
                total = sum(parts)
                return busy, total, parts[0] + parts[1], parts[2], parts[6]
    return 0, 0, 0, 0, 0


def unmount(mnt):
    subprocess.run(["fusermount3", "-u", "-z", mnt], stderr=subprocess.DEVNULL)
    subprocess.run(["umount", "-l", mnt], stderr=subprocess.DEVNULL)
    time.sleep(0.5)


def main():
    parser = argparse.ArgumentParser(
        description="Run GCSFuse gRPC FIO read benchmark with CPU telemetry"
    )
    parser.add_argument(
        "--bucket",
        default=os.environ.get(
            "BUCKET", os.environ.get("BUCKET_NAME", "kislayk-c4-usw4-hns")
        ),
    )
    parser.add_argument("--mnt", default="/mnt/gcs")
    parser.add_argument(
        "--flags",
        default=os.environ.get("GCSFUSE_FLAGS", "--client-protocol=grpc"),
    )
    parser.add_argument(
        "--gcsfuse-cpus", default=os.environ.get("GCSFUSE_CPUS", None)
    )
    parser.add_argument("--fio-cpus", default=os.environ.get("FIO_CPUS", None))
    parser.add_argument(
        "--numjobs", type=int, default=int(os.environ.get("NUMJOBS", "56"))
    )
    parser.add_argument(
        "--nrfiles",
        type=int,
        default=int(
            os.environ.get("NR_FILES", os.environ.get("NRFILES", "3"))
        ),
    )
    parser.add_argument("--bs", default=os.environ.get("BLOCK_SIZE", "1M"))
    parser.add_argument("--filesize", default=os.environ.get("FILE_SIZE", "1G"))
    parser.add_argument(
        "--label", default=os.environ.get("LABEL", "docker_grpc_opt")
    )
    args = parser.parse_args()

    avail_cpus = os.sched_getaffinity(0)
    avail_cpu_str = f"{format_cpu_list(avail_cpus)} ({len(avail_cpus)} vCPUs)"

    os.makedirs(args.mnt, exist_ok=True)
    unmount(args.mnt)

    log_file = f"/tmp/gcsfuse_{args.label}.log"
    cmd = (
        ["gcsfuse"]
        + shlex.split(args.flags)
        + [
            "-o",
            "allow_other",
            f"--log-file={log_file}",
            "--log-format=json",
            args.bucket,
            args.mnt,
        ]
    )
    if args.gcsfuse_cpus:
        cmd = ["taskset", "-c", args.gcsfuse_cpus] + cmd

    print(
        f"[*] Mounting gs://{args.bucket} at {args.mnt} (active CPUs: {args.gcsfuse_cpus or avail_cpu_str})..."
    )
    subprocess.run(cmd, check=True)
    time.sleep(1.5)

    pids = [
        p
        for p in subprocess.check_output(["pgrep", "-f", args.bucket])
        .decode()
        .split()
        if int(p) != os.getpid()
        and "python" not in open(f"/proc/{p}/comm").read()
    ]
    gcsfuse_pid = int(pids[-1])

    samples = []
    stop_mon = threading.Event()

    def monitor():
        u0, s0, _ = get_proc_ticks(gcsfuse_pid)
        sb0, st0, su0, ss0, si0 = get_system_cpu_ticks()
        t_prev = time.time()
        while not stop_mon.wait(1.0):
            t_now = time.time()
            dt = t_now - t_prev
            if dt <= 0:
                continue
            u1, s1, rss = get_proc_ticks(gcsfuse_pid)
            sb1, st1, su1, ss1, si1 = get_system_cpu_ticks()
            gf_cores = ((u1 - u0) + (s1 - s0)) / CLK_TCK / dt
            gf_ucores = (u1 - u0) / CLK_TCK / dt
            gf_scores = (s1 - s0) / CLK_TCK / dt
            sys_cores = (sb1 - sb0) / CLK_TCK / dt
            softirq_cores = (si1 - si0) / CLK_TCK / dt
            samples.append(
                (gf_cores, gf_ucores, gf_scores, sys_cores, softirq_cores, rss)
            )
            u0, s0, sb0, st0, su0, ss0, si0, t_prev = (
                u1,
                s1,
                sb1,
                st1,
                su1,
                ss1,
                si1,
                t_now,
            )

    u_start, s_start, _ = get_proc_ticks(gcsfuse_pid)
    syscr_start, syscw_start = get_proc_io(gcsfuse_pid)
    sb_start, st_start, _, _, si_start = get_system_cpu_ticks()
    t_start = time.time()
    mon_thread = threading.Thread(target=monitor, daemon=True)
    mon_thread.start()

    fio_out = f"/tmp/fio_{args.label}.json"
    fio_cmd = [
        "fio",
        "/bench/read.fio",
        "--output-format=json",
        f"--output={fio_out}",
        f"--directory={args.mnt}",
    ]
    if args.fio_cpus:
        fio_cmd = ["taskset", "-c", args.fio_cpus] + fio_cmd

    fio_env = os.environ.copy()
    fio_env.update(
        {
            "DIR": args.mnt,
            "NUMJOBS": str(args.numjobs),
            "NR_FILES": str(args.nrfiles),
            "BLOCK_SIZE": args.bs,
            "FILE_SIZE": args.filesize,
            "READ_TYPE": "read",
        }
    )

    print(
        f"[*] Running FIO read workload (bs={args.bs}, filesize={args.filesize}, numjobs={args.numjobs}, nrfiles={args.nrfiles})..."
    )
    subprocess.run(fio_cmd, check=True, env=fio_env)
    t_end = time.time()
    stop_mon.set()
    mon_thread.join(timeout=2.0)

    u_end, s_end, rss_end = get_proc_ticks(gcsfuse_pid)
    syscr_end, syscw_end = get_proc_io(gcsfuse_pid)
    sb_end, st_end, _, _, si_end = get_system_cpu_ticks()
    wall = t_end - t_start

    gf_avg_cores = ((u_end - u_start) + (s_end - s_start)) / CLK_TCK / wall
    gf_u_cores = (u_end - u_start) / CLK_TCK / wall
    gf_s_cores = (s_end - s_start) / CLK_TCK / wall
    sys_avg_cores = (sb_end - sb_start) / CLK_TCK / wall
    softirq_avg_cores = (si_end - si_start) / CLK_TCK / wall

    steady = samples[2:-1] if len(samples) > 4 else samples
    gf_steady = (
        sum(x[0] for x in steady) / len(steady) if steady else gf_avg_cores
    )
    sys_steady = (
        sum(x[3] for x in steady) / len(steady) if steady else sys_avg_cores
    )
    max_rss_mb = max((x[5] for x in samples), default=rss_end) / (1024 * 1024)

    with open(fio_out) as f:
        content = f.read()
        data = json.loads(content[content.find("{") :])
    read_stats = data["jobs"][0]["read"]
    bw_mib = read_stats["bw"] / 1024.0
    bw_gb = read_stats["bw"] * 1024.0 / 1e9
    iops = read_stats["iops"]
    lat_ms = read_stats["lat_ns"]["mean"] / 1e6

    unmount(args.mnt)

    res = {
        "label": args.label,
        "container_cpus": avail_cpu_str,
        "gcsfuse_cpus": args.gcsfuse_cpus or avail_cpu_str,
        "fio_cpus": args.fio_cpus or avail_cpu_str,
        "numjobs": args.numjobs,
        "nrfiles": args.nrfiles,
        "bw_mib_s": round(bw_mib, 2),
        "bw_gb_s": round(bw_gb, 3),
        "iops": round(iops, 1),
        "mean_lat_ms": round(lat_ms, 2),
        "wall_s": round(wall, 2),
        "gcsfuse_avg_cores": round(gf_avg_cores, 2),
        "gcsfuse_user_cores": round(gf_u_cores, 2),
        "gcsfuse_sys_cores": round(gf_s_cores, 2),
        "gcsfuse_steady_cores": round(gf_steady, 2),
        "total_sys_avg_cores": round(sys_avg_cores, 2),
        "total_sys_steady_cores": round(sys_steady, 2),
        "softirq_avg_cores": round(softirq_avg_cores, 2),
        "max_rss_mb": round(max_rss_mb, 1),
        "syscr_per_s": round((syscr_end - syscr_start) / wall, 1),
        "syscw_per_s": round((syscw_end - syscw_start) / wall, 1),
        "flags": args.flags,
    }
    print(
        "\n================ GCSFuse gRPC FIO Read Benchmark Summary ================"
    )
    print(f"  Bucket                : gs://{args.bucket}")
    print(f"  Active CPU Affinity   : {res['gcsfuse_cpus']}")
    print(
        f"  Read Throughput       : {res['bw_mib_s']:,.2f} MiB/s ({res['bw_gb_s']:.3f} GB/s)"
    )
    print(
        f"  Read IOPS / Mean Lat  : {res['iops']:,.1f} IOPS / {res['mean_lat_ms']:.2f} ms"
    )
    print(
        f"  GCSFuse Avg CPU Cores : {res['gcsfuse_avg_cores']:.2f} cores ({res['gcsfuse_user_cores']:.2f} user + {res['gcsfuse_sys_cores']:.2f} sys)"
    )
    print(f"  GCSFuse Steady Cores  : {res['gcsfuse_steady_cores']:.2f} cores")
    print(
        f"  Total System Cores    : {res['total_sys_avg_cores']:.2f} avg / {res['total_sys_steady_cores']:.2f} steady (softirq: {res['softirq_avg_cores']:.2f})"
    )
    print(f"  GCSFuse Max RSS       : {res['max_rss_mb']:.1f} MB")
    print(
        "=========================================================================\n"
    )
    print("RESULT_JSON:", json.dumps(res))


if __name__ == "__main__":
    main()
