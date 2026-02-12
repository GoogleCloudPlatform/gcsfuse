# GCSFuse Network Clients and Retry Strategies

## Overview

GCSFuse uses multiple network clients to communicate with Google Cloud Storage (GCS) and related services. Each client has specific retry strategies tailored to its use case and the operations it performs.

## Network Clients

### 1. HTTP Storage Clients (HTTP/1.1 and HTTP/2)

**Purpose:** Primary storage operations using HTTP protocol

**Creation:** `createHTTPClientHandle()` in `internal/storage/storage_handle.go`

**Protocols:**
- **HTTP/1.1** (`cfg.HTTP1`): Default protocol, optimized for performance with connection pooling
- **HTTP/2** (`cfg.HTTP2`): Alternative protocol for better multiplexing

**Configuration Options:**
- `MaxConnsPerHost`: Maximum TCP connections per server (HTTP/1.1 only)
- `MaxIdleConnsPerHost`: Maximum idle connections per host
- `HttpClientTimeout`: Timeout for HTTP operations
- `ExperimentalEnableJsonRead`: Use JSON API instead of XML API for reads
- `ReadStallRetryConfig`: Configuration for handling read stalls with dynamic timeouts

**When Used:**
- Default client when `--client-protocol=http1` or `--client-protocol=http2`

---

### 2. gRPC Storage Client (Standard)

**Purpose:** Storage operations using gRPC protocol for better performance and features

**Creation:** `createGRPCClientHandle()` in `internal/storage/storage_handle.go`

**Configuration Options:**
- `GrpcConnPoolSize`: Number of gRPC connections in the pool
- `EnableGrpcMetrics`: Enable gRPC metrics collection (GKE only)
- `ExperimentalLocalSocketAddress`: Bind to specific local address

**Features:**
- DirectPath support via `GOOGLE_CLOUD_ENABLE_DIRECT_PATH_XDS` environment variable
- Connection pooling for better throughput
- gRPC interceptors for tracing and metrics

**When Used:**
- When `--client-protocol=grpc` is set
- Normal mode without DirectPath enforcement

---

### 3. gRPC Storage Client with Bidi Configuration

**Purpose:** gRPC client optimized for zonal buckets with bidirectional streaming

**Creation:** `createGRPCClientHandle()` with `enableBidiConfig=true`

**Configuration:**
- Includes `experimental.WithGRPCBidiReads()` option
- Optimized for zonal bucket operations

**When Used:**
- Automatically used for zonal buckets regardless of `--client-protocol` setting
- Provides better performance for zonal bucket operations

---

### 4. Storage Control Client (Without GAX Retries)

**Purpose:** HNS (Hierarchical Namespace) operations without default retry logic

**Creation:** `storageutil.CreateGRPCControlClient()` with `disableDefaultGaxRetries=true`

**Operations:**
- GetStorageLayout (to determine bucket type)
- Folder operations (when wrapped with custom retry logic)

**Retry Strategy:**
- No default GAX retries (custom retry logic applied via wrappers)
- Wrapped with `withRetryOnStorageLayout()` for GetStorageLayout calls
- Wrapped with `withRetryOnAllAPIs()` for zonal buckets

---

### 5. Storage Control Client (With GAX Retries)

**Purpose:** HNS folder operations with default retry logic

**Creation:** `storageutil.CreateGRPCControlClient()` with `disableDefaultGaxRetries=false`

**Operations:**
- CreateFolder
- DeleteFolder
- GetFolder
- RenameFolder

**Retry Strategy:**
- GAX retries with configurable backoff
- Custom retry codes: ResourceExhausted, Unavailable, DeadlineExceeded, Internal, Unknown

---

## Retry Strategies

### Standard Retry Configuration

**Function:** `setRetryConfig()` in `internal/storage/storage_handle.go`

**Applied To:**
- All HTTP storage clients
- All gRPC storage clients

**Parameters:**
```go
Max Backoff:         clientConfig.MaxRetrySleep (from config)
Multiplier:          clientConfig.RetryMultiplier (from config)
Max Attempts:        clientConfig.MaxRetryAttempts (0 = unlimited)
Policy:              storage.RetryAlways (retry all operations)
Error Function:      storageutil.ShouldRetryWithMonitoring()
```

