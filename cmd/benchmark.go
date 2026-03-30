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
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/benchmark"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	internalstorage "github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	storageutil "github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ExecuteBenchmarkCmd is the standalone entry point for the gcsfuse-bench
// benchmark subcommand. It does not mount a FUSE file system — it reads GCS
// objects directly via the Go storage client, wrapped with the instrumented
// bucket to capture latency distributions.
var ExecuteBenchmarkCmd = func() {
	rootCmd := newBenchmarkRootCmd()
	rootCmd.SetArgs(os.Args[1:])
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("benchmark command failed: %v", err)
	}
}

// newBenchmarkRootCmd builds the Cobra command tree for the benchmark binary.
func newBenchmarkRootCmd() *cobra.Command {
	var (
		cfgFile        string
		duration       time.Duration
		warmup         time.Duration
		concurrency    int
		objectPrefix   string
		outputPath     string
		outputFormat   string
		keyFile        string
		customEndpoint string
		dryRun         bool
		mode           string
		workerID       int
		numWorkers     int
		startAt        int64
		rapidMode      string
		verbosity      int
	)

	rootCmd := &cobra.Command{
		Use:          "gcs-bench bench [flags]",
		Short:        "Standalone GCS storage benchmark (no FUSE mount required)",
		Long:         benchmarkLongDescription,
		Version:      common.GetVersion(),
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --- Load config file (required — all parameters including bucket live here) ---
			benchCfg := cfg.DefaultBenchmarkConfig()
			var configData []byte // raw YAML bytes; saved into the results directory later
			if cfgFile != "" {
				var err error
				configData, err = os.ReadFile(cfgFile)
				if err != nil {
					return fmt.Errorf("reading config file: %w", err)
				}
				// Use yaml.v3 directly so that hyphenated keys (op-type,
				// object-size-min, etc.) are handled via struct tags correctly.
				var envelope struct {
					Benchmark cfg.BenchmarkConfig `yaml:"benchmark"`
				}
				if err := yaml.Unmarshal(configData, &envelope); err != nil {
					return fmt.Errorf("parsing config file %s: %w", cfgFile, err)
				}
				benchCfg = envelope.Benchmark
			}

			// Bucket must be set in the config file.
			bucketName := benchCfg.Bucket
			if bucketName == "" {
				return fmt.Errorf("bucket name required: set 'bucket:' in the config file (--config)")
			}

			// --- CLI flags override config file values ---
			if cmd.Flags().Changed("duration") {
				benchCfg.Duration = duration
			}
			if cmd.Flags().Changed("warmup") {
				benchCfg.WarmupDuration = warmup
			}
			if cmd.Flags().Changed("concurrency") {
				benchCfg.TotalConcurrency = concurrency
			}
			if cmd.Flags().Changed("object-prefix") {
				benchCfg.ObjectPrefix = objectPrefix
			}
			if cmd.Flags().Changed("output-path") {
				benchCfg.OutputPath = outputPath
			}
			if cmd.Flags().Changed("output-format") {
				benchCfg.OutputFormat = outputFormat
			}
			if cmd.Flags().Changed("mode") {
				benchCfg.Mode = mode
			}
			if cmd.Flags().Changed("worker-id") {
				benchCfg.WorkerID = workerID
			}
			if cmd.Flags().Changed("num-workers") {
				benchCfg.NumWorkers = numWorkers
			}
			if cmd.Flags().Changed("start-at") {
				benchCfg.StartAt = startAt
			}
			if cmd.Flags().Changed("rapid-mode") {
				benchCfg.RapidMode = rapidMode
			}

			// Apply defaults for histogram config.
			if benchCfg.Histograms.MinValueMicros == 0 {
				benchCfg.Histograms.MinValueMicros = 1
			}
			if benchCfg.Histograms.MaxValueMicros == 0 {
				benchCfg.Histograms.MaxValueMicros = 60_000_000
			}
			if benchCfg.Histograms.SignificantDigits == 0 {
				benchCfg.Histograms.SignificantDigits = 3
			}

			// Require at least one track.
			if len(benchCfg.Tracks) == 0 {
				return fmt.Errorf("no benchmark tracks defined; specify tracks in config file")
			}

			// --- Dry-run: validate config and describe what would run ---
			if dryRun {
				return printDryRun(bucketName, benchCfg)
			}

			// --- Configure logging ---
			// Default: WARN (only errors/warnings visible).
			// -v:   INFO  — phase transitions, prepare progress
			// -vv:  DEBUG — per-object errors, retry detail
			// -vvv: TRACE — every individual GCS call
			logSeverity := cfg.LogSeverity(cfg.WARNING)
			switch {
			case verbosity >= 3:
				logSeverity = cfg.LogSeverity(cfg.TRACE)
			case verbosity == 2:
				logSeverity = cfg.LogSeverity(cfg.DEBUG)
			case verbosity == 1:
				logSeverity = cfg.LogSeverity(cfg.INFO)
			}
			if err := logger.InitLogFile(cfg.LoggingConfig{
				Severity:  logSeverity,
				Format:    "text",
				LogRotate: cfg.DefaultLoggingConfig().LogRotate,
			}, "gcs-bench"); err != nil {
				return fmt.Errorf("initializing logger: %w", err)
			}

			// --- Resolve rapid-mode ---
			// RAPID (zonal) GCS buckets require the bidi-streaming gRPC client.
			// Using HTTP/2 against a RAPID bucket causes 100% I/O errors.
			//   auto (default): call GetStorageLayout to detect zonal bucket type.
			//   on:             force bidi-gRPC without detection.
			//   off:            use HTTP/2; no detection call.
			effectiveRapidMode := strings.ToLower(benchCfg.RapidMode)
			if effectiveRapidMode == "" {
				effectiveRapidMode = "auto"
			}
			if effectiveRapidMode != "auto" && effectiveRapidMode != "on" && effectiveRapidMode != "off" {
				return fmt.Errorf("invalid rapid-mode %q: must be auto, on, or off", effectiveRapidMode)
			}

			// --- Create GCS storage client ---
			userAgent := fmt.Sprintf("gcsfuse-bench/%s", common.GetVersion())
			storageClientConfig := storageutil.StorageClientConfig{
				ClientProtocol:  cfg.HTTP2,
				MaxConnsPerHost: 0, // unlimited
				UserAgent:       userAgent,
				KeyFile:         keyFile,
				CustomEndpoint:  customEndpoint,
			}
			finalizeForRapid := false
			switch effectiveRapidMode {
			case "on":
				// Force bidi-gRPC; skip GetStorageLayout.
				storageClientConfig.ForceZonal = true
				finalizeForRapid = true
				logger.Infof("RAPID mode: on (bidi-gRPC forced, skipping detection)\n")
			case "auto":
				// Enable the storage control client so GetStorageLayout can detect
				// whether this is a RAPID/zonal bucket.
				storageClientConfig.EnableHNS = true
				finalizeForRapid = true // no-op for non-zonal buckets
				logger.Infof("RAPID mode: auto (detecting bucket type via GetStorageLayout)\n")
			case "off":
				// HTTP/2 only; no detection. Use for non-RAPID buckets.
				logger.Infof("RAPID mode: off (HTTP/2, no detection)\n")
			}

			logger.Infof("Creating GCS storage handle (bucket=%s)...\n", bucketName)
			sh, err := internalstorage.NewStorageHandle(context.Background(), storageClientConfig, "")
			if err != nil {
				return fmt.Errorf("NewStorageHandle: %w", err)
			}

			// --- Get a raw bucket handle ---
			ctx := context.Background()
			bh, err := sh.BucketHandle(ctx, bucketName, "", finalizeForRapid)
			if err != nil {
				return fmt.Errorf("BucketHandle(%q): %w", bucketName, err)
			}

			// Log the resolved transport so the operator can confirm RAPID/bidi-gRPC
			// is actually active (not just requested).
			bt := bh.BucketType()
			if bt.Zonal {
				logger.Infof("RAPID mode: CONFIRMED — bucket %q is zonal; bidi-gRPC (RAPID) transport is ACTIVE\n", bucketName)
			} else {
				logger.Infof("RAPID mode: bucket %q is NOT zonal; using standard HTTP/2 transport\n", bucketName)
			}

			// --- Pre-flight check ---
			// Uses the raw (un-wrapped) bucket handle so that pre-flight I/O
			// does not pollute the benchmark histograms.
			//
			// Set up a tee writer so all console output is captured in logBuf
			// and saved as console.log in the results directory at the end of
			// the run.  consoleOut tees stdout; progressOut tees stderr (for
			// the live [warmup]/[bench] progress lines).
			var logBuf bytes.Buffer
			consoleOut := io.MultiWriter(os.Stdout, &logBuf)
			progressOut := io.MultiWriter(os.Stderr, &logBuf)

			if err := benchmark.RunPreflight(ctx, bh, bucketName, benchCfg.ObjectPrefix, benchCfg.Mode, consoleOut); err != nil {
				return err
			}

			// --- Wrap each track with its own instrumented bucket and run ---
			globalHists := benchmark.NewTrackHistograms(benchCfg.Histograms)
			wrappedBucket := internalstorage.NewInstrumentedBucket(bh, internalstorage.InstrumentedBucketParams{
				TrackName: "all",
				Hists:     globalHists,
				Events:    nil,
			})

			_ = metrics.NewNoopMetrics()

			// --- Create and run engine ---
			engine, err := benchmark.NewEngine(wrappedBucket, benchCfg, verbosity, progressOut)
			if err != nil {
				return fmt.Errorf("NewEngine: %w", err)
			}

			summary, err := engine.Run(ctx)
			if err != nil {
				return fmt.Errorf("benchmark run: %w", err)
			}

			// Write HDR histogram .hgrm files for plotting (always, alongside other outputs).
			if err := benchmark.ExportHgrm(summary, engine.Histograms(), benchCfg.OutputPath, consoleOut); err != nil {
				fmt.Printf("Warning: could not write histogram files: %v\n", err)
			}

			// Copy the config file into the results directory so the run is fully reproducible.
			if cfgFile != "" {
				if err := benchmark.ExportConfig(summary, benchCfg.OutputPath, cfgFile, configData, consoleOut); err != nil {
					fmt.Printf("Warning: could not save config file: %v\n", err)
				}
			}

			// --- Output ---
			// Prepare mode omits the console summary table (progress already printed
			// live) but always writes the result file so runs are traceable.
			if strings.ToLower(benchCfg.Mode) != "prepare" {
				benchmark.PrintSummary(consoleOut, summary)
			}
			format := benchCfg.OutputFormat
			if format == "" {
				format = "yaml"
			}
			if err := benchmark.Export(summary, benchCfg.OutputPath, format, consoleOut); err != nil {
				return fmt.Errorf("export results: %w", err)
			}
			// Save everything that was printed to the terminal into console.log
			// inside the results directory so the run is fully self-contained.
			if err := benchmark.ExportConsoleLog(summary, benchCfg.OutputPath, logBuf.Bytes()); err != nil {
				fmt.Printf("Warning: could not save console log: %v\n", err)
			}
			return nil
		},
	}

	rootCmd.Flags().StringVar(&cfgFile, "config", "", "Path to benchmark YAML config file")
	rootCmd.Flags().DurationVar(&duration, "duration", 30*time.Second, "Measurement phase duration")
	rootCmd.Flags().DurationVar(&warmup, "warmup", 5*time.Second, "Warmup phase duration (stats discarded)")
	rootCmd.Flags().IntVar(&concurrency, "concurrency", 8, "Total I/O goroutines across all tracks")
	rootCmd.Flags().StringVar(&objectPrefix, "object-prefix", "", "Prefix prepended to all object names")
	rootCmd.Flags().StringVar(&outputPath, "output-path", "", "Directory for result files (default: cwd)")
	rootCmd.Flags().StringVar(&outputFormat, "output-format", "yaml", "Result format: yaml|tsv|both")
	rootCmd.Flags().StringVar(&keyFile, "key-file", "", "Path to service account key JSON (default: ADC)")
	rootCmd.Flags().StringVar(&customEndpoint, "endpoint", "", "Custom GCS endpoint (for testing proxies)")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate config and print planned workload without connecting to GCS")
	rootCmd.Flags().StringVar(&mode, "mode", "", "Override execution mode: 'benchmark' (time-bounded, default) or 'prepare' (write all objects once, then exit)")
	rootCmd.Flags().IntVar(&workerID, "worker-id", 0, "0-based worker index in a distributed run (used with --num-workers)")
	rootCmd.Flags().IntVar(&numWorkers, "num-workers", 1, "Total number of workers in a distributed run; partitions prepare-mode writes")
	rootCmd.Flags().Int64Var(&startAt, "start-at", 0, "Unix epoch timestamp to sleep until before starting (synchronized multi-worker start)")
	rootCmd.Flags().StringVar(&rapidMode, "rapid-mode", "", "RAPID/zonal bucket handling: auto (detect via GetStorageLayout), on (force bidi-gRPC), off (HTTP/2 only)")
	rootCmd.Flags().CountVarP(&verbosity, "verbose", "v", "Increase log verbosity: -v=INFO, -vv=DEBUG, -vvv=TRACE")

	return rootCmd
}

const benchmarkLongDescription = `gcs-bench is a standalone GCS I/O benchmark tool that measures
real network latency distributions without requiring a FUSE mount.

It uses Google's Go storage client (gRPC or HTTP/2) to issue reads and writes
directly against GCS, recording time-to-first-byte (TTFB) and total latency
in HDR histograms. Results are reported as accurate percentiles (p50/p90/p95/
p99/p999/max) — never averaged across histogram buckets.

All parameters — including the target bucket — are specified in a YAML config
file. CLI flags can override individual config file values.

Example:
  gcs-bench bench --config bench.yaml
  gcs-bench bench --config bench.yaml --dry-run
  gcs-bench bench --config bench.yaml --duration 120s --output-format both

See examples/benchmark-configs/ for complete YAML config examples.`

// printDryRun validates the resolved config and prints a human-readable
// description of what the benchmark would do without connecting to GCS.
func printDryRun(bucketName string, c cfg.BenchmarkConfig) error {
	fmt.Println("=== DRY RUN — no GCS connection will be made ===")
	fmt.Println()
	mode := strings.ToLower(c.Mode)
	if mode == "" {
		mode = "benchmark"
	}
	fmt.Printf("  Mode:             %s\n", mode)
	fmt.Printf("  Bucket:           gs://%s\n", bucketName)
	if c.ObjectPrefix != "" {
		fmt.Printf("  Object prefix:    %s\n", c.ObjectPrefix)
	}
	fmt.Printf("  Warmup:           %s (stats discarded)\n", c.WarmupDuration)
	fmt.Printf("  Measurement:      %s\n", c.Duration)
	fmt.Printf("  Total goroutines: %d\n", c.TotalConcurrency)
	outPath := c.OutputPath
	if outPath == "" {
		outPath = "<cwd>"
	}
	fmt.Printf("  Output path:      %s\n", outPath)
	fmt.Printf("  Output format:    %s\n", c.OutputFormat)
	rapidModeDisplay := strings.ToLower(c.RapidMode)
	if rapidModeDisplay == "" {
		rapidModeDisplay = "auto"
	}
	fmt.Printf("  RAPID mode:       %s\n", rapidModeDisplay)
	fmt.Printf("  Histogram range:  %d µs – %d µs (%d significant digits)\n",
		c.Histograms.MinValueMicros,
		c.Histograms.MaxValueMicros,
		c.Histograms.SignificantDigits,
	)
	fmt.Println()
	fmt.Printf("  Tracks (%d):\n", len(c.Tracks))

	// Compute total weight for goroutine distribution display.
	totalWeight := 0
	for _, t := range c.Tracks {
		totalWeight += t.Weight
	}

	for i, t := range c.Tracks {
		name := t.Name
		if name == "" {
			name = fmt.Sprintf("track-%d", i)
		}
		fmt.Printf("\n  [%d] %s\n", i+1, name)
		fmt.Printf("      op-type:        %s\n", t.OpType)
		fmt.Printf("      access-pattern: %s\n", t.AccessPattern)

		// Object size
		if t.SizeSpec != nil {
			switch strings.ToLower(t.SizeSpec.Type) {
			case "lognormal":
				fmt.Printf("      object-size:    lognormal (mean=%s, σ=%s, [%s – %s])\n",
					formatBytes(int64(t.SizeSpec.Mean)), formatBytes(int64(t.SizeSpec.StdDev)),
					formatBytes(t.SizeSpec.Min), formatBytes(t.SizeSpec.Max))
			case "fixed":
				fmt.Printf("      object-size:    %s (fixed)\n", formatBytes(t.SizeSpec.Min))
			default: // uniform
				fmt.Printf("      object-size:    %s – %s (uniform)\n",
					formatBytes(t.SizeSpec.Min), formatBytes(t.SizeSpec.Max))
			}
		} else if t.ObjectSizeMin == t.ObjectSizeMax {
			fmt.Printf("      object-size:    %s\n", formatBytes(t.ObjectSizeMin))
		} else {
			fmt.Printf("      object-size:    %s – %s\n",
				formatBytes(t.ObjectSizeMin), formatBytes(t.ObjectSizeMax))
		}

		// Read size
		if t.ReadSize > 0 {
			fmt.Printf("      read-size:      %s\n", formatBytes(t.ReadSize))
		} else {
			fmt.Printf("      read-size:      full object\n")
		}

		// Goroutine count
		goroutines := t.Concurrency
		if goroutines == 0 && totalWeight > 0 {
			goroutines = (t.Weight * c.TotalConcurrency) / totalWeight
			if goroutines < 1 {
				goroutines = 1
			}
		}
		fmt.Printf("      goroutines:     %d", goroutines)
		if t.Concurrency == 0 {
			fmt.Printf(" (weight %d / %d)", t.Weight, totalWeight)
		} else {
			fmt.Printf(" (direct override)")
		}
		fmt.Println()

		if t.DirectoryStructure != nil {
			ds := t.DirectoryStructure
			total := intPow(ds.Width, ds.Depth) * ds.FilesPerDir
			fmt.Printf("      directory-tree: width=%d, depth=%d, files-per-dir=%d → %d total objects\n",
				ds.Width, ds.Depth, ds.FilesPerDir, total)
		} else if t.ObjectCount > 0 {
			fmt.Printf("      object-count:   %d\n", t.ObjectCount)
		}
	}
	fmt.Println()
	fmt.Println("Config is valid. Re-run without --dry-run to execute.")
	return nil
}

// intPow computes base^exp for non-negative integers.
func intPow(base, exp int) int {
	result := 1
	for range exp {
		result *= base
	}
	return result
}

// formatBytes formats a byte count as a human-readable string (KiB / MiB / GiB).
func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.0f GiB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.0f MiB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0f KiB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
