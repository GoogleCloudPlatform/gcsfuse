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
	"strconv"
	"strings"
	"time"
)

// Rapid bucket defaults (zonal, Rapid Cache Ultra)
const (
	rapidMaxBackground    = 192
	rapidMaxReadAheadKb   = 16 * 1024 // 16 MiB
	rapidMaxRequestSizeKb = 1024      // 1 MiB
)

// Non-rapid bucket defaults (flat, hierarchical)
const (
	nonRapidMaxBackground    = 256
	nonRapidMaxReadAheadKb   = 128 * 1024 // 128 MiB
	nonRapidMaxRequestSizeKb = 16 * 1024  // 16 MiB
)

var kernelPageSize int = os.Getpagesize()

func (sc StorageClass) DefaultFuseMaxRequestSizeKb() int {
	if sc != StorageClassRapid {
		return nonRapidMaxRequestSizeKb
	}
	return rapidMaxRequestSizeKb
}

// MaxPagesForRequestSizeKb converts a request size in KiB to the number of kernel pages.
func MaxPagesForRequestSizeKb(requestSizeKb int) int {
	return ((requestSizeKb * 1024) + kernelPageSize - 1) / kernelPageSize
}

// systemTotalMemoryKb returns the total system memory in KiB from /proc/meminfo on Linux,
// or 0 on non-Linux systems or if /proc/meminfo cannot be read or parsed.
func systemTotalMemoryKb() int64 {
	if runtime.GOOS != "linux" {
		return 0
	}
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "MemTotal:"))
		if len(fields) == 0 {
			return 0
		}
		kb, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil || kb <= 0 {
			return 0
		}
		return kb
	}
	return 0
}

func (sc StorageClass) defaultMaxBackgroundFor(cpus int, totalMemKb int64) int {
	limit := nonRapidMaxBackground
	if sc == StorageClassRapid {
		limit = rapidMaxBackground
	}
	scaled := min(max(12, 8*cpus), limit)
	if totalMemKb > 0 {
		memLimit := max(12, int((totalMemKb/4)/int64(sc.DefaultFuseMaxRequestSizeKb())))
		scaled = min(scaled, memLimit)
	}
	return scaled
}

func (sc StorageClass) DefaultMaxBackground() int {
	return sc.defaultMaxBackgroundFor(runtime.NumCPU(), systemTotalMemoryKb())
}

func (sc StorageClass) DefaultCongestionThreshold() int {
	return (3 * sc.DefaultMaxBackground()) / 4
}

func (sc StorageClass) DefaultMaxReadAheadKb() int {
	if sc != StorageClassRapid {
		return nonRapidMaxReadAheadKb
	}
	return rapidMaxReadAheadKb
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
// The priority order is: Zonal > Rapid Cache Ultra > Hierarchical > Flat.
// This is used to determine bucket-specific optimizations for kernel configs.
// TODO (b/472597952): Make BucketType in bucket_handle.go as enum
func GetBucketType(hierarchical, zonal, rcu bool) BucketType {
	if zonal {
		return BucketTypeZonal
	}
	if rcu {
		return BucketTypeRCU
	}
	if hierarchical {
		return BucketTypeHierarchical
	}
	return BucketTypeFlat
}

// IsRapid returns true for rapid bucket types (zonal and Rapid Cache Ultra).
func (bt BucketType) IsRapid() bool {
	return bt == BucketTypeZonal || bt == BucketTypeRCU
}

// StorageClass returns the storage performance class corresponding to the bucket type.
func (bt BucketType) StorageClass() StorageClass {
	if bt.IsRapid() {
		return StorageClassRapid
	}
	return StorageClassStandard
}
