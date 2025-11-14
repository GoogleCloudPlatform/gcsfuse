# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
GCSFuse. Below tables shows performance metrics of GCSFuse for different
workloads for the given test setup:

## Test setup:

* Infra: GCP VM [C4-standard-192](https://cloud.google.com/compute/docs/general-purpose-machines#c4_series)
* Network: [Tier_1](https://cloud.google.com/compute/docs/networking/configure-vm-with-high-bandwidth-configuration) Networking enabled on VM providing 200Gpbs egress bandwidth.
* OS Version: [Ubuntu 22.04 LTS](https://cloud.google.com/compute/docs/images/os-details#notable-difference-ubuntu)
* Image Family: [ubuntu-2204-lts](https://cloud.google.com/compute/docs/images/os-details#notable-difference-ubuntu)
* Disk Type: [Hyperdisk Balanced](https://cloud.google.com/compute/docs/disks/hd-types/hyperdisk-balanced)
* VM Region: us-south1
* GCS Bucket Region: us-south1
* Framework: FIO (version 3.39)
* GCSFuse version: 3.4.3

## FIO workloads
Please read the details about the FIO specification [here](https://fio.readthedocs.io/en/latest/).
### Sequential Reads
| File Size | BlockSize | NRFiles | NumJobs | **Avg Bandwidth (MB/s)** | **Avg IOPS** | **Avg Latency (msec)** |
| :--- | :--- | ---: | ---: | ---: | ---: | ---: |
| 128 KiB | 128 KiB | 192 | 30 | 1,303.87 | | |
| 256 KiB | 128 KiB | 192 | 30| 2,539.67 | | |
| 1 MiB | 1 MiB | 192 | 30 | 6,204.93 | | |
| 5 MiB | 1 MiB | 192 | 20 | 12,394.90 | | |
| 10 MiB | 1 MiB | 192 | 20 | 14,489.90 | | |
| 50 MiB | 1 MiB | 192 | 20 | 13,808.20 | | |
| 100 MiB | 1 MiB | 144 | 10 | 13,433.40 | | |
| 200 MiB | 1 MiB | 144 | 10 | 13,261.70 | | |
| 1 GiB | 1 MiB | 144 | 10 | 14,198.00 | | |

#### GCSFuse Mount Option and fio configuration
<details>
  <summary> Click to expand </summary> 

##### GCSFuse Mount Options
```text
--implicit-dirs
--metadata-cache-ttl-secs=-1
```
##### Fio templated configuration
```ini
[global]
ioengine=libaio
direct=1
fadvise_hint=0
iodepth=64
invalidate=1
thread=1
openfiles=1
group_reporting=1
create_serialize=0
allrandrepeat=0
file_service_type=random
rw=read
filename_format=$jobname.$jobnum.$filenum.size-${FILESIZE}

[seq_read]
directory=${DIR}
filesize=${FILESIZE}
bs=${BS}
numjobs=${NUMJOBS}
nrfiles=${NRFILES}
```
</details>

---

### Random Reads
| File Size | BlockSize | NRFiles | NumJobs | **Avg Bandwidth (MB/s)** | **Avg IOPS** | **Avg Latency (msec)** |
| :--- | :--- | ---: | ---: | ---: | ---: | ---: |
| 256 KiB | 128 KiB | 192 | 30| 1,591.49 | | |
| 5 MiB | 1 MiB | 192 | 20 | 5,014.54 | | |
| 10 MiB | 1 MiB | 192 | 20 | 4,197.65 | | |
| 50 MiB | 1 MiB | 192 | 20 | 4,421.05 | | |
| 100 MiB | 1 MiB | 192 | 10 | 4,454.59 | | |
| 200 MiB | 1 MiB | 192 | 10 | 4,205.02 | | |
| 1 GiB | 1 MiB | 192 | 10 | 4,107.23 | | |

#### GCSFuse Mount Option and fio configuration
<details>
  <summary> Click to expand </summary> 

##### GCSFuse Mount Options
```text
--implicit-dirs
--metadata-cache-ttl-secs=-1
```
##### Fio templated configuration
```ini
[global]
ioengine=libaio
direct=1
fadvise_hint=0
iodepth=64
invalidate=1
thread=1
openfiles=1
group_reporting=1
create_serialize=0
allrandrepeat=0
file_service_type=random
rw=randread
filename_format=$jobname.$jobnum.$filenum.size-${FILESIZE}

[rand_read]
directory=${DIR}
filesize=${FILESIZE}
bs=${BS}
numjobs=${NUMJOBS}
nrfiles=${NRFILES}
```
</details>



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

<!-- Benchmarks start -->

## GCSFuse Benchmarks on c4-standard-192 machine-type

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