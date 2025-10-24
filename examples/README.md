# Read Pattern Example

This example demonstrates how GCSFuse readers adapt to different file access patterns by showing:

1. **Random reads** (7-8 times) at various offsets in the file
2. **Sequential reads** (multiple) starting from the beginning of the file  
3. **Random reads again** to show the pattern switching back

## What it demonstrates

- **BufferedReader behavior**: The BufferedReader initially tries to prefetch blocks for sequential access
- **Reader switching**: When random access is detected (exceeding the RandomSeekThreshold), it falls back to a GCSReader
- **Access pattern adaptation**: Different readers are optimized for different access patterns

## Key Configuration

The example uses these important settings:
- `RandomSeekThreshold: 3` - Low threshold to quickly demonstrate reader switching
- `EnableBufferedRead: true` - Enables the BufferedReader for prefetching
- File size: 10MB with 4KB read buffer size

## Building and Running

```bash
# Build the example
cd /path/to/gcsfuse
go build -o read_pattern_example ./examples/read_pattern_example.go

# Run the example
./read_pattern_example
```

## Expected Output

You should see output like:

```
Phase 1: Performing random reads...
Random Read 1: offset=8541426, bytes_read=4096
Random Read 2: offset=5378803, bytes_read=4096
...
WARNING: Fallback to another reader for object "test-read-pattern-object"...

Phase 2: Switching to sequential reads...
Sequential Read 1: offset=0, bytes_read=4096
Sequential Read 2: offset=4096, bytes_read=4096
...

Phase 3: Returning to random reads...
Random Read 29: offset=6519724, bytes_read=4096
...
```

## Key Observations

- **Random access detection**: After a few random reads, you'll see a warning about fallback to another reader
- **Sequential access optimization**: Sequential reads show consecutive offsets (0, 4096, 8192, etc.)
- **Reader adaptation**: The system automatically chooses the best reader for the current access pattern

This example helps understand how GCSFuse optimizes performance for different file access patterns in real-world scenarios.
