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
)

const (
	// maxBackgroundRapidLimit configures the upper limit for max-background kernel fuse
	// setting for rapid buckets. This is more than sufficient to saturate the 200 Gbps network bandwidth
	// on a single VM. Revise the numbers if you plan to support higher bandwidth VMs.
	maxBackgroundRapidLimit = 192

	// maxBackgroundNonRapidLimit configures the default max-background kernel fuse
	// setting for non-rapid (regional and multi-regional) buckets.
	maxBackgroundNonRapidLimit = 96

	// maxReadAheadRapidKb configures the default max-read-ahead-kb setting for rapid buckets (16 MiB).
	maxReadAheadRapidKb = 16 * 1024

	// maxReadAheadNonRapidKb configures the default max-read-ahead-kb setting for non-rapid buckets (128 MiB).
	maxReadAheadNonRapidKb = 128 * 1024

	// fuseMaxRequestSizeRapidKb configures the target maximum FUSE request size in KiB (1 MiB)
	// for rapid buckets when calculating max_pages based on kernel page size (currently used for read requests).
	fuseMaxRequestSizeRapidKb = 1024

	// fuseMaxRequestSizeNonRapidKb configures the target maximum FUSE request size in KiB (16 MiB)
	// for non-rapid buckets when calculating max_pages based on kernel page size (currently used for read requests).
	fuseMaxRequestSizeNonRapidKb = 16 * 1024
)

var kernelPageSize int = os.Getpagesize()

func (bt BucketType) DefaultFuseMaxRequestSizeKb() int {
	if !bt.IsRapid() {
		return fuseMaxRequestSizeNonRapidKb
	}
	return fuseMaxRequestSizeRapidKb
}

// MaxPagesForRequestSizeKb converts a request size in KiB to the number of kernel pages.
func MaxPagesForRequestSizeKb(requestSizeKb int64) int {
	if requestSizeKb <= 0 {
		return 0
	}
	return int((requestSizeKb * 1024) / int64(kernelPageSize))
}

func (bt BucketType) DefaultMaxBackground() int {
	limit := maxBackgroundNonRapidLimit
	if bt.IsRapid() {
		limit = maxBackgroundRapidLimit
	}
	return min(max(12, 2*runtime.NumCPU()), limit)
}

func (bt BucketType) DefaultCongestionThreshold() int {
	return (3 * bt.DefaultMaxBackground()) / 4
}

func (bt BucketType) DefaultMaxReadAheadKb() int {
	if !bt.IsRapid() {
		return maxReadAheadNonRapidKb
	}
	return maxReadAheadRapidKb
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

// IsRapid returns true for rapid bucket types (zonal and pirlo).
func (bt BucketType) IsRapid() bool {
	return bt == BucketTypeZonal || bt == BucketTypePirlo
}
