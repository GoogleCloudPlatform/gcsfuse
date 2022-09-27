# GCSFuse Performance Benchmarks

FIO is used to perform load tests on GCSFuse. Below tables shows
performance metrics of GCSFuse for different workloads for the given 
test setup:

## Test setup:

* Infra: GCP VM
* VM Type: n2-standard-48
* VM Bandwidth: 32Gbps
* VM location: us-central1-c
* GCS Bucket location: us-central1-c
* Framework: FIO
* Test runtime: 60sec

## Reads
### Sequential reads

| File Size | Bandwidth in GiB/sec |
|-----------|----------------------|
| 100KB     | 0.077                |
| 1MB       | 0.836                |
| 2MB       | 1.56                 |
| 5MB       | 3.55                 |
| 10MB      | 4.9                  |
| 50MB      | 6.39                 |
| 100MB     | 6.02                 |
| 200MB     | 6.17                 |
| 1GB       | 5.78                 |
| 5GB       | 6.23                 |

### Random reads

| File Size | Bandwidth in GiB/sec |
|-----------|----------------------|
| 100KB     | 0.079                |
| 1MB       | 0.812                |
| 2MB       | 1.23                 |
| 5MB       | 1.02                 |
| 10MB      | 0.9                  |
| 50MB      | 0.84                 |
| 100MB     | 0.82                 |
| 200MB     | 0.69                 |
| 1GB       | 0.52                 |
| 5GB       | 0.43                 |


### Recommendation for reads
GCSFuse performs well for sequential reads and recommendation is to use GCSFuse
for doing sequential reads on small files.