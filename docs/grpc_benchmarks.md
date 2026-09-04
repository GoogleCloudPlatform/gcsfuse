# GCSFuse gRPC Performance Benchmarks

This document compares the performance of GCSFuse using the standard HTTP protocol versus the gRPC protocol for various FIO workloads. The tables below show performance metrics (bandwidth in MB/s) under different test configurations.

For more details on the testing environment and how to run benchmarks manually, please refer to the [GCSFuse Performance Benchmarks](../docs/benchmarks.md) document.

---

## 1. Single-Threaded Read Workloads (NumJobs = 1)

| Io Type | Block Size | File Size | Num Jobs | Num Files | Io Depth | Direct | HTTP Read Bw (MB/s) | gRPC Read Bw (MB/s) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| read | 4k | 4k | 1 | 500 | 1 | 1 | 0.08 | 0.08 |
| read | 4k | 4k | 1 | 500 | 1 | 0 | 0.13 | 0.12 |
| read | 128k | 128k | 1 | 500 | 1 | 1 | 2.57 | 2.55 |
| read | 128k | 128k | 1 | 500 | 1 | 0 | 3.89 | 4.14 |
| read | 1m | 1m | 1 | 500 | 1 | 1 | 16.43 | 17.34 |
| read | 1m | 1m | 1 | 500 | 1 | 0 | 24.86 | 26.75 |
| read | 1m | 50m | 1 | 500 | 1 | 1 | 118.11 | 140.55 |
| read | 1m | 50m | 1 | 500 | 1 | 0 | 138.56 | 208.63 |
| read | 1m | 1g | 1 | 20 | 1 | 1 | 157.38 | 194.52 |
| read | 1m | 1g | 1 | 20 | 1 | 0 | 180.12 | 278.23 |
| randread | 1m | 1g | 1 | 15 | 4 | 0 | 20.33 | 24.36 |
| randread | 1m | 1g | 1 | 15 | 1 | 1 | 20.75 | 23.09 |
| randread | 1m | 1g | 1 | 15 | 4 | 1 | 21.97 | 25.15 |
| randread | 1m | 1g | 1 | 15 | 1 | 0 | 15.23 | 17.51 |
| randread | 4m | 1g | 1 | 15 | 1 | 1 | 64.87 | 75.40 |
| randread | 4m | 1g | 1 | 15 | 1 | 0 | 57.86 | 69.82 |
| randread | 4m | 1g | 1 | 15 | 4 | 1 | 66.30 | 79.01 |
| randread | 4m | 1g | 1 | 15 | 4 | 0 | 59.56 | 68.44 |
| randread | 16m | 1g | 1 | 15 | 1 | 1 | 105.47 | 114.60 |
| randread | 16m | 1g | 1 | 15 | 1 | 0 | 104.10 | 109.72 |
| randread | 16m | 1g | 1 | 15 | 4 | 1 | 105.42 | 116.64 |
| randread | 16m | 1g | 1 | 15 | 4 | 0 | 105.52 | 110.05 |

---

## 2. Parallel Read Workloads (NumJobs = 48)


