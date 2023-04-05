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
* FIO creates a number of threads or processes doing a particular type of I/O action specified by the user.
* The use of fio is to write a job file matching the I/O load one wants to simulate.
* For testing different sizes of the file, change file size parameters.
* Blocksize defines how large size we are issuing for I/O. For 256kb we used 16k block size other than that we used 1M blocksize.
* We have a fsync parameter for writes that defines fio will sync the file after every fsync number of writes issued. When the Writefile operation called GCSFuse will write the data to disk. When the SyncFile operation called GCSFuse will write the data from disk to GCS bucket. So after fsync write operation GCSFuse will write the data into the GCS bucket. We tested the write operation for fsync value 1, 3, 10 and equal to file size. So fsync value one means after one write operation GCSFuse will write the data into the GCS bucket.

### GCSFuse command
```
gcsfuse --implicit-dirs  --client-protocol=http1 --max-conns-per-host=100 <bucket-name> <path-to-mount-point>
```

### Sequential Write

## 256KB (Block Size: 16k)
| Threads | Fsync | Bandwidth in MiB/sec | IOPS(avg) | Avg Latency (msec) | CPU Utilization(%) | Network Traffic Send(GiB/s)| Network Traffic Receive(GiB/s) |
|---------|-----|------------------|----------|------------------|------------------|--------------------------|--------------------------------|
| 48      |1|3.104|279.18|37.617|1.19|0.053|0.056|
| 64|1|3.531|354.36|38.747|1.13|0.072|0.065|
| 96|1|4.982|523.3|41.262|1.67|0.069|0.063|
| 112|1|4.985|613.64|37.756|1.69|0.078|0.0.68|
| 48|3|4.425|787.363|12.076|0.87|0.038|0.032|
| 64|3|5.754|1008.8|12.415|0.95|0.032|0.026|
| 96|3|5.618|1602.83|11.998|1.19|0.051|0.041|
| 112|3|8.314|1851.1|11.931|1.5|0.055|0.046|
| 48|10|11.269|2967.87|3.301|0.88|0.016|0.013|
| 64|10|16.56|3554.19|3.567|1.01|0.039|0.03|
| 96|10|20.73|5293.73|3.599|1.41|0.048|0.039|
| 112|10|24.76|6186.59|3.573|1.55|0.038|0.029|
| 48|16|30.4|4353.34|2.287|1.72|0.04|0.04|
| 64|16|38.7|5681.36|2.268|1.3|0.03|0.02|
| 96|16|46.6|8489.89|2.307|2.63|0.07|0.06|
| 112|16|62.3|9872.44|2.278|2.58|0.03|0.03|

## 1MB (Block Size: 1M)
| Threads | Fsync | Bandwidth in MiB/sec | IOPS(avg) | Avg Latency (msec) | CPU Utilization(%) | Network Traffic Send(GiB/s)| Network Traffic Receive(GiB/s) |
|---------|-----|------------------|----------|------------------|------------------|--------------------------|--------------------------------|
| 48|1|123|239.11|43.004|1.02|0.1|0.08|
| 64|1|197|324.3|44.528|1.38|0.15|0.13|
| 96|1|200|471.31|43.127|3.13|0.28|0.25|
| 112|1|359|564.89|47.064|8|1.13|1.12|
| 48|10|1091|2302.28|4.996|2.62|0.14|0.12|
| 64|10|1127|3199.25|4.815|2.38|0.14|0.11|
| 96|10|1726|4574.98|6.306|3.17|0.17|0.15|
| 112|10|2524|3871.71|15.15|4.94|0.28|0.25|

## 50MB (Block Size: 1M)
| Threads | Fsync | Bandwidth in MiB/sec | IOPS(avg) | Avg Latency (msec) | CPU Utilization(%) | Network Traffic Send(GiB/s)| Network Traffic Receive(GiB/s) |
|---------|-----|------------------|----------|------------------|------------------|--------------------------|--------------------------------|
| 48|1|54.16|91.48|303.95|12.04|1.37|2.31|
| 64|1|58.36|122.58|349.442|13.26|2.33|2.42|
| 96|1|66.33|182.18|345.304|17.71|2.57|3.3|
| 112|1|71.96|212.48|359.847|15.42|3.09|3.16|
| 48|3|155.33|282.39|107.183|13.31|2.35|2.43|
| 64|3|171.66|376.56|119.382|13.9|2.5|2.59|
| 96|3|201.66|563.45|118.74|17.17|3.34|3.44|
| 112|3|206.33|657.88|131.707|16.17|3.06|3.14|
| 48|10|506|939.6|32.463|13.26|2.37|2.46|
| 64|10|614.66|1249.94|30.995|14.96|2.55|2.33|
| 96|10|658|1869.02|32.214|14.19|2.27|2.39|
| 112|10|704.33|2183.96|34.921|18.17|3.15|3.3|
| 48|50|2628|4181.78|6.815|11.06|1.85|1.89|
| 64|50|2941|5068.06|8.081|12.99|1.94|2.02|
| 96|50|3010|4664.81|16.11|15.37|2.25|2.37|
| 112|50|3025|4588.38|19.991|14.08|2.27|2.3|

## 100MB (Block Size: 1M)
| Threads | Fsync | Bandwidth in MiB/sec | IOPS(avg) | Avg Latency (msec) | CPU Utilization(%) | Network Traffic Send(GiB/s)| Network Traffic Receive(GiB/s) |
|---------|-----|------------------|----------|------------------|------------------|--------------------------|--------------------------------|
| 48|1|66.6|98.09|327.395|12.85|1.7|1.76|
| 64|1|48.5|123.24|411.865|13.8|2.47|2.8|
| 96|1|35.3|181.51|500.443|15.77|3.04|3.22|
| 112|1|35.5|210.59|603.911|10.85|1.87|2.01|
| 48|10|288|940.62|46.432|13.6|2.48|2.6|
| 64|10|331|1243.46|49.953|14.12|3.2|3.32|
| 96|10|366|1868.85|58.043|16.57|3.11|3.57|
| 112|10|357|2193.88|66.585|17.12|3.41|3.45|
| 48|100|2696|6864.94|6.443|13.79|2.2|2.3|
| 64|100|2974|6746.49|9.199|17.3|2.63|2.41|
| 96|100|2331|6358.86|15.026|13.32|1.86|2.01|
| 112|100|2904|6242.3|18.648|16.75|2.48|2.53|

## 1GB (Block Size: 1M)
| Threads | Fsync | Bandwidth in MiB/sec | IOPS(avg) | Avg Latency (msec) | CPU Utilization(%) | Network Traffic Send(GiB/s)| Network Traffic Receive(GiB/s) |
|---------|-----|------------------|----------|------------------|------------------|--------------------------|--------------------------------|
| 48|1|2.394|91.79|4367.78|11.7|2.59|2.54|
| 64|1|2.488|121.49|4085.435|10.96|2.36|2.43|
| 96|1|2.609|182.28|22540.331|12.4|2.66|2.77|
| 112|1|2.455|210.82|36481.528|11.15|2.4|2.49|
| 48|1024|2811.66|18212.23|6.845|17.486|2.77|2.82|
| 64|1024|2875|11155.96|9.789|18.45|3.02|3.13|
| 96|1024|2371|28454.36|30.737|16.5|2.66|2.77|
| 112|1024|766.66|1726.71|133.27|9.12|2.17|1.21|

## 4GB (Block Size: 1M)
| Threads | Fsync | Bandwidth in MiB/sec | IOPS(avg) | Avg Latency (msec) | CPU Utilization(%) | Network Traffic Send(GiB/s)| Network Traffic Receive(GiB/s) |
|---------|-----|------------------|----------|------------------|------------------|--------------------------|--------------------------------|
| 48|1|0.346|89.85|114342.26|5.18|1.48|1.43|
| 64|1|0.306|117.27|179542.39|5.72|1.94|1.04|
| 96|1|0.282|164.83|284485.29|7.33|2.83|1.6|
| 112|1|0.294|174.17|293227.72|5.82|2.3|1.04|
| 48|10|3.205|816.75|12475.076|5.71|1.37|1.38|
| 64|10|2.613|1001.34|20581.715|6.24|1.22|2.03|
| 96|10|2.168|1312.38|36136.499|5.99|1.99|1.06|
| 112|10|2.372|873.11|36962.433|5.67|2.13|1.11|
| 48|4096|476|1088.53|89.894|7.18|1.77|1.27|
| 64|4096|423|980.65|129.703|6.97|2.08|1.18|
| 96|4096|477|850.86|175.337|9.3|3.09|1.36|
| 112|4096|335|944.84|211.772|6.94|1.28|0.75|

### Comparison of Sequential Writes Vs. Random Writes
In the case of Random writes, only offset will change; there are no changes in gcs calls, for that reason bandwidth and other results remain the same.
Please refer below results for comparison.

## 50MB (Block Size: 1M)
| Threads| Fsync | Bandwidth in MiB/sec Random Writes |Bandwidth in MiB/sec Seq. Writes |  IOPS(avg) Random Writes |IOPS(avg) Seq. Writes| Avg Latency Random Writes (msec) |Avg Latency Seq. Writes (msec) | CPU Utilization Random Writes(%) | CPU Utilization Seq. Writes(%) | Network Traffic Send(GiB/s) Random Writes  |Network Traffic Send(GiB/s) Seq. Writes| Network Traffic Receive(GiB/s) Random Writes |Network Traffic Receive(GiB/s) Seq. Writes |
|--------|-------|------------------------------------|---------------------------------|--------------------------|---------------------|----------------------------------|-------------------------------|----------------------------------|--------------------------------|--------------------------------------------|---------------------------------------|----------------------------------------------|-------------------------------------------|
|48|1|56.8|51.4|92.43|91.82|284.381|293.842| 13.55                            | 10.64                          | 2.09                                       |2.19|2.18|2.26|
|64|1|55.1|55.1|122.11|121.83|265.07|313.158| 12.73                            | 11.2                           | 1.91                                       |1.69|1.99|1.76|
|96|1|58.7|56.3|182.49|182.36|386.499|358.22| 12.37                            | 11.5                           | 2.5                                        |2.76|2.59|2.86|
|112|1|63.6|60.1|212.35|213.06|300.625|339.086| 14.36                            | 9.27                           | 2.35                                       |2.43|2.47|2.49|
|48|50|2543|2518|4274.96|4255.35|7.955|8.023| 13.73                            | 13.32                          | 2.41                                       |2.34|2.5|2.42|
|64|50|2668|2726|5564.39|5384.53|8.807|9.212| 16.15                            | 16.22                          | 2.42                                       |2.67|2.52|2.75|
|96|50|2535|2535|7728.4|7840.93|8.829|8.901| 13.25                            | 13.76                          | 2.2                                        |2.55|2.25|2.63|
|112|50|2854|2872|7354.27|6834.71|13.057|14.568| 18.02                            | 16.95                          | 2.53                                       |2.47|2.62|2.6|

## 1GB (Block Size: 1M)
| Threads| Fsync | Bandwidth in MiB/sec Random Writes |Bandwidth in MiB/sec Seq. Writes |  IOPS(avg) Random Writes |IOPS(avg) Seq. Writes| Avg Latency Random Writes (msec) |Avg Latency Seq. Writes (msec) | CPU Utilization Random Writes(%) | CPU Utilization Seq. Writes(%) | Network Traffic Send(GiB/s) Random Writes  |Network Traffic Send(GiB/s) Seq. Writes| Network Traffic Receive(GiB/s) Random Writes |Network Traffic Receive(GiB/s) Seq. Writes |
|--------|-------|------------------------------------|---------------------------------|--------------------------|---------------------|----------------------------------|-------------------------------|----------------------------------|--------------------------------|--------------------------------------------|---------------------------------------|----------------------------------------------|-------------------------------------------|
|48|1|2.56|2.394|92.34|91.79|5251.695|4367.78| 11.39                            | 11.7                           | 2.8                                        |2.59|2.93|2.54|
|64|1|2.64|2.488|120.2|121.49|5445.196|4085.435| 12.98                            | 10.96                          | 2.6                                        |2.36|2.71|2.43|
|96|1|2.478|2.609|179.25|182.28|23445.837|22540.331| 11.47                            | 12.4                           | 2.45                                       |2.66|2.6|2.77|
|112|1|2.373|2.455|206.05|210.82|35504.022|36481.528| 11.49                            | 11.15                          | 2.32                                       |2.4|2.41|2.49|
|48|1024|2267|2193|17345.88|17827.81|7.04|6.918| 14.58                            | 15.5                           | 2.34                                       |2.43|2.46|2.63|
|64|1024|2262|2315|15058.66|18127.58|8.494|8.013| 15.85                            | 15.84                          | 2.32                                       |2.3|2.46|2.53|
|96|1024|2208|2320|21962.9|24371.07|31.778|31.915| 14.32                            | 15.27                          | 2.25                                       |2.33|2.27|2.45|
|112|1024|1337|1815|7995.19|9875.59|71.904|50.426| 9.69                             | 12.73                          | 2.26                                       |2.05|2.35|2.13|

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