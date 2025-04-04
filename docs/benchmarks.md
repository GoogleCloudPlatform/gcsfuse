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
rw=<read/randread> 
thread=1
filename_format=$jobname.$jobnum/$filenum

[experiment]
stonewall
directory=${DIR}
bs=128K
filesize=128K
nrfiles=30
  ```
**Note:** Please note an update to our FIO read workload. This change accounts for the difference between the updated and existing n2 benchmarks.
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
# Every file is written only once. Set nrfiles per thread in such a way that the test runs for 1-2 min. 
# This will vary based on file size. 
nrfiles=2
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
nrfiles=30
filesize=256K
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
* Hyperdisk balanced 
* GCS Bucket location: us-south1

### Sequential Reads
| File Size | BlockSize | nrfiles |Bandwidth in (MiB/sec) | Avg Latency (msec) | IOPs  |
|---|---|---|---|---|---|
| 128K | 128K | 30  |  |  |  |
| 256K  | 128K  | 30  |  |  |  |
| 1M | 1M | 30 |  |  |  |
| 5M | 1M  | 20  |  |  |  |
| 10M | 1M | 20 |  |  |  |
| 50M | 1M | 20 |  |  |  |
| 100M |1M | 10 |  |  |  |
| 200M  | 1M | 10  |  |  |  |
| 1G | 1M | 10 |  |  |  |



### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (MiB/sec) | Avg Latency (msec) | IOPs  |
|---|---|---|---|---|---|
| 256K  | 128K  | 30  |  |  |  |
| 5M | 1M  | 20  |  |  |  |
| 10M | 1M | 20 |  |  |  |
| 50M | 1M | 20 |  |  |  |
| 100M |1M | 10 |  |  |  |
| 200M  | 1M | 10  |  |  |  |
| 1G | 1M | 10 |  |  |  |


### Sequential Writes

| File Size | BlockSize | nrfiles | Bandwidth in MiB/sec | IOPS(avg) | Avg Latency (msec) | Network Send Traffic (GiB/s) |
|-----------|-----------|---------|----------------------|-----------|--------------------|------------------------------|
| 256KB     | 16K       | 30      |                   |  |              |                         |
| 1MB       | 1M        | 30      |                   |  |              |          
| 50MB      | 1M        | 20      |                   |  |              |                               |
| 100MB     | 1M        | 10     |                   |  |              |                                 |
| 1GB       | 1M        | 2      |                   |  |              |          


## Benchmarking on n2 machine-type
* VM Type: n2-standard-96
* VM location: us-south1
* Networking: gVNIC+  tier_1 networking (100Gbps)
* SSD persistent disk  
* GCS Bucket location: us-south1
### Sequential Reads
| File Size | BlockSize | nrfiles |Bandwidth in (MiB/sec) | Avg Latency (msec) | IOPs  |
|---|---|---|---|---|---|
| 128K | 128K | 30  |  |  |  |
| 256K  | 128K  | 30  |  |  |  |
| 1M | 1M | 30 |  |  |  |
| 5M | 1M  | 20  |  |  |  |
| 10M | 1M | 20 |  |  |  |
| 50M | 1M | 20 |  |  |  |
| 100M |1M | 10 |  |  |  |
| 200M  | 1M | 10  |  |  |  |
| 1G | 1M | 10 |  |  |  |



### Random Reads
| File Size | BlockSize | nrfiles |Bandwidth in (MiB/sec) | Avg Latency (msec) | IOPs  |
|---|---|---|---|---|---|
| 256K  | 128K  | 30  |  |  |  |
| 5M | 1M  | 20  |  |  |  |
| 10M | 1M | 20 |  |  |  |
| 50M | 1M | 20 |  |  |  |
| 100M |1M | 10 |  |  |  |
| 200M  | 1M | 10  |  |  |  |
| 1G | 1M | 10 |  |  |  |


### Sequential Writes

| File Size | BlockSize | nrfiles | Bandwidth in MiB/sec | IOPS(avg) | Avg Latency (msec) | Network Send Traffic (GiB/s) |
|-----------|-----------|---------|----------------------|-----------|--------------------|------------------------------|
| 256KB     | 16K       | 30      |                   |  |              |                         |
| 1MB       | 1M        | 30      |                   |  |              |          
| 50MB      | 1M        | 20      |                   |  |              |                               |
| 100MB     | 1M        | 10     |                   |  |              |                                 |
| 1GB       | 1M        | 2      |                   |  |              |     



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
    fio samplejobspec.fio
    ```

9. Metrics will be displayed on the terminal after test is completed.