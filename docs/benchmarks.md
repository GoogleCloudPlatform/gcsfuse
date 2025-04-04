# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
GCSFuse. Below tables shows performance metrics of GCSFuse for different
workloads for the given test setup:

## Test setup:

* Infra: GCP VM
* OS: ubuntu-20.04
* Framework: FIO (version 3.39)
* GCSFuse version: 2.11.1

## FIO workloads

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
# Change this to randread to test random reads.
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
**Note:** Benchmarking is done by writing out new files to GCS. Performance
numbers will be different for edits/appends to existing files.

**Note:** Random writes and sequential write performance will generally be the same, as
all writes are first staged to a local temporary directory before being written
to GCS on close/fsync.

## GCSFuse Benchmarking on c4 machine-type
* VM Type: c4-standard-96
* VM location: us-south1
* Networking: gVNIC+  tier_1 networking (200Gbps)
* Disk Type: Hyperdisk balanced 
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles |Bandwidth in (GiB/sec) | IOPs  |  Avg Latency (msec) |
|---|---|---|---|---|---|
| 128K | 128K | 30  | 0.45 |  3650 | 30  |
| 256K  | 128K  | 30  | 0.81 | 6632 | 16 |
| 1M | 1M | 30 | 2.83 | 2902  | 38 |
| 5M | 1M  | 20  | 6.72  | 6874 | 17 |
| 10M | 1M | 20 | 9.33  | 9548 | 15 |
| 50M | 1M | 20 | 15.6 |15.9k | 14 |
| 100M |1M | 10 | 13.2 | 13.5k | 33 |
| 200M  | 1M | 10  | 12.4 |  12.7k| 38 |
| 1G | 1M | 10 | 14.5 | 14.8k  | 60 |



### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (MiB/sec) | IOPs  |  Avg Latency (msec)  |
|---|---|---|---|---|---|
| 256K  | 128K  | 30  | 626 | 5009   | 24 |
| 5M | 1M  | 20  | 4291 | 4290 | 30 |
| 10M | 1M | 20 | 4138 | 4137 | 37  |
| 50M | 1M | 20 | 3552 |3552  | 83 |
| 100M |1M | 10 | 3327 | 3327 | 211 |
| 200M  | 1M | 10  | 3139 | 3138 | 286 |
| 1G | 1M | 10 | 3320  | 3320 | 345 |


### Sequential Writes
| File Size | BlockSize | nrfiles |Bandwidth in (MiB/sec) | IOPs  |  Avg Latency (msec)  |
|---|---|---|---|---|---|
| 256K  | 16K  | 30  | 215 | 13.76k | 0.23 |
| 1M | 1M  | 30  |  718 | 717 | 1.12 |
| 50M | 1M | 20 | 3592 | 3592 | 2.35 |
| 100M |1M | 10 | 4549 | 4549 | 7.04 |
| 1G | 1M | 2 | 2398 | 2398 | 37.07  |

## GCSFuse Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+  tier_1 networking (100Gbps)
* Disk Type: SSD persistent disk  
* GCS Bucket location: us-south1
### Sequential Reads
| File Size | BlockSize | nrfiles |Bandwidth in (MiB/sec) | IOPs |  Avg Latency (msec)  |
|---|---|---|---|---|---|
| 128K | 128K | 30  |  443 | 3545 | 29 |
| 256K  | 128K  | 30  |  821 | 6569 | 16 |
| 1M | 1M | 30 | 2710 | 2709 | 40 |
| 5M | 1M  | 20  | 5666 | 5666 | 20 |
| 10M | 1M | 20 | 5994 | 5993 | 20 |
| 50M | 1M | 20 | 7986 | 7985 | 28 |
| 100M |1M | 10 | 6469 | 6468 | 68 |
| 200M  | 1M | 10  | 6955  | 6954 | 92 |
| 1G | 1M | 10 | 7470  | 7469 | 131 |



### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (MiB/sec) | IOPs  |  Avg Latency (msec)  |
|---|---|---|---|---|---|
| 256K  | 128K  | 30  | 562  | 4499 | 24  |
| 5M | 1M  | 20  | 3608 | 3607 | 34 |
| 10M | 1M | 20 | 3185 | 3184  | 45 |
| 50M | 1M | 20 | 3386  | 3386 | 84 |
| 100M |1M | 10 | 3297 | 3297 | 207 |
| 200M  | 1M | 10  | 3150 | 3150 | 279 |
| 1G | 1M | 10 | 2730 | 2730  | 457  |


### Sequential Writes
| File Size | BlockSize | nrfiles |Bandwidth in (MiB/sec) | IOPs  |  Avg Latency (msec)  |
|---|---|---|---|---|---|
| 256K  | 16K  | 30  | 192 | 12.27k | 0.27 |
| 1M | 1M  | 30  |  683 | 682 | 1.23 |
| 50M | 1M | 20 | 3429 | 3429 | 2.88 |
| 100M |1M | 10 | 3519 | 3518 | 11.83 |
| 1G | 1M | 2 | 1892 | 1891 | 45.40  |



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

7. Create a FIO job spec file.
   The FIO content referred to above. Please read the details about the FIO
   specification
   [here](https://fio.readthedocs.io/en/latest/).
    ```
    vi samplejobspec.fio
    ```

8. Run the FIO test using following command.

    ```
    DIR=<path-to-mount-point> fio samplejobspec.fio
    ```

9. Metrics will be displayed on the terminal after test is completed.