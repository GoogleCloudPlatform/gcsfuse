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

package benchmark

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Export writes summary to disk in the requested format ("yaml", "tsv", "both").
// Files are named bench-YYYYMMDD-HHMMSS.{yaml,tsv} under outputPath.
// When outputPath is empty the current working directory is used.
func Export(summary RunSummary, outputPath, format string) error {
	if outputPath == "" {
		var err error
		outputPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd: %w", err)
		}
	}
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outputPath, err)
	}

	timestamp := summary.StartTime.UTC().Format("20060102-150405")
	stem := filepath.Join(outputPath, "bench-"+timestamp)

	switch format {
	case "yaml":
		return writeYAML(summary, stem+".yaml")
	case "tsv":
		return writeTSV(summary, stem+".tsv")
	case "both":
		if err := writeYAML(summary, stem+".yaml"); err != nil {
			return err
		}
		return writeTSV(summary, stem+".tsv")
	default:
		return fmt.Errorf("unknown output format %q; expected yaml|tsv|both", format)
	}
}

// PrintSummary writes a human-readable summary table to stdout.
func PrintSummary(summary RunSummary) {
	fmt.Printf("\n=== Benchmark Results (%s, %.1fs) ===\n",
		summary.StartTime.UTC().Format(time.RFC3339),
		summary.MeasurementDuration.Seconds())

	for _, t := range summary.Tracks {
		fmt.Printf("\nTrack: %s\n", t.TrackName)
		fmt.Printf("  Ops/s:       %.1f    Errors: %d / %d\n", t.OpsPerSec, t.Errors, t.TotalOps)
		fmt.Printf("  Throughput:  %.2f MB/s    Avg op size: %.2f MiB\n",
			t.ThroughputBytesPerSec/1e6, t.AvgOpSizeBytes/(1024*1024))
		fmt.Printf("  TTFB (us)   p50=%.0f  p90=%.0f  p95=%.0f  p99=%.0f  p999=%.0f  max=%.0f  mean=%.1f\n",
			t.TTFB.P50, t.TTFB.P90, t.TTFB.P95, t.TTFB.P99, t.TTFB.P999, t.TTFB.Max, t.TTFB.Mean)
		fmt.Printf("  Total (us)  p50=%.0f  p90=%.0f  p95=%.0f  p99=%.0f  p999=%.0f  max=%.0f  mean=%.1f\n",
			t.TotalLatency.P50, t.TotalLatency.P90, t.TotalLatency.P95, t.TotalLatency.P99,
			t.TotalLatency.P999, t.TotalLatency.Max, t.TotalLatency.Mean)
	}
	fmt.Println()
}

// writeYAML marshals summary to a YAML file.
func writeYAML(summary RunSummary, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(summary); err != nil {
		return fmt.Errorf("yaml encode: %w", err)
	}
	fmt.Printf("Results written to %s\n", path)
	return nil
}

// writeTSV writes one row per track to a TSV file.
func writeTSV(summary RunSummary, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = '\t'

	header := []string{
		"track", "ops_total", "errors", "ops_per_sec", "throughput_mb_s", "avg_op_size_bytes",
		"ttfb_p50_us", "ttfb_p90_us", "ttfb_p95_us", "ttfb_p99_us", "ttfb_p999_us", "ttfb_max_us", "ttfb_mean_us",
		"total_p50_us", "total_p90_us", "total_p95_us", "total_p99_us", "total_p999_us", "total_max_us", "total_mean_us",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("tsv write header: %w", err)
	}

	for _, t := range summary.Tracks {
		row := []string{
			t.TrackName,
			strconv.FormatInt(t.TotalOps, 10),
			strconv.FormatInt(t.Errors, 10),
			strconv.FormatFloat(t.OpsPerSec, 'f', 2, 64),
			strconv.FormatFloat(t.ThroughputBytesPerSec/1e6, 'f', 3, 64),
			strconv.FormatFloat(t.AvgOpSizeBytes, 'f', 1, 64),
			fmtF(t.TTFB.P50), fmtF(t.TTFB.P90), fmtF(t.TTFB.P95),
			fmtF(t.TTFB.P99), fmtF(t.TTFB.P999), fmtF(t.TTFB.Max), fmtF(t.TTFB.Mean),
			fmtF(t.TotalLatency.P50), fmtF(t.TotalLatency.P90), fmtF(t.TotalLatency.P95),
			fmtF(t.TotalLatency.P99), fmtF(t.TotalLatency.P999), fmtF(t.TotalLatency.Max), fmtF(t.TotalLatency.Mean),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("tsv write row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("tsv flush: %w", err)
	}
	fmt.Printf("Results written to %s\n", path)
	return nil
}

func fmtF(v float64) string {
	return strconv.FormatFloat(v, 'f', 1, 64)
}
