# Cloud Storage FUSE Metrics Reference Guide

For instructions on how to enable and export Cloud Storage FUSE metrics, refer to the official Google Cloud metrics guide at <https://cloud.google.com/storage/docs/cloud-storage-fuse/metrics>.

This document provides a detailed technical reference for the **experimental client-side metrics** introduced in the `read-split` tracking branches to monitor GCS reader lifetimes, strategy transitions, and connection preemption cancellations.

---

## Experimental GCS Metrics

These metrics are prefixed with `gcs/experimental_` and are designed to capture connection preemption telemetries to help align GCSFuse client-side connection states with GCS server-side request logs.

### 1. `gcs/experimental_reader_cancellation_count`

* **Metric Type**: Cumulative Integer Counter (`int_counter`)
* **Description**: Tracks the cumulative number of times an in-flight open GCS object reader connection is aborted or explicitly closed before the requested GCS byte range has been fully read. Each preemption registers as an aborted/cancelled request in GCS server-side logs; this metric maps the client-side preemption trigger directly to those telemetry cancellations.
* **Attributes**:
  * **`reason`** (string): Identifies the direct trigger of the GCS stream preemption cancellation:
    * **`"canceled"`**: The calling FUSE kernel context was explicitly canceled (e.g., manual read preemption, SIGINT, client interrupt, or aborted threads).
    * **`"deadline_exceeded"`**: The calling context expired because of a read timeout or deadline breach.
    * **`"inactive_timeout"`**: The background inactivity monitoring loop explicitly reclaimed and closed the idle connection because no read operations were seen during the timeout window.
    * **`"seek"`**: A FUSE offset jump (seek) within the sequential read strategy required closing the active GCS HTTP stream before it was fully read to realign the offset.
    * **`"sequential_to_random"`**: The classifier transitioned the active read strategy from sequential to random access, preempting and closing the active RangeReader sequential connection.
    * **`"explicit_close"`**: The client explicitly closed/released the local file handle before consuming the full prefetch range requested from GCS.
    * **`"forced_recreate"`**: GCSFuse forced the recreation/refresh of the active range reader before the GCS stream range was fully read.
    * **`"unknown"`**: A fallback indicator for untracked socket closures.

---

## Experimental Read Operation Metrics

These metrics are prefixed with `read/experimental_` and monitor read strategy heuristics, seeking behaviors, and runtime strategy modifications.

### 1. `read/experimental_read_type_transitions_count`

* **Metric Type**: Cumulative Integer Counter (`int_counter`)
* **Description**: Tracks the cumulative number of read strategy state-transitions (from sequential-to-random or random-to-sequential) executed by the adaptive read strategy classifier.
* **Attributes**:
  * **`transition_type`** (string): The direction of the strategy transition:
    * **`"sequential_to_random"`**: Transitioned from sequential prefetch reads to random-access reads.
    * **`"random_to_sequential"`**: Transitioned from random-access reads back to sequential prefetch access.
  * **`reason`** (string): The concrete read pattern heuristic that triggered the transition:
    * **`"initial_offset_non_zero"`**: The first read request arrived at a non-zero starting offset, causing GCSFuse to initialize the read strategy as random access.
    * **`"backward_seek"`**: A backward jump in read offset triggered a random-access state.
    * **`"forward_seek"`**: A forward jump beyond the sequential prefetch window (8MB) triggered a random-access state.
    * **`"average_read_size_large_enough"`**: The average block read size successfully crossed the prefetch window limit (8MB), triggering a switch back to sequential prefetch access.
