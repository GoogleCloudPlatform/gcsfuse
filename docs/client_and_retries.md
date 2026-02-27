# GCSFuse Network Clients and Retry Strategies

## Overview

GCSFuse uses multiple network clients to communicate with Google Cloud Storage (GCS). Each client has specific retry strategies tailored to its use case and the operations it performs.

## Network Clients

### 1. HTTP Storage Clients (HTTP/1.1 and HTTP/2)

**Purpose:** Primary storage operations using HTTP protocol

**Protocols:**
- **HTTP/1.1** (`cfg.HTTP1`): Default protocol
- **HTTP/2** (`cfg.HTTP2`): Alternative protocol for better multiplexing

**Configuration Options:**
- `MaxConnsPerHost`: The max number of TCP connections allowed per host. This is effective when client-protocol is set to 'http1'. A value of 0 indicates no limit on TCP connections (limited by the machine specifications). (Default: 0)
- `MaxIdleConnsPerHost`: The number of maximum idle connections allowed per host. (Default: 100)
- `HttpClientTimeout`: The time duration that http client will wait to get response from the server. A value of 0 indicates no timeout. (Default: 0s)
- `ExperimentalEnableJsonRead`: By default, GCSFuse uses the GCS XML API to read objects. When this flag is specified, GCSFuse uses the GCS JSON API instead. (Default: false)
- `ReadStallRetryConfig`: To turn on/off retries for stalled read requests. This is based on a timeout that changes depending on how long similar requests took in the past. (Default: true)

### 2. gRPC Storage Client (Standard)

**Purpose:** Storage operations using gRPC protocol for better performance and features.
**Configuration Options:**
- `GrpcConnPoolSize`: The number of gRPC channel in grpc client. (Default: 1)
- `EnableGrpcMetrics`: Enables support for gRPC metrics. (Default: false)

**When Used:**
- When `--client-protocol=grpc` is set
- First tries DirectPath, defaulting to CloudPath as a secondary failover.

### 3. gRPC Storage Client with Bidi Configuration

**Purpose:** gRPC client optimized for rapid buckets with bidirectional read streaming

**When Used:**
- Automatically used for rapid buckets regardless of `--client-protocol` setting.

### 4. Storage Control Client

**Purpose:** Handles HNS folder operations and utilizes GetStorageLayout to determine the bucket type using default retry logic.

**Operations:**
- GetStorageLayout
- CreateFolder
- DeleteFolder
- GetFolder
- RenameFolder

**When Used:**
- For HNS buckets folder operations.

---

## Retry Strategies

### Standard Retry Configuration

**Applied To:**
- All HTTP storage clients
- All gRPC storage clients

**Parameters:**
```go
Max Backoff:         30 seconds (Configurable via `--max-retry-sleep`)
Multiplier:          2 (Configurable via `--retry-multiplier`)
Max Attempts:        0 (unlimited) (Configurable via `--max-retry-attempts`)
Policy:              storage.RetryAlways
```

---

### GCSFuse-Level Control Client Retry with Stall Detection

Implemented [custom retry logic](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/internal/storage/storageutil/retry.go) for Folder APIs to mitigate stall issues and improve request reliability.

**Applied To:**
- GetStorageLayout calls (all buckets)
- All control client operations (rapid buckets)

**Default Parameters:**
```go
Retry Deadline:      30 seconds
Total Budget:        5 minutes
Initial Backoff:     1 second
```

**Features:**
- Time-bound retry approach
- Exponential backoff with jitter
- Stall detection with deadline per attempt
- Retries on timeout and retryable errors

---

### Read Stall Retry Configuration (HTTP Only)

**Configuration:** `ReadStallRetryConfig` in config

**Applied To:**
- HTTP storage clients when `ReadStallRetryConfig.Enable = true`

**Default Parameters:**
```go
Min Timeout:         1.5 seconds (Configurable via `--read-stall-min-req-timeout`)
Target Percentile:   0.99 (Configurable via `--read-stall-req-target-percentile`)
Initial Timeout:     20 seconds (Configurable via `--read-stall-initial-req-timeout`)
Max Timeout:         20 minutes (Configurable via `--read-stall-max-req-timeout`)
Increase Rate:       15 (Configurable via `--read-stall-req-increase-rate`)
```

**Purpose:**
- Handle stalled read operations
- Dynamic timeout adjustment based on request performance

### Write Stall Retry Configuration (HTTP Only)

**Default Parameters:**
```go
Chunk Transfer Timeout: 10 seconds (Configurable via `--chunk-transfer-timeout-secs`)
```

**Purpose:**
- Detect and retry (without exponential backoff) stalled chunk write operations within 10 seconds for resumable uploads.

---

## Client Selection Decision Tree

```
┌─────────────────────────────────────────────────────────────┐
│                    Bucket Access Request                    │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
                  ┌────────────────────┐
                  │  Operation Type?   │
                  └──────────┬─────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
         Storage Ops                    Control Ops
    (Read/Write/List/Stat)       (GetStorageLayout/Folders)
              │                             │
              ▼                             ▼
      ┌───────────────┐            ┌─────────────────────┐
      │ Lookup Bucket │            │  Storage Control    │
      │     Type      │            │      Client         │
      └───────┬───────┘            └──────────┬──────────┘
              │                               │
              ▼                               ▼
        Is rapid? ─── YES ──▶ gRPC Client     │
              │               with Bidi       │
              │               Config          │
              NO                              │
              │                               │
              ▼                               │
    ┌─────────────────────┐                   │
    │ Client Protocol?    │                   │
    └──────────┬──────────┘                   │
               │                              │
      ┌────────┴────────┬─────────────┐       │
      │                 │             │       │
    HTTP1/2           GRPC         OTHER      │
      │                 │             │       │
      │                 │             │       │
      ▼                 ▼             ▼       ▼
      │                 │           Error     │
      │                 │                     │
      │    Standard gRPC Client               │
      │                                       │
      ▼                                       │
HTTP Storage Client                           │
                                              │
                                              ▼
                                ┌─────────────────────────┐
                                │ Enable HNS?             │
                                └────────┬────────────────┘
                                         │
                                    ┌────┴────┐
                                    │         │
                                    YES       NO
                                    │         │
                                    │         └──▶ No Control Client
                                    │
                                    ▼
                            ┌─────────────────┐
                            │   Bucket Type?  │
                            └────────┬────────┘
                                     │
                                ┌────┴────┐
                                │         │
                                Rapid    Non-rapid
                                │         │
                                │         │
                                ▼         ▼
                        GAX + GCSFuse   GAX Retries
                            Retries      for Folders
                        (All APIs)      + GCSFuse for
                                        GetStorageLayout
```

---