| Io Type | Block Size | File Size | Num Jobs | Num Files | Io Depth | Direct | HTTP Read Bw (MB/s) | gRPC Read Bw (MB/s) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| read | 4k | 4k | 48 | 500 | 1 | 1 | 4.14 | 4.10 |
| read | 4k | 4k | 48 | 500 | 1 | 0 | 6.55 | 6.08 |
| read | 128k | 128k | 48 | 500 | 1 | 1 | 131.17 | 128.71 |
| read | 128k | 128k | 48 | 500 | 1 | 0 | 203.30 | 200.61 |
| read | 1m | 1m | 48 | 500 | 1 | 1 | 850.02 | 883.44 |
| read | 1m | 1m | 48 | 500 | 1 | 0 | 1332.21 | 1305.24 |
| read | 1m | 50m | 48 | 500 | 1 | 1 | 5696.50 | 7126.20 |
| read | 1m | 50m | 48 | 500 | 1 | 0 | 7704.17 | 10782.95 |
| read | 1m | 1g | 48 | 20 | 1 | 1 | 6200.94 | 9164.57 |
| read | 1m | 1g | 48 | 20 | 1 | 0 | 8730.94 | 12547.74 |
| randread | 1m | 1g | 48 | 15 | 4 | 0 | 1038.20 | 1067.21 |
| randread | 1m | 1g | 48 | 15 | 1 | 1 | 1198.31 | 1072.71 |
| randread | 1m | 1g | 48 | 15 | 4 | 1 | 1151.46 | 1146.52 |
| randread | 1m | 1g | 48 | 15 | 1 | 0 | 787.62 | 809.60 |
| randread | 4m | 1g | 48 | 15 | 1 | 1 | 3415.72 | 3836.66 |
| randread | 4m | 1g | 48 | 15 | 1 | 0 | 2988.02 | 3193.09 |
| randread | 4m | 1g | 48 | 15 | 4 | 1 | 3328.73 | 3792.83 |
| randread | 4m | 1g | 48 | 15 | 4 | 0 | 2963.69 | 3178.80 |
| randread | 16m | 1g | 48 | 15 | 1 | 1 | 4518.64 | 4909.91 |
| randread | 16m | 1g | 48 | 15 | 1 | 0 | 4506.98 | 4600.54 |
| randread | 16m | 1g | 48 | 15 | 4 | 1 | 4604.31 | 5020.27 |
| randread | 16m | 1g | 48 | 15 | 4 | 0 | 4210.09 | 4890.98 |

<details>
  <summary>Fio configuration for read benchmarks</summary>

##### Fio templated configuration
```ini
[global]
ioengine=libaio
fadvise_hint=0
fallocate=none
invalidate=1
thread=1◊
openfiles=1
group_reporting=1
create_serialize=0
allrandrepeat=0
file_service_type=random
rw=${IO_TYPE}
iodepth=${IO_DEPTH}
direct=${DIRECT}
filename_format=$jobname.$jobnum.$filenum.size-${FILE_SIZE}

[test]
directory=${TEST_DATA_DIR}
filesize=${FILE_SIZE}
bs=${BS}
numjobs=${THREADS}
nrfiles=${NRFILES}
```
</details>

---

## 3. Write Workloads

These benchmarks evaluate GCSFuse's sequential write performance using both single-threaded and parallel workloads.

| Io Type | Block Size | File Size | Num Jobs | Num Files | Io Depth | Direct | HTTP Write Bw (MB/s) | gRPC Write Bw (MB/s) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| write | 4k | 4k | 1 | 500 | 1 | 0 | 0.04 | 0.04 |
| write | 4k | 4k | 48 | 500 | 1 | 0 | 0.16 | 0.16 |
| write | 16k | 128k | 48 | 500 | 1 | 0 | 5.02 | 5.08 |
| write | 16k | 128k | 1 | 500 | 1 | 0 | 1.22 | 1.16 |
| write | 1m | 1m | 1 | 500 | 1 | 0 | 8.47 | 7.50 |
| write | 1m | 1m | 48 | 500 | 1 | 0 | 40.54 | 38.39 |
| write | 1m | 50m | 1 | 200 | 1 | 0 | 52.82 | 47.46 |
| write | 1m | 50m | 48 | 200 | 1 | 0 | 2216.97 | 2098.04 |
| write | 1m | 1g | 1 | 15 | 1 | 0 | 66.79 | 60.49 |
| write | 1m | 1g | 48 | 15 | 1 | 0 | 2183.00 | 2374.04 |

<details>
  <summary>Fio configuration for write benchmarks</summary>

##### Fio templated configuration
```ini
[global]
ioengine=libaio
fadvise_hint=0
invalidate=1
thread=1
openfiles=1
group_reporting=1
allrandrepeat=1
verify=0
invalidate=1
file_append=0
create_on_open=1
end_fsync=1
rw=${IO_TYPE}
iodepth=${IO_DEPTH}
direct=${DIRECT}
filename_format=$jobname.$jobnum.$filenum.size-${FILE_SIZE}

[test]
directory=${TEST_DATA_DIR}
filesize=${FILE_SIZE}
bs=${BS}
numjobs=${THREADS}
nrfiles=${NRFILES}
```
</details>
