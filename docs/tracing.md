## GCSFuse Tracing

### Introduction

GCSFuse traces each of its FUSE file system operations. It further traces the Google Cloud Storage Go SDK client operations along with the underlying HTTP and gRPC client calls inside every FUSE file system operation. GCSFuse supports exporting the traces locally to the process output stream and also to Google Cloud Trace. For information on permissions, requirements for exporting traces to Google Cloud Trace, and basic tracing concepts, refer to the [Google Cloud Trace Docs](https://docs.cloud.google.com/trace/docs/overview).

### Enabling tracing

To enable tracing in GCSFuse and visualize the data as a waterfall/gantt chart in the Google Cloud Trace Explorer, add the required configuration to your GCSFuse config-file as shown below:

```
trace:
  sampling-ratio: 0.1
```

OR

GCSFuse command line option below:

`--trace-sampling-ratio=0.1` Sets the **sampling rate**. This means **10%** (1 out of 10) of all traces will be exported. This helps manage cost and overhead in high-traffic applications.

### Accessing and Viewing Trace Exports

Once you have run GCSFuse with tracing enabled for some time (using either `--trace-sampling-ratio` or the config file), you can view the collected traces using the Google Cloud Console:

1. Navigate to the **Trace Explorer** page in the Google Cloud Console.
2. In the Trace Explorer UI, you will see various filters and a timeline showing trace requests.
3. **Locate Your GCSFuse Traces:**
   * Look for traces with the root span name **GCSFuse FUSE Operations** (e.g., `fs.inode.open`, `fs.file.read`, `fs.dir.read`).
   * You can use the search/filter bar to find traces matching your specific GCSFuse instance.
   * For instance, you can filter by the **Instance ID**, which is structured as `<bucket-name>-<mount-id>`. Notice that `<mount-id>` is a unique random identifier generated automatically by GCSFuse, which can also be configured using the `--mount-id` flag or config-file option.

#### Example Trace Explorer Views

Once you click on a trace from the list, you will be able to inspect its full hierarchy (the waterfall view).

![](https://github.com/user-attachments/assets/26baaa4c-083f-486e-8db3-6f2f38cb2ecc)

Trace Explorer view filtering by mount instance ID

#### Waterfall/Gantt Chart Breakdown

Clicking into a single trace shows the breakdown of a single FUSE operation and its underlying GCS calls:

![](https://github.com/user-attachments/assets/4956965e-1b7c-4bce-b6d0-18def4fc239e)

Waterfall/Gantt chart view of a single trace

### Best practices

1. **Performance Impact:** Enabling high sampling rates (e.g., `--trace-sampling-ratio=1.0`, meaning 100%) on high-traffic mounts can introduce minor CPU/network overhead.
2. **Sampling Strategy:** A low ratio (e.g., `0.01` to `0.1`) is recommended for production workloads to capture representative performance profiles without excessive overhead or cost. Use higher ratios (`1.0`) only for targeted, short-term debugging.

### Interpreting GCSFuse Spans

Here is a list of the key span names you might see when viewing a trace waterfall in Cloud Trace Explorer, categorized by component:

| Span Group / Span Name | Description |
| :---- | :---- |
| **FUSE File System Operations** | These spans represent the top-level requests sent by the Linux Kernel (via FUSE) to GCSFuse. They correspond to standard POSIX calls. |
| **fs.inode.lookup** | Searching for a file or directory name inside a parent directory. |
| **fs.inode.get_attrs** | Retrieving file or directory metadata (equivalent to the stat system call). |
| **fs.inode.set_attrs** | Modifying file or directory metadata (e.g., changing permissions or timestamps via chmod/chown/utimens). |
| **fs.inode.unlink** | Deleting a file (directory entry removal). |
| **fs.inode.forget** | Kernel telling GCSFuse it no longer caches the inode in memory. |
| **fs.dir.open** | Opening a directory handle for reading its contents. |
| **fs.dir.read** | Reading directory entries (e.g., running the `ls` command). |
| **fs.dir.mk** | Creating a new directory (`mkdir`). |
| **fs.dir.rm** | Deleting an empty directory (`rmdir`). |
| **fs.dir.rename** | Moving or renaming a file or directory. |
| **fs.file.open** | Opening an existing file for I/O. |
| **fs.file.create** | Creating and opening a new file. |
| **fs.file.read** | Reading a chunk of data from an open file. |
| **fs.file.write** | Writing a chunk of data to an open file. |
| **fs.file.flush** | Flushing dirty data/writes from GCSFuse buffers (often happens on file close). |
| **fs.file.release** | Final cleanup when the last handle to an open file is closed. |
| **fs.file.sync** | Explicitly forcing dirty file data and metadata to persistent storage (`fsync`). |
| **Read/Write Operations** | These internal GCSFuse spans break down the I/O performance into caching and GCS client logic. |
| **read.buffered** | Read fulfilled using GCSFuse's internal read-ahead or buffered reader. |
| **read.file_cache** | Read fulfilled from the local File Cache (SSD/RAM). Latency here is local. |
| **read.gcs** | Read that missed local cache and is fulfilled directly by fetching data from GCS. |
| **read.file_cache.async_download** | An asynchronous background task downloading the full object into the local File Cache. |
| **write.streaming** | Write using streaming upload directly to GCS via a background writer. |
| **write.staged** | Write staged to a local temporary file first, uploaded to GCS on flush/sync. |
| **write.streaming.upload.block** | Time spent writing an individual block of data to the GCS streaming upload channel. |
| **write.streaming.upload.finalize** | Time spent completing and finalizing the upload to GCS on flush/sync. |
| **GCS Client Library (Go SDK)** | Client-side spans emitted by `cloud.google.com/go/storage` around API requests, chunked uploads, and retries. |
| **cloud.google.com/go/storage.Object.Reader** | Client-side operation to read object data. |
| **cloud.google.com/go/storage.Object.MultiRangeDownloader** | Client-side operation for reading multiple byte ranges concurrently. |
| **cloud.google.com/go/storage.Object.Writer** | Client-side operation to write or upload object data. |
| **cloud.google.com/go/storage.Object.WriterFromAppendableObject** | Client-side operation to resume writing to an appendable object. |
| **cloud.google.com/go/storage.Object.Attrs** | Client-side operation to fetch an object's metadata. |
| **cloud.google.com/go/storage.Object.Update** | Client-side operation to update an object's metadata. |
| **cloud.google.com/go/storage.Object.Delete** | Client-side operation to delete an object. |
| **cloud.google.com/go/storage.Copier.Run** | Client-side operation to copy or rewrite an object. |
| **cloud.google.com/go/storage.Composer.Run** | Client-side operation to compose multiple source objects into one object. |
| **cloud.google.com/go/storage.Storage.UploadChunk** | Individual chunk upload within a resumable write. |
| **cloud.google.com/go/storage.RetryBackoff** | Backoff wait time between retryable Storage request attempts. |
| **GCS Operations (gRPC)** | |
| **google.storage.v2.Storage/ListObjects** | The gRPC call to list a collection of objects (like a directory listing) within a Google Cloud Storage bucket. |
| **cloud.google.com/go/storage.grpcStorageClient.ObjectsListCall** | The gRPC client-side function call within the Go library that initiates the object listing operation. |
| **cloud.google.com/go/storage.grpcStorageClient.NewRangeReader** | Initializes a gRPC range read stream for an object. |
| **cloud.google.com/go/storage.grpcStorageClient.NewRangeReaderReadObject** | Executes the underlying gRPC object read stream with retry handling. |
| **google.storage.v2.Storage/ReadObject** | The gRPC call to stream the content (data) of a specific GCS object. |
| **google.storage.v2.Storage/BidiReadObject** | Bidirectional gRPC stream for reading object ranges (such as multi-range downloads). |
| **google.storage.v2.Storage/GetObject** | The gRPC call to retrieve the **metadata/attributes** (not the data content) of a single GCS object. |
| **google.storage.v2.Storage/BidiWriteObject** | Bidirectional gRPC stream for writing or appending object data. |
| **google.storage.v2.Storage/StartResumableWrite** | Initiates a resumable upload session over gRPC. |
| **google.storage.control.v2.StorageControl/GetFolder** | A specific gRPC call to retrieve metadata for a GCS Folder (using the Storage Control API). |
| **google.storage.control.v2.StorageControl/GetStorageLayout** | A Storage Control gRPC call to inspect bucket storage layout properties. |
| **GCS Operations (HTTP)** | |
| **HTTP GET** | An entire end-to-end trace for a client's GET request. |
| **HTTP POST** | An entire end-to-end trace for a client's POST request. |
| **HTTP PUT** | An HTTP request used to upload chunks in resumable uploads. |
| **cloud.google.com/go/storage.httpStorageClient.ObjectsListCall** | A specific operation to list objects within a Google Cloud Storage (GCS) bucket. |
| **cloud.google.com/go/storage.httpStorageClient.NewRangeReader** | Initiates an HTTP range read request for an object. |
| **cloud.google.com/go/storage.Object.Attrs** | A specific operation to get the metadata/attributes of a single GCS object. |
| **Low-level HTTP Transport** | |
| **http.dns** | Time spent resolving the domain name to an IP address. |
| **http.getconn** | Time spent obtaining a connection (either from a connection pool or establishing a new TCP connection). |
| **http.headers** | Time spent writing HTTP request headers to the wire. |
| **http.send** | Time spent sending the entire request body to the server. |
| **http.receive** | Time spent receiving the entire response body from the server. |

### Differentiating gRPC from HTTP Spans

The easiest way to tell whether a trace is for a gRPC or HTTP call in Google Cloud Trace Explorer is by looking at the **Span Naming Convention** and the **Trace Attributes**.

#### Naming Convention

| Trace Type | Naming Pattern | Example |
| :---- | :---- | :---- |
| **gRPC** | Uses the full **Service/Method** format, often starting with the API version and service name. | google.storage.v2.Storage/ListObjects |
| **HTTP** | Uses the **HTTP method** or lower-level network phases. | HTTP GET, http.dns, http.send |

#### Trace Content and Attributes

The attributes (tags) attached to the span clearly indicate the protocol and operation context:

* **gRPC Spans** contain OpenTelemetry attributes starting with `rpc.`:
  * `rpc.system.name`: Typically `"grpc"`
  * `rpc.method`: The full service and method name (e.g., `google.storage.v2.Storage/GetObject`)
  * `rpc.response.status_code`: The gRPC status name or code (e.g., `OK`, `NOT_FOUND`, `CANCELLED`)
* **HTTP Spans** contain OpenTelemetry attributes starting with `http.` and `url.`:
  * `http.request.method`: The HTTP verb (`GET`, `POST`, `PUT`)
  * `url.full`: The full request URL
  * `http.response.status_code`: The three-digit HTTP status code (e.g., `200`, `206`, `404`)
* **Google Cloud Storage SDK Spans** may also include helpful storage and retry attributes:
  * `gcp.storage.uri`: The target bucket or object URI (`gs://<bucket>/<object>`)
  * `gcp.storage.read.mode` / `gcp.storage.write.mode`: The transfer mode used (e.g., `range`, `multi_range`, `oneshot`, `resumable`, `appendable`)
  * `gcp.storage.upload.id` / `gcp.storage.upload.chunk.number`: Resumable upload identifier and chunk sequence number
  * `gcp.storage.payload.offset` / `gcp.storage.payload.size_bytes`: Byte offset and payload size for a read or chunk upload
  * `gcp.client.retry.attempt_number`: Attempt counter recorded on retried requests (`1`, `2`, ...)
  * `error.type`: Error classification attached when an operation fails (e.g., `NOT_FOUND`)

### Sampling Ratios

When choosing a sampling ratio for tracing, consider the following:

* `--trace-sampling-ratio=0.0` (Default): Tracing is disabled. No spans will be sampled or exported.
* `--trace-sampling-ratio=0.1` (10% Sampling): Good for general monitoring in production. Captures enough data for anomaly detection without overwhelming trace storage or incurring high costs.
* `--trace-sampling-ratio=1.0` (100% Sampling): Exports every trace. Best suited for short-term debugging, reproducing specific issues, or low-volume testing.

By examining the waterfall view of a trace, you can visually see how these spans nest and which ones contribute most to the overall latency of the parent FUSE operation.
