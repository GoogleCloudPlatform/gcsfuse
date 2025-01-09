# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on
GCSFuse. Below tables shows performance metrics of GCSFuse for different
workloads for the given test setup:

## Test setup:

* Infra: GCP VM
* VM Type: n2-standard-96
* OS: ubuntu-20.04
* VM Bandwidth: 100Gbps
* VM location: us-west1-b
* Disk Type: SSD persistent disk
* GCS Bucket location: us-west1
* Framework: FIO
* GCSFuse version: 2.2.0

## Reads

### FIO spec

  ```
    [global]
    ioengine=sync
    direct=1
    fadvise_hint=0
    verify=0
    iodepth=64
    invalidate=1
    ramp_time=10s
    runtime=60s
    time_based=1
    thread=1
    openfiles=1
    group_reporting=1
    allrandrepeat=1
    # Change this to randread to test random reads.
    rw=read  
    # Update the block size value from the table for different experiments.
    bs=1M  
    # Update the file size value from table(file size) for different experiments.
    filesize=10M  
    # Change the test directory (1mb) for different experiments. The directory must exist within the mounted directory.
    directory=/mnt/1mb  
    filename_format=$jobname.$jobnum.$filenum
    [experiment]
    stonewall
    # Number of threads
    numjobs=128 
```

### Results

#### Sequential Reads

| File Size | BlockSize | Bandwidth in (MiB/sec) | Avg Latency (msec) | IOPs     |
|-----------|-----------|------------------------|--------------------|----------|
| 128KB     | 128K      | 862                    | 18.54              | 6898.27  |
| 256KB     | 128K      | 1548                   | 10.325             | 12386.03 |
| 1MB       | 1M        | 5108                   | 24.99              | 5113.21  |
| 5MB       | 1M        | 7282                   | 17.505             | 7308.51  |
| 10MB      | 1M        | 7946                   | 16.092             | 7946.63  |
| 50MB      | 1M        | 7810                   | 16.356             | 7818.17  |
| 100MB     | 1M        | 7839                   | 16.295             | 7840.17  |
| 200MB     | 1M        | 7879                   | 16.217             | 7884.45  |
| 1GB       | 1M        | 7911                   | 16.162             | 7910.19  |

#### Random Reads

| File Size | BlockSize | Bandwidth in MiB/sec | Avg Latency (msec) | IOPs     |
|-----------|-----------|----------------------|--------------------|----------|
| 256KB     | 128K      | 1264                 | 12.648             | 10109.62 |
| 5MB       | 1M        | 4367                 | 29.129             | 4449.03  |
| 10MB      | 1M        | 3810                 | 33.496             | 3825.54  |
| 50MB      | 1M        | 4370                 | 29.185             | 4426.73  |
| 100MB     | 1M        | 3504                 | 36.421             | 3505.01  |
| 200MB     | 1M        | 3048                 | 41.919             | 3044.43  |
| 1GB       | 1M        | 2120                 | 60.246             | 2114.33  |

## Writes

### FIO spec

  ```
    [global]
    ioengine=sync
    direct=1
    fadvise_hint=0
    verify=0
    iodepth=64
    invalidate=1
    time_based=0
    file_append=0
    # By default fio creates all files first and then starts writing to them. This option is to disable that behavior. 
    create_on_open=1 
    thread=1
    openfiles=1
    group_reporting=1
    allrandrepeat=1
    # Every file is written only once. Set nrfiles per thread in such a way that the test runs for 1-2 min. 
    # This will vary based on file size. Change the value from table to get provided results.
    nrfiles=2
    filename_format=$jobname.$jobnum.$filenum
    # Change this to randwrite to test random writes.
    rw=write   
    # Update the block size value from the table for different 
    bs=1M
    # Update the file size value from table(file size) for different experiments.
    filesize=1G  
    [experiment]
    stonewall
    # Change the test directory (1mb) for different experiments. The directory must exist within the mounted directory.
    directory=gcs/1gb
    numjobs=112
 ```

**Note:** Benchmarking is done by writing out new files to GCS. Performance
numbers will be different for edits/appends to existing files.

### Results

#### Sequential Write

| File Size | BlockSize | nrfiles | Bandwidth in MiB/sec | IOPS(avg) | Avg Latency (msec) | Network Send Traffic (GiB/s) |
|-----------|-----------|---------|----------------------|-----------|--------------------|------------------------------|
| 256KB     | 16K       | 30      | 212                  | 14976.95  | 3.206              | 0.027                        |
| 1MB       | 1M        | 30      | 772                  | 794.32    | 1.150              | 0.036                        |
| 50MB      | 1M        | 20      | 3611                 | 5948.63   | 8.929              | 1.33                         |
| 100MB     | 1M        | 10      | 3577                 | 4672.64   | 1.911              | 1.41                         |
| 1GB       | 1M        | 2       | 1766                 | 2121.66   | 49.114             | 1.77                         |

#### Random Write

Random writes and sequential write performance will generally be the same, as
all writes are first staged to a local temporary directory before being written
to GCS on close/fsync.

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