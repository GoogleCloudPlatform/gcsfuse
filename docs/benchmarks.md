# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
GCSFuse. Below tables shows performance metrics of GCSFuse for different
workloads for the given test setup:

## Test setup:

* Infra: GCP VM
* OS: ubuntu-20.04
* Framework: FIO (version 3.39)
* GCSFuse version: change-default-configs-for-streaming-writes

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
| 128K | 128K | 30 | 0.735 | 6.02K | 16.27ms |
| 256K | 128K | 30 | 1.552 | 12.72K | 8.51ms |
| 1MB | 1M | 30 | 5.304 | 5.43K | 20.96ms |
| 5MB | 1M | 20 | 10.073 | 10.31K | 16.02ms |
| 10MB | 1M | 20 | 14.493 | 14.84K | 23.74ms |
| 50MB | 1M | 20 | 18.477 | 18.92K | 133.99ms |
| 100MB | 1M | 10 | 16.569 | 16.97K | 235.64ms |
| 200MB | 1M | 10 | 17.670 | 18.09K | 303.14ms |
| 1GB | 1M | 2 | 16.020 | 16.40K | 362.72ms |

### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.691 | 5.66K | 17.88ms |
| 256K | 128K | 30 | 0.974 | 7.98K | 13.89ms |
| 1MB | 1M | 30 | 4.705 | 4.82K | 21.48ms |
| 5MB | 1M | 20 | 5.200 | 5.32K | 39.29ms |
| 10MB | 1M | 20 | 5.262 | 5.39K | 91.53ms |
| 50MB | 1M | 20 | 4.030 | 4.13K | 693.41ms |
| 100MB | 1M | 10 | 3.554 | 3.64K | 1391.84ms |
| 200MB | 1M | 10 | 3.158 | 3.23K | 1970.93ms |
| 1GB | 1M | 2 | 3.090 | 3.16K | 2454.43ms |

### Sequential Writes
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.018 | 1.18K | 0.06ms |
| 1M | 1M | 30 | 0.067 | 0.07K | 0.71ms |
| 50M | 1M | 20 | 3.659 | 3.75K | 0.94ms |
| 100M | 1M | 10 | 2.503 | 2.56K | 3.22ms |
| 1G | 1M | 2 | 3.120 | 3.20K | 21.74ms |

 
## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+ tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.784 | 6.42K | 16.47ms |
| 256K | 128K | 30 | 1.515 | 12.41K | 8.54ms |
| 1MB | 1M | 30 | 4.139 | 4.24K | 22.41ms |
| 5MB | 1M | 20 | 5.933 | 6.07K | 33.49ms |
| 10MB | 1M | 20 | 6.708 | 6.87K | 70.96ms |
| 50MB | 1M | 20 | 6.214 | 6.36K | 393.72ms |
| 100MB | 1M | 10 | 6.237 | 6.39K | 704.54ms |
| 200MB | 1M | 10 | 6.873 | 7.04K | 904.78ms |
| 1GB | 1M | 2 | 6.178 | 6.33K | 1058.68ms |

### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30 | 0.739 | 6.06K | 17.55ms |
| 256K | 128K | 30 | 0.897 | 7.35K | 13.88ms |
| 1MB | 1M | 30 | 4.006 | 4.10K | 23.02ms |
| 5MB | 1M | 20 | 3.935 | 4.03K | 54.28ms |
| 10MB | 1M | 20 | 3.201 | 3.28K | 149.23ms |
| 50MB | 1M | 20 | 3.683 | 3.77K | 762.47ms |
| 100MB | 1M | 10 | 3.634 | 3.72K | 1372.29ms |
| 200MB | 1M | 10 | 3.439 | 3.52K | 1849.72ms |
| 1GB | 1M | 2 | 3.393 | 3.47K | 2195.99ms |

### Sequential Writes
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 256K | 16K | 30 | 0.020 | 1.29K | 0.06ms |
| 1M | 1M | 30 | 0.077 | 0.08K | 0.86ms |
| 50M | 1M | 20 | 3.535 | 3.62K | 1.09ms |
| 100M | 1M | 10 | 4.268 | 4.37K | 4.00ms |
| 1G | 1M | 2 | 0.981 | 1.00K | 93.29ms |

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