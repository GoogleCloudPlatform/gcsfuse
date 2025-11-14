# GCSFuse Performance Benchmarks

[fio](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
GCSFuse. The tables below show performance metrics of GCSFuse for different
workloads for the given test setup:

## Test setup:

* Infra: GCP VM [C4-standard-192](https://cloud.google.com/compute/docs/general-purpose-machines#c4_series)
* Network: [Tier_1](https://cloud.google.com/compute/docs/networking/configure-vm-with-high-bandwidth-configuration) Networking enabled on VM providing 200 Gbps egress bandwidth.
* OS Version: [Ubuntu 22.04 LTS](https://cloud.google.com/compute/docs/images/os-details#notable-difference-ubuntu)
* Image Family: [ubuntu-2204-lts](https://cloud.google.com/compute/docs/images/os-details#notable-difference-ubuntu)
* Disk Type: [Hyperdisk Balanced](https://cloud.google.com/compute/docs/disks/hd-types/hyperdisk-balanced)
* VM Region: us-south1
* GCS Bucket ([HNS enabled](https://cloud.google.com/storage/docs/hns-overview)) Region: us-south1
* Framework: fio (version 3.39)
* GCSFuse version: [v3.4.3](https://github.com/GoogleCloudPlatform/gcsfuse/releases/tag/v3.4.3)

## Fio workloads
Please read the details about the fio specification [here](https://fio.readthedocs.io/en/latest/).

<!-- Benchmarks start -->
---

### Sequential Reads
| File Size | BlockSize | NRFiles | NumJobs | **Avg Bandwidth (MB/s)** | **Avg IOPS** | **Avg Latency (msec)** |
| :---      | :---      | ---:    | ---:    | ---:                     | ---:         | ---:                   |
| 128 KiB   | 128 KiB   | 192     | 30      | 1,303.87                 | 9,947.74     | 0.10                   |
| 256 KiB   | 128 KiB   | 192     | 30      | 2,539.67                 | 19,375.51    | 0.05                   |
| 1 MiB     | 1 MiB     | 192     | 30      | 6,204.93                 | 5,917.49     | 0.17                   |
| 5 MiB     | 1 MiB     | 192     | 20      | 12,394.90                | 11,820.68    | 0.08                   |
| 10 MiB    | 1 MiB     | 192     | 20      | 14,489.90                | 13,818.61    | 0.07                   |
| 50 MiB    | 1 MiB     | 192     | 20      | 13,808.20                | 13,168.52    | 0.08                   |
| 100 MiB   | 1 MiB     | 144     | 10      | 13,433.40                | 12,811.09    | 0.08                   |
| 200 MiB   | 1 MiB     | 144     | 10      | 13,261.70                | 12,647.36    | 0.08                   |
| 1 GiB     | 1 MiB     | 144     | 10      | 14,198.00                | 13,540.29    | 0.07                   |

#### GCSFuse Mount Option and fio configuration
<details>
  <summary> Click to expand </summary> 

##### GCSFuse Mount Options
See more details about each option [here](https://cloud.google.com/storage/docs/cloud-storage-fuse/cli-options#options).
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
| :---      | :---      | ---:    | ---:    | ---:                     | ---:         | ---:                   |
| 256 KiB   | 128 KiB   | 192     | 30      | 1,591.49                 | 12,142.02    | 0.08                   |
| 5 MiB     | 1 MiB     | 192     | 20      | 5,014.54                 | 4,782.28     | 0.21                   |
| 10 MiB    | 1 MiB     | 192     | 20      | 4,197.65                 | 4,003.21     | 0.25                   |
| 50 MiB    | 1 MiB     | 192     | 20      | 4,421.05                 | 4,216.24     | 0.24                   |
| 100 MiB   | 1 MiB     | 192     | 10      | 4,454.59                 | 4,248.22     | 0.24                   |
| 200 MiB   | 1 MiB     | 192     | 10      | 4,205.02                 | 4,009.95     | 0.25                   |
| 1 GiB     | 1 MiB     | 192     | 10      | 4,107.23                 | 3,916.94     | 0.26                   |

#### GCSFuse Mount Option and fio configuration
<details>
  <summary> Click to expand </summary> 

##### GCSFuse Mount Options
See more details about each option [here](https://cloud.google.com/storage/docs/cloud-storage-fuse/cli-options#options).
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
| :---      | :---      | ---:    | ---:    | ---:                     | ---:         | ---:                   |
| 256 KiB   | 16 KiB    | 96      | 30      | 170.88                   | 10,430.00    | 0.10                   |
| 1 MiB     | 1 MiB     | 96      | 30      | 528.49                   | 504.00       | 1.98                   |
| 50 MiB    | 1 MiB     | 96      | 30      | 3,581.07                 | 3,415.15     | 0.29                   |
| 100 MiB   | 1 MiB     | 96      | 20      | 4,061.29                 | 3,873.18     | 0.26                   |
| 500 MiB   | 1 MiB     | 96      | 20      | 4,569.04                 | 4,357.39     | 0.23                   |
| 1 GiB     | 1 MiB     | 96      | 10      | 4,624.59                 | 4,410.39     | 0.23                   |

#### GCSFuse Mount Option and fio configuration
<details>
  <summary> Click to expand </summary> 

##### GCSFuse Mount Options
See more details about each option [here](https://cloud.google.com/storage/docs/cloud-storage-fuse/cli-options#options).
```bash
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

Note: Peformance for Random Writes to an existing file would be performance of complete file read and performance of complete file write after append operations.
---
<!-- Benchmarks end -->

## Steps to benchmark GCSFuse performance

> [!IMPORTANT]
> GCSFuse performance may differ based on region of VM and GCS Bucket region and the GCSFuse version in use. To reproduce above benchmark please use the exact testing infra setup mentioned above. Use new GCS Bucket for each fio run which ensures for sequential write objects are not being overwritten.

1. [Create](https://cloud.google.com/compute/docs/instances/create-start-instance#publicimage)
   a GCP VM instance
2. [Connect](https://cloud.google.com/compute/docs/instances/connecting-to-instance)
   to the VM instance
3. Install FIO.

    ```bash
    sudo apt-get update
    sudo apt-get install fio
    ```

4. [Install GCSFuse](https://cloud.google.com/storage/docs/gcsfuse-install)
5. [Create GCS Bucket](https://cloud.google.com/storage/docs/creating-buckets)
6. Create a directory on the VM and then mount the gcs bucket to that directory with the mount options provided in benchmark result section.

    ```bash
    mkdir <path-to-mount-point>
    gcsfuse <mount options> <bucket-name> <path-to-mount-point>
    ```

7. Create a fio job file with the templated fio configuration content provided in benchmark result section.
    ```bash
    # Copy content of fio configuration to this file.
    vi samplejobspec.fio
    ```

8. Run the FIO tool using following command.

    ```bash
    # See the values of these variables from the respective benchmark result table.
    DIR=<path-to-mount-point> \
    NUMJOBS="" \
    BS="" \
    FILESIZE="" \
    NRFILES="" fio samplejobspec.fio

    # Example command for last row of sequential write benchmark result table.
    DIR=<path-to-mount-point> \
    NUMJOBS="10" \
    BS="1M" \
    FILESIZE="1G" \
    NRFILES="96" fio samplejobspec.fio
    ```

9. Metrics will be displayed on the terminal after test is completed.