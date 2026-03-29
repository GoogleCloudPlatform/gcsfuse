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
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Export writes summary to disk in the requested format ("yaml", "tsv", "both").
// A timestamped subdirectory bench-YYYYMMDD-HHMMSS/ is created under outputPath
// and all result files are written inside it.
// When outputPath is empty the current working directory is used.
// notify receives the "Results written to ..." status lines; pass os.Stdout to
// match the old behaviour, or an io.MultiWriter to tee into a console log.
func Export(summary RunSummary, outputPath, format string, notify io.Writer) error {
	if outputPath == "" {
		var err error
		outputPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd: %w", err)
		}
	}

	// Create a timestamped subdirectory so all result files stay together.
	timestamp := summary.StartTime.UTC().Format("20060102-150405")
	subDir := filepath.Join(outputPath, "bench-"+timestamp)
	if err := os.MkdirAll(subDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", subDir, err)
	}

	// Always write the human-readable text file alongside whatever machine format.
	if err := writeHumanText(summary, filepath.Join(subDir, "bench.txt"), notify); err != nil {
		return err
	}

	switch format {
	case "yaml":
		return writeYAML(summary, filepath.Join(subDir, "bench.yaml"), notify)
	case "tsv":
		return writeTSV(summary, filepath.Join(subDir, "bench.tsv"), notify)
	case "both":
		if err := writeYAML(summary, filepath.Join(subDir, "bench.yaml"), notify); err != nil {
			return err
		}
		return writeTSV(summary, filepath.Join(subDir, "bench.tsv"), notify)
	default:
		return fmt.Errorf("unknown output format %q; expected yaml|tsv|both", format)
	}
}

// PrintSummary writes a human-readable summary to w (typically os.Stdout or a
// tee writer that also captures to a log buffer).
func PrintSummary(w io.Writer, summary RunSummary) {
	printHumanSummary(w, summary)
}

// writeYAML marshals summary to a YAML file.
func writeYAML(summary RunSummary, path string, notify io.Writer) error {
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
	fmt.Fprintf(notify, "Results written to %s\n", path)
	return nil
}

// writeTSV writes one row per track to a TSV file.
func writeTSV(summary RunSummary, path string, notify io.Writer) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = '\t'

	header := []string{
		"track", "goroutines", "ops_total", "errors", "ops_per_sec", "throughput_mb_s", "avg_op_size_bytes",
		"ttfb_p50_us", "ttfb_p90_us", "ttfb_p95_us", "ttfb_p99_us", "ttfb_p999_us", "ttfb_max_us", "ttfb_mean_us",
		"total_p50_us", "total_p90_us", "total_p95_us", "total_p99_us", "total_p999_us", "total_max_us", "total_mean_us",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("tsv write header: %w", err)
	}

	for _, t := range summary.Tracks {
		row := []string{
			t.TrackName,
			strconv.Itoa(t.Goroutines),
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
	fmt.Fprintf(notify, "Results written to %s\n", path)
	return nil
}

func fmtF(v float64) string {
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// ---------------------------------------------------------------------------
// Human-readable formatting helpers
// ---------------------------------------------------------------------------

// insertCommas adds thousands separators to an integer string.
func insertCommas(s string) string {
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	n := len(s)
	if n <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	b.Grow(n + n/3)
	start := n % 3
	if start == 0 {
		start = 3
	}
	b.WriteString(s[:start])
	for i := start; i < n; i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// commaInt formats an int64 with thousands separators.
func commaInt(n int64) string {
	return insertCommas(strconv.FormatInt(n, 10))
}

// commaFloat formats a float64 with thousands separators in the integer part.
func commaFloat(f float64, decimals int) string {
	s := strconv.FormatFloat(f, 'f', decimals, 64)
	parts := strings.SplitN(s, ".", 2)
	parts[0] = insertCommas(parts[0])
	if len(parts) == 2 {
		return parts[0] + "." + parts[1]
	}
	return parts[0]
}

// humanLatency converts a value in microseconds to a readable string.
// Values >= 10,000 µs are shown in milliseconds; smaller values in µs.
func humanLatency(us float64) string {
	if us <= 0 {
		return "0 µs"
	}
	if us >= 10_000 {
		return commaFloat(us/1000, 2) + " ms"
	}
	return commaFloat(us, 1) + " µs"
}

// humanBytes formats a byte count with IEC (binary) units.
func humanBytes(b float64) string {
	const (
		KiB = 1 << 10
		MiB = 1 << 20
		GiB = 1 << 30
	)
	switch {
	case b >= GiB:
		return commaFloat(b/GiB, 3) + " GiB"
	case b >= MiB:
		return commaFloat(b/MiB, 3) + " MiB"
	case b >= KiB:
		return commaFloat(b/KiB, 1) + " KiB"
	default:
		return commaFloat(b, 0) + " B"
	}
}

// humanThroughput formats bytes/sec with IEC units.
func humanThroughput(bps float64) string {
	return humanBytes(bps) + "/s"
}

// humanDuration formats a time.Duration for human reading.
func humanDuration(d time.Duration) string {
	if d < time.Second {
		return commaFloat(float64(d)/float64(time.Millisecond), 1) + " ms"
	}
	if d < time.Minute {
		return commaFloat(d.Seconds(), 2) + " s"
	}
	m := int(d.Minutes())
	sec := d.Seconds() - float64(m)*60
	return fmt.Sprintf("%dm %s", m, commaFloat(sec, 2)+" s")
}

// capitalize returns s with the first rune upper-cased.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// printHumanSummary writes a fully formatted, human-readable report to w.
// Latency is shown in ms when >= 10,000 µs; throughput uses GiB/MiB/KiB.
// TTFB is shown only for read tracks; other op types display "N/A".
func printHumanSummary(w io.Writer, summary RunSummary) {
	fmt.Fprintf(w, "\n=== GCS Benchmark Results ===\n")
	fmt.Fprintf(w, "Run started:  %s\n", summary.StartTime.UTC().Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(w, "Duration:     %s\n", humanDuration(summary.MeasurementDuration))

	for _, t := range summary.Tracks {
		opLabel := t.OpType
		if opLabel == "" {
			opLabel = "unknown"
		}

		fmt.Fprintf(w, "\n--- Track: %s (%s) ---\n\n", t.TrackName, opLabel)
		fmt.Fprintf(w, "  Threads:          %d goroutines\n", t.Goroutines)
		fmt.Fprintf(w, "  Throughput:       %s\n", humanThroughput(t.ThroughputBytesPerSec))
		fmt.Fprintf(w, "  Ops/sec:          %s  (%s total, %s errors)\n",
			commaFloat(t.OpsPerSec, 2), commaInt(t.TotalOps), commaInt(t.Errors))
		fmt.Fprintf(w, "  Avg object size:  %s\n", humanBytes(t.AvgOpSizeBytes))

		// Total (end-to-end) latency
		fmt.Fprintf(w, "\n  %s latency (end-to-end):\n", capitalize(opLabel))
		fmt.Fprintf(w, "    P50      %s\n", humanLatency(t.TotalLatency.P50))
		fmt.Fprintf(w, "    P90      %s\n", humanLatency(t.TotalLatency.P90))
		fmt.Fprintf(w, "    P95      %s\n", humanLatency(t.TotalLatency.P95))
		fmt.Fprintf(w, "    P99      %s\n", humanLatency(t.TotalLatency.P99))
		fmt.Fprintf(w, "    P99.9    %s\n", humanLatency(t.TotalLatency.P999))
		fmt.Fprintf(w, "    Max      %s\n", humanLatency(t.TotalLatency.Max))
		fmt.Fprintf(w, "    Mean     %s\n", humanLatency(t.TotalLatency.Mean))

		// TTFB — only meaningful for reads
		if strings.ToLower(opLabel) == "read" {
			fmt.Fprintf(w, "\n  Time-to-First-Byte (TTFB):\n")
			fmt.Fprintf(w, "    P50      %s\n", humanLatency(t.TTFB.P50))
			fmt.Fprintf(w, "    P90      %s\n", humanLatency(t.TTFB.P90))
			fmt.Fprintf(w, "    P95      %s\n", humanLatency(t.TTFB.P95))
			fmt.Fprintf(w, "    P99      %s\n", humanLatency(t.TTFB.P99))
			fmt.Fprintf(w, "    P99.9    %s\n", humanLatency(t.TTFB.P999))
			fmt.Fprintf(w, "    Max      %s\n", humanLatency(t.TTFB.Max))
			fmt.Fprintf(w, "    Mean     %s\n", humanLatency(t.TTFB.Mean))
		} else {
			fmt.Fprintf(w, "\n  TTFB: N/A (%s operation)\n", opLabel)
		}
	}
	fmt.Fprintln(w)
}

// writeHumanText writes the human-readable report to a .txt file.
func writeHumanText(summary RunSummary, path string, notify io.Writer) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	printHumanSummary(f, summary)
	fmt.Fprintf(notify, "Results written to %s\n", path)
	return nil
}

// ExportHgrm writes HDR histogram percentile-distribution files (.hgrm) for
// each track. Files are written into the same bench-YYYYMMDD-HHMMSS/
// subdirectory as Export so all result files are co-located.
// notify receives the "Results written to ..." status lines.
func ExportHgrm(summary RunSummary, hists []*TrackHistograms, outputPath string, notify io.Writer) error {
	if len(hists) == 0 {
		return nil
	}
	if outputPath == "" {
		var err error
		outputPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd: %w", err)
		}
	}
	// Use the same timestamped subdirectory as Export.
	timestamp := summary.StartTime.UTC().Format("20060102-150405")
	subDir := filepath.Join(outputPath, "bench-"+timestamp)
	if err := os.MkdirAll(subDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", subDir, err)
	}
	for i, t := range summary.Tracks {
		if i >= len(hists) {
			break
		}
		safeName := strings.NewReplacer("/", "_", " ", "_").Replace(t.TrackName)
		ttfbPath := filepath.Join(subDir, safeName+"-ttfb.hgrm")
		totalPath := filepath.Join(subDir, safeName+"-total-latency.hgrm")

		ttfbF, err := os.Create(ttfbPath)
		if err != nil {
			return fmt.Errorf("create %s: %w", ttfbPath, err)
		}
		totalF, err := os.Create(totalPath)
		if err != nil {
			ttfbF.Close()
			return fmt.Errorf("create %s: %w", totalPath, err)
		}

		werr := hists[i].WritePercentileDistribution(ttfbF, totalF)
		ttfbF.Close()
		totalF.Close()
		if werr != nil {
			return fmt.Errorf("track %q histogram write: %w", t.TrackName, werr)
		}
		fmt.Fprintf(notify, "Results written to %s\n", ttfbPath)
		fmt.Fprintf(notify, "Results written to %s\n", totalPath)
	}
	return nil
}

// ExportConfig copies the benchmark YAML config file (configPath) into the
// same bench-YYYYMMDD-HHMMSS/ subdirectory as the other result files.
// The destination is always named "config.yaml" so it is easy to find.
// configData is the raw file contents; configPath is used only to derive a
// useful source label for the printed status line.
// notify receives the "Config saved to ..." status line.
func ExportConfig(summary RunSummary, outputPath string, configPath string, configData []byte, notify io.Writer) error {
	if len(configData) == 0 {
		return nil
	}
	if outputPath == "" {
		var err error
		outputPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd: %w", err)
		}
	}
	timestamp := summary.StartTime.UTC().Format("20060102-150405")
	subDir := filepath.Join(outputPath, "bench-"+timestamp)
	if err := os.MkdirAll(subDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", subDir, err)
	}
	dest := filepath.Join(subDir, "config.yaml")
	if err := os.WriteFile(dest, configData, 0644); err != nil {
		return fmt.Errorf("writing config to %s: %w", dest, err)
	}
	_ = configPath // used only for documentation; destination name is always config.yaml
	fmt.Fprintf(notify, "Config saved to   %s\n", dest)
	return nil
}

// ExportConsoleLog writes the captured console output (everything printed to
// stdout/stderr during the run) to console.log in the bench-YYYYMMDD-HHMMSS/
// results subdirectory.
func ExportConsoleLog(summary RunSummary, outputPath string, content []byte) error {
	if len(content) == 0 {
		return nil
	}
	if outputPath == "" {
		var err error
		outputPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd: %w", err)
		}
	}
	timestamp := summary.StartTime.UTC().Format("20060102-150405")
	subDir := filepath.Join(outputPath, "bench-"+timestamp)
	if err := os.MkdirAll(subDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", subDir, err)
	}
	dest := filepath.Join(subDir, "console.log")
	if err := os.WriteFile(dest, content, 0644); err != nil {
		return fmt.Errorf("writing console log to %s: %w", dest, err)
	}
	fmt.Printf("Console log saved to %s\n", dest)
	return nil
}
