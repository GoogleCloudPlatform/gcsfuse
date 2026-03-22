# GCSFuse Write Flow Traces

This document outlines the OpenTelemetry traces added to GCSFuse's write flows (staged and streaming writes) to aid in debugging performance and operational issues. These traces can provide insights into local buffering latency, out-of-order write fallbacks, and the actual upload time to Google Cloud Storage (GCS).

## Useful Traces for Debugging

| Trace Name / Span Name | Component | Description | Useful For Debugging |
| :--- | :--- | :--- | :--- |
| `fs.file.write.staged` | `internal/fs/inode/file.go` | Traces a write operation using the legacy staged writes (`writeUsingTempFile`). | Understanding the latency of writing to the local temporary file buffer, distinguishing it from streaming writes latency. |
| `fs.file.write.streaming` | `internal/fs/inode/file.go` | Traces a write operation using the buffered writes handler (`writeUsingBufferedWrites`). | Tracking the time taken to buffer the write locally before uploading. Helps identify if the buffer pool is exhausted and the write is blocking. |
| `fs.file.sync.staged` | `internal/fs/inode/file.go` | Traces the synchronization of the staged temp file to GCS (`syncUsingContent`). | Measuring the total time taken to upload the staged file to GCS. Helps distinguish local write time from upload time, identifying slow GCS connections. |
| `fs.file.sync.streaming` | `internal/fs/inode/file.go` | Traces the synchronization of buffered writes to GCS (`SyncPendingBufferedWrites`). | Measuring the time waiting for in-flight streaming blocks to finish uploading to GCS during a file sync or close. |
| `streaming.block.upload` | `internal/bufferedwrites/upload_handler.go` | Traces the upload of a single streaming block to GCS (`uploadBlock`). | Tracking individual chunk upload latency to GCS. Helps identify if specific chunks are taking too long due to network or GCS latency. |
| `streaming.upload.finalize` | `internal/bufferedwrites/upload_handler.go` | Traces the finalization of the streaming upload (`Finalize`). | Measuring the wait time for all pending blocks to complete uploading and closing the object writer during file close/finalize. |
| `streaming.upload.flush` | `internal/bufferedwrites/upload_handler.go` | Traces the flushing of pending streaming writes (`FlushPendingWrites`). | Measuring the time spent ensuring the writer is created and waiting for background uploads during a flush operation. |

## Common Debugging Scenarios using these Traces

### 1. High latency during write operations
* **Symptom:** Operations that write data (like `cp`, `rsync`) are taking longer than expected.
* **Trace Analysis:** Check the time spent in `fs.file.write.staged` vs `fs.file.write.streaming`. If streaming writes are enabled but you see lots of `fs.file.write.staged` spans instead, the application might be doing out-of-order writes or concurrent writes limit (`--write-global-max-blocks`) has been reached.

### 2. High latency during file close/sync
* **Symptom:** Saving a file takes a long time, freezing the application briefly.
* **Trace Analysis:** Check `fs.file.sync.staged` and `fs.file.sync.streaming`. High latency in `SyncFileStaged` means the entire file is being uploaded to GCS at the time of closing. High latency in `SyncFileStreaming` indicates that the background upload process is slow, or `StreamingUploadFinalize` is blocking waiting for blocks. Check `streaming.block.upload` for individual block latencies.

### 3. Investigating "legacy staged writes fallback"
* **Symptom:** You see warnings like "Out of order write detected. File ... will now use legacy staged writes."
* **Trace Analysis:** You will notice a sequence of `fs.file.write.streaming` spans immediately followed by `fs.file.sync.streaming` (flushing the current buffer) and then `fs.file.write.staged`. This pinpoints exactly when and where the application did a non-sequential write, triggering the fallback.
