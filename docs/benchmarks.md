# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
GCSFuse. Below tables shows performance metrics of GCSFuse for different
workloads for the given test setup:

## Test setup:

* Infra: GCP VM
* OS: ubuntu-20.04
* Framework: FIO (version 3.39)
* GCSFuse version: v3.0.0

## FIO workloads
Please read the details about the FIO specification [here](https://fio.readthedocs.io/en/latest/).
### Reads 
  ```
[global]
allrandrepeat=0
create_serialize=0
direct=1
fadvise_hint=0
file_service_type=random
group_reporting=1
iodepth=64
ioengine=libaio
invalidate=1
numjobs=128
openfiles=1
# Change "read" to "randread" to test random reads.
rw=read 
thread=1
filename_format=$jobname.$jobnum/$filenum

[experiment]
stonewall
directory=${DIR}
# Update the block size value from the table for different experiments.
bs=128K
# Update the file size value from table(file size) for different experiments.
filesize=128K
# Set nrfiles per thread in such a way that the test runs for 1-2 min.
nrfiles=30
  ```
**Note:** Please note an update to our FIO read workload. This change accounts for the bandwidth difference between the current and [previous](https://github.com/GoogleCloudPlatform/gcsfuse/blob/26bc07f3dd210e05a7030954bb3e6070e957bfca/docs/benchmarks.md#sequential-read) n2 benchmarks.
### Writes
```
[global]
allrandrepeat=1
# By default fio creates all files first and then starts writing to them. This option is to disable that behavior. 
create_on_open=1
direct=1
fadvise_hint=0
file_append=0
group_reporting=1
iodepth=64
ioengine=sync
invalidate=1
numjobs=112
openfiles=1
rw=write
thread=1
time_based=0
verify=0
filename_format=$jobname.$jobnum.$filenum

 
[experiment]
stonewall
directory=${DIR}
# Every file is written only once. Set nrfiles per thread in such a way that the test runs for 1-2 min. 
# This will vary based on file size. 
nrfiles=30
# Update the file size value from table(file size) for different experiments.
filesize=256K
# Update the block size value from the table for different experiments.
bs=16K
```
**Note:** 
* Benchmarking is done by writing out new files to GCS. Performance
numbers will be different for edits/appends to existing files.

* Random writes and sequential write performance will generally be the same, as
all writes are first staged to a local temporary directory before being written
to GCS on close/fsync.

<!-- Benchmarks start -->

## GCSFuse Benchmarking on c4 machine-type
* VM Type: c4-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (200Gbps)
* Disk Type: Hyperdisk balanced
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.822 | 6.27K | 16.50ms |
| 256K | 128K | 30 | 1.637 | 12.49K | 8.42ms |
| 1MB | 1M | 30 | 5.670 | 5.41K | 20.09ms |
| 5MB | 1M | 20 | 12.357 | 11.78K | 9.80ms |
| 10MB | 1M | 20 | 17.290 | 16.49K | 8.45ms |
| 50MB | 1M | 20 | 16.767 | 15.99K | 13.05ms |
| 100MB | 1M | 10 | 15.908 | 15.17K | 28.15ms |
| 200MB | 1M | 10 | 16.101 | 15.36K | 36.28ms |
| 1GB | 1M | 10 | 17.623 | 16.81K | 51.08ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 794.532 | 6.06K | 16.92ms |
| 256K | 128K | 30 | 1028.940 | 7.85K | 13.96ms |
| 1MB | 1M | 30 | 5642.883 | 5.38K | 20.74ms |
| 5MB | 1M | 20 | 5173.230 | 4.93K | 26.45ms |
| 10MB | 1M | 20 | 4851.292 | 4.63K | 34.09ms |
| 50MB | 1M | 20 | 4224.960 | 4.03K | 74.34ms |
| 100MB | 1M | 10 | 3939.360 | 3.76K | 187.47ms |
| 200MB | 1M | 10 | 3799.176 | 3.62K | 252.86ms |
| 1GB | 1M | 10 | 3083.442 | 2.94K | 408.55ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 266.985 | 16.30K | 2.61ms |
| 1M | 1M | 30 | 939.672 | 0.90K | 49.40ms |
| 50M | 1M | 20 | 3869.910 | 3.69K | 15.67ms |
| 100M | 1M | 10 | 4012.270 | 3.83K | 16.31ms |
| 1G | 1M | 2 | 2105.270 | 2.01K | 45.28ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.841 | 6.42K | 16.05ms |
| 256K | 128K | 30 | 1.852 | 14.13K | 7.92ms |
| 1MB | 1M | 30 | 4.335 | 4.13K | 20.68ms |
| 5MB | 1M | 20 | 6.563 | 6.26K | 18.13ms |
| 10MB | 1M | 20 | 7.315 | 6.98K | 20.80ms |
| 50MB | 1M | 20 | 7.408 | 7.07K | 33.49ms |
| 100MB | 1M | 10 | 7.366 | 7.02K | 75.98ms |
| 200MB | 1M | 10 | 7.022 | 6.70K | 89.70ms |
| 1GB | 1M | 10 | 7.689 | 7.33K | 136.42ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 808.471 | 6.17K | 16.11ms |
| 256K | 128K | 30 | 1150.783 | 8.78K | 12.88ms |
| 1MB | 1M | 30 | 3921.740 | 3.74K | 21.94ms |
| 5MB | 1M | 20 | 4022.110 | 3.84K | 29.91ms |
| 10MB | 1M | 20 | 3509.580 | 3.35K | 42.20ms |
| 50MB | 1M | 20 | 3914.330 | 3.73K | 73.89ms |
| 100MB | 1M | 10 | 3556.750 | 3.39K | 211.13ms |
| 200MB | 1M | 10 | 3999.990 | 3.81K | 237.04ms |
| 1GB | 1M | 10 | 3090.847 | 2.95K | 403.28ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 269.107 | 16.43K | 2.72ms |
| 1M | 1M | 30 | 977.337 | 0.93K | 49.54ms |
| 50M | 1M | 20 | 3162.307 | 3.02K | 16.90ms |
| 100M | 1M | 10 | 2917.150 | 2.78K | 20.77ms |
| 1G | 1M | 2 | 177.634 | 0.17K | 645.42ms |


<!-- Benchmarks end -->

## Steps to benchmark GCSFuse performance

1. [Create](https://cloud.google.com/compute/docs/instances/create-start-instance#publicimage)
   a GCP VM instance. 
2. [Connect](https://cloud.google.com/compute/docs/instances/connecting-to-instance)
   to the VM instance.
3. Install FIO.

    ```
    sudo apt-get update
    sudo apt-get install fio
    ```

5. [Install GCSFuse](https://cloud.google.com/storage/docs/gcsfuse-install).
6. Create a directory on the VM and then mount the gcs bucket to that directory.

    ```
    mkdir <path-to-mount-point>
    gcsfuse <bucket-name> <path-to-mount-point>
    ```

7. Create a FIO job spec file.\
   The fio workload files can be found [above](#fio-workloads). 
    ```
    vi samplejobspec.fio
    ```

8. Run the FIO test using following command.

    ```
    DIR=<path-to-mount-point> fio samplejobspec.fio
    ```

9. Metrics will be displayed on the terminal after test is completed.