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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadRange(t *testing.T) {
	tests := []struct {
		name    string
		start   int64
		end     int64
		wantStr string
		wantLen int64
	}{
		{"Basic range", 0, 100, "[0, 100)", 100},
		{"Mid-file range", 500, 1024, "[500, 1024)", 524},
		{"Single byte", 42, 43, "[42, 43)", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := ReadRange{Start: tt.start, End: tt.end}

			if got := rr.String(); got != tt.wantStr {
				t.Errorf("ReadRange.String() = %v, want %v", got, tt.wantStr)
			}

			if got := rr.Length(); got != tt.wantLen {
				t.Errorf("ReadRange.Length() = %v, want %v", got, tt.wantLen)
			}
		})
	}
}

func TestNewReadPatternVisualizer(t *testing.T) {
	rpv := NewReadPatternVisualizer()

	if rpv == nil {
		t.Fatal("NewReadPatternVisualizer() returned nil")
	}

	if len(rpv.ranges) != 0 {
		t.Errorf("New visualizer should have empty ranges, got %d ranges", len(rpv.ranges))
	}

	if rpv.maxOffset != 0 {
		t.Errorf("New visualizer should have maxOffset = 0, got %d", rpv.maxOffset)
	}

	if rpv.scaleUnit != 1024 {
		t.Errorf("Default scaleUnit should be 1024, got %d", rpv.scaleUnit)
	}

	if rpv.graphWidth != 100 {
		t.Errorf("Default graphWidth should be 100, got %d", rpv.graphWidth)
	}
}

func TestNewReadPatternVisualizerWithConfig(t *testing.T) {
	scaleUnit := int64(1024 * 1024) // 1MB
	graphWidth := 80
	description := "Test pattern"

	rpv := NewReadPatternVisualizerWithConfig(scaleUnit, graphWidth, description)

	if rpv.scaleUnit != scaleUnit {
		t.Errorf("Expected scaleUnit %d, got %d", scaleUnit, rpv.scaleUnit)
	}

	if rpv.graphWidth != graphWidth {
		t.Errorf("Expected graphWidth %d, got %d", graphWidth, rpv.graphWidth)
	}

	if rpv.description != description {
		t.Errorf("Expected description %q, got %q", description, rpv.description)
	}
}

func TestAcceptRange(t *testing.T) {
	rpv := NewReadPatternVisualizer()

	// Test valid ranges
	rpv.AcceptRange(0, 100)
	rpv.AcceptRange(200, 300)
	rpv.AcceptRange(150, 250)

	ranges := rpv.GetRanges()
	if len(ranges) != 3 {
		t.Errorf("Expected 3 ranges, got %d", len(ranges))
	}

	if rpv.GetMaxOffset() != 300 {
		t.Errorf("Expected maxOffset 300, got %d", rpv.GetMaxOffset())
	}

	// Test invalid ranges (should be ignored)
	initialCount := len(rpv.ranges)
	rpv.AcceptRange(-10, 50)  // negative start
	rpv.AcceptRange(100, 50)  // end <= start
	rpv.AcceptRange(100, 100) // end == start

	if len(rpv.ranges) != initialCount {
		t.Errorf("Invalid ranges should be ignored, but range count changed from %d to %d",
			initialCount, len(rpv.ranges))
	}
}

func TestAcceptRange_Merging(t *testing.T) {
	rpv := NewReadPatternVisualizer()

	// Test consecutive ranges that should be merged
	rpv.AcceptRange(0, 100)   // First range
	rpv.AcceptRange(100, 200) // Adjacent range - should merge
	rpv.AcceptRange(200, 300) // Another adjacent range - should merge

	ranges := rpv.GetRanges()
	if len(ranges) != 1 {
		t.Errorf("Expected 1 merged range, got %d ranges", len(ranges))
		for i, r := range ranges {
			t.Logf("Range %d: %s", i, r.String())
		}
	}

	expectedRange := ReadRange{Start: 0, End: 300}
	if len(ranges) > 0 && ranges[0] != expectedRange {
		t.Errorf("Expected merged range %s, got %s", expectedRange.String(), ranges[0].String())
	}

	// Test non-consecutive ranges that should NOT be merged
	rpv.Reset()
	rpv.AcceptRange(0, 100)
	rpv.AcceptRange(150, 250) // Gap between ranges - should not merge

	ranges = rpv.GetRanges()
	if len(ranges) != 2 {
		t.Errorf("Expected 2 separate ranges, got %d ranges", len(ranges))
	}

	// Test overlapping ranges that should NOT be merged (only consecutive merging)
	rpv.Reset()
	rpv.AcceptRange(0, 150)
	rpv.AcceptRange(100, 200) // Overlaps but doesn't start at end - should not merge

	ranges = rpv.GetRanges()
	if len(ranges) != 2 {
		t.Errorf("Expected 2 separate ranges for overlapping case, got %d ranges", len(ranges))
	}

	// Test complex scenario with some merging and some not
	rpv.Reset()
	rpv.AcceptRange(0, 50)    // Range 1
	rpv.AcceptRange(50, 100)  // Adjacent to range 1 - should merge
	rpv.AcceptRange(200, 250) // Gap - separate range
	rpv.AcceptRange(250, 300) // Adjacent to range 3 - should merge

	ranges = rpv.GetRanges()
	if len(ranges) != 2 {
		t.Errorf("Expected 2 ranges after complex merging, got %d ranges", len(ranges))
		for i, r := range ranges {
			t.Logf("Range %d: %s", i, r.String())
		}
	}

	// Verify the merged ranges are correct
	if len(ranges) >= 1 {
		expected1 := ReadRange{Start: 0, End: 100}
		if ranges[0] != expected1 {
			t.Errorf("Expected first merged range %s, got %s", expected1.String(), ranges[0].String())
		}
	}

	if len(ranges) >= 2 {
		expected2 := ReadRange{Start: 200, End: 300}
		if ranges[1] != expected2 {
			t.Errorf("Expected second merged range %s, got %s", expected2.String(), ranges[1].String())
		}
	}
}

func TestReset(t *testing.T) {
	rpv := NewReadPatternVisualizer()

	// Add some ranges
	rpv.AcceptRange(0, 100)
	rpv.AcceptRange(200, 300)

	if len(rpv.ranges) == 0 || rpv.maxOffset == 0 {
		t.Fatal("Setup failed - no ranges added")
	}

	// Reset
	rpv.Reset()

	if len(rpv.ranges) != 0 {
		t.Errorf("After reset, expected 0 ranges, got %d", len(rpv.ranges))
	}

	if rpv.maxOffset != 0 {
		t.Errorf("After reset, expected maxOffset 0, got %d", rpv.maxOffset)
	}
}

func TestSetters(t *testing.T) {
	rpv := NewReadPatternVisualizer()

	// Test SetScaleUnit
	rpv.SetScaleUnit(2048)
	if rpv.scaleUnit != 2048 {
		t.Errorf("SetScaleUnit failed, expected 2048, got %d", rpv.scaleUnit)
	}

	// Test invalid scale unit
	rpv.SetScaleUnit(-100)
	if rpv.scaleUnit != 2048 {
		t.Errorf("SetScaleUnit should ignore negative values, scaleUnit changed to %d", rpv.scaleUnit)
	}

	// Test SetGraphWidth
	rpv.SetGraphWidth(120)
	if rpv.graphWidth != 120 {
		t.Errorf("SetGraphWidth failed, expected 120, got %d", rpv.graphWidth)
	}

	// Test invalid graph width
	rpv.SetGraphWidth(-50)
	if rpv.graphWidth != 120 {
		t.Errorf("SetGraphWidth should ignore negative values, graphWidth changed to %d", rpv.graphWidth)
	}

	// Test SetDescription
	desc := "Custom description"
	rpv.SetDescription(desc)
	if rpv.description != desc {
		t.Errorf("SetDescription failed, expected %q, got %q", desc, rpv.description)
	}
}

func TestDumpGraph_EmptyRanges(t *testing.T) {
	rpv := NewReadPatternVisualizer()

	output := rpv.DumpGraph()
	expected := "No read ranges recorded."

	if output != expected {
		t.Errorf("DumpGraph() for empty ranges = %q, want %q", output, expected)
	}
}

func TestDumpGraph_WithRanges(t *testing.T) {
	rpv := NewReadPatternVisualizer()
	rpv.SetDescription("Test Sequential Read")
	rpv.SetGraphWidth(50) // Smaller width for testing

	// Add sequential ranges
	rpv.AcceptRange(0, 1024)
	rpv.AcceptRange(1024, 2048)
	rpv.AcceptRange(2048, 3072)

	output := rpv.DumpGraph()

	// Verify the output contains expected elements
	expectedElements := []string{
		"Read Pattern: Test Sequential Read",
		"Total ranges added: 3",         // 3 ranges were added
		"Final ranges (after merge): 1", // Consecutive ranges get merged into 1
		"Max offset: 3072",
		"Summary Statistics:",
		"Sequential (merged)", // Sequential ranges were merged
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("DumpGraph() output missing expected element: %q", element)
		}
	}

	// Verify the merged range appears in the output
	rangeMarker := strings.TrimSpace(strings.Split(output, "\n")[7]) // Adjust for header lines (now has one more line)
	if !strings.Contains(rangeMarker, "█") {
		t.Errorf("Merged range should be visualized with █ characters")
	}
}

func TestDumpGraph_RandomPattern(t *testing.T) {
	rpv := NewReadPatternVisualizer()
	rpv.SetDescription("Random Read Pattern")
	rpv.SetGraphWidth(40)

	// Add random access pattern
	rpv.AcceptRange(0, 512)
	rpv.AcceptRange(2048, 2560)
	rpv.AcceptRange(1024, 1536)
	rpv.AcceptRange(512, 1024)

	output := rpv.DumpGraph()

	// Should detect random pattern due to non-sequential temporal order
	if !strings.Contains(output, "Random") {
		t.Errorf("DumpGraph() should detect Random pattern, got: %s", output)
	}
}

func TestAnalyzePattern(t *testing.T) {
	tests := []struct {
		name     string
		ranges   []ReadRange
		expected string
	}{
		{
			name:     "Empty ranges",
			ranges:   []ReadRange{},
			expected: "Insufficient data",
		},
		{
			name:     "Single range",
			ranges:   []ReadRange{{0, 100}},
			expected: "Insufficient data",
		},
		{
			name: "Sequential pattern",
			ranges: []ReadRange{
				{0, 100},
				{100, 200},
				{200, 300},
			},
			expected: "Sequential (merged)",
		},
		{
			name: "Random with gaps",
			ranges: []ReadRange{
				{0, 100},
				{200, 300},
				{400, 500},
			},
			expected: "Random",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpv := NewReadPatternVisualizer()
			for _, r := range tt.ranges {
				rpv.AcceptRange(r.Start, r.End)
			}

			result := rpv.analyzePattern()
			if !strings.Contains(result, tt.expected) {
				t.Errorf("analyzePattern() = %q, want to contain %q", result, tt.expected)
			}
		})
	}
}

func TestScaleUnitNames(t *testing.T) {
	tests := []struct {
		scaleUnit  int64
		wantName   string
		wantAbbrev string
	}{
		{1, "bytes", "B"},
		{1024, "KB", "K"},
		{1024 * 1024, "MB", "M"},
		{1024 * 1024 * 1024, "GB", "G"},
		{2048, "2048-byte units", "U"},
	}

	for _, tt := range tests {
		t.Run(tt.wantName, func(t *testing.T) {
			rpv := NewReadPatternVisualizer()
			rpv.SetScaleUnit(tt.scaleUnit)

			if got := rpv.getScaleUnitName(); got != tt.wantName {
				t.Errorf("getScaleUnitName() = %q, want %q", got, tt.wantName)
			}

			if got := rpv.getScaleUnitAbbrev(); got != tt.wantAbbrev {
				t.Errorf("getScaleUnitAbbrev() = %q, want %q", got, tt.wantAbbrev)
			}
		})
	}
}

// Example usage demonstrating the visualizer
func ExampleReadPatternVisualizer() {
	// Create a new visualizer
	visualizer := NewReadPatternVisualizer()
	visualizer.SetDescription("File Reading Pattern")
	visualizer.SetGraphWidth(60)

	// Simulate some read operations
	visualizer.AcceptRange(0, 4096)      // Read first 4KB
	visualizer.AcceptRange(8192, 12288)  // Skip 4KB, read next 4KB
	visualizer.AcceptRange(4096, 8192)   // Go back and read the gap
	visualizer.AcceptRange(12288, 16384) // Continue sequential

	// Generate and display the graph
	graph := visualizer.DumpGraph()
	println(graph)
}

// Benchmark test for performance
func BenchmarkAcceptRange(b *testing.B) {
	rpv := NewReadPatternVisualizer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rpv.AcceptRange(int64(i*1024), int64((i+1)*1024))
	}
}

func BenchmarkDumpGraph(b *testing.B) {
	rpv := NewReadPatternVisualizer()

	// Setup with 1000 ranges
	for i := 0; i < 1000; i++ {
		rpv.AcceptRange(int64(i*1024), int64((i+1)*1024))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rpv.DumpGraph()
	}
}

func TestDumpGraphToFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "reader_pattern_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	rpv := NewReadPatternVisualizerWithReader("File Test Reader")
	rpv.SetDescription("Test File Output")
	rpv.SetGraphWidth(40)

	// Add some test ranges
	rpv.AcceptRange(0, 1024)
	rpv.AcceptRange(1024, 2048)
	rpv.AcceptRange(3072, 4096) // Gap to avoid complete merge

	// Test writing to file
	testFile := filepath.Join(tempDir, "test_output.txt")
	err = rpv.DumpGraphToFile(testFile)
	if err != nil {
		t.Fatalf("DumpGraphToFile() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("Output file was not created")
		return
	}

	// Read the file content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Verify the content matches what DumpGraph() would return
	expectedContent := rpv.DumpGraph()
	if string(content) != expectedContent {
		t.Errorf("File content does not match DumpGraph() output")
		t.Logf("Expected:\n%s", expectedContent)
		t.Logf("Got:\n%s", string(content))
	}

	// Verify expected elements are in the file
	contentStr := string(content)
	expectedElements := []string{
		"Reader: File Test Reader",
		"Read Pattern: Test File Output",
		"Total ranges added:",
		"Final ranges (after merge):",
		"Summary Statistics:",
	}

	for _, element := range expectedElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("File output missing expected element: %q", element)
		}
	}
}

func TestDumpGraphToFile_InvalidPath(t *testing.T) {
	rpv := NewReadPatternVisualizer()
	rpv.AcceptRange(0, 1024)

	// Test with invalid path (directory that doesn't exist)
	invalidPath := "/nonexistent/directory/output.txt"
	err := rpv.DumpGraphToFile(invalidPath)
	if err == nil {
		t.Errorf("Expected error when writing to invalid path, got nil")
	}
}