**Retryable Conditions:**
- Determined by `ShouldRetryWithMonitoring()` which checks error codes
- Metrics are recorded for each retry attempt

**Default Values (from config):**
- MaxRetrySleep: Configurable via `--max-retry-sleep`
- RetryMultiplier: Configurable via `--multiplier`
- MaxRetryAttempts: Configurable via `--max-retry-attempts` (default: 0 = unlimited)

---

### GAX Retry Configuration (Control Client)

**Function:** `storageControlClientGaxRetryOptions()` in `internal/storage/control_client_wrapper.go`

**Applied To:**
- Storage Control Client folder APIs
- CreateFolder, DeleteFolder, GetFolder, RenameFolder

**Parameters:**
```go
Total Timeout:       5 minutes (DefaultTotalRetryBudget)
Max Backoff:         clientConfig.MaxRetrySleep
Multiplier:          clientConfig.RetryMultiplier
Retry Codes:         ResourceExhausted, Unavailable, DeadlineExceeded, 
                     Internal, Unknown
```

**Purpose:**
- Handle transient failures in control plane operations
- Folder API operations are critical for HNS buckets

---

### GCSFuse-Level Retry with Stall Detection

**Function:** `newRetryWrapper()` in `internal/storage/control_client_wrapper.go`

**Applied To:**
- GetStorageLayout calls (all buckets)
- All control client operations (zonal buckets)

**Parameters:**
```go
Retry Deadline:      30 seconds (DefaultRetryDeadline)
Total Budget:        5 minutes (DefaultTotalRetryBudget)
Initial Backoff:     1 second (DefaultInitialBackoff)
Max Backoff:         clientConfig.MaxRetrySleep
Multiplier:          clientConfig.RetryMultiplier
```

**Features:**
- Time-bound retry approach
- Exponential backoff with jitter
- Stall detection with deadline per attempt
- Retries on timeout and retryable errors

**Wrapper Types:**
- `withRetryOnStorageLayout()`: Retries only GetStorageLayout
- `withRetryOnAllAPIs()`: Retries all control client operations (zonal buckets)

---

### Read Stall Retry Configuration (HTTP Only)

**Configuration:** `ReadStallRetryConfig` in config

**Applied To:**
- HTTP storage clients when `ReadStallRetryConfig.Enable = true`

**Parameters:**
```go
Min Timeout:         ReadStallRetryConfig.MinReqTimeout
Target Percentile:   ReadStallRetryConfig.ReqTargetPercentile
Initial Timeout:     ReadStallRetryConfig.InitialReqTimeout (via env var)
Increase Rate:       ReadStallRetryConfig.ReqIncreaseRate (via env var)
```

**Purpose:**
- Handle stalled read operations
- Dynamic timeout adjustment based on request performance
- Uses `experimental.WithReadStallTimeout()` option

---

### Write Stall Retry Configuration

**Configuration:** `WriteStallRetryConfig` in config

**Applied To:**
- All storage clients (HTTP and gRPC) when `WriteStallRetryConfig.Enable = true`

**Parameters:**
```go
Min Timeout:         WriteStallRetryConfig.MinReqTimeout
Target Percentile:   WriteStallRetryConfig.ReqTargetPercentile
Initial Timeout:     WriteStallRetryConfig.InitialReqTimeout
Increase Rate:       WriteStallRetryConfig.ReqIncreaseRate
```

**Purpose:**
- Detect and retry stalled write operations
- Dynamic timeout adjustment for upload operations
- Applies to object writes and uploads

---

## Client Selection Decision Tree

```
┌─────────────────────────────────────────────────────────────┐
│                    Bucket Access Request                     │
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
        Is Zonal? ─── YES ──▶ gRPC Client     │
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
                                ┌──────────────────────────┐
                                │ Enable HNS?              │
                                └─────────┬────────────────┘
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
                                Zonal    Non-Zonal
                                │         │
                                │         │
                                ▼         ▼
                        GAX + GCSFuse   GAX Retries
                            Retries      for Folders
                        (All APIs)      + GCSFuse for
                                        GetStorageLayout
```

---
