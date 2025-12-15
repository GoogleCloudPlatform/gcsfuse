# GCSFuse Metrics Debugging Playbook

This playbook provides a guide on how to use GCSFuse metrics for debugging performance issues and errors.


## Introduction
GCSFuse metrics provide deep visibility into the file system's behavior and its interaction with Google Cloud Storage. By monitoring these metrics, you can:
- **Identify Performance Bottlenecks:** Distinguish between network latency, GCS API latency, and local processing delays.
- **Optimize Configuration:** Tune GCSFuse flags (like caching or concurrency) based on observed throughput and operation counts.

## Prerequisites
To use metrics for debugging, you must first enable and configure them.

Please refer to the official documentation for detailed instructions on setting up Cloud Storage FUSE metrics:
[Cloud Storage FUSE Metrics Guide](https://docs.cloud.google.com/storage/docs/cloud-storage-fuse/metrics)

## Common Debugging Scenarios

### High Latency
If you observe slow file operations, check `gcs/request_latency`:

1.  **Filter by `method`:** Look at specific methods like `NewReader` (read) or `CreateObject` (write).
2.  **Analyze `NewReader` Latency:**
    *   If `NewReader` latency is high, it suggests a delay in establishing the connection or receiving the first byte.
    *   **Potential Causes:**
        *   High network latency between the client and GCS.
        *   DNS resolution issues.
        *   GCS server-side delays.

### Client-side Latency & Tracing
Sometimes, high latency isn't due to GCS itself but stems from client-side issues like DNS resolution or connection establishment.

*   **Enable Tracing:**
    To dive deeper into client-side latency, you can enable experimental tracing. This allows you to visualize the breakdown of a request, including DNS lookup, TLS handshake, and connection wait times.
    
    Refer to the [GCSFuse Tracing Guide](tracing.md) for instructions on how to enable tracing and interpret the results (e.g., `http.dns`, `http.tls`, `http.getconn` spans).
