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

package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/benchmark"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ExecuteMergeResultsCmd is the standalone entry point for the
// 'gcs-bench merge-results' subcommand.
var ExecuteMergeResultsCmd = func() {
	rootCmd := newMergeResultsCmd()
	rootCmd.SetArgs(os.Args[1:])
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("merge-results command failed: %v", err)
	}
}

// newMergeResultsCmd builds the Cobra command for 'gcs-bench merge-results'.
func newMergeResultsCmd() *cobra.Command {
	var (
		outputPath   string
		outputFormat string
	)

	cmd := &cobra.Command{
		Use:   "merge-results <worker-result.yaml> [<worker-result.yaml> ...]",
		Short: "Merge per-worker YAML result files into a combined summary",
		Long: `merge-results loads the YAML output files produced by multiple
gcs-bench workers (identified by --worker-id / --num-workers) and statistically
merges their HDR histograms to produce accurate combined percentiles.

The merge is performed using the HDR histogram Merge() method — the same
approach used by sai3-bench — so no accuracy is lost (unlike averaging).

Per-worker summaries require raw histogram data (RawTTFB / RawTotal fields)
which are automatically included when gcs-bench runs with --num-workers > 1.

Example:
  gcs-bench merge-results worker-0.yaml worker-1.yaml worker-2.yaml
  gcs-bench merge-results results/*.yaml --output-format both`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMergeResults(args, outputPath, outputFormat)
		},
	}

	cmd.Flags().StringVar(&outputPath, "output-path", "", "Directory for merged result files (default: cwd)")
	cmd.Flags().StringVar(&outputFormat, "output-format", "yaml", "Output format: yaml|tsv|both")

	return cmd
}

// runMergeResults loads per-worker YAML files, merges HDR histograms, and
// outputs a consolidated RunSummary alongside a per-worker comparison table.
func runMergeResults(paths []string, outputPath, outputFormat string) error {
	if len(paths) == 0 {
		return fmt.Errorf("no input files specified")
	}

	// Load all per-worker summaries.
	workers := make([]benchmark.RunSummary, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		var s benchmark.RunSummary
		if err := yaml.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("parsing %s: %w", p, err)
		}
		workers = append(workers, s)
	}

	if len(workers) == 0 {
		return fmt.Errorf("no valid summaries loaded")
	}

	// Validate that all workers have the same track layout.
	numTracks := len(workers[0].Tracks)
	for i, w := range workers[1:] {
		if len(w.Tracks) != numTracks {
			return fmt.Errorf("worker %d (%s) has %d tracks; expected %d from first file",
				i+1, paths[i+1], len(w.Tracks), numTracks)
		}
	}

	// Build merged summary via HDR histogram Merge().
	merged, err := mergeWorkerSummaries(workers)
	if err != nil {
		return fmt.Errorf("merging summaries: %w", err)
	}

	// Print merged results.
	fmt.Printf("\n=== Merged Results (%d workers) ===\n", len(workers))
	benchmark.PrintSummary(merged)

	// Print per-worker comparison table.
	printWorkerComparison(workers, paths)

	// Export merged summary.
	if outputFormat == "" {
		outputFormat = "yaml"
	}
	if err := benchmark.Export(merged, outputPath, outputFormat); err != nil {
		return fmt.Errorf("export merged results: %w", err)
	}

	return nil
}

// mergeWorkerSummaries merges multiple per-worker RunSummary structs into a
// single combined RunSummary using HDR histogram merging for accurate
// cross-worker percentiles.
func mergeWorkerSummaries(workers []benchmark.RunSummary) (benchmark.RunSummary, error) {
	if len(workers) == 0 {
		return benchmark.RunSummary{}, fmt.Errorf("no workers to merge")
	}

	// Use the earliest start time and the longest measurement duration.
	earliest := workers[0].StartTime
	longestDuration := workers[0].MeasurementDuration
	for _, w := range workers[1:] {
		if w.StartTime.Before(earliest) {
			earliest = w.StartTime
		}
		if w.MeasurementDuration > longestDuration {
			longestDuration = w.MeasurementDuration
		}
	}

	numTracks := len(workers[0].Tracks)

	// Accumulate per-track merged histograms and aggregate counters.
	type trackAccum struct {
		name      string
		totalOps  int64
		errors    int64
		bytes     float64 // throughput * duration for each worker, summed
		opsPerSec float64 // sum across workers
		hists     *benchmark.TrackHistograms
		histOK    bool // true if at least one worker contributed histogram data
	}

	accums := make([]trackAccum, numTracks)
	histCfg := cfg.HistogramConfig{
		MinValueMicros:    1,
		MaxValueMicros:    60_000_000,
		SignificantDigits: 3,
	}
	for i, t := range workers[0].Tracks {
		accums[i] = trackAccum{
			name:  t.TrackName,
			hists: benchmark.NewTrackHistograms(histCfg),
		}
	}

	for _, w := range workers {
		dur := w.MeasurementDuration.Seconds()
		for i, t := range w.Tracks {
			accums[i].totalOps += t.TotalOps
			accums[i].errors += t.Errors
			// Reconstruct total bytes from throughput × duration to aggregate.
			accums[i].bytes += t.ThroughputBytesPerSec * dur
			accums[i].opsPerSec += t.OpsPerSec

			// Merge raw histograms when available.
			if t.RawTTFB != "" && t.RawTotal != "" {
				if err := accums[i].hists.MergeFromBase64(t.RawTTFB, t.RawTotal); err != nil {
					return benchmark.RunSummary{}, fmt.Errorf(
						"merging histogram for track %q from worker %d: %w",
						t.TrackName, w.WorkerID, err)
				}
				accums[i].histOK = true
			}
		}
	}

	// Build output tracks from accumulated data.
	tracks := make([]benchmark.TrackStats, numTracks)
	for i, acc := range accums {
		var ttfb, total benchmark.LatencyPercentiles
		if acc.histOK {
			// Compute percentiles from the merged HDR histogram — accurate.
			ttfb, total = acc.hists.Snapshot()
		} else {
			// Fall back to the first worker's percentiles with a warning.
			fmt.Printf("warning: no raw histogram data for track %q; "+
				"percentiles are from worker 0 only (run with --num-workers to enable merging)\n",
				acc.name)
			ttfb = workers[0].Tracks[i].TTFB
			total = workers[0].Tracks[i].TotalLatency
		}

		avgDur := longestDuration.Seconds()
		var throughput float64
		if avgDur > 0 {
			// Total bytes across all workers / measurement window.
			throughput = acc.bytes / avgDur
		}

		tracks[i] = benchmark.TrackStats{
			TrackName:             acc.name,
			WorkerID:              -1, // -1 = merged
			Goroutines:            workers[0].Tracks[i].Goroutines,
			TotalOps:              acc.totalOps,
			Errors:                acc.errors,
			ThroughputBytesPerSec: throughput,
			OpsPerSec:             acc.opsPerSec,
			TTFB:                  ttfb,
			TotalLatency:          total,
		}
	}

	return benchmark.RunSummary{
		StartTime:           earliest,
		MeasurementDuration: longestDuration,
		WorkerID:            -1,
		Tracks:              tracks,
	}, nil
}

// printWorkerComparison prints a per-worker summary table to stdout.
func printWorkerComparison(workers []benchmark.RunSummary, paths []string) {
	fmt.Println("\n=== Per-Worker Summary ===")

	// Header for each track that appeared across workers.
	if len(workers) == 0 || len(workers[0].Tracks) == 0 {
		return
	}

	for _, ref := range workers[0].Tracks {
		trackName := ref.TrackName
		fmt.Printf("\nTrack: %s\n", trackName)
		fmt.Printf("  %-30s  %8s  %10s  %10s  %10s  %10s  %10s\n",
			"Worker file", "ops/s", "MB/s", "ttfb_p50", "ttfb_p99", "total_p50", "total_p99")
		fmt.Printf("  %-30s  %8s  %10s  %10s  %10s  %10s  %10s\n",
			strings.Repeat("-", 30), "--------", "----------", "----------", "----------", "----------", "----------")

		for wi, w := range workers {
			for _, t := range w.Tracks {
				if t.TrackName != trackName {
					continue
				}
				label := paths[wi]
				if len(label) > 30 {
					label = "..." + label[len(label)-27:]
				}
				fmt.Printf("  %-30s  %8.1f  %10.2f  %10.0f  %10.0f  %10.0f  %10.0f\n",
					label,
					t.OpsPerSec,
					t.ThroughputBytesPerSec/1e6,
					t.TTFB.P50,
					t.TTFB.P99,
					t.TotalLatency.P50,
					t.TotalLatency.P99,
				)
				break
			}
		}
	}

	// Wall time range.
	if len(workers) > 0 {
		earliest := workers[0].StartTime
		latest := workers[0].StartTime
		for _, w := range workers[1:] {
			if w.StartTime.Before(earliest) {
				earliest = w.StartTime
			}
			if w.StartTime.After(latest) {
				latest = w.StartTime
			}
		}
		skew := latest.Sub(earliest)
		fmt.Printf("\nWorker start skew: %s  (earliest: %s)\n",
			skew.Round(time.Millisecond),
			earliest.UTC().Format(time.RFC3339))
	}
	fmt.Println()
}
