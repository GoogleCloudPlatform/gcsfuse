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
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.770 | 6.31K | 17.21ms | 0.00ms | 17.22ms |
| 256K | 128K | 30 | 1.616 | 13.24K | 8.76ms | 0.03ms | 8.80ms |
| 1MB | 1M | 30 | 4.864 | 4.98K | 22.20ms | 0.00ms | 22.21ms |
| 5MB | 1M | 20 | 8.295 | 8.49K | 11.05ms | 6.70ms | 17.75ms |
| 10MB | 1M | 20 | 13.774 | 14.10K | 7.79ms | 18.44ms | 26.23ms |
| 50MB | 1M | 20 | 13.043 | 13.36K | 7.36ms | 131.79ms | 139.15ms |
| 100MB | 1M | 10 | 12.364 | 12.66K | 7.57ms | 252.49ms | 260.06ms |
| 200MB | 1M | 10 | 13.015 | 13.33K | 7.47ms | 341.91ms | 349.38ms |
| 1GB | 1M | 10 | 13.599 | 13.93K | 7.89ms | 469.07ms | 476.96ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.678 | 5.56K | 18.83ms | 0.00ms | 18.83ms |
| 256K | 128K | 30 | 0.922 | 7.55K | 14.07ms | 0.64ms | 14.72ms |
| 1MB | 1M | 30 | 4.624 | 4.73K | 22.26ms | 0.00ms | 22.27ms |
| 5MB | 1M | 20 | 4.309 | 4.41K | 25.28ms | 24.10ms | 49.39ms |
| 10MB | 1M | 20 | 3.596 | 3.68K | 28.43ms | 94.53ms | 122.96ms |
| 50MB | 1M | 20 | 3.278 | 3.36K | 36.11ms | 764.64ms | 800.75ms |
| 100MB | 1M | 10 | 3.113 | 3.19K | 38.64ms | 1499.11ms | 1537.75ms |
| 200MB | 1M | 10 | 3.010 | 3.08K | 39.93ms | 2029.30ms | 2069.22ms |
| 1GB | 1M | 10 | 1.818 | 1.86K | 67.38ms | 4079.86ms | 4147.24ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.019 | 1.24K | 0.00ms | 0.06ms | 0.06ms |
| 1M | 1M | 30 | 0.075 | 0.08K | 0.00ms | 0.71ms | 0.76ms |
| 50M | 1M | 20 | 3.485 | 3.57K | 0.00ms | 0.96ms | 0.98ms |
| 100M | 1M | 10 | 4.818 | 4.93K | 0.00ms | 3.13ms | 3.15ms |
| 1G | 1M | 2 | 2.434 | 2.49K | 0.00ms | 33.28ms | 33.31ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.720 | 5.90K | 18.04ms | 0.00ms | 18.05ms |
| 256K | 128K | 30 | 1.385 | 11.34K | 9.16ms | 0.03ms | 9.19ms |
| 1MB | 1M | 30 | 3.447 | 3.53K | 23.50ms | 0.00ms | 23.50ms |
| 5MB | 1M | 20 | 5.211 | 5.34K | 17.31ms | 13.76ms | 31.07ms |
| 10MB | 1M | 20 | 6.612 | 6.77K | 16.95ms | 50.57ms | 67.52ms |
| 50MB | 1M | 20 | 6.873 | 7.04K | 17.28ms | 389.87ms | 407.15ms |
| 100MB | 1M | 10 | 6.583 | 6.74K | 16.99ms | 696.81ms | 713.80ms |
| 200MB | 1M | 10 | 6.546 | 6.70K | 17.28ms | 895.10ms | 912.38ms |
| 1GB | 1M | 10 | 7.024 | 7.19K | 17.26ms | 1049.65ms | 1066.91ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.682 | 5.59K | 18.73ms | 0.00ms | 18.73ms |
| 256K | 128K | 30 | 0.901 | 7.38K | 14.18ms | 0.64ms | 14.82ms |
| 1MB | 1M | 30 | 3.713 | 3.80K | 24.21ms | 0.00ms | 24.22ms |
| 5MB | 1M | 20 | 3.817 | 3.91K | 27.75ms | 27.37ms | 55.12ms |
| 10MB | 1M | 20 | 3.149 | 3.22K | 33.29ms | 113.21ms | 146.50ms |
| 50MB | 1M | 20 | 3.477 | 3.56K | 34.39ms | 735.83ms | 770.23ms |
| 100MB | 1M | 10 | 3.197 | 3.27K | 37.17ms | 1454.88ms | 1492.05ms |
| 200MB | 1M | 10 | 2.809 | 2.88K | 43.34ms | 2194.08ms | 2237.42ms |
| 1GB | 1M | 10 | 1.915 | 1.96K | 63.38ms | 3840.20ms | 3903.58ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (GiB/sec) | IOPs | slat mean | clat mean | lat mean |
|---|---|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.019 | 1.26K | 0.00ms | 0.07ms | 0.07ms |
| 1M | 1M | 30 | 0.077 | 0.08K | 0.00ms | 0.92ms | 0.98ms |
| 50M | 1M | 20 | 3.505 | 3.59K | 0.00ms | 1.11ms | 1.16ms |
| 100M | 1M | 10 | 3.929 | 4.02K | 0.00ms | 4.63ms | 4.69ms |
| 1G | 1M | 2 | 0.713 | 0.73K | 0.00ms | 141.76ms | 141.82ms |


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