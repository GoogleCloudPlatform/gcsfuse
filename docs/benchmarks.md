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
| 128K | 128K | 30 | 0.791 | 6.04K | 16.52ms |
| 256K | 128K | 30 | 1.878 | 14.33K | 7.85ms |
| 1MB | 1M | 30 | 6.092 | 5.81K | 19.20ms |
| 5MB | 1M | 20 | 11.166 | 10.65K | 10.84ms |
| 10MB | 1M | 20 | 12.912 | 12.31K | 10.39ms |
| 50MB | 1M | 20 | 19.033 | 18.15K | 12.66ms |
| 100MB | 1M | 10 | 16.132 | 15.38K | 26.86ms |
| 200MB | 1M | 10 | 18.945 | 18.07K | 32.89ms |
| 1GB | 1M | 10 | 13.041 | 12.44K | 46.60ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 788.897 | 6.02K | 17.23ms |
| 256K | 128K | 30 | 1118.481 | 8.53K | 13.26ms |
| 1MB | 1M | 30 | 6000.793 | 5.72K | 19.78ms |
| 5MB | 1M | 20 | 5338.812 | 5.09K | 25.89ms |
| 10MB | 1M | 20 | 5605.251 | 5.35K | 30.16ms |
| 50MB | 1M | 20 | 4680.980 | 4.46K | 68.13ms |
| 100MB | 1M | 10 | 4290.848 | 4.09K | 178.68ms |
| 200MB | 1M | 10 | 4057.490 | 3.87K | 237.73ms |
| 1GB | 1M | 10 | 2666.191 | 2.54K | 455.52ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 257.922 | 15.74K | 2.67ms |
| 1M | 1M | 30 | 964.208 | 0.92K | 49.18ms |
| 50M | 1M | 20 | 3809.911 | 3.63K | 13.78ms |
| 100M | 1M | 10 | 4076.946 | 3.89K | 16.57ms |
| 1G | 1M | 2 | 3501.689 | 3.34K | 9.45ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.748 | 5.71K | 17.93ms |
| 256K | 128K | 30 | 1.760 | 13.43K | 8.17ms |
| 1MB | 1M | 30 | 3.468 | 3.31K | 23.37ms |
| 5MB | 1M | 20 | 6.084 | 5.80K | 18.69ms |
| 10MB | 1M | 20 | 7.302 | 6.96K | 20.89ms |
| 50MB | 1M | 20 | 7.507 | 7.16K | 35.63ms |
| 100MB | 1M | 10 | 7.295 | 6.96K | 77.80ms |
| 200MB | 1M | 10 | 7.083 | 6.75K | 96.66ms |
| 1GB | 1M | 10 | 7.613 | 7.26K | 132.52ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 802.738 | 6.12K | 17.05ms |
| 256K | 128K | 30 | 985.928 | 7.52K | 14.21ms |
| 1MB | 1M | 30 | 4439.396 | 4.23K | 22.75ms |
| 5MB | 1M | 20 | 4040.269 | 3.85K | 32.30ms |
| 10MB | 1M | 20 | 3723.616 | 3.55K | 43.01ms |
| 50MB | 1M | 20 | 3860.603 | 3.68K | 79.59ms |
| 100MB | 1M | 10 | 3777.801 | 3.60K | 198.15ms |
| 200MB | 1M | 10 | 3783.819 | 3.61K | 259.07ms |
| 1GB | 1M | 10 | 2536.874 | 2.42K | 466.11ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 260.902 | 15.92K | 2.80ms |
| 1M | 1M | 30 | 954.284 | 0.91K | 51.50ms |
| 50M | 1M | 20 | 3166.878 | 3.02K | 15.49ms |
| 100M | 1M | 10 | 2835.631 | 2.70K | 21.63ms |
| 1G | 1M | 2 | 2502.955 | 2.39K | 14.02ms |


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