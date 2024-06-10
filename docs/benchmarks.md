# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
GCSFuse. Below tables shows performance metrics of GCSFuse for different
workloads for the given test setup:

## Test setup:

* Infra: GCP VM
* VM Type: n2-standard-96
* OS:  Debian-11
* VM Bandwidth: 100Gbps
* VM location: asia-southeast1-a
* Disk Type: Local ssd
* GCS Bucket location: asia-southeast1
* Framework: FIO

## Reads
### FIO spec
```
  [global]
  ioengine=libaio
  direct=1
  fadvise_hint=0
  verify=0
  fsync=1  // For write tests only
  rw=read  // Change this to randread to test random reads.
  bs=1M  // Update the block size value from the table for different experiments.
  iodepth=64
  invalidate=1
  ramp_time=10s
  runtime=60s
  time_based=0
  thread=1
  filesize=10M  // Update the file size value from table(file size) for different experiments.
  openfiles=1
  group_reporting=1
  allrandrepeat=1
  directory=/mnt/1mb  // Change the test directory (1mb) for different experiments. The directory must exist within the mounted directory.
  filename_format=$jobname.$jobnum.$filenum
  
  [100_thread]
  stonewall
  numjobs=128 // Number of threads
 ```

### Results

#### Sequential Reads

| File Size | BlockSize | Bandwidth in (MiB/sec) | Avg Latency (msec) |
|-----------|-----------|------------------------|--------------------|
| 128KB     | 128K      | 826                    | 1235.02            |
| 256KB     | 128K      | 1235.02                | 653.332            |
| 1MB       | 1M        | 1235.02                | 1671.34            |
| 5MB       | 1M        | 7635                   | 1080.48            |
| 10MB      | 1M        | 8102                   | 1017.55            |
| 50MB      | 1M        | 8081                   | 1020.52            |
| 100MB     | 1M        | 8145                   | 1014.493           |
| 200MB     | 1M        | 8131                   | 1013.38            |
| 1GB       | 1M        | 8131                   | 1017.30            |

#### Random Reads

| File Size | BlockSize | Bandwidth in MiB/sec | Avg Latency (msec) |
|-----------|-----------|----------------------|--------------------|
| 128KB     | 128K      | 864                  | 1196.64            |
| 256KB     | 128K      | 1274                 | 808.451            |
| 1MB       | 1M        | 4860                 | 1709.91            |
| 5MB       | 1M        | 5929                 | 1394.41            |
| 10MB      | 1M        | 5013                 | 1654.96            |
| 50MB      | 1M        | 3304                 | 2524.56            |
| 100MB     | 1M        | 3265                 | 2557.20            |
| 200MB     | 1M        | 3071                 | 2557.20            |
| 1GB       | 1M        | 2716                 | 3041.41            |

### Recommendation for reads

GCSFuse performs well for sequential reads and recommendation is to use GCSFuse
for doing sequential reads on file sizes > 10MB and < 1GB. Always use http1 (
`--client-protocol=http1`, enabled by default) gives better throughput.

## Writes
### FIO spec
```
  [global]
  ioengine=libaio
  direct=1
  fadvise_hint=0
  verify=0
  fsync=1  // For write tests only. Update the fsync value from the table for different experiments.
  rw=write  // Change this to randwrite to test random reads.
  bs=1M  // Update the block size value from the table for different experiments.
  nrfiles=30
  iodepth=64
  invalidate=1
  ramp_time=10s
  time_based=0
  thread=1
  filesize=10M  // Update the file size value from table(file size) for different experiments.
  openfiles=1
  group_reporting=1
  allrandrepeat=1
  directory=/mnt/1mb  // Change the test directory (1mb) for different experiments. The directory must exist within the mounted directory.
  filename_format=$jobname.$jobnum.$filenum
  
  [100_thread]
  stonewall
  numjobs=112 // Number of threads
 ```

* We have a fsync parameter for writes that defines fio will sync the file after
  every fsync number of writes issued. When the writeFile operation is invoked,
  gcsfuse will write data to disk. When syncFile is invoked, gcsfuse will write
  the
  data from disk to GCS bucket. So after fsync number of write operations, sync
  call
  will be issued to gcsfuse i.e, data will get written to GCS bucket.

## Write

### Sequential Write

| File Size | BlockSize | Fsync | Bandwidth in MiB/sec | IOPS(avg) | Avg Latency (msec) | Network Send Traffic (GiB/s) |
|-----------|-----------|-------|----------------------|-----------|--------------------|------------------------------|
| 256KB     | 16K       | 16    | 62.3                 | 9872.44   | 2.278              | 0.03                         |
| 1MB       | 1M        | 10    | 2524                 | 3871.71   | 15.150             | 0.25                         |
| 50MB      | 1M        | 50    | 3025                 | 4588.38   | 19.991             | 2.3                          |
| 100MB     | 1M        | 100   | 2904                 | 6242.30   | 18.648             | 2.53                         |
| 1GB       | 1M        | 1024  | 1815                 | 9875.59   | 50.426             | 2.05                         |

### Random Write

In case of random writes, only offset will change in calls issued by fio.
GCSFuse behaviour will
remain the same and there are no changes in the way gcs calls are being made.
Hence the bandwidth will be same
as sequential writes.

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

5. [Install GCSFuse](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/installing.md#linux).
6. Create a directory on the VM and then mount the gcs bucket to that directory.

    ```
      mkdir <path-to-mount-point> 
      
      gcsfuse <bucket-name> <path-to-mount-point>
    ```

7. Create a FIO job spec file.
   The FIO content referred to above. Please read the details about the FIO specification
   [here](https://fio.readthedocs.io/en/latest/).
    ```
    vi samplejobspec.fio
    ```

8. Run the FIO test using following command.

    ```
    fio samplejobspec.fio
    ```

9. Metrics will be displayed on the terminal after test is completed.