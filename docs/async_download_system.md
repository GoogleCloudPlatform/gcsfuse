# Async Block Download System

This document describes the asynchronous block download functionality added to gcsfuse, which enables efficient concurrent downloading of data blocks using worker pools.

## Overview

The async download system provides non-blocking downloads for block cache operations, improving performance by allowing multiple downloads to run concurrently while maintaining priority ordering and state tracking.

## Architecture

### Key Components

1. **AsyncBlockDownloadTask** - Individual download task implementation
2. **AsyncDownloadManager** - Coordinates multiple downloads
3. **BlockDownloadRequest** - Specification for download operations
4. **DownloadState** - State tracking throughout download lifecycle
5. **Worker Pool Integration** - Uses existing worker pool infrastructure
6. **Block Cache Integration** - Seamless integration with block caching

### File Structure

```
internal/block/
├── async_download.go      # Core async download implementation
├── async_download_test.go # Comprehensive unit tests
├── block_cache.go         # Enhanced cache with async support
└── block.go              # Block definitions and types
```

## Key Features

### ✅ Non-blocking Downloads
- Downloads execute asynchronously in background worker threads
- Callers can proceed immediately or wait for completion
- Multiple downloads can run concurrently

### ✅ Priority Queue Support
- High-priority downloads use urgent worker pool scheduling
- Normal downloads use regular scheduling
- Priority downloads complete faster under load

### ✅ State Tracking
- Complete download lifecycle monitoring
- States: NotStarted → InProgress → Completed/Failed/Cancelled
- Thread-safe status access with timestamps

### ✅ Error Handling & Cancellation
- Robust error handling with detailed error reporting
- Downloads can be cancelled at any time
- Context cancellation propagates to ongoing downloads
- Automatic cleanup of completed/failed downloads

### ✅ Block Cache Integration
- `GetOrScheduleDownload()` checks cache first
- Automatic storage of downloaded blocks in cache
- Seamless fallback to async downloads when blocks missing

### ✅ Callback-based Completion
- OnComplete callbacks notify when downloads finish
- Callbacks include final state and any errors
- Enables reactive programming patterns

## Usage Examples

### Basic Download Request

```go
request := &block.BlockDownloadRequest{
    Key:         block.CacheKey("my-block"),
    ObjectName:  "path/to/object",
    Generation:  123456,
    StartOffset: 0,
    EndOffset:   1024,
    Priority:    false,
    OnComplete: func(key block.CacheKey, state block.DownloadState, err error) {
        if err != nil {
            log.Printf("Download failed: %v", err)
        } else {
            log.Printf("Download completed: %s", key)
        }
    },
}
```

### Cache Integration Pattern

```go
// Try to get from cache or schedule download
block, task, err := cache.GetOrScheduleDownload(ctx, request)
if err != nil {
    return err
}

if block != nil {
    // Block was in cache, use immediately
    defer cache.Release(block)
    // Process block data...
    return nil
}

if task != nil {
    // Block not in cache, download was scheduled
    // Option 1: Wait for download completion
    // Option 2: Return and handle via callback
    // Option 3: Monitor progress asynchronously
}
```

### Priority Downloads

```go
// High-priority download (urgent)
priorityRequest := &block.BlockDownloadRequest{
    Key:      block.CacheKey("urgent-block"),
    // ... other fields
    Priority: true,  // Uses urgent worker pool queue
}

// Normal priority download
normalRequest := &block.BlockDownloadRequest{
    Key:      block.CacheKey("normal-block"), 
    // ... other fields
    Priority: false, // Uses normal worker pool queue
}
```

### Download Monitoring

```go
func monitorDownload(task AsyncDownloadTask) {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            status := task.GetStatus()
            switch status.State {
            case block.DownloadStateCompleted:
                fmt.Println("Download completed!")
                return
            case block.DownloadStateFailed:
                fmt.Printf("Download failed: %v\n", status.Error)
                return
            case block.DownloadStateInProgress:
                fmt.Printf("In progress... (started %v ago)\n", time.Since(status.StartTime))
            }
        }
    }
}
```

## API Reference

### DownloadState

```go
type DownloadState int

const (
    DownloadStateNotStarted DownloadState = iota
    DownloadStateInProgress
    DownloadStateCompleted
    DownloadStateFailed
    DownloadStateCancelled
)
```

### BlockDownloadRequest

```go
type BlockDownloadRequest struct {
    Key         CacheKey                // Unique cache identifier
    ObjectName  string                  // GCS object name
    Generation  int64                   // Object generation
    StartOffset int64                   // Download start byte offset
    EndOffset   int64                   // Download end byte offset
    Priority    bool                    // High priority flag
    OnComplete  func(CacheKey, DownloadState, error) // Completion callback
}
```

### AsyncDownloadManager Methods

```go
// Schedule a new download
ScheduleDownload(ctx context.Context, req *BlockDownloadRequest) (AsyncDownloadTask, error)

// Cancel an ongoing download
CancelDownload(key CacheKey) error

// List all active downloads
ListActiveDownloads() []CacheKey

// Clean up completed downloads
CleanupCompletedDownloads() int
```

### Block Cache Methods

```go
// Get block from cache or schedule download
GetOrScheduleDownload(ctx context.Context, req *BlockDownloadRequest) (*Block, AsyncDownloadTask, error)

// Schedule async download (cache miss)
ScheduleAsyncDownload(ctx context.Context, req *BlockDownloadRequest) (AsyncDownloadTask, error)

// Set async download manager
SetAsyncDownloadManager(manager *AsyncDownloadManager)
```

## Implementation Details

### Worker Pool Integration

The async download system integrates with gcsfuse's existing worker pool infrastructure:

- **Priority downloads** use `WorkerPool.Schedule(task, true)` for urgent processing
- **Normal downloads** use `WorkerPool.Schedule(task, false)` for regular processing
- **Concurrent execution** allows multiple downloads simultaneously
- **Task interface** implementation ensures compatibility

### State Management

Download state is managed with thread-safe operations:

```go
type DownloadStatus struct {
    State     DownloadState
    Error     error
    StartTime time.Time
}
```

State transitions are atomic and include:
- Start time tracking for performance monitoring
- Error details for debugging and handling
- Thread-safe access via mutexes

### GCS Integration

Downloads use GCS bucket's `NewReaderWithReadHandle()` for optimal performance:

- Range-based downloads for specific byte offsets
- ReadHandle optimization for reduced latency
- Automatic retry and error handling
- Context-based cancellation support

### Memory Management

The system includes careful memory management:

- Block allocation through semaphore-controlled pools
- Automatic cleanup of completed downloads
- Resource cleanup on cancellation or failure
- Integration with existing cache eviction policies

## Performance Benefits

### Concurrency
- Multiple blocks can download simultaneously
- Non-blocking operations prevent UI/API freezing
- Worker pool manages optimal concurrency levels

### Priority Handling
- Critical downloads complete faster
- Background prefetching doesn't block urgent requests
- Intelligent queue management

### Caching Integration
- Downloaded blocks persist in cache
- Subsequent access is immediate
- Automatic LRU eviction management

### Resource Efficiency
- Shared worker pool reduces thread overhead
- Efficient memory allocation patterns
- Automatic cleanup prevents resource leaks

## Testing

The system includes comprehensive tests covering:

- ✅ State transitions and tracking
- ✅ Concurrent download operations
- ✅ Priority queue ordering
- ✅ Error handling and recovery
- ✅ Cancellation scenarios
- ✅ Block cache integration
- ✅ Memory safety and cleanup

Run tests with:
```bash
go test -v ./internal/block -run Test.*Download
```

## Migration Guide

### From Synchronous Downloads

**Before:**
```go
// Synchronous download - blocks until complete
block, err := downloadBlockSync(ctx, request)
if err != nil {
    return err
}
defer cache.Release(block)
// Process block...
```

**After:**
```go
// Asynchronous download - non-blocking
block, task, err := cache.GetOrScheduleDownload(ctx, request)
if err != nil {
    return err
}

if block != nil {
    // Already cached
    defer cache.Release(block)
    // Process block...
} else if task != nil {
    // Download in progress - handle via callback or monitoring
    // Process continues without blocking
}
```

### Adding Async Support to Existing Code

1. **Replace direct downloads** with `GetOrScheduleDownload()`
2. **Add completion callbacks** for async notification
3. **Handle cache hits** and misses appropriately
4. **Consider priority levels** for different use cases
5. **Add progress monitoring** where needed

## Best Practices

### Request Design
- Use meaningful cache keys for debugging
- Set appropriate priority levels
- Include completion callbacks for error handling
- Validate request parameters before scheduling

### Error Handling
- Always check for errors from scheduling operations
- Implement robust completion callbacks
- Handle cancellation scenarios gracefully
- Log errors with sufficient context for debugging

### Performance Optimization
- Group related downloads when possible
- Use priority sparingly to avoid queue starvation
- Monitor active download counts to prevent overload
- Clean up completed downloads periodically

### Resource Management
- Release blocks promptly after use
- Cancel unnecessary downloads to free resources
- Monitor memory usage with high download volumes
- Use context cancellation for request timeouts

## Troubleshooting

### Common Issues

**Downloads not starting:**
- Verify worker pool is running
- Check request validation
- Ensure sufficient semaphore capacity

**High memory usage:**
- Check for unreleased blocks
- Monitor active download counts
- Verify cleanup is running

**Performance issues:**
- Balance priority vs normal downloads
- Monitor worker pool utilization
- Check for excessive concurrent downloads

### Debugging

Enable debug logging and monitor:
- Download state transitions
- Worker pool queue depths
- Cache hit/miss ratios
- Error rates and patterns

### Monitoring

Key metrics to track:
- Active download count
- Average download duration
- Error rates by type
- Cache efficiency ratios
- Worker pool utilization

## Future Enhancements

Potential improvements for future versions:

- **Download progress tracking** with byte counts
- **Bandwidth throttling** for rate limiting
- **Download grouping** for batch operations
- **Predictive prefetching** based on access patterns
- **Adaptive priority adjustment** based on system load
- **Enhanced metrics** and monitoring
- **Download deduplication** for identical requests

## Contributing

When contributing to the async download system:

1. **Maintain thread safety** in all operations
2. **Add comprehensive tests** for new functionality
3. **Update documentation** for API changes
4. **Consider backward compatibility** for existing code
5. **Follow existing patterns** for consistency
6. **Test edge cases** thoroughly
7. **Verify performance impact** under load

## License

This code is part of the gcsfuse project and is licensed under the Apache License 2.0.
