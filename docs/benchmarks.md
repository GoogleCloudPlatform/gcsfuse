# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
GCSFuse. Below tables shows performance metrics of GCSFuse for different
workloads for the given test setup:

## Test setup:

* Infra: GCP VM
* OS: ubuntu-20.04
* Framework: FIO (version 3.39)
* GCSFuse version: enable_read_manager_flag

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
| 128K | 128K | 30 | 0.828 | 6.32K | 16.20ms |
| 256K | 128K | 30 | 1.428 | 10.89K | 8.93ms |
| 1MB | 1M | 30 | 5.569 | 5.31K | 20.52ms |
| 5MB | 1M | 20 | 8.772 | 8.37K | 14.16ms |
| 10MB | 1M | 20 | 11.920 | 11.37K | 12.13ms |
| 50MB | 1M | 20 | 17.170 | 16.37K | 12.73ms |
| 100MB | 1M | 10 | 16.957 | 16.17K | 27.02ms |
| 200MB | 1M | 10 | 17.182 | 16.39K | 33.86ms |
| 1GB | 1M | 10 | 15.517 | 14.80K | 54.49ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 940.778 | 7.18K | 15.19ms |
| 256K | 128K | 30 | 997.654 | 7.61K | 14.96ms |
| 1MB | 1M | 30 | 4793.490 | 4.57K | 21.69ms |
| 5MB | 1M | 20 | 5144.413 | 4.91K | 26.56ms |
| 10MB | 1M | 20 | 5336.689 | 5.09K | 31.88ms |
| 50MB | 1M | 20 | 4499.572 | 4.29K | 70.25ms |
| 100MB | 1M | 10 | 4029.836 | 3.84K | 177.93ms |
| 200MB | 1M | 10 | 3801.233 | 3.63K | 246.28ms |
| 1GB | 1M | 10 | 2454.249 | 2.34K | 519.39ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 266.749 | 16.28K | 2.91ms |
| 1M | 1M | 30 | 924.971 | 0.88K | 54.90ms |
| 50M | 1M | 20 | 3697.982 | 3.53K | 15.26ms |
| 100M | 1M | 10 | 3873.112 | 3.69K | 16.83ms |
| 1G | 1M | 2 | 2221.056 | 2.12K | 41.95ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.757 | 5.77K | 18.11ms |
| 256K | 128K | 30 | 1.440 | 10.99K | 9.40ms |
| 1MB | 1M | 30 | 4.199 | 4.00K | 23.58ms |
| 5MB | 1M | 20 | 6.493 | 6.19K | 18.89ms |
| 10MB | 1M | 20 | 7.166 | 6.83K | 20.68ms |
| 50MB | 1M | 20 | 7.453 | 7.11K | 33.12ms |
| 100MB | 1M | 10 | 7.220 | 6.89K | 76.12ms |
| 200MB | 1M | 10 | 6.730 | 6.42K | 88.51ms |
| 1GB | 1M | 10 | 7.606 | 7.25K | 138.89ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 758.007 | 5.78K | 17.61ms |
| 256K | 128K | 30 | 893.197 | 6.81K | 14.88ms |
| 1MB | 1M | 30 | 4087.850 | 3.90K | 23.12ms |
| 5MB | 1M | 20 | 3630.450 | 3.46K | 33.95ms |
| 10MB | 1M | 20 | 4032.985 | 3.85K | 41.64ms |
| 50MB | 1M | 20 | 3684.264 | 3.51K | 83.52ms |
| 100MB | 1M | 10 | 3595.920 | 3.43K | 205.67ms |
| 200MB | 1M | 10 | 3353.850 | 3.20K | 283.69ms |
| 1GB | 1M | 10 | 1494.391 | 1.43K | 585.08ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 253.907 | 15.50K | 3.12ms |
| 1M | 1M | 30 | 908.514 | 0.87K | 55.94ms |
| 50M | 1M | 20 | 3095.346 | 2.95K | 18.69ms |
| 100M | 1M | 10 | 2719.097 | 2.59K | 23.61ms |
| 1G | 1M | 2 | 175.054 | 0.17K | 654.34ms |


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