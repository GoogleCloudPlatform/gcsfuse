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
| File Size | BlockSize | nrfiles | Bandwidth in (GB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.853 | 6.51K | 17.11ms |
| 256K | 128K | 30 | 1.810 | 13.81K | 8.24ms |
| 1MB | 1M | 30 | 2.747 | 2.62K | 29.46ms |
| 5MB | 1M | 20 | 10.176 | 9.70K | 11.60ms |
| 10MB | 1M | 20 | 14.448 | 13.78K | 9.66ms |
| 50MB | 1M | 20 | 14.346 | 13.68K | 12.28ms |
| 100MB | 1M | 10 | 17.449 | 16.64K | 26.29ms |
| 200MB | 1M | 10 | 13.975 | 13.33K | 34.58ms |
| 1GB | 1M | 10 | 19.565 | 18.66K | 51.77ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 798.915 | 6.10K | 17.37ms |
| 256K | 128K | 30 | 1124.730 | 8.58K | 13.44ms |
| 1MB | 1M | 30 | 5463.408 | 5.21K | 20.38ms |
| 5MB | 1M | 20 | 5317.660 | 5.07K | 25.89ms |
| 10MB | 1M | 20 | 4814.122 | 4.59K | 34.13ms |
| 50MB | 1M | 20 | 4116.856 | 3.93K | 78.18ms |
| 100MB | 1M | 10 | 3851.850 | 3.67K | 195.23ms |
| 200MB | 1M | 10 | 3450.904 | 3.29K | 274.62ms |
| 1GB | 1M | 10 | 2216.132 | 2.11K | 538.78ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|---|---|
| 256K | 16K | 30 | 264.268 | 16.13K | 2.98ms |
| 1M | 1M | 30 | 924.001 | 0.88K | 56.43ms |
| 50M | 1M | 20 | 3911.424 | 3.73K | 14.50ms |
| 100M | 1M | 10 | 4102.725 | 3.91K | 15.91ms |
| 1G | 1M | 2 | 2116.958 | 2.02K | 45.31ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.850 | 6.49K | 16.06ms |
| 256K | 128K | 30 | 1.394 | 10.64K | 8.67ms |
| 1MB | 1M | 30 | 3.979 | 3.79K | 21.46ms |
| 5MB | 1M | 20 | 6.837 | 6.52K | 18.44ms |
| 10MB | 1M | 20 | 7.179 | 6.85K | 20.30ms |
| 50MB | 1M | 20 | 7.527 | 7.18K | 33.18ms |
| 100MB | 1M | 10 | 7.227 | 6.89K | 73.70ms |
| 200MB | 1M | 10 | 7.279 | 6.94K | 96.41ms |
| 1GB | 1M | 10 | 7.655 | 7.30K | 127.85ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|---|---|
| 128K | 128K | 30 | 663.131 | 5.06K | 17.66ms |
| 256K | 128K | 30 | 1095.357 | 8.36K | 13.02ms |
| 1MB | 1M | 30 | 4225.112 | 4.03K | 22.40ms |
| 5MB | 1M | 20 | 3701.537 | 3.53K | 31.46ms |
| 10MB | 1M | 20 | 3587.750 | 3.42K | 42.94ms |
| 50MB | 1M | 20 | 3797.146 | 3.62K | 84.38ms |
| 100MB | 1M | 10 | 3669.256 | 3.50K | 206.60ms |
| 200MB | 1M | 10 | 3450.061 | 3.29K | 282.24ms |
| 1GB | 1M | 10 | 2257.131 | 2.15K | 545.64ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|---|---|
| 256K | 16K | 30 | 263.635 | 16.09K | 3.10ms |
| 1M | 1M | 30 | 946.846 | 0.90K | 55.98ms |
| 50M | 1M | 20 | 3031.270 | 2.89K | 18.62ms |
| 100M | 1M | 10 | 2854.794 | 2.72K | 21.54ms |
| 1G | 1M | 2 | 174.296 | 0.17K | 657.36ms |


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