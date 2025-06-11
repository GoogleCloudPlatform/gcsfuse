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
| 128K | 128K | 30 | 0.849 | 6.48K | 16.40ms |
| 256K | 128K | 30 | 1.730 | 13.20K | 8.53ms |
| 1MB | 1M | 30 | 5.679 | 5.42K | 20.71ms |
| 5MB | 1M | 20 | 11.148 | 10.63K | 11.10ms |
| 10MB | 1M | 20 | 14.417 | 13.75K | 9.13ms |
| 50MB | 1M | 20 | 19.147 | 18.26K | 12.43ms |
| 100MB | 1M | 10 | 14.870 | 14.18K | 28.02ms |
| 200MB | 1M | 10 | 19.736 | 18.82K | 33.90ms |
| 1GB | 1M | 10 | 19.670 | 18.76K | 51.06ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 848.763 | 6.48K | 16.87ms |
| 256K | 128K | 30 | 821.741 | 6.27K | 14.90ms |
| 1MB | 1M | 30 | 5639.400 | 5.38K | 20.99ms |
| 5MB | 1M | 20 | 5311.347 | 5.07K | 26.75ms |
| 10MB | 1M | 20 | 5059.093 | 4.82K | 33.67ms |
| 50MB | 1M | 20 | 4028.385 | 3.84K | 81.14ms |
| 100MB | 1M | 10 | 4057.123 | 3.87K | 186.50ms |
| 200MB | 1M | 10 | 3566.869 | 3.40K | 252.39ms |
| 1GB | 1M | 10 | 2601.852 | 2.48K | 464.79ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 244.872 | 14.95K | 3.09ms |
| 1M | 1M | 30 | 849.992 | 0.81K | 58.18ms |
| 50M | 1M | 20 | 3849.751 | 3.67K | 13.98ms |
| 100M | 1M | 10 | 3945.723 | 3.76K | 15.50ms |
| 1G | 1M | 2 | 2062.498 | 1.97K | 46.05ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.843 | 6.43K | 17.71ms |
| 256K | 128K | 30 | 1.568 | 11.96K | 8.99ms |
| 1MB | 1M | 30 | 4.459 | 4.25K | 23.00ms |
| 5MB | 1M | 20 | 6.844 | 6.53K | 18.33ms |
| 10MB | 1M | 20 | 7.419 | 7.08K | 20.72ms |
| 50MB | 1M | 20 | 7.608 | 7.26K | 34.26ms |
| 100MB | 1M | 10 | 7.439 | 7.09K | 80.85ms |
| 200MB | 1M | 10 | 6.823 | 6.51K | 93.66ms |
| 1GB | 1M | 10 | 7.722 | 7.36K | 148.45ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 785.205 | 5.99K | 17.84ms |
| 256K | 128K | 30 | 957.786 | 7.31K | 14.37ms |
| 1MB | 1M | 30 | 4329.604 | 4.13K | 22.87ms |
| 5MB | 1M | 20 | 3951.066 | 3.77K | 30.44ms |
| 10MB | 1M | 20 | 3744.914 | 3.57K | 42.27ms |
| 50MB | 1M | 20 | 4013.448 | 3.83K | 77.93ms |
| 100MB | 1M | 10 | 3771.856 | 3.60K | 193.84ms |
| 200MB | 1M | 10 | 3481.066 | 3.32K | 264.10ms |
| 1GB | 1M | 10 | 2649.078 | 2.53K | 462.64ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 232.771 | 14.21K | 3.42ms |
| 1M | 1M | 30 | 863.322 | 0.82K | 58.85ms |
| 50M | 1M | 20 | 3153.019 | 3.01K | 16.20ms |
| 100M | 1M | 10 | 2886.864 | 2.75K | 20.42ms |
| 1G | 1M | 2 | 173.222 | 0.17K | 660.83ms |


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