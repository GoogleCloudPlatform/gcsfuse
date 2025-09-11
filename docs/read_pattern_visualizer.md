# Read Pattern Visualizer Utility

The Read Pattern Visualizer is a utility class in the `common` package that provides functionality for tracking and visualizing file read patterns. It's particularly useful for analyzing I/O performance, debugging read access patterns, and understanding how applications access files.

## Features

- **Range Tracking**: Record read operations with start and end offsets
- **Visual Graphs**: Generate ASCII-based visual representations of read patterns
- **Pattern Analysis**: Automatic detection of sequential vs random read patterns
- **Flexible Scaling**: Support for different scale units (bytes, KB, MB, GB)
- **Summary Statistics**: Detailed statistics about read patterns

## Basic Usage

```go
package main

import (
    "fmt"
    "github.com/googlecloudplatform/gcsfuse/v3/common"
)

func main() {
    // Create a new visualizer
    visualizer := common.NewReadPatternVisualizer()
    
    // Set description and customize appearance
    visualizer.SetDescription("My File Read Pattern")
    visualizer.SetGraphWidth(60)
    visualizer.SetScaleUnit(1024) // 1KB units
    
    // Record some read operations
    visualizer.AcceptRange(0, 4096)      // Read 0-4KB
    visualizer.AcceptRange(4096, 8192)   // Read 4-8KB (sequential)
    visualizer.AcceptRange(12288, 16384) // Read 12-16KB (gap)
    visualizer.AcceptRange(8192, 12288)  // Read 8-12KB (fill gap)
    
    // Generate and display the visual graph
    fmt.Println(visualizer.DumpGraph())
}
```

## API Reference

### Types

#### `ReadRange`
Represents a single read operation range.

```go
type ReadRange struct {
    Start int64  // Starting offset in bytes
    End   int64  // Ending offset in bytes (exclusive)
}

// Methods
func (rr ReadRange) String() string  // Returns "[start, end)" format
func (rr ReadRange) Length() int64   // Returns end - start
```

#### `ReadPatternVisualizer`
Main utility class for tracking and visualizing read patterns.

### Constructor Functions

```go
// Create with default settings (1KB scale, 100 char width)
func NewReadPatternVisualizer() *ReadPatternVisualizer

// Create with custom configuration
func NewReadPatternVisualizerWithConfig(
    scaleUnit int64,    // Scale unit for display (1, 1024, 1024*1024, etc.)
    graphWidth int,     // Width of the graph in characters
    description string  // Description for the pattern
) *ReadPatternVisualizer
```

### Core Methods

#### Recording Ranges
```go
// Add a new read range (invalid ranges are ignored)
func (rpv *ReadPatternVisualizer) AcceptRange(start, end int64)

// Get all recorded ranges
func (rpv *ReadPatternVisualizer) GetRanges() []ReadRange

// Get the maximum offset seen
func (rpv *ReadPatternVisualizer) GetMaxOffset() int64

// Clear all ranges and reset state
func (rpv *ReadPatternVisualizer) Reset()
```

#### Configuration
```go
// Set the scale unit for display (1=bytes, 1024=KB, 1024*1024=MB, etc.)
func (rpv *ReadPatternVisualizer) SetScaleUnit(unit int64)

// Set the width of the graph display
func (rpv *ReadPatternVisualizer) SetGraphWidth(width int)

// Set a description for this read pattern
func (rpv *ReadPatternVisualizer) SetDescription(desc string)
```

#### Visualization
```go
// Generate a visual graph of the read pattern
func (rpv *ReadPatternVisualizer) DumpGraph() string
```

## Graph Format

The generated graph has the following structure:

```
Read Pattern: [Description]
Total ranges: [N]
Max offset: [X] bytes ([Y] [units])

Range# | [X-axis labels showing offsets]
-------|-----------------------------------
     0 | ████████                          | [start, end) (len: size)
     1 |         ███████                   | [start, end) (len: size)
     2 |                   ████            | [start, end) (len: size)
     ...

Summary Statistics:
  Total bytes read: [total] ([scaled units])
  Average range size: [avg] bytes
  Min range size: [min] bytes
  Max range size: [max] bytes
  Read pattern analysis: [Sequential|Random|Random with overlaps]
```

### Graph Components

- **X-axis**: Shows file offsets from 0 to maximum offset
- **Y-axis**: Shows ranges in temporal order (top to bottom)
- **Bars**: `█` characters represent the span of each read operation
- **Range Info**: Each line shows the exact range and length
- **Summary**: Statistical analysis of the read pattern

## Examples

### Sequential Reading
```go
visualizer := common.NewReadPatternVisualizer()
visualizer.SetDescription("Sequential File Reading")

// Read file in 4KB chunks sequentially
for i := 0; i < 8; i++ {
    start := int64(i * 4096)
    end := int64((i + 1) * 4096)
    visualizer.AcceptRange(start, end)
}

fmt.Println(visualizer.DumpGraph())
// Output shows continuous bars with "Sequential" pattern analysis
```

### Random Access
```go
visualizer := common.NewReadPatternVisualizer()
visualizer.SetDescription("Random File Access")

// Random read operations
visualizer.AcceptRange(0, 2048)
visualizer.AcceptRange(8192, 12288)
visualizer.AcceptRange(4096, 6144)
visualizer.AcceptRange(16384, 20480)

fmt.Println(visualizer.DumpGraph())
// Output shows scattered bars with "Random" pattern analysis
```

### Large Files with Custom Scale
```go
visualizer := common.NewReadPatternVisualizerWithConfig(
    1024*1024*1024, // 1GB scale unit
    50,             // 50 character width
    "Large File Processing")

// Read chunks of a multi-GB file
gbSize := int64(1024 * 1024 * 1024)
for i := int64(0); i < 5; i++ {
    start := i * gbSize
    end := (i + 1) * gbSize
    visualizer.AcceptRange(start, end)
}

fmt.Println(visualizer.DumpGraph())
// Output shows pattern scaled in GB units
```

## Pattern Analysis

The utility automatically analyzes read patterns and categorizes them as:

- **Sequential**: Reads occur in order without gaps
- **Random**: Reads occur with gaps or out of temporal order
- **Random with overlaps**: Reads have overlapping ranges

This analysis helps identify:
- Efficient sequential access patterns
- Random access that might benefit from buffering
- Overlapping reads that might indicate prefetching

## Performance

The utility is designed to be lightweight:
- `AcceptRange`: ~47 ns/op
- `DumpGraph`: ~1.2 ms/op for 1000 ranges

It's suitable for production use with reasonable numbers of read operations.

## Use Cases

1. **Performance Analysis**: Understanding how applications read files
2. **Debugging**: Visualizing unexpected read patterns
3. **Cache Optimization**: Identifying patterns that would benefit from caching
4. **Prefetch Tuning**: Understanding read-ahead requirements
5. **I/O Pattern Documentation**: Creating visual documentation of access patterns

## Demo

Run the included demo to see various patterns:

```bash
go run examples/reader_pattern_demo.go
```

For an interactive demo where you can input your own ranges:

```bash
go run examples/reader_pattern_demo.go --interactive
```
