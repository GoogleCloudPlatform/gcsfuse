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
	"path"
	"path/filepath"
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

// systemTotalMemoryKbFrom parses MemTotal in KiB from a Linux /proc/meminfo-formatted file.
func systemTotalMemoryKbFrom(meminfoPath string) int64 {
	data, err := os.ReadFile(meminfoPath)
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

// parseCgroupPath returns the cleaned relative cgroup directory path for either
// cgroup v2 ("0::<path>") or a cgroup v1 controller (e.g. "memory").
func parseCgroupPath(procCgroupPath, controller string) string {
	data, err := os.ReadFile(procCgroupPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 3)
		if len(parts) != 3 {
			continue
		}
		controllers, rawPath := parts[1], parts[2]
		matched := false
		if controller == "" {
			matched = parts[0] == "0" && controllers == ""
		} else {
			for _, c := range strings.Split(controllers, ",") {
				if c == controller {
					matched = true
					break
				}
			}
		}
		if matched {
			return strings.TrimPrefix(path.Clean("/"+rawPath), "/")
		}
	}
	return ""
}

func minFiniteCgroupLimitKb(root, relPath, filename string) int64 {
	if root == "" {
		return 0
	}
	rootClean := filepath.Clean(root)
	cur := rootClean
	if relPath != "" && relPath != "." {
		cur = filepath.Join(rootClean, filepath.FromSlash(relPath))
	}
	var minKb int64
	for {
		data, err := os.ReadFile(filepath.Join(cur, filename))
		if err == nil {
			s := strings.TrimSpace(string(data))
			if bytes, pErr := strconv.ParseInt(s, 10, 64); pErr == nil && bytes > 0 {
				if kb := bytes / 1024; kb > 0 && (minKb == 0 || kb < minKb) {
					minKb = kb
				}
			}
		}
		if cur == rootClean {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur || !strings.HasPrefix(parent, rootClean) {
			break
		}
		cur = parent
	}
	return minKb
}

func cgroupMemoryLimitKb(procCgroupPath, cgroupV2Root, cgroupV1Root string) int64 {
	if limitKb := minFiniteCgroupLimitKb(cgroupV2Root, parseCgroupPath(procCgroupPath, ""), "memory.max"); limitKb > 0 {
		return limitKb
	}
	if limitKb := minFiniteCgroupLimitKb(cgroupV1Root, parseCgroupPath(procCgroupPath, "memory"), "memory.limit_in_bytes"); limitKb > 0 {
		return limitKb
	}
	return 0
}

func effectiveTotalMemoryKbFrom(meminfoPath, procCgroupPath, cgroupV2Root, cgroupV1Root string) int64 {
	memTotalKb := systemTotalMemoryKbFrom(meminfoPath)
	cgroupLimitKb := cgroupMemoryLimitKb(procCgroupPath, cgroupV2Root, cgroupV1Root)
	if cgroupLimitKb > 0 && (memTotalKb == 0 || cgroupLimitKb < memTotalKb) {
		return cgroupLimitKb
	}
	return memTotalKb
}

// effectiveTotalMemoryKb returns the effective memory limit in KiB on Linux,
// taking the minimum of /proc/meminfo MemTotal and any configured cgroup v1/v2
// memory limit, or 0 on non-Linux systems.
func effectiveTotalMemoryKb() int64 {
	if runtime.GOOS != "linux" {
		return 0
	}
	return effectiveTotalMemoryKbFrom("/proc/meminfo", "/proc/self/cgroup", "/sys/fs/cgroup", "/sys/fs/cgroup/memory")
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
	return sc.defaultMaxBackgroundFor(runtime.NumCPU(), effectiveTotalMemoryKb())
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
