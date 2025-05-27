# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
GCSFuse. Below tables shows performance metrics of GCSFuse for different
workloads for the given test setup:

## Test setup:

* Infra: GCP VM
* OS: ubuntu-20.04
* Framework: FIO (version 3.39)
* GCSFuse version: PR:change-default-configs-for-streaming-writes

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

# Benchmarks start

## GCSFuse Benchmarking on c4 machine-type
* VM Type: c4-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (200Gbps)
* Disk Type: Hyperdisk balanced
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.735 | 6.02K | 0.00ms |
| 256K | 128K | 30 | 1.552 | 12.72K | 0.03ms |
| 1MB | 1M | 30 | 5.304 | 5.43K | 0.00ms |
| 5MB | 1M | 20 | 10.073 | 10.31K | 5.95ms |
| 10MB | 1M | 20 | 14.493 | 14.84K | 16.77ms |
| 50MB | 1M | 20 | 18.477 | 18.92K | 127.96ms |
| 100MB | 1M | 10 | 16.569 | 16.97K | 229.86ms |
| 200MB | 1M | 10 | 17.670 | 18.09K | 297.29ms |
| 1GB | 1M | 2 | 16.020 | 16.40K | 356.80ms |

### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.691 | 5.66K | 0.00ms |
| 256K | 128K | 30 | 0.974 | 7.98K | 0.60ms |
| 1MB | 1M | 30 | 4.705 | 4.82K | 0.00ms |
| 5MB | 1M | 20 | 5.200 | 5.32K | 19.53ms |
| 10MB | 1M | 20 | 5.262 | 5.39K | 70.63ms |
| 50MB | 1M | 20 | 4.030 | 4.13K | 663.78ms |
| 100MB | 1M | 10 | 3.554 | 3.64K | 1358.20ms |
| 200MB | 1M | 10 | 3.158 | 3.23K | 1933.01ms |
| 1GB | 1M | 2 | 3.090 | 3.16K | 2414.61ms |

### Sequential Writes
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.018 | 1.18K | 0.06ms |
| 1M | 1M | 30 | 0.067 | 0.07K | 0.66ms |
| 50M | 1M | 20 | 3.659 | 3.75K | 0.92ms |
| 100M | 1M | 10 | 2.503 | 2.56K | 3.20ms |
| 1G | 1M | 2 | 3.120 | 3.20K | 21.72ms |


## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.784 | 6.42K | 0.00ms |
| 256K | 128K | 30 | 1.515 | 12.41K | 0.03ms |
| 1MB | 1M | 30 | 4.139 | 4.24K | 0.00ms |
| 5MB | 1M | 20 | 5.933 | 6.07K | 16.82ms |
| 10MB | 1M | 20 | 6.708 | 6.87K | 54.35ms |
| 50MB | 1M | 20 | 6.214 | 6.36K | 377.13ms |
| 100MB | 1M | 10 | 6.237 | 6.39K | 687.87ms |
| 200MB | 1M | 10 | 6.873 | 7.04K | 887.72ms |
| 1GB | 1M | 2 | 6.178 | 6.33K | 1041.52ms |

### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.739 | 6.06K | 0.00ms |
| 256K | 128K | 30 | 0.897 | 7.35K | 0.60ms |
| 1MB | 1M | 30 | 4.006 | 4.10K | 0.00ms |
| 5MB | 1M | 20 | 3.935 | 4.03K | 28.82ms |
| 10MB | 1M | 20 | 3.201 | 3.28K | 116.61ms |
| 50MB | 1M | 20 | 3.683 | 3.77K | 730.80ms |
| 100MB | 1M | 10 | 3.634 | 3.72K | 1340.03ms |
| 200MB | 1M | 10 | 3.439 | 3.52K | 1814.87ms |
| 1GB | 1M | 2 | 3.393 | 3.47K | 2160.51ms |

### Sequential Writes
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.020 | 1.29K | 0.06ms |
| 1M | 1M | 30 | 0.077 | 0.08K | 0.80ms |
| 50M | 1M | 20 | 3.535 | 3.62K | 1.04ms |
| 100M | 1M | 10 | 4.268 | 4.37K | 3.94ms |
| 1G | 1M | 2 | 0.981 | 1.00K | 93.24ms |

# Benchmarks end

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