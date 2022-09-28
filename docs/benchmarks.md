# GCSFuse Performance Benchmarks

FIO is used to perform load tests on GCSFuse. Below tables shows performance metrics of GCSFuse for different workloads for the given test setup:

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
* Block Size: 1MB

### GCSFuse command
```
gcsfuse  --max-conns-per-host=100 --implicit-dirs --stat-cache-ttl=60s 
--type-cache-ttl=60s --disable-http2 <bucket-name> ~/gcs
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
for doing sequential reads on file sizes > 10MB and < 1GB.