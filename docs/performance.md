General performance considerations & best practices

# Best practices

- Sequential vs Random reads and writes:
     - Performance will be significantly better for sequential workloads vs random read workloads. Specifically for reads, Cloud Storage FUSE has a read-ahead feature that further improves performance for sequential reads:  
          - Cloud Storage FUSE uses a heuristic to detect when a file is being read sequentially, and will issue fewer, larger read requests to Cloud Storage in this case, using the same open TCP connection.
          - The consequence of this is that Cloud Storage FUSE is relatively efficient when reading or writing entire large files, but will not be particularly fast for small numbers of random writes within larger files, and to a lesser extent the same is true of small random reads.
     - Cloud Storage FUSE has higher latency than a local file system. As such, latency and throughput will be reduced when reading or writing one small file at a time.
     - Sequential reads: It is recommended to use file sizes that are 5MB>x>200MB for best performance.
     - Random reads: Sequential reads will always perform better. If random reads are required, it is recommended to use file sizes around 2MB for best performance.
     - Writes: Sequential and random writes will behave roughly the same, as new and modified files are fully staged in the local temporary directory until they are written out to Cloud Storage from being closed or fsync'd. 
- Cloud Storage can provide very high throughput, especially aggregated across multiple objects. Using larger files and working across multiple different files at a time will help to increase throughput.

# IOPS

Workloads that require high instantaneous IOPS (called queries-per-second in Cloud Storage), very high IOPS on a single file system, or best-in-class latency are best served by FileStore or potentially other file server solutions.

List operations (“ls”) can be expensive and slow as these details need to be fetched from Cloud Storage. The more objects in a bucket, the more expensive the operation. 

# Latency and rsync

Cloud Storage FUSE has higher latency than a local file system. As such, latency and throughput may be reduced when reading or writing one small file at a time. Using larger files and/or transferring multiple files at a time will help to increase throughput.

To achieve high throughput, larger files should be used to smooth across latency hiccups or read/write multiple files at a time.

Note in particular that this heavily affects rsync, which reads and writes only one file at a time. You might try using gsutil -m rsync to transfer multiple files to or from your bucket in parallel instead of plain rsync with Cloud Storage FUSE.

# Rate limiting

If you would like to rate limit traffic on Cloud Storage FUSE to/from Cloud Storage, the following flags can be used:

- The flag ```--limit-ops-per-sec``` controls the rate at which Cloud Storage FUSE will send requests to Cloud Storage.
- The flag ```--limit-bytes-per-sec``` controls the egress bandwidth from Cloud Storage FUSE to Cloud Storage.

All rate limiting is approximate, and is performed over an 8-hour window. By default, there are no limits applied.

# Upload procedure control

An upload procedure is implemented as a retry loop with exponential backoff for failed requests to the Cloud Storage backend. Once the backoff duration exceeds this limit, the retry stops. Flag ```--max-retry-sleep``` controls such behavior. The default is 1 minute. A value of 0 disables retries.

# Cloud Storage round trips

By default, Cloud Storage FUSE uses two forms of caching to save round trips to Cloud Storage, at the cost of consistency guarantees. These caching behaviors can be controlled with the flags ```--stat-cache-capacity```, ```--stat-cache-ttl``` and ```--type-cache-ttl```. See semantics.md for more information.

# Benchmarks

