// Copyright 2024 Google LLC
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

package cfg

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	// maxBackgroundLimit configures the upper limit for max-background kernel fuse
	// setting. This is more than sufficient to saturate the 200 Gbps network bandwidth
	// on a single VM. Revise the numbers if you plan to support higher bandwidth VMs.
	maxBackgroundLimit = 192
)

var kernelPageSize int = os.Getpagesize()

func DefaultFuseMaxPagesLimit() int {
	// The default limit is calculated to achieve a 1 MiB maximum request size
	// based on the kernel page size.
	return (1024 * 1024) / kernelPageSize
}

func DefaultMaxBackground() int {
	return min(max(12, 2*runtime.NumCPU()), maxBackgroundLimit)
}

func DefaultMaxBackgroundForNonRapid() int {
	return 96
}

func DefaultCongestionThreshold() int {
	// 75 % of DefaultMaxBackground
	return (3 * DefaultMaxBackground()) / 4
}

func DefaultCongestionThresholdForNonRapid() int {
	// 75 % of DefaultMaxBackgroundForNonRapid
	return (3 * DefaultMaxBackgroundForNonRapid()) / 4
}

func DefaultMaxReadAheadKbForNonRapid() int {
	// 128 Kib
	return 128 * 1024
}

func DefaultMaxParallelDownloads() int {
	return max(16, 2*runtime.NumCPU())
}

func IsFileCacheEnabled(mountConfig *Config) bool {
	return mountConfig.FileCache.MaxSizeMb != 0 && string(mountConfig.CacheDir) != ""
}

func IsParallelDownloadsEnabled(mountConfig *Config) bool {
	return IsFileCacheEnabled(mountConfig) && mountConfig.FileCache.EnableParallelDownloads
}

// IsTracingEnabled returns true if tracing is enabled.
func IsTracingEnabled(mountConfig *Config) bool {
	return len(mountConfig.Trace.Exporters) > 0 && mountConfig.Trace.SamplingRatio > 0
}

// ListCacheTTLSecsToDuration converts TTL in seconds to time.Duration.
func ListCacheTTLSecsToDuration(secs int64) time.Duration {
	err := isTTLInSecsValid(secs)
	if err != nil {
		panic(fmt.Sprintf("invalid argument: %d, %v", secs, err))
	}

	if secs == -1 {
		return maxSupportedTTL
	}

	return time.Duration(secs * int64(time.Second))
}

// IsMetricsEnabled returns true if metrics are enabled.
func IsMetricsEnabled(c *MetricsConfig) bool {
	return c.CloudMetricsExportIntervalSecs > 0 || c.PrometheusPort > 0
}

// IsGKEEnvironment returns true for /dev/fd/N mountpoints.
func IsGKEEnvironment(mountPoint string) bool {
	return strings.HasPrefix(mountPoint, "/dev/fd/")
}

// GetBucketType converts BucketType boolean flags to a BucketType enum.
// The priority order is: Zonal > Pirlo > Hierarchical > Flat.
// This is used to determine bucket-specific optimizations for kernel configs.
// TODO (b/472597952): Make BucketType in bucket_handle.go as enum
func GetBucketType(hierarchical, zonal, pirlo bool) BucketType {
	if zonal {
		return BucketTypeZonal
	}
	if pirlo {
		return BucketTypePirlo
	}
	if hierarchical {
		return BucketTypeHierarchical
	}
	return BucketTypeFlat
}

func DefaultFuseMaxPagesLimitForNonRapid() int {
	// 16 MiB maximum request size based on the kernel page size.
	return (16 * 1024 * 1024) / kernelPageSize
}

// IsNonRapid returns true for regional and multi-regional bucket types (flat and hierarchical).
func (bt BucketType) IsNonRapid() bool {
	return bt == BucketTypeFlat || bt == BucketTypeHierarchical
}

func (c *Config) shouldEnableKernelReaderForNonRapidBuckets(v *viper.Viper, input *OptimizationInput) bool {
	if input == nil || !input.BucketType.IsNonRapid() || v == nil {
		return false
	}

	// Do not enable if file cache or buffered read are enabled explicitly by user
	// or if kernel does not support increasing max pages limit beyond 1MiB.
	isFileCacheSet := v.IsSet("cache-dir")
	isBufferedReadSet := v.IsSet("read.enable-buffered-read")
	return /*c.Profile == "aiml-serving" &&*/ !isFileCacheSet && !isBufferedReadSet && input.HasCapability(CapabilityFuseMaxPagesLimit)
}

func (c *Config) applyRegionalKernelReaderOptimizations(v *viper.Viper, input *OptimizationInput, optimizedFlags map[string]OptimizationResult) {
	if c.DisableAutoconfig || v == nil || input == nil || !input.BucketType.IsNonRapid() {
		return
	}

	// 1. Automatically enable kernel reader for non-rapid buckets if appropriate.
	if !v.IsSet("file-system.enable-kernel-reader") && !optimizedFlags["file-system.enable-kernel-reader"].Optimized {
		if c.shouldEnableKernelReaderForNonRapidBuckets(v, input) {
			c.FileSystem.EnableKernelReader = true
			optimizedFlags["file-system.enable-kernel-reader"] = OptimizationResult{
				Optimized:  true,
				FinalValue: true,
			}
		}
	}

	// 2. If kernel reader is enabled, apply parameter optimizations cleanly using a table-driven loop.
	if c.FileSystem.EnableKernelReader {
		type flagOpt struct {
			name   string
			val    int64
			setter func(int64)
		}
		opts := []flagOpt{
			{"file-system.max-read-ahead-kb", int64(DefaultMaxReadAheadKbForNonRapid()), func(v int64) { c.FileSystem.MaxReadAheadKb = v }},
			{"file-system.max-background", int64(DefaultMaxBackgroundForNonRapid()), func(v int64) { c.FileSystem.MaxBackground = v }},
			{"file-system.congestion-threshold", int64(DefaultCongestionThresholdForNonRapid()), func(v int64) { c.FileSystem.CongestionThreshold = v }},
			{"file-system.fuse-max-pages-limit", int64(DefaultFuseMaxPagesLimitForNonRapid()), func(v int64) { c.FileSystem.FuseMaxPagesLimit = v }},
		}
		for _, opt := range opts {
			if !v.IsSet(opt.name) && !optimizedFlags[opt.name].Optimized {
				opt.setter(opt.val)
				optimizedFlags[opt.name] = OptimizationResult{
					Optimized:  true,
					FinalValue: opt.val,
				}
			}
		}
	}
}
