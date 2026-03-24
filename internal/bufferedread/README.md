# Buffered Reader and Async Reads

This package implements the buffered reading logic to prefetch data from GCS asynchronously, improving performance for large sequential and slightly out-of-order reads.

## Enabling Parallel Async Reads

To fully utilize async reads and instruct the kernel to send FUSE read requests in parallel, you must pass the following flags when mounting GCSFuse:

1. `--enable-async-reads`: Enables the FUSE async reads feature.
2. `--max-read-ahead-kb`: Set this to a value **greater than 1024** (e.g., `--max-read-ahead-kb=2048`). A higher read-ahead threshold prompts the kernel to issue requests in parallel.

## Configuration and Limits

### Max Blocks Per Handle (`--read-max-blocks-per-handle`)

The total memory footprint for prefetching for a single file handle is strictly limited by the `max-blocks-per-handle` limit. The reader initializes a block pool based on this limit, and all active downloads and cached blocks count against it.

### Retired Blocks Per Handle (`--read-retired-blocks-per-handle`)

To efficiently handle out-of-order reads or concurrent parallel requests without re-downloading data or falling back to another reader, consumed blocks are placed into an **LRU (Least Recently Used) cache** known as "retired blocks".

**How it works:**
* When the current read offset moves past an active block, the block is moved to the retired blocks LRU cache rather than being immediately discarded.
* If a parallel or out-of-order read requests an offset that was recently processed, it can be served directly from this cache.
* **Important Constraint:** Retired blocks share the exact same underlying memory pool as active prefetch blocks. Therefore, the actual number of retired blocks you can hold at any given time is strictly bounded by `--read-max-blocks-per-handle`. Setting `--read-retired-blocks-per-handle` higher than `--read-max-blocks-per-handle` provides no benefit, as the shared pool will exhaust its block supply before the retired cache limit is ever reached.
