# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
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

<!-- Benchmarks start -->
---

### Sequential Reads
| File Size | BlockSize | NumJobs | NRFiles | **Avg Bandwidth (GB/s)** | **Avg IOPS (K)** | **Avg Latency (msec)** |
| :---      | :---      | ---:    | ---:    | ---:                     | ---:             | ---:                   |
| 128 KiB   | 128 KiB   | 192     | 30      |                     1.30 |             9.95 | 0.10                   |
| 256 KiB   | 128 KiB   | 192     | 30      |                     2.54 |            19.38 | 0.05                   |
| 1 MiB     | 1 MiB     | 192     | 30      |                     6.20 |             5.92 | 0.17                   |
| 5 MiB     | 1 MiB     | 192     | 20      |                    12.39 |            11.82 | 0.08                   |
| 10 MiB    | 1 MiB     | 192     | 20      |                    14.49 |            13.82 | 0.07                   |
| 50 MiB    | 1 MiB     | 192     | 20      |                    13.81 |            13.17 | 0.08                   |
| 100 MiB   | 1 MiB     | 144     | 10      |                    13.43 |            12.81 | 0.08                   |
| 200 MiB   | 1 MiB     | 144     | 10      |                    13.26 |            12.65 | 0.08                   |
| 1 GiB     | 1 MiB     | 144     | 10      |                    14.20 |            13.54 | 0.07                   |

<details>
  <summary>GCSFuse Mount Option and fio configuration</summary>

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
| File Size | BlockSize | NumJobs | NRFiles | **Avg Bandwidth (GB/s)** | **Avg IOPS (K)** | **Avg Latency (msec)** |
| :---      | :---      | ---:    | ---:    | ---:                     | ---:             | ---:                   |
| 256 KiB   | 128 KiB   | 192     | 30      |                     1.59 |            12.14 | 0.08                   |
| 5 MiB     | 1 MiB     | 192     | 20      |                     5.01 |             4.78 | 0.21                   |
| 10 MiB    | 1 MiB     | 192     | 20      |                     4.20 |             4.00 | 0.25                   |
| 50 MiB    | 1 MiB     | 192     | 20      |                     4.42 |             4.22 | 0.24                   |
| 100 MiB   | 1 MiB     | 192     | 10      |                     4.45 |             4.25 | 0.24                   |
| 200 MiB   | 1 MiB     | 192     | 10      |                     4.21 |             4.01 | 0.25                   |
| 1 GiB     | 1 MiB     | 192     | 10      |                     4.11 |             3.92 | 0.26                   |

<details>
  <summary>GCSFuse Mount Option and fio configuration</summary>

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
| File Size | BlockSize | NumJobs | NRFiles | **Avg Bandwidth (GB/s)** | **Avg IOPS (K)** | **Avg Latency (msec)** |
| :---      | :---      | ---:    | ---:    | ---:                     | ---:             | ---:                   |
| 256 KiB   | 16 KiB    | 96      | 30      |                     0.17 |            10.43 | 0.10                   |
| 1 MiB     | 1 MiB     | 96      | 30      |                     0.53 |             0.50 | 1.98                   |
| 50 MiB    | 1 MiB     | 96      | 30      |                     3.58 |             3.42 | 0.29                   |
| 100 MiB   | 1 MiB     | 96      | 20      |                     4.06 |             3.87 | 0.26                   |
| 500 MiB   | 1 MiB     | 96      | 20      |                     4.57 |             4.36 | 0.23                   |
| 1 GiB     | 1 MiB     | 96      | 10      |                     4.62 |             4.41 | 0.23                   |

<details>
  <summary>GCSFuse Mount Option and fio configuration</summary>

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

> [!NOTE]
> Edits and appends to existing files are handled by first downloading the entire file. After changes are made locally, the entire file is uploaded again on `close` or `sync`. This results in performance similar to a full file read followed by a full file write.

> [!NOTE]
> The bandwidth observed during benchmark runs can vary by up to 10%. This variation is expected and is due to normal fluctuations in Google Cloud Storage. To obtain more statistically reliable results, we recommend running the benchmarks multiple times. The numbers published above are the average of 5 runs.

---
<!-- Benchmarks end -->

## Steps to benchmark GCSFuse performance

> [!IMPORTANT]
> GCSFuse performance may differ based on region of VM and GCS Bucket region and GCSFuse version in use. To reproduce above benchmark please use the exact testing infra setup mentioned above. Use new GCS Bucket for each fio run which ensures for sequential write objects are not being overwritten.

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
    NUMJOBS="96" \
    BS="1M" \
    FILESIZE="1G" \
    NRFILES="10" fio samplejobspec.fio
    ```

9. Metrics will be displayed on the terminal after test is completed.
