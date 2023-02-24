# GCSFuse Performance Benchmarks

[FIO](https://fio.readthedocs.io/en/latest/) is used to perform load tests on GCSFuse. Below tables shows performance metrics of GCSFuse for different workloads for the given test setup:

## Test setup:

* Infra: GCP VM
* VM Type: n2-standard-48
* VM Bandwidth: 32Gbps
* OS: debian-11
* VM location: us-west1-c
* GCS Bucket location: us-west1
* Framework: FIO

### FIO Spec
* Test runtime: 60sec
* Thread count: 40
* Block Size: 256KB for 256KB files and 1MB for all other files.

### GCSFuse command
```
gcsfuse --implicit-dirs --stat-cache-ttl=60s --type-cache-ttl=60s 
--client-protocol http1 --max-conns-per-host=100 <bucket-name> <path-to-mount-point>
```
## Reads
### Sequential reads

| File Size | Bandwidth in GiB/sec | IOPS | Avg Latency (msec) |
|-----------|----------------------|------|--------------------|
| 256KB     | 0.47                 | 1875 | 1350               |
| 1MB       | 1.56                 | 1554 | 1623               |
| 2MB       | 2.27                 | 2279 | 1110               |
| 5MB       | 3.75                 | 3796 | 670                |
| 10MB      | 4.02                 | 4071 | 625                |
| 50MB      | 4.37                 | 4433 | 574                |
| 100MB     | 4.33                 | 4390 | 579.94             |
| 200MB     | 4.31                 | 4371 | 582.89             |
| 1GB       | 4.32                 | 4383 | 580.67             |

### Random reads

| File Size | Bandwidth in GiB/sec | IOPS  | Avg Latency (msec) |
|-----------|----------------------|-------|--------------------|
| 256KB     | 0.45                 | 1780  | 1418               |
| 1MB       | 1.46                 | 1456  | 1728               |
| 2MB       | 2.00                 | 2010  | 1256               |
| 5MB       | 1.64                 | 1641  | 1539               |
| 10MB      | 1.44                 | 1435  | 1761               |
| 50MB      | 1.31                 | 1301  | 1934               |
| 100MB     | 1.07                 | 1056  | 2365               |
| 200MB     | 1.06                 | 1044  | 2405               |
| 1GB       | 0.91                 | 889   | 2809               |


### Recommendation for reads
GCSFuse performs well for sequential reads and recommendation is to use GCSFuse
for doing sequential reads on file sizes > 10MB and < 1GB. Always use http1 
(--client-protocol=http1, enabled by default) and --max-connections-per-host  
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
  
  gcsfuse --implicit-dirs --stat-cache-ttl=60s --type-cache-ttl=60s --client-protocol http1 
  --max-conns-per-host=100 <bucket-name> <path-to-mount-point>
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
rw=read
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