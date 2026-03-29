// Copyright 2015 Google LLC
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

// A fuse file system for Google Cloud Storage buckets.
//
// Usage:
//
//	gcsfuse [flags] bucket mount_point
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/cmd"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/perf"
)

func logPanic() {
	// Detect if panic happens in main go routine.
	a := recover()
	if a != nil {
		logger.Fatal("Panic: %v", a)
	}
}

// Don't remove the comment below. It's used to generate config.go file
// which is used for flag and config file parsing.
// Refer https://go.dev/blog/generate for details.
//
//go:generate go run -C tools/config-gen . --paramsFile=../../cfg/params.yaml --outDir=../../cfg --templateDir=templates
//go:generate go run -C tools/metrics-gen . --input=../../metrics/metrics.yaml --outDir=../../metrics
const gcsBenchUsage = `gcs-bench — GCS I/O benchmark tool

USAGE:
  gcs-bench bench          [--config <file>] [flags]   Run a storage benchmark
  gcs-bench benchmark      [--config <file>] [flags]   Alias for 'bench'
  gcs-bench merge-results  <worker0.yaml> ...          Merge distributed results

QUICK START:
  # Validate a config without touching GCS
  gcs-bench bench --config examples/benchmark-configs/unet3d-like.yaml --dry-run

  # Run a benchmark
  gcs-bench bench --config examples/benchmark-configs/unet3d-like.yaml

  # Show bench-specific flags
  gcs-bench bench --help

SUBCOMMAND FLAGS:
  --config          Path to benchmark YAML config file (required)
  --duration        Measurement phase length (default: 30s)
  --warmup          Warm-up period, stats discarded (default: 5s)
  --concurrency     Total I/O goroutines across all tracks (default: 8)
  --object-prefix   Prefix added to every object path
  --output-path     Directory for result files (default: cwd)
  --output-format   yaml | tsv | both  (default: yaml)
  --mode            benchmark (default) | prepare
  --dry-run         Validate config and print plan; no GCS connection
  --key-file        Service account key JSON (default: Application Default Credentials)
  --worker-id       0-based worker index for distributed runs
  --num-workers     Total workers for distributed runs
  --start-at        Unix epoch: sleep until this time before starting

DISTRIBUTED RUNS:
  gcs-bench bench --config bench.yaml --worker-id 0 --num-workers 4 --start-at <epoch>
  gcs-bench merge-results worker-0/bench-*.yaml worker-1/bench-*.yaml ...

For full documentation see docs/bench-user-guide.md
`

func main() {
	// Common configuration for all commands
	defer logPanic()
	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	// Set up profiling handlers.
	go perf.HandleCPUProfileSignals()
	go perf.HandleMemoryProfileSignals()

	// Show friendly usage when called with no arguments, or with a bare
	// --help / -h that does not correspond to a recognised subcommand.
	if len(os.Args) == 1 {
		fmt.Print(gcsBenchUsage)
		os.Exit(0)
	}
	if len(os.Args) == 2 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Print(gcsBenchUsage)
		os.Exit(0)
	}

	// Dispatch to the benchmark engine or merge-results subcommand.
	// Scan past any leading flags (e.g. -v, -vv, --flag value) so that
	//   gcs-bench -v bench --config ...    works as well as
	//   gcs-bench bench -v --config ...
argScan:
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "bench", "benchmark":
			os.Args = append(os.Args[:i], os.Args[i+1:]...)
			cmd.ExecuteBenchmarkCmd()
			return
		case "merge-results":
			os.Args = append(os.Args[:i], os.Args[i+1:]...)
			cmd.ExecuteMergeResultsCmd()
			return
		default:
			if !strings.HasPrefix(arg, "-") {
				// First positional arg is not a known subcommand — fall through to mount.
				break argScan
			}
		}
	}

	cmd.ExecuteMountCmd()
}
