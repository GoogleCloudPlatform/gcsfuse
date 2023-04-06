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
* Thread count: 40
* Block Size: 256KB for 256KB files and 1MB for all other files.
* We have a fsync parameter for writes that defines fio will sync the file after 
every fsync number of writes issued. When the writeFile operation is invoked, 
gcsfuse will write data to disk. When syncFile is invoked, gcsfuse will write the
data from disk to GCS bucket. So after fsync number of write operations, sync call
will be issued to gcsfuse i.e, data will get written to GCS bucket.
```
gcsfuse --implicit-dirs  --client-protocol=http1 --max-conns-per-host=100 <bucket-name> <path-to-mount-point>
```

## Write
### Sequential Write

| File Size | BlockSize | Fsync | Bandwidth in MiB/sec  | IOPS(avg) | Avg Latency (msec) |
|-----------|-----------|-------|-----------------------|-----------|--------------------|
| 256KB     | 16k       | 16    | 62.3                  | 9872.44   | 2.278              |
| 1MB       | 1M        | 10    | 2524                  | 3871.71   | 15.150             |
| 50MB      | 1M        | 50    | 3025                  | 4588.38   | 19.991             |
| 100MB     | 1M        | 100   | 2904                  | 6242.30   | 18.648             |
| 1GB       | 1M        | 1024  | 2875                  | 11155.96  | 9.789              |
| 4GB       | 1M        | 4096  | 477                   | 850.86    | 175.337            |

### Random Write
In case of random writes, only offset will change in calls issued by fio. GCSFuse behaviour will
remain the same and there are no changes in the way gcs calls are being made. Hence the bandwidth will be same
as sequential writes.

## Read
### Sequential Read
| File Size | Bandwidth in MiB/sec  | Avg Latency (msec) |
|-----------|-----------------------|--------------------|
| 128KB     | 788                   | 25.37              |
| 256KB     | 1579                  | 10.089             |
| 1MB       | 4655                  | 27.23              |
| 5MB       | 7545                  | 21.191             |
| 10MB      | 7622                  | 20.959             |
| 50MB      | 7706                  | 16.598             |
| 100MB     | 7741                  | 16.518             |
| 200MB     | 7700                  | 12.460             |
| 1GB       | 7971                  | 8.023              |

### Random Read
| File Size | Bandwidth in MiB/sec | Avg Latency (msec) |
|-----------|----------------------|--------------------|
| 128KB     | 707                  | 28.27              |
| 256KB     | 982                  | 20.347             |
| 1MB       | 4428                 | 28.90              |
| 5MB       | 3314                 | 28.930             |
| 10MB      | 3667                 | 26.139             |
| 50MB      | 2893                 | 33.160             |
| 100MB     | 2685                 | 59.544             |
| 200MB     | 2317                 | 68.819             |
| 1GB       | 2068                 | 61.858             |

### Recommendation for reads
GCSFuse performs well for sequential reads and recommendation is to use GCSFuse for doing sequential reads on file sizes > 10MB and < 1GB. Always use http1 (--client-protocol=http1, enabled by default) and --max-connections-per-host
flag, it gives better throughput.

## Steps to benchmark GCSFuse performance
1. [Create](https://cloud.google.com/compute/docs/instances/create-start-instance#publicimage) a GCP VM instance.
2. [Connect](https://cloud.google.com/compute/docs/instances/connecting-to-instance) to the VM instance.
3. Install FIO.
```
sudo apt-get update
sudo apt-get install fio
```
5. [Install GCSFuse](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/installing.md#linux).
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
fsyc=1  // For write tests only
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
numjobs=40
```

8. Run the FIO test using following command. 
```
fio samplejobspec.fio
```
9. Metrics will be displayed on the terminal after test is completed.