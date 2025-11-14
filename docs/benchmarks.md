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

<!-- Benchmarks start -->
---

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
```bash
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
```bash
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

---

### Sequential Writes
| File Size | BlockSize | NRFiles | NumJobs | **Avg Bandwidth (MB/s)** | **Avg IOPS** | **Avg Latency (msec)** |
| :--- | :--- | ---: | ---: | ---: | ---: | ---: |
| 256 KiB | 16K KiB | 96 | 30| 170.88 | | |
| 1 MiB | 1 MiB | 96 | 30 | 528.49 | | |
| 50 MiB | 1 MiB | 96 | 30 | 3,581.07 | | |
| 100 MiB | 1 MiB | 96 | 20 | 4,061.29 | | |
| 500 MiB | 1 MiB | 96 | 20 | 4,569.04 | | |
| 1 GiB | 1 MiB | 96 | 10 | 4,624.59 | | |

#### GCSFuse Mount Option and fio configuration
<details>
  <summary> Click to expand </summary> 

##### GCSFuse Mount Options
```text
--implicit-dirs
--metadata-cache-ttl-secs=-1
--write-global-max-blocks=-1
```
##### Fio templated configuration
```ini
[global]
ioengine=libaio
direct=1
fadvise_hint=0
iodepth=64
verify=0
invalidate=1
file_append=0
create_on_open=1
end_fsync=1
thread=1
openfiles=1
group_reporting=1
allrandrepeat=1
filename_format=$jobname.$jobnum.$filenum.size-${FILESIZE}
rw=write

[write_seq]
directory=${DIR}
filesize=${FILESIZE}
bs=${BS}
numjobs=${NUMJOBS}
nrfiles=${NRFILES}
```
</details>

---
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