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

package main

import (
	"fmt"
	"os"

	"github.com/googlecloudplatform/gcsfuse/v3/common"
)

func main() {
	fmt.Println("=== Read Pattern Visualizer Demo ===")

	// Demo 1: Sequential Reading Pattern
	fmt.Println("Demo 1: Sequential Reading Pattern")
	fmt.Println("-----------------------------------")

	sequential := common.NewReadPatternVisualizerWithReader("Sequential Reader")
	sequential.SetDescription("Sequential File Reading")
	sequential.SetGraphWidth(60)
	sequential.SetScaleUnit(1024) // 1KB scale

	// Simulate sequential reads
	for i := 0; i < 8; i++ {
		start := int64(i * 4096) // 4KB blocks
		end := int64((i + 1) * 4096)
		sequential.AcceptRange(start, end)
	}

	fmt.Println(sequential.DumpGraph())
	fmt.Println()

	// Demo 2: Random Access Pattern
	fmt.Println("Demo 2: Random Access Pattern")
	fmt.Println("------------------------------")

	random := common.NewReadPatternVisualizerWithReader("Random Access Reader")
	random.SetDescription("Random File Access")
	random.SetGraphWidth(60)
	random.SetScaleUnit(1024) // 1KB scale

	// Simulate random reads
	randomOffsets := []struct{ start, end int64 }{
		{0, 2048},
		{8192, 12288},
		{4096, 6144},
		{16384, 20480},
		{2048, 4096},
		{12288, 14336},
		{20480, 22528},
		{6144, 8192},
	}

	for _, offset := range randomOffsets {
		random.AcceptRange(offset.start, offset.end)
	}

	fmt.Println(random.DumpGraph())
	fmt.Println()

	// Demo 3: Mixed Pattern with Overlaps
	fmt.Println("Demo 3: Overlapping Reads Pattern")
	fmt.Println("----------------------------------")

	overlapping := common.NewReadPatternVisualizerWithReader("Overlapping Prefetch Reader")
	overlapping.SetDescription("Overlapping Read Operations")
	overlapping.SetGraphWidth(80)
	overlapping.SetScaleUnit(1024 * 1024) // 1MB scale

	// Simulate overlapping reads (common in prefetch scenarios)
	overlappingRanges := []struct{ start, end int64 }{
		{0, 1024 * 1024},           // 0-1MB
		{512 * 1024, 1536 * 1024},  // 0.5-1.5MB (overlaps with previous)
		{1024 * 1024, 2048 * 1024}, // 1-2MB
		{1536 * 1024, 2560 * 1024}, // 1.5-2.5MB (overlaps)
		{2048 * 1024, 3072 * 1024}, // 2-3MB
	}

	for _, offset := range overlappingRanges {
		overlapping.AcceptRange(offset.start, offset.end)
	}

	fmt.Println(overlapping.DumpGraph())
	fmt.Println()

	// Demo 4: Large File Reading with Custom Scale
	fmt.Println("Demo 4: Large File Reading (GB scale)")
	fmt.Println("--------------------------------------")

	largefile := common.NewReadPatternVisualizerWithFullConfig(
		1024*1024*1024, // 1GB scale unit
		50,             // narrower graph
		"Large File Processing",
		"Large File Reader")

	// Simulate reading chunks of a very large file
	gbSize := int64(1024 * 1024 * 1024)
	for i := int64(0); i < 10; i += 2 { // Read every other GB
		start := i * gbSize
		end := (i + 1) * gbSize
		largefile.AcceptRange(start, end)
	}

	fmt.Println(largefile.DumpGraph())
	fmt.Println()

	// Demo 5: Range Merging Demonstration
	fmt.Println("Demo 5: Range Merging Demonstration")
	fmt.Println("------------------------------------")

	// Show how consecutive ranges get merged
	merging := common.NewReadPatternVisualizerWithReader("Range Merging Reader")
	merging.SetDescription("Range Merging Example")
	merging.SetGraphWidth(50)
	merging.SetScaleUnit(1024) // 1KB scale

	fmt.Println("Adding consecutive ranges that will be merged:")
	ranges_to_add := []struct{ start, end int64 }{
		{0, 1024},    // Range 1
		{1024, 2048}, // Adjacent to range 1 - will merge
		{2048, 3072}, // Adjacent to merged range - will merge
		{4096, 5120}, // Gap - new separate range
		{5120, 6144}, // Adjacent to range 4 - will merge
	}

	for i, r := range ranges_to_add {
		fmt.Printf("  Step %d: Adding [%d, %d)\n", i+1, r.start, r.end)
		merging.AcceptRange(r.start, r.end)
		ranges := merging.GetRanges()
		fmt.Printf("           Current ranges: %d\n", len(ranges))
		if len(ranges) <= 3 { // Only show details if not too many
			for j, range_ := range ranges {
				fmt.Printf("           Range %d: %s\n", j+1, range_.String())
			}
		}
		fmt.Println()
	}

	fmt.Printf("Final result: %d ranges instead of %d individual ranges\n",
		len(merging.GetRanges()), len(ranges_to_add))
	fmt.Println("\nFinal Graph:")
	fmt.Println(merging.DumpGraph())
	fmt.Println()

	// Demo 6: File Output Demonstration
	fmt.Println("Demo 6: File Output Demonstration")
	fmt.Println("----------------------------------")

	fileDemo := common.NewReadPatternVisualizerWithReader("File Output Reader")
	fileDemo.SetDescription("Demonstration of File Output Feature")
	fileDemo.SetGraphWidth(50)

	// Add some sample ranges
	fileDemo.AcceptRange(0, 2048)
	fileDemo.AcceptRange(4096, 6144)
	fileDemo.AcceptRange(8192, 10240)

	// Write to file
	outputFile := "read_pattern_output.txt"
	err := fileDemo.DumpGraphToFile(outputFile)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
	} else {
		fmt.Printf("Graph successfully written to: %s\n", outputFile)
		fmt.Println("File contents:")
		fmt.Println(fileDemo.DumpGraph())
	}
	fmt.Println()

	// Demo 7: Interactive example allowing user input
	if len(os.Args) > 1 && os.Args[1] == "--interactive" {
		runInteractiveDemo()
	}

	fmt.Println("=== End of Demo ===")
}

func runInteractiveDemo() {
	fmt.Println("Demo 6: Interactive Range Input")
	fmt.Println("-------------------------------")
	fmt.Println("Enter ranges in format 'start end' (e.g., '0 1024')")
	fmt.Println("Enter 'done' to finish and display graph")
	fmt.Println()

	interactive := common.NewReadPatternVisualizerWithReader("Interactive User Reader")
	interactive.SetDescription("User-defined Read Pattern")
	interactive.SetGraphWidth(70)

	var start, end int64
	for {
		fmt.Print("Enter range (start end) or 'done': ")
		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		if input == "done" {
			break
		}

		// Try to parse two integers
		n, err := fmt.Sscanf(input, "%d %d", &start, &end)
		if n != 2 || err != nil {
			fmt.Println("Invalid format. Use: start end (e.g., '0 1024')")
			continue
		}

		interactive.AcceptRange(start, end)
		fmt.Printf("Added range [%d, %d)\n", start, end)
	}

	fmt.Println("\nGenerated Graph:")
	fmt.Println(interactive.DumpGraph())
}
