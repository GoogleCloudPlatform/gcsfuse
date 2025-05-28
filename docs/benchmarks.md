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
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.781 | 6.40K | 16.83ms | 0.00ms | 16.83ms |
| 256K | 128K | 30 | 1.614 | 13.22K | 8.71ms | 0.03ms | 8.74ms |
| 1MB | 1M | 30 | 5.130 | 5.25K | 21.17ms | 0.00ms | 21.17ms |
| 5MB | 1M | 20 | 11.770 | 12.05K | 9.11ms | 5.83ms | 14.93ms |
| 10MB | 1M | 20 | 14.237 | 14.58K | 7.54ms | 19.03ms | 26.57ms |
| 50MB | 1M | 20 | 14.341 | 14.69K | 7.14ms | 130.89ms | 138.03ms |
| 100MB | 1M | 10 | 13.821 | 14.15K | 6.76ms | 243.77ms | 250.53ms |
| 200MB | 1M | 10 | 14.114 | 14.45K | 6.63ms | 307.81ms | 314.44ms |
| 1GB | 1M | 10 | 13.526 | 13.85K | 6.89ms | 411.91ms | 418.81ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.702 | 5.75K | 18.14ms | 0.00ms | 18.14ms |
| 256K | 128K | 30 | 0.957 | 7.84K | 13.80ms | 0.64ms | 14.44ms |
| 1MB | 1M | 30 | 4.889 | 5.01K | 21.70ms | 0.00ms | 21.70ms |
| 5MB | 1M | 20 | 4.988 | 5.11K | 20.87ms | 22.79ms | 43.67ms |
| 10MB | 1M | 20 | 5.067 | 5.19K | 22.42ms | 77.91ms | 100.33ms |
| 50MB | 1M | 20 | 4.287 | 4.39K | 28.36ms | 647.03ms | 675.40ms |
| 100MB | 1M | 10 | 4.104 | 4.20K | 29.81ms | 1234.70ms | 1264.51ms |
| 200MB | 1M | 10 | 3.881 | 3.97K | 31.63ms | 1658.25ms | 1689.88ms |
| 1GB | 1M | 10 | 2.365 | 2.42K | 51.86ms | 3130.62ms | 3182.48ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.019 | 1.22K | 0.00ms | 0.09ms | 0.09ms |
| 1M | 1M | 30 | 0.074 | 0.08K | 0.00ms | 0.70ms | 0.75ms |
| 50M | 1M | 20 | 3.604 | 3.69K | 0.00ms | 1.14ms | 1.16ms |
| 100M | 1M | 10 | 4.859 | 4.98K | 0.00ms | 5.35ms | 5.37ms |
| 1G | 1M | 2 | 2.398 | 2.46K | 0.00ms | 33.49ms | 33.52ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.704 | 5.77K | 17.56ms | 0.00ms | 17.57ms |
| 256K | 128K | 30 | 1.460 | 11.96K | 9.24ms | 0.04ms | 9.28ms |
| 1MB | 1M | 30 | 4.218 | 4.32K | 23.31ms | 0.00ms | 23.31ms |
| 5MB | 1M | 20 | 5.418 | 5.55K | 17.23ms | 17.16ms | 34.39ms |
| 10MB | 1M | 20 | 6.591 | 6.75K | 17.23ms | 54.88ms | 72.11ms |
| 50MB | 1M | 20 | 6.809 | 6.97K | 17.47ms | 399.08ms | 416.56ms |
| 100MB | 1M | 10 | 6.203 | 6.35K | 17.27ms | 715.21ms | 732.48ms |
| 200MB | 1M | 10 | 6.330 | 6.48K | 17.53ms | 910.93ms | 928.46ms |
| 1GB | 1M | 10 | 6.803 | 6.97K | 17.39ms | 1057.62ms | 1075.01ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.741 | 6.07K | 18.03ms | 0.00ms | 18.04ms |
| 256K | 128K | 30 | 0.979 | 8.02K | 13.44ms | 0.62ms | 14.07ms |
| 1MB | 1M | 30 | 3.918 | 4.01K | 23.60ms | 0.00ms | 23.60ms |
| 5MB | 1M | 20 | 3.595 | 3.68K | 26.11ms | 27.85ms | 53.96ms |
| 10MB | 1M | 20 | 3.022 | 3.09K | 33.69ms | 117.79ms | 151.49ms |
| 50MB | 1M | 20 | 3.687 | 3.78K | 32.37ms | 737.52ms | 769.89ms |
| 100MB | 1M | 10 | 3.449 | 3.53K | 34.17ms | 1410.70ms | 1444.87ms |
| 200MB | 1M | 10 | 3.498 | 3.58K | 34.13ms | 1770.72ms | 1804.84ms |
| 1GB | 1M | 10 | 2.298 | 2.35K | 53.19ms | 3216.83ms | 3270.02ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.018 | 1.20K | 0.00ms | 0.11ms | 0.11ms |
| 1M | 1M | 30 | 0.073 | 0.07K | 0.00ms | 0.99ms | 1.06ms |
| 50M | 1M | 20 | 3.393 | 3.47K | 0.00ms | 1.71ms | 1.76ms |
| 100M | 1M | 10 | 3.429 | 3.51K | 0.00ms | 10.48ms | 10.54ms |
| 1G | 1M | 2 | 0.590 | 0.60K | 0.00ms | 171.58ms | 171.64ms |


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