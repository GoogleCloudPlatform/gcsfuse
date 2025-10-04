// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

// Global mutex to protect while dumping read pattern visual to file.
var gMutex sync.Mutex
func init() {
	gMutex = sync.Mutex{}
}

// ReadRange represents a read operation range with start and end offsets.
type ReadRange struct {
	Start int64
	End   int64
}

// String returns a string representation of the range.
func (rr ReadRange) String() string {
	return fmt.Sprintf("[%d, %d)", rr.Start, rr.End)
}

// Length returns the length of the range.
func (rr ReadRange) Length() int64 {
	return rr.End - rr.Start
}

// ReadPatternVisualizer provides utilities for tracking and visualizing read patterns.
type ReadPatternVisualizer struct {
	ranges           []ReadRange
	maxOffset        int64
	scaleUnit        int64 // Unit for scaling the graph (default: 1KB)
	graphWidth       int   // Maximum width for the graph display
	description      string
	readerName       string // Name of the reader
	totalRangesAdded int    // Total number of ranges added (before merging)
}

// NewReadPatternVisualizer creates a new read pattern visualizer.
func NewReadPatternVisualizer() *ReadPatternVisualizer {
	return &ReadPatternVisualizer{
		ranges:           make([]ReadRange, 0),
		scaleUnit:        4 * 1024, // 4KB default scale
		graphWidth:       100,      // 100 characters width
		readerName:       "",
		totalRangesAdded: 0,
	}
}

// NewReadPatternVisualizerWithReader creates a new read pattern visualizer with reader name.
func NewReadPatternVisualizerWithReader(readerName string) *ReadPatternVisualizer {
	return &ReadPatternVisualizer{
		ranges:           make([]ReadRange, 0),
		scaleUnit:        4 * 1024, // 4KB default scale
		graphWidth:       100,      // 100 characters width
		readerName:       readerName,
		totalRangesAdded: 0,
	}
}

// NewReadPatternVisualizerWithConfig creates a new read pattern visualizer with custom configuration.
func NewReadPatternVisualizerWithConfig(scaleUnit int64, graphWidth int, description string) *ReadPatternVisualizer {
	return &ReadPatternVisualizer{
		ranges:           make([]ReadRange, 0),
		scaleUnit:        scaleUnit,
		graphWidth:       graphWidth,
		description:      description,
		readerName:       "",
		totalRangesAdded: 0,
	}
}

// NewReadPatternVisualizerWithFullConfig creates a new read pattern visualizer with full configuration.
func NewReadPatternVisualizerWithFullConfig(scaleUnit int64, graphWidth int, description string, readerName string) *ReadPatternVisualizer {
	return &ReadPatternVisualizer{
		ranges:           make([]ReadRange, 0),
		scaleUnit:        scaleUnit,
		graphWidth:       graphWidth,
		description:      description,
		readerName:       readerName,
		totalRangesAdded: 0,
	}
}

// AcceptRange adds a new read range to the pattern tracker.
// The ranges are stored in the order they are added to maintain temporal sequence.
// If the new range starts exactly where the last range ends, they will be merged.
// This helps create cleaner visualizations and more accurate sequential read detection.
func (rpv *ReadPatternVisualizer) AcceptRange(start, end int64) {
	if start < 0 || end <= start {
		return // Invalid range, ignore
	}

	// Increment total ranges added counter
	rpv.totalRangesAdded++

	newRange := ReadRange{
		Start: start,
		End:   end,
	}

	// Try to merge with the last range if possible
	if len(rpv.ranges) > 0 {
		lastRange := &rpv.ranges[len(rpv.ranges)-1]

		// Check if ranges are adjacent or overlapping
		if rpv.canMergeRanges(*lastRange, newRange) {
			// Merge the ranges by extending the last range
			mergedRange := rpv.mergeRanges(*lastRange, newRange)
			rpv.ranges[len(rpv.ranges)-1] = mergedRange

			// Update max offset if necessary
			if mergedRange.End > rpv.maxOffset {
				rpv.maxOffset = mergedRange.End
			}
			return
		}
	}

	// No merge possible, add as new range
	rpv.ranges = append(rpv.ranges, newRange)

	// Update max offset
	if end > rpv.maxOffset {
		rpv.maxOffset = end
	}
}

// canMergeRanges checks if the new range can be merged with the last range.
// Merge only happens when the new range starts exactly where the last range ends.
func (rpv *ReadPatternVisualizer) canMergeRanges(lastRange, newRange ReadRange) bool {
	// Only merge if the new range starts exactly where the last range ends
	return lastRange.End == newRange.Start
}

// mergeRanges combines two ranges into a single range.
func (rpv *ReadPatternVisualizer) mergeRanges(range1, range2 ReadRange) ReadRange {
	start := range1.Start
	if range2.Start < start {
		start = range2.Start
	}

	end := range1.End
	if range2.End > end {
		end = range2.End
	}

	return ReadRange{Start: start, End: end}
}

// GetRanges returns a copy of all stored ranges.
func (rpv *ReadPatternVisualizer) GetRanges() []ReadRange {
	result := make([]ReadRange, len(rpv.ranges))
	copy(result, rpv.ranges)
	return result
}

// GetMaxOffset returns the maximum offset encountered across all ranges.
func (rpv *ReadPatternVisualizer) GetMaxOffset() int64 {
	return rpv.maxOffset
}

// Reset clears all stored ranges and resets the visualizer state.
func (rpv *ReadPatternVisualizer) Reset() {
	rpv.ranges = rpv.ranges[:0]
	rpv.maxOffset = 0
	rpv.totalRangesAdded = 0
}

// SetScaleUnit sets the unit for scaling the graph display.
func (rpv *ReadPatternVisualizer) SetScaleUnit(unit int64) {
	if unit > 0 {
		rpv.scaleUnit = unit
	}
}

// SetGraphWidth sets the maximum width for the graph display.
func (rpv *ReadPatternVisualizer) SetGraphWidth(width int) {
	if width > 0 {
		rpv.graphWidth = width
	}
}

// SetDescription sets a description for the read pattern.
func (rpv *ReadPatternVisualizer) SetDescription(desc string) {
	rpv.description = desc
}

// SetReaderName sets the name of the reader for the read pattern.
func (rpv *ReadPatternVisualizer) SetReaderName(name string) {
	rpv.readerName = name
}

// DumpGraph generates and returns a visual representation of the read pattern.
// The graph shows:
// - X axis: file offsets (0 to max offset)
// - Y axis: downward progression of read operations (0th range at top)
//
// Each range is represented as a horizontal bar showing the span of bytes read.
// The graph is scaled based on the scaleUnit and graphWidth settings.
func (rpv *ReadPatternVisualizer) DumpGraph() string {
	if len(rpv.ranges) == 0 {
		return "No read ranges recorded."
	}

	var result strings.Builder

	// Add header information
	if rpv.readerName != "" {
		result.WriteString(fmt.Sprintf("Reader: %s\n", rpv.readerName))
	}
	if rpv.description != "" {
		result.WriteString(fmt.Sprintf("Read Pattern: %s\n", rpv.description))
	}
	result.WriteString(fmt.Sprintf("Total ranges added: %d\n", rpv.totalRangesAdded))
	result.WriteString(fmt.Sprintf("Final ranges (after merge): %d\n", len(rpv.ranges)))
	result.WriteString(fmt.Sprintf("Max offset: %d bytes (%.2f %s)\n",
		rpv.maxOffset,
		float64(rpv.maxOffset)/float64(rpv.scaleUnit),
		rpv.getScaleUnitName()))
	result.WriteString("\n")

	// Calculate scaling factor
	scaleFactor := float64(rpv.maxOffset) / float64(rpv.graphWidth)
	if scaleFactor < 1 {
		scaleFactor = 1
	}

	// Create the graph header (X-axis labels)
	result.WriteString(rpv.createXAxisLabels(scaleFactor))
	result.WriteString("\n")

	// Create separator line
	result.WriteString(strings.Repeat("-", rpv.graphWidth+10))
	result.WriteString("\n")

	// Generate each range row
	for i, range_ := range rpv.ranges {
		result.WriteString(rpv.createRangeRow(i, range_, scaleFactor))
		result.WriteString("\n")
	}

	// Add summary statistics
	result.WriteString("\n")
	result.WriteString(rpv.generateSummary())

	return result.String()
}

// createXAxisLabels creates the X-axis labels showing offset positions.
func (rpv *ReadPatternVisualizer) createXAxisLabels(scaleFactor float64) string {
	var labels strings.Builder
	labels.WriteString("Range# | ")

	// Create position markers every 10 characters
	for i := 0; i < rpv.graphWidth; i += 10 {
		offset := int64(float64(i) * scaleFactor)
		if i == 0 {
			labels.WriteString("0")
		} else {
			unitValue := float64(offset) / float64(rpv.scaleUnit)
			if unitValue < 1 {
				labels.WriteString(fmt.Sprintf("%d", offset))
			} else {
				labels.WriteString(fmt.Sprintf("%.1f%s", unitValue, rpv.getScaleUnitAbbrev()))
			}
		}
		// Add spacing
		remaining := 10 - len(fmt.Sprintf("%.1f%s", float64(offset)/float64(rpv.scaleUnit), rpv.getScaleUnitAbbrev()))
		if i == 0 {
			remaining = 9
		}
		if remaining > 0 {
			labels.WriteString(strings.Repeat(" ", remaining))
		}
	}

	return labels.String()
}

// createRangeRow creates a visual representation of a single range.
func (rpv *ReadPatternVisualizer) createRangeRow(index int, range_ ReadRange, scaleFactor float64) string {
	var row strings.Builder

	// Range number (left margin)
	row.WriteString(fmt.Sprintf("%6d | ", index))

	// Calculate start and end positions in graph coordinates
	startPos := int(float64(range_.Start) / scaleFactor)
	endPos := int(float64(range_.End) / scaleFactor)

	// Ensure positions are within bounds
	if startPos >= rpv.graphWidth {
		startPos = rpv.graphWidth - 1
	}
	if endPos >= rpv.graphWidth {
		endPos = rpv.graphWidth - 1
	}
	if endPos <= startPos {
		endPos = startPos + 1
	}

	// Build the visual bar
	for i := 0; i < rpv.graphWidth; i++ {
		if i >= startPos && i < endPos {
			row.WriteString("█")
		} else {
			row.WriteString(" ")
		}
	}

	// Add range information
	row.WriteString(fmt.Sprintf(" | %s (len: %d)", range_.String(), range_.Length()))

	return row.String()
}

// generateSummary creates summary statistics about the read pattern.
func (rpv *ReadPatternVisualizer) generateSummary() string {
	if len(rpv.ranges) == 0 {
		return ""
	}

	var summary strings.Builder
	summary.WriteString("Summary Statistics:\n")

	// Calculate total bytes read
	totalBytes := int64(0)
	minRangeSize := rpv.ranges[0].Length()
	maxRangeSize := rpv.ranges[0].Length()

	for _, range_ := range rpv.ranges {
		length := range_.Length()
		totalBytes += length
		if length < minRangeSize {
			minRangeSize = length
		}
		if length > maxRangeSize {
			maxRangeSize = length
		}
	}

	avgRangeSize := totalBytes / int64(len(rpv.ranges))

	summary.WriteString(fmt.Sprintf("  Total bytes read: %d (%.2f %s)\n",
		totalBytes,
		float64(totalBytes)/float64(rpv.scaleUnit),
		rpv.getScaleUnitName()))
	summary.WriteString(fmt.Sprintf("  Average range size: %d bytes\n", avgRangeSize))
	summary.WriteString(fmt.Sprintf("  Min range size: %d bytes\n", minRangeSize))
	summary.WriteString(fmt.Sprintf("  Max range size: %d bytes\n", maxRangeSize))

	// Analyze read pattern
	summary.WriteString(fmt.Sprintf("  Read pattern analysis: %s\n", rpv.analyzePattern()))

	return summary.String()
}

// analyzePattern provides basic analysis of the read pattern.
func (rpv *ReadPatternVisualizer) analyzePattern() string {
	// Special case: if we have only one range but multiple ranges were added,
	// it likely means sequential ranges were merged
	if len(rpv.ranges) == 1 && rpv.totalRangesAdded > 1 {
		return "Sequential (merged)"
	}

	if len(rpv.ranges) <= 1 {
		return "Insufficient data"
	}

	// Check if reads are sequential
	sequential := true
	overlapping := false
	gaps := 0

	sortedRanges := make([]ReadRange, len(rpv.ranges))
	copy(sortedRanges, rpv.ranges)
	sort.Slice(sortedRanges, func(i, j int) bool {
		return sortedRanges[i].Start < sortedRanges[j].Start
	})

	for i := 1; i < len(sortedRanges); i++ {
		prev := sortedRanges[i-1]
		curr := sortedRanges[i]

		if curr.Start < prev.End {
			overlapping = true
		} else if curr.Start > prev.End {
			gaps++
			sequential = false
		}

		// Check temporal order vs spatial order
		if rpv.ranges[i].Start < rpv.ranges[i-1].End && rpv.ranges[i].Start >= rpv.ranges[i-1].Start {
			continue
		} else if rpv.ranges[i].Start != rpv.ranges[i-1].End {
			sequential = false
		}
	}

	if sequential && !overlapping {
		return "Sequential"
	} else if overlapping {
		return fmt.Sprintf("Random with overlaps (gaps: %d)", gaps)
	} else {
		return fmt.Sprintf("Random (gaps: %d)", gaps)
	}
}

// getScaleUnitName returns the human-readable name for the current scale unit.
func (rpv *ReadPatternVisualizer) getScaleUnitName() string {
	switch rpv.scaleUnit {
	case 1:
		return "bytes"
	case 1024:
		return "KB"
	case 1024 * 1024:
		return "MB"
	case 1024 * 1024 * 1024:
		return "GB"
	default:
		return fmt.Sprintf("%d-byte units", rpv.scaleUnit)
	}
}

// getScaleUnitAbbrev returns the abbreviated name for the current scale unit.
func (rpv *ReadPatternVisualizer) getScaleUnitAbbrev() string {
	switch rpv.scaleUnit {
	case 1:
		return "B"
	case 1024:
		return "K"
	case 1024 * 1024:
		return "M"
	case 1024 * 1024 * 1024:
		return "G"
	default:
		return "U"
	}
}

// DumpGraphToFile writes the visual representation of the read pattern to a file.
// It uses the same format as DumpGraph() but writes directly to the specified file path.
// The file will be created if it doesn't exist, or overwritten if it does exist.
func (rpv *ReadPatternVisualizer) DumpGraphToFile(filePath string) error {
	graphOutput := rpv.DumpGraph()

	// Lock to ensure only one goroutine writes to the read-pattern file at a time,
	// to avoid interleaved writes.
	gMutex.Lock()
	defer gMutex.Unlock()

	// Create or overwrite the file
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Write the graph output to the file
	_, err = file.WriteString(graphOutput)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filePath, err)
	}

	return nil
}
