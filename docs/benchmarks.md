# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on GCSFuse. Below tables shows performance metrics of GCSFuse for different workloads for the given test setup:

## Test setup:

* Infra: GCP VM
* VM Type: n2-standard-96
* OS:  ubuntu-20.04
* VM Bandwidth: 100Gbps
* VM location: us-west1-b
* Disk Type: SSD persistent disk
* GCS Bucket location: us-west1
* Framework: FIO

### FIO Spec
* Test runtime: 60sec
* Thread count
  * Reads - 128
  * Writes - 112
* We have a fsync parameter for writes that defines fio will sync the file after 
every fsync number of writes issued. When the writeFile operation is invoked, 
gcsfuse will write data to disk. When syncFile is invoked, gcsfuse will write the
data from disk to GCS bucket. So after fsync number of write operations, sync call
will be issued to gcsfuse i.e, data will get written to GCS bucket.

### GCSFuse command
```
gcsfuse --implicit-dirs  --client-protocol=http1 --max-conns-per-host=100 <bucket-name> <path-to-mount-point>
```

## Read
### Sequential Read
| File Size | BlockSize | Bandwidth in (MiB/sec) | Avg Latency (msec) |
|-----------|-----------|------------------------|--------------------|
| 128KB     | 128K      | 765                    | 20.90              |
| 256KB     | 128K      | 1579                   | 10.089             |
| 1MB       | 1M        | 4655                   | 27.23              |
| 5MB       | 1M        | 7564                   | 16.915             |
| 10MB      | 1M        | 7564                   | 16.915             |
| 50MB      | 1M        | 7706                   | 16.598             |
| 100MB     | 1M        | 7741                   | 16.518             |
| 200MB     | 1M        | 7683                   | 16.639             |
| 1GB       | 1M        | 7714                   | 16.573             |

### Random Read
| File Size | BlockSize | Bandwidth in MiB/sec | Avg Latency (msec) |
|-----------|-----------|----------------------|--------------------|
| 128KB     | 128K      | 733                  | 21.77              |
| 256KB     | 128K      | 956                  | 16.735             |
| 1MB       | 1M        | 4428                 | 28.90              |
| 5MB       | 1M        | 2876                 | 44.463             |
| 10MB      | 1M        | 3629                 | 35.238             |
| 50MB      | 1M        | 2630                 | 48.643             |
| 100MB     | 1M        | 2644                 | 48.388             |
| 200MB     | 1M        | 2279                 | 56.104             |
| 1GB       | 1M        | 2068                 | 61.858             |

### Recommendation for reads
GCSFuse performs well for sequential reads and recommendation is to use GCSFuse for doing sequential reads on file sizes > 10MB and < 1GB. Always use http1 (--client-protocol=http1, enabled by default) and --max-connections-per-host
flag, it gives better throughput.

## Write
### Sequential Write

| File Size | BlockSize | Fsync | Bandwidth in MiB/sec   | IOPS(avg)     | Avg Latency (msec)  | Network Send Traffic (GiB/s) |
|-----------|-----------|-------|------------------------|---------------|---------------------|------------------------------|
| 256KB     | 16K       | 16    | 62.3                   | 9872.44       | 2.278               | 0.03                         |
| 1MB       | 1M        | 10    | 2524                   | 3871.71       | 15.150              | 0.25                         |
| 50MB      | 1M        | 50    | 3025                   | 4588.38       | 19.991              | 2.3                          |
| 100MB     | 1M        | 100   | 2904                   | 6242.30       | 18.648              | 2.53                         |
| 1GB       | 1M        | 1024  | 1815                   | 9875.59       | 50.426              | 2.05                         |

### Random Write
In case of random writes, only offset will change in calls issued by fio. GCSFuse behaviour will
remain the same and there are no changes in the way gcs calls are being made. Hence the bandwidth will be same
as sequential writes.

## Steps to benchmark GCSFuse performance
1. [Create](https://cloud.google.com/compute/docs/instances/create-start-instance#publicimage) a GCP VM instance.
2. [Connect](https://cloud.google.com/compute/docs/instances/connecting-to-instance) to the VM instance.
3. Install FIO.
```
sudo apt-get update
sudo apt-get install fio
```
5. [Install GCSFuse](https://github.com/googlecloudplatform/gcsfuse/v2/blob/master/docs/installing.md#linux).
6. Create a directory on the VM and then mount the gcs bucket to that directory.
```
  mkdir <path-to-mount-point> 
  
  gcsfuse --implicit-dirs --client-protocol=http1 --max-conns-per-host=100 <bucket-name> <path-to-mount-point>
```
7. Create a FIO job spec file.
```
vi samplejobspec.fio
```
Copy the following contents into the job spec file. Read the details about FIO spec
[here](https://fio.readthedocs.io/en/latest/).
```
[global]
ioengine=libaio
direct=1
fadvise_hint=0
verify=0
fsync=10  // For write tests only
rw=write
bs=1M
iodepth=64
invalidate=1
ramp_time=10s
runtime=60s
time_based=1
nrfiles=1
thread=1
filesize=10M 
openfiles=1
group_reporting=1
allrandrepeat=1
directory=<path-to-mount-point>
filename_format=$jobname.$jobnum.$filenum

[40_thread]
stonewall
numjobs=112
```
8. Run the FIO test using following command. 
```
fio samplejobspec.fio
```
9. Metrics will be displayed on the terminal after test is completed.