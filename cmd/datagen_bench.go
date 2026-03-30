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

// Package cmd provides the CLI subcommands for gcs-bench.
package cmd

// datagen_bench.go — gcs-bench datagen-bench
//
// Measures raw Xoshiro256++ data-generation throughput at increasing
// GOMAXPROCS values (1, 2, 4, 8, …, runtime.NumCPU()).  The output table
// shows how well generation scales with the number of OS threads.
//
// This command exists to verify that the prerequisite condition for the
// write-benchmark pipeline holds:
//
//	generation throughput ≥ 1.5 × maximum network write rate
//
// If generation is the bottleneck, the DataPool's background producer will
// stall and the write benchmark will be CPU-limited rather than network-limited.
//
// Usage:
//
//	gcs-bench datagen-bench                  # full scaling sweep on this machine
//	gcs-bench datagen-bench --total 5        # quick 5-GiB run per GOMAXPROCS step
//	gcs-bench datagen-bench --buf-size 16    # 16 MiB fill buffer (default: 64 MiB)

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/benchmark"
	"github.com/spf13/cobra"
)

// ExecuteDatagenBenchCmd is the entry point called from main.go.
var ExecuteDatagenBenchCmd = func() {
	c := newDatagenBenchCmd()
	c.SetArgs(os.Args[1:])
	if err := c.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newDatagenBenchCmd() *cobra.Command {
	var (
		bufSizeMiB int
		totalGiB   int
		warmupGiB  int
	)

	c := &cobra.Command{
		Use:   "datagen-bench",
		Short: "Benchmark Xoshiro256++ data-generation throughput and CPU scaling",
		Long: `datagen-bench measures how fast gcs-bench can fill memory with
incompressible pseudo-random bytes (Xoshiro256++) at varying GOMAXPROCS.

Each step:
  1. Warm up (--warmup GiB, discarded).
  2. Fill --total GiB in --buf-size MiB chunks.
  3. Report GB/s, GiB/s, and elapsed seconds.

Goal: verify generation rate ≥ 1.5× expected GCS write throughput so that
the DataPool producer never stalls the write benchmark.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDatagenBench(bufSizeMiB, totalGiB, warmupGiB)
		},
	}

	c.Flags().IntVar(&bufSizeMiB, "buf-size", 64, "Fill-buffer size in MiB per goroutine chunk")
	c.Flags().IntVar(&totalGiB, "total", 20, "Total GiB to generate per GOMAXPROCS step (measurement)")
	c.Flags().IntVar(&warmupGiB, "warmup", 1, "Warm-up GiB per step (discarded, warms OS page tables)")
	return c
}

func runDatagenBench(bufSizeMiB, totalGiB, warmupGiB int) error {
	maxProcs := runtime.NumCPU()
	bufSize := int64(bufSizeMiB) * 1024 * 1024
	totalBytes := int64(totalGiB) * 1024 * 1024 * 1024
	warmupBytes := int64(warmupGiB) * 1024 * 1024 * 1024

	fmt.Printf("gcs-bench datagen-bench — Xoshiro256++ generation throughput\n")
	fmt.Printf("System:      %d logical CPUs\n", maxProcs)
	fmt.Printf("Buffer size: %d MiB per goroutine chunk\n", bufSizeMiB)
	fmt.Printf("Measurement: %d GiB per GOMAXPROCS step\n", totalGiB)
	fmt.Printf("Warm-up:     %d GiB per step\n\n", warmupGiB)

	// GOMAXPROCS values to test: powers of two up to numCPU, always include numCPU.
	procs := cpuSteps(maxProcs)

	// Fixed entropy so results are reproducible; actual benchmarks use crypto seed.
	const entropy uint64 = 0xf00dcafe_12345678

	orig := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(orig)

	hdr := fmt.Sprintf("%-8s  %-12s  %-12s  %-10s", "CPUs", "GB/s", "GiB/s", "Time(s)")
	sep := strings.Repeat("─", len(hdr))
	fmt.Println(hdr)
	fmt.Println(sep)

	for _, ncpu := range procs {
		runtime.GOMAXPROCS(ncpu)

		// Allocate one buffer sized for the number of active goroutines.
		// fillRandomLoop will internally spawn ≤ ncpu goroutines (one per 1 MiB block),
		// so the buffer should hold enough blocks to saturate all threads.
		buf := make([]byte, bufSize)

		// Warm-up: fill warmupGiB, discard timing.
		fillRandomLoop(buf, entropy, 0, warmupBytes)

		// Measurement: fill totalGiB, record time.
		startSeq := uint64(warmupBytes/int64(benchmark.GenBlockSize)) + 1
		start := time.Now()
		fillRandomLoop(buf, entropy, startSeq, totalBytes)
		elapsed := time.Since(start).Seconds()

		gbps := float64(totalBytes) / elapsed / 1e9
		gibps := float64(totalBytes) / elapsed / (1024 * 1024 * 1024)
		fmt.Printf("%-8d  %-12.2f  %-12.2f  %-10.3f\n", ncpu, gbps, gibps, elapsed)
	}
	return nil
}

// fillRandomLoop repeatedly calls benchmark.FillRandom to fill totalBytes
// using buf as a scratch buffer.  startSeq gives the initial block sequence
// so successive calls never reuse the same RNG state.
func fillRandomLoop(buf []byte, entropy uint64, startSeq uint64, totalBytes int64) {
	bufLen := int64(len(buf))
	nBlocks := (bufLen + int64(benchmark.GenBlockSize) - 1) / int64(benchmark.GenBlockSize)
	seq := startSeq
	remaining := totalBytes
	for remaining > 0 {
		fill := bufLen
		if fill > remaining {
			fill = remaining
		}
		benchmark.FillRandom(buf[:fill], entropy, seq)
		seq += uint64(nBlocks)
		remaining -= fill
	}
}

// cpuSteps returns the GOMAXPROCS values to test: 1, 2, 4, 8, …, up to max,
// plus max itself if it isn't already a power of two in the sequence.
func cpuSteps(max int) []int {
	seen := map[int]bool{}
	var steps []int
	for v := 1; v <= max; v *= 2 {
		if !seen[v] {
			steps = append(steps, v)
			seen[v] = true
		}
	}
	if !seen[max] {
		steps = append(steps, max)
	}
	return steps
}
