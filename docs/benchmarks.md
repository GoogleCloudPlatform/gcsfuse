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
| 128K | 128K | 30 | 0.324 | 2.47K | 16.39ms |
| 256K | 128K | 30 | 0.659 | 5.03K | 8.44ms |
| 1MB | 1M | 30 | 2.277 | 2.17K | 20.26ms |
| 5MB | 1M | 20 | 6.291 | 6.00K | 8.02ms |
| 10MB | 1M | 20 | 7.557 | 7.21K | 6.81ms |
| 50MB | 1M | 20 | 10.673 | 10.18K | 9.03ms |
| 100MB | 1M | 10 | 8.269 | 7.89K | 21.35ms |
| 200MB | 1M | 10 | 8.381 | 7.99K | 26.20ms |
| 1GB | 1M | 10 | 10.409 | 9.93K | 37.74ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 261.418 | 1.99K | 16.92ms |
| 256K | 128K | 30 | 413.458 | 3.15K | 13.65ms |
| 1MB | 1M | 30 | 2138.739 | 2.04K | 20.32ms |
| 5MB | 1M | 20 | 2233.983 | 2.13K | 23.59ms |
| 10MB | 1M | 20 | 2110.784 | 2.01K | 30.90ms |
| 50MB | 1M | 20 | 1771.680 | 1.69K | 69.63ms |
| 100MB | 1M | 10 | 1649.947 | 1.57K | 170.25ms |
| 200MB | 1M | 10 | 1552.081 | 1.48K | 232.90ms |
| 1GB | 1M | 10 | 1193.533 | 1.14K | 390.13ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 114.982 | 7.02K | 2.76ms |
| 1M | 1M | 30 | 407.984 | 0.39K | 51.17ms |
| 50M | 1M | 20 | 2134.053 | 2.04K | 6.85ms |
| 100M | 1M | 10 | 2512.437 | 2.40K | 6.64ms |
| 1G | 1M | 2 | 3260.453 | 3.11K | 9.19ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.332 | 2.54K | 17.29ms |
| 256K | 128K | 30 | 0.642 | 4.90K | 9.01ms |
| 1MB | 1M | 30 | 2.211 | 2.11K | 20.45ms |
| 5MB | 1M | 20 | 6.101 | 5.82K | 8.18ms |
| 10MB | 1M | 20 | 6.991 | 6.67K | 8.01ms |
| 50MB | 1M | 20 | 7.504 | 7.16K | 12.61ms |
| 100MB | 1M | 10 | 7.377 | 7.04K | 26.02ms |
| 200MB | 1M | 10 | 7.464 | 7.12K | 37.08ms |
| 1GB | 1M | 10 | 7.773 | 7.41K | 51.49ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 309.924 | 2.36K | 17.83ms |
| 256K | 128K | 30 | 405.029 | 3.09K | 14.23ms |
| 1MB | 1M | 30 | 2085.566 | 1.99K | 20.81ms |
| 5MB | 1M | 20 | 2169.468 | 2.07K | 21.13ms |
| 10MB | 1M | 20 | 2087.151 | 1.99K | 29.51ms |
| 50MB | 1M | 20 | 1720.034 | 1.64K | 69.37ms |
| 100MB | 1M | 10 | 1698.156 | 1.62K | 171.62ms |
| 200MB | 1M | 10 | 1599.150 | 1.53K | 227.46ms |
| 1GB | 1M | 10 | 1156.546 | 1.10K | 394.10ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 102.024 | 6.23K | 3.13ms |
| 1M | 1M | 30 | 373.103 | 0.36K | 56.82ms |
| 50M | 1M | 20 | 1691.593 | 1.61K | 7.06ms |
| 100M | 1M | 10 | 1425.866 | 1.36K | 7.62ms |
| 1G | 1M | 2 | 1925.131 | 1.84K | 11.24ms |


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