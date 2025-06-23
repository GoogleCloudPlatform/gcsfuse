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
| 128K | 128K | 30 | 0.939 | 7.16K | 15.50ms |
| 256K | 128K | 30 | 1.807 | 13.79K | 7.97ms |
| 1MB | 1M | 30 | 6.010 | 5.73K | 19.09ms |
| 5MB | 1M | 20 | 12.532 | 11.95K | 10.16ms |
| 10MB | 1M | 20 | 15.763 | 15.03K | 9.06ms |
| 50MB | 1M | 20 | 17.850 | 17.02K | 12.07ms |
| 100MB | 1M | 10 | 14.818 | 14.13K | 26.72ms |
| 200MB | 1M | 10 | 18.483 | 17.63K | 34.38ms |
| 1GB | 1M | 10 | 9.446 | 9.01K | 59.64ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 1018.859 | 7.77K | 14.93ms |
| 256K | 128K | 30 | 1229.100 | 9.38K | 12.53ms |
| 1MB | 1M | 30 | 6411.675 | 6.11K | 18.67ms |
| 5MB | 1M | 20 | 5491.724 | 5.24K | 23.64ms |
| 10MB | 1M | 20 | 5383.784 | 5.13K | 30.31ms |
| 50MB | 1M | 20 | 4651.777 | 4.44K | 68.71ms |
| 100MB | 1M | 10 | 3969.412 | 3.79K | 180.57ms |
| 200MB | 1M | 10 | 3750.985 | 3.58K | 264.71ms |
| 1GB | 1M | 10 | 2585.558 | 2.47K | 475.66ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 281.947 | 17.21K | 2.47ms |
| 1M | 1M | 30 | 989.945 | 0.94K | 46.02ms |
| 50M | 1M | 20 | 3868.902 | 3.69K | 15.08ms |
| 100M | 1M | 10 | 4077.371 | 3.89K | 16.38ms |
| 1G | 1M | 2 | 2225.948 | 2.12K | 41.93ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles | Bandwidth in (GB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.910 | 6.94K | 15.71ms |
| 256K | 128K | 30 | 1.844 | 14.07K | 8.09ms |
| 1MB | 1M | 30 | 3.948 | 3.76K | 22.75ms |
| 5MB | 1M | 20 | 6.459 | 6.16K | 19.60ms |
| 10MB | 1M | 20 | 5.952 | 5.68K | 21.36ms |
| 50MB | 1M | 20 | 6.901 | 6.58K | 34.80ms |
| 100MB | 1M | 10 | 6.478 | 6.18K | 78.73ms |
| 200MB | 1M | 10 | 7.038 | 6.71K | 112.09ms |
| 1GB | 1M | 10 | 6.811 | 6.50K | 123.85ms |

### Random Reads
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 773.144 | 5.90K | 17.58ms |
| 256K | 128K | 30 | 1087.077 | 8.29K | 13.79ms |
| 1MB | 1M | 30 | 4459.061 | 4.25K | 22.08ms |
| 5MB | 1M | 20 | 3638.323 | 3.47K | 34.51ms |
| 10MB | 1M | 20 | 3308.708 | 3.16K | 46.76ms |
| 50MB | 1M | 20 | 3740.844 | 3.57K | 83.59ms |
| 100MB | 1M | 10 | 3649.303 | 3.48K | 193.13ms |
| 200MB | 1M | 10 | 3629.664 | 3.46K | 265.36ms |
| 1GB | 1M | 10 | 2310.423 | 2.20K | 521.91ms |

### Sequential Writes
| File Size | BlockSize | nrfiles | Bandwidth in (MB/sec) | IOPs | IOPs Avg Latency (ms) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 258.908 | 15.80K | 2.81ms |
| 1M | 1M | 30 | 939.524 | 0.90K | 52.23ms |
| 50M | 1M | 20 | 3408.123 | 3.25K | 18.22ms |
| 100M | 1M | 10 | 2772.504 | 2.64K | 23.23ms |
| 1G | 1M | 2 | 174.815 | 0.17K | 657.60ms |


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