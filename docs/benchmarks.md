# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
GCSFuse. Below tables shows performance metrics of GCSFuse for different
workloads for the given test setup:

## Test setup:

* Infra: GCP VM
* OS: ubuntu-20.04
* Framework: FIO (version 3.39)
* GCSFuse version: v2.11.1

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
| 128K | 128K | 30 | 0.734 | 6.01K | 17.56ms |
| 256K | 128K | 30 | 1.597 | 13.08K | 8.86ms |
| 1MB | 1M | 30 | 5.034 | 5.15K | 22.05ms |
| 5MB | 1M | 20 | 8.356 | 8.56K | 16.06ms |
| 10MB | 1M | 20 | 15.654 | 16.03K | 24.05ms |
| 50MB | 1M | 20 | 16.270 | 16.66K | 134.57ms |
| 100MB | 1M | 10 | 15.283 | 15.65K | 235.96ms |
| 200MB | 1M | 10 | 12.332 | 12.63K | 340.10ms |
| 1GB | 1M | 2 | 10.983 | 11.25K | 390.11ms |

### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.713 | 5.84K | 17.99ms |
| 256K | 128K | 30 | 0.981 | 8.03K | 14.06ms |
| 1MB | 1M | 30 | 5.027 | 5.15K | 21.55ms |
| 5MB | 1M | 20 | 5.259 | 5.38K | 42.10ms |
| 10MB | 1M | 20 | 4.751 | 4.87K | 104.25ms |
| 50MB | 1M | 20 | 3.364 | 3.44K | 775.79ms |
| 100MB | 1M | 10 | 3.606 | 3.69K | 1354.41ms |
| 200MB | 1M | 10 | 2.875 | 2.94K | 2132.17ms |
| 1GB | 1M | 2 | 2.616 | 2.68K | 2676.60ms |

### Sequential Writes
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.019 | 1.22K | 0.09ms |
| 1M | 1M | 30 | 0.075 | 0.08K | 0.73ms |
| 50M | 1M | 20 | 3.630 | 3.72K | 1.16ms |
| 100M | 1M | 10 | 4.924 | 5.04K | 5.25ms |
| 1G | 1M | 2 | 2.486 | 2.55K | 33.34ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.742 | 6.08K | 17.78ms |
| 256K | 128K | 30 | 1.515 | 12.41K | 9.29ms |
| 1MB | 1M | 30 | 4.098 | 4.20K | 23.49ms |
| 5MB | 1M | 20 | 6.050 | 6.20K | 33.46ms |
| 10MB | 1M | 20 | 6.647 | 6.81K | 71.40ms |
| 50MB | 1M | 20 | 6.131 | 6.28K | 362.17ms |
| 100MB | 1M | 10 | 6.427 | 6.58K | 601.54ms |
| 200MB | 1M | 10 | 6.465 | 6.62K | 901.24ms |
| 1GB | 1M | 2 | 6.321 | 6.47K | 1056.84ms |

### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.662 | 5.42K | 19.05ms |
| 256K | 128K | 30 | 0.959 | 7.85K | 14.66ms |
| 1MB | 1M | 30 | 3.453 | 3.54K | 23.96ms |
| 5MB | 1M | 20 | 3.457 | 3.54K | 56.16ms |
| 10MB | 1M | 20 | 3.226 | 3.30K | 152.75ms |
| 50MB | 1M | 20 | 2.934 | 3.00K | 928.62ms |
| 100MB | 1M | 10 | 3.114 | 3.19K | 1598.25ms |
| 200MB | 1M | 10 | 2.696 | 2.76K | 2293.23ms |
| 1GB | 1M | 2 | 2.725 | 2.79K | 2707.27ms |

### Sequential Writes
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.018 | 1.20K | 0.10ms |
| 1M | 1M | 30 | 0.072 | 0.07K | 0.97ms |
| 50M | 1M | 20 | 3.383 | 3.46K | 1.39ms |
| 100M | 1M | 10 | 3.763 | 3.85K | 8.87ms |
| 1G | 1M | 2 | 0.608 | 0.62K | 167.73ms |


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