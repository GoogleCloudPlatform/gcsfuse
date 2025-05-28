# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
GCSFuse. Below tables shows performance metrics of GCSFuse for different
workloads for the given test setup:

## Test setup:

* Infra: GCP VM
* OS: ubuntu-20.04
* Framework: FIO (version 3.39)
* GCSFuse version: master

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
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.679 | 5.57K | 17.47ms |
| 256K | 128K | 30 | 1.479 | 12.11K | 8.92ms |
| 1MB | 1M | 30 | 5.074 | 5.20K | 21.56ms |
| 5MB | 1M | 20 | 12.231 | 12.52K | 14.18ms |
| 10MB | 1M | 20 | 15.244 | 15.61K | 23.39ms |
| 50MB | 1M | 20 | 17.436 | 17.85K | 134.23ms |
| 100MB | 1M | 10 | 16.234 | 16.62K | 240.22ms |
| 200MB | 1M | 10 | 14.998 | 15.36K | 313.32ms |
| 1GB | 1M | 2 | 12.344 | 12.64K | 473.47ms |

### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.636 | 5.21K | 18.74ms |
| 256K | 128K | 30 | 0.966 | 7.92K | 14.40ms |
| 1MB | 1M | 30 | 4.967 | 5.09K | 21.46ms |
| 5MB | 1M | 20 | 5.533 | 5.67K | 40.90ms |
| 10MB | 1M | 20 | 5.163 | 5.29K | 99.89ms |
| 50MB | 1M | 20 | 3.608 | 3.69K | 733.98ms |
| 100MB | 1M | 10 | 3.600 | 3.69K | 1346.87ms |
| 200MB | 1M | 10 | 2.833 | 2.90K | 2130.03ms |
| 1GB | 1M | 2 | 2.886 | 2.95K | 2523.21ms |

### Sequential Writes
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.019 | 1.25K | 0.06ms |
| 1M | 1M | 30 | 0.076 | 0.08K | 0.74ms |
| 50M | 1M | 20 | 3.637 | 3.72K | 0.96ms |
| 100M | 1M | 10 | 5.051 | 5.17K | 3.03ms |
| 1G | 1M | 2 | 3.015 | 3.09K | 25.36ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.735 | 6.02K | 17.59ms |
| 256K | 128K | 30 | 1.534 | 12.57K | 8.88ms |
| 1MB | 1M | 30 | 3.890 | 3.98K | 23.74ms |
| 5MB | 1M | 20 | 6.514 | 6.67K | 32.94ms |
| 10MB | 1M | 20 | 6.495 | 6.65K | 71.25ms |
| 50MB | 1M | 20 | 6.871 | 7.04K | 395.47ms |
| 100MB | 1M | 10 | 5.971 | 6.11K | 630.02ms |
| 200MB | 1M | 10 | 6.897 | 7.06K | 875.07ms |
| 1GB | 1M | 2 | 6.361 | 6.51K | 1051.87ms |

### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.694 | 5.69K | 18.67ms |
| 256K | 128K | 30 | 0.968 | 7.93K | 14.26ms |
| 1MB | 1M | 30 | 3.834 | 3.93K | 24.00ms |
| 5MB | 1M | 20 | 3.841 | 3.93K | 56.43ms |
| 10MB | 1M | 20 | 2.576 | 2.64K | 158.10ms |
| 50MB | 1M | 20 | 3.037 | 3.11K | 899.11ms |
| 100MB | 1M | 10 | 3.064 | 3.14K | 1538.63ms |
| 200MB | 1M | 10 | 2.737 | 2.80K | 2252.88ms |
| 1GB | 1M | 2 | 2.735 | 2.80K | 2681.07ms |

### Sequential Writes
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.019 | 1.25K | 0.06ms |
| 1M | 1M | 30 | 0.075 | 0.08K | 0.88ms |
| 50M | 1M | 20 | 3.516 | 3.60K | 1.08ms |
| 100M | 1M | 10 | 4.193 | 4.29K | 4.14ms |
| 1G | 1M | 2 | 0.912 | 0.93K | 103.90ms |


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