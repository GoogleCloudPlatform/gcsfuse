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
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DefaultMaxBackground(t *testing.T) {
	cpus := runtime.NumCPU()
	memKb := effectiveTotalMemoryKb()
	assert.Equal(t, StorageClassRapid.defaultMaxBackgroundFor(cpus, memKb), StorageClassRapid.DefaultMaxBackground())
	assert.Equal(t, StorageClassStandard.defaultMaxBackgroundFor(cpus, memKb), StorageClassStandard.DefaultMaxBackground())
}

func Test_DefaultMaxBackgroundFor(t *testing.T) {
	const mibKb = int64(1024)
	const gibKb = int64(1024 * 1024)

	testCases := []struct {
		name         string
		cpus         int
		totalMemKb   int64
		wantRapid    int
		wantStandard int
	}{
		{
			name:         "1 vCPU / 512 MiB RAM (floor at 12)",
			cpus:         1,
			totalMemKb:   512 * mibKb,
			wantRapid:    12,
			wantStandard: 12,
		},
		{
			name:         "2 vCPU / 2 GiB RAM",
			cpus:         2,
			totalMemKb:   2 * gibKb,
			wantRapid:    16,
			wantStandard: 16,
		},
		{
			name:         "4 vCPU / 4 GiB RAM",
			cpus:         4,
			totalMemKb:   4 * gibKb,
			wantRapid:    32,
			wantStandard: 32,
		},
		{
			name:         "4 vCPU / 1 GiB RAM (memory-capped non-rapid)",
			cpus:         4,
			totalMemKb:   1 * gibKb,
			wantRapid:    32,
			wantStandard: 16,
		},
		{
			name:         "4 vCPU / 512 MiB RAM (memory floor at 12)",
			cpus:         4,
			totalMemKb:   512 * mibKb,
			wantRapid:    32,
			wantStandard: 12,
		},
		{
			name:         "24 vCPU / 64 GiB RAM (rapid reaches cap)",
			cpus:         24,
			totalMemKb:   64 * gibKb,
			wantRapid:    rapidMaxBackground,
			wantStandard: 192,
		},
		{
			name:         "32 vCPU / 64 GiB RAM (high-throughput VM)",
			cpus:         32,
			totalMemKb:   64 * gibKb,
			wantRapid:    rapidMaxBackground,
			wantStandard: nonRapidMaxBackground,
		},
		{
			name:         "64 vCPU / 256 GiB RAM (high-throughput VM)",
			cpus:         64,
			totalMemKb:   256 * gibKb,
			wantRapid:    rapidMaxBackground,
			wantStandard: nonRapidMaxBackground,
		},
		{
			name:         "2 vCPU / unknown memory (0 KiB)",
			cpus:         2,
			totalMemKb:   0,
			wantRapid:    16,
			wantStandard: 16,
		},
		{
			name:         "32 vCPU / unknown memory (0 KiB)",
			cpus:         32,
			totalMemKb:   0,
			wantRapid:    rapidMaxBackground,
			wantStandard: nonRapidMaxBackground,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantRapid, StorageClassRapid.defaultMaxBackgroundFor(tc.cpus, tc.totalMemKb))
			assert.Equal(t, tc.wantStandard, StorageClassStandard.defaultMaxBackgroundFor(tc.cpus, tc.totalMemKb))
		})
	}
}

func Test_EffectiveTotalMemoryKbFrom_CgroupLimits(t *testing.T) {
	writeFile := func(t *testing.T, p, content string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}

	t.Run("CgroupV2NestedTakesTighterParentLimit", func(t *testing.T) {
		dir := t.TempDir()
		meminfo := filepath.Join(dir, "meminfo")
		procCgroup := filepath.Join(dir, "cgroup")
		cgV2 := filepath.Join(dir, "sys/fs/cgroup")
		cgV1 := filepath.Join(dir, "sys/fs/cgroup/memory")

		writeFile(t, meminfo, "MemTotal:       134217728 kB\n")
		writeFile(t, procCgroup, "0::/kubepods.slice/pod123/container\n")
		writeFile(t, filepath.Join(cgV2, "kubepods.slice/pod123/memory.max"), "2147483648\n") // 2 GiB
		writeFile(t, filepath.Join(cgV2, "kubepods.slice/pod123/container/memory.max"), "max\n")

		got := effectiveTotalMemoryKbFrom(meminfo, procCgroup, cgV2, cgV1)
		assert.Equal(t, int64(2*1024*1024), got)
		// On a 32-vCPU host with 2 GiB cgroup limit, non-rapid (16 MiB req) caps at 2 GiB/4/16 MiB = 32.
		assert.Equal(t, 32, StorageClassStandard.defaultMaxBackgroundFor(32, got))
		assert.Equal(t, rapidMaxBackground, StorageClassRapid.defaultMaxBackgroundFor(32, got))
	})

	t.Run("CgroupV2MaxFallsBackToHostMemTotal", func(t *testing.T) {
		dir := t.TempDir()
		meminfo := filepath.Join(dir, "meminfo")
		procCgroup := filepath.Join(dir, "cgroup")
		cgV2 := filepath.Join(dir, "sys/fs/cgroup")
		cgV1 := filepath.Join(dir, "sys/fs/cgroup/memory")

		writeFile(t, meminfo, "MemTotal:       67108864 kB\n")
		writeFile(t, procCgroup, "0::/\n")
		writeFile(t, filepath.Join(cgV2, "memory.max"), "max\n")

		got := effectiveTotalMemoryKbFrom(meminfo, procCgroup, cgV2, cgV1)
		assert.Equal(t, int64(67108864), got)
	})

	t.Run("CgroupV1FiniteVsUnlimitedSentinel", func(t *testing.T) {
		dir := t.TempDir()
		meminfo := filepath.Join(dir, "meminfo")
		procCgroup := filepath.Join(dir, "cgroup")
		cgV2 := filepath.Join(dir, "sys/fs/cgroup")
		cgV1 := filepath.Join(dir, "sys/fs/cgroup/memory")

		writeFile(t, meminfo, "MemTotal:       67108864 kB\n")
		writeFile(t, procCgroup, "4:memory:/docker/abc\n")
		// Tight finite limit (1 GiB).
		writeFile(t, filepath.Join(cgV1, "docker/abc/memory.limit_in_bytes"), "1073741824\n")
		assert.Equal(t, int64(1024*1024), effectiveTotalMemoryKbFrom(meminfo, procCgroup, cgV2, cgV1))

		// Kernel unlimited sentinel (> MemTotal) falls back to host MemTotal.
		writeFile(t, filepath.Join(cgV1, "docker/abc/memory.limit_in_bytes"), "9223372036854771712\n")
		assert.Equal(t, int64(67108864), effectiveTotalMemoryKbFrom(meminfo, procCgroup, cgV2, cgV1))
	})

	t.Run("NonContainerizedFallback", func(t *testing.T) {
		dir := t.TempDir()
		meminfo := filepath.Join(dir, "meminfo")
		writeFile(t, meminfo, "MemTotal:       33554432 kB\n")
		assert.Equal(t, int64(33554432), effectiveTotalMemoryKbFrom(
			meminfo,
			filepath.Join(dir, "missing_cgroup"),
			filepath.Join(dir, "missing_v2"),
			filepath.Join(dir, "missing_v1"),
		))
	})
}

func Test_DefaultCongestionThreshold(t *testing.T) {
	assert.Equal(t, (3*StorageClassRapid.DefaultMaxBackground())/4, StorageClassRapid.DefaultCongestionThreshold())
	assert.Equal(t, (3*StorageClassStandard.DefaultMaxBackground())/4, StorageClassStandard.DefaultCongestionThreshold())
}

func Test_DefaultFuseMaxRequestSizeKb(t *testing.T) {
	assert.Equal(t, rapidMaxRequestSizeKb, StorageClassRapid.DefaultFuseMaxRequestSizeKb())
	assert.Equal(t, nonRapidMaxRequestSizeKb, StorageClassStandard.DefaultFuseMaxRequestSizeKb())
}

func Test_DefaultMaxReadAheadKb(t *testing.T) {
	assert.Equal(t, rapidMaxReadAheadKb, StorageClassRapid.DefaultMaxReadAheadKb())
	assert.Equal(t, nonRapidMaxReadAheadKb, StorageClassStandard.DefaultMaxReadAheadKb())
}

func Test_DefaultMaxParallelDownloads(t *testing.T) {
	assert.GreaterOrEqual(t, DefaultMaxParallelDownloads(), 16)
}

func TestIsFileCacheEnabled(t *testing.T) {
	testCases := []struct {
		name                       string
		config                     *Config
		expectedIsFileCacheEnabled bool
	}{
		{
			name: "Config with CacheDir set and cache size non zero.",
			config: &Config{
				CacheDir: "/tmp/folder/",
				FileCache: FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedIsFileCacheEnabled: true,
		},
		{
			name:                       "Empty Config.",
			config:                     &Config{},
			expectedIsFileCacheEnabled: false,
		},
		{
			name: "Config with CacheDir unset",
			config: &Config{
				CacheDir: "",
				FileCache: FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedIsFileCacheEnabled: false,
		},
		{
			name: "Config with CacheDir set and cache size zero.",
			config: &Config{
				CacheDir: "//tmp//folder//",
				FileCache: FileCacheConfig{
					MaxSizeMb: 0,
				},
			},
			expectedIsFileCacheEnabled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedIsFileCacheEnabled, IsFileCacheEnabled(tc.config))
		})
	}

}

func TestIsParallelDownloadsEnabled(t *testing.T) {
	testCases := []struct {
		name                               string
		config                             *Config
		expectedIsParallelDownloadsEnabled bool
	}{
		{
			name: "Config with file-cache enabled",
			config: &Config{
				CacheDir: "/tmp/folder/",
				FileCache: FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedIsParallelDownloadsEnabled: false,
		},
		{
			name:                               "Empty Config.",
			config:                             &Config{},
			expectedIsParallelDownloadsEnabled: false,
		},
		{
			name: "Config with file-cache disabled but enable parallel downloads is set.",
			config: &Config{
				CacheDir: "",
				FileCache: FileCacheConfig{
					MaxSizeMb:               -1,
					EnableParallelDownloads: true,
				},
			},
			expectedIsParallelDownloadsEnabled: false,
		},
		{
			name: "Config with file-cache and parallel downloads enabled.",
			config: &Config{
				CacheDir: "//tmp//folder//",
				FileCache: FileCacheConfig{
					MaxSizeMb:               -1,
					EnableParallelDownloads: true,
				},
			},
			expectedIsParallelDownloadsEnabled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedIsParallelDownloadsEnabled, IsParallelDownloadsEnabled(tc.config))
		})
	}

}

func Test_ListCacheTtlSecsToDuration(t *testing.T) {
	var testCases = []struct {
		testName         string
		ttlInSecs        int64
		expectedDuration time.Duration
	}{
		{"-1", -1, maxSupportedTTL},
		{"0", 0, time.Duration(0)},
		{"max_supported_positive", 9223372036, maxSupportedTTL},
		{"positive", 1, time.Second},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.expectedDuration, ListCacheTTLSecsToDuration(tt.ttlInSecs))
		})
	}
}

func Test_ListCacheTtlSecsToDuration_InvalidCall(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	// Calling with invalid argument to trigger panic.
	ListCacheTTLSecsToDuration(-3)
}

func TestIsTracingEnabled(t *testing.T) {
	t.Parallel()
	var testCases = []struct {
		testName           string
		traceMode          []string
		traceSamplingRatio float64
		expected           bool
	}{
		{"empty_non_zero_sampling_ratio", []string{}, 0.5, false},
		{"non_empty_non_zero_sampling_ratio", []string{"gcpexporter"}, 0.4, true},
		{"non_empty_zero_sampling_ratio", []string{"gcpexporter"}, 0.0, false},
		{"empty_zero_samping_ratio", []string{}, 0.0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, IsTracingEnabled(&Config{Trace: TraceConfig{
				Exporters:     tc.traceMode,
				SamplingRatio: tc.traceSamplingRatio,
			}}))
		})
	}
}

func TestIsMetricsEnabled(t *testing.T) {
	t.Parallel()
	var testCases = []struct {
		testName string
		m        *MetricsConfig
		enabled  bool
	}{
		{"cloud_metrics_export_interval_set", &MetricsConfig{CloudMetricsExportIntervalSecs: 100}, true},
		{"prom_port_set", &MetricsConfig{PrometheusPort: 10000}, true},
		{"none_set", &MetricsConfig{CloudMetricsExportIntervalSecs: 0, PrometheusPort: 0}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.enabled, IsMetricsEnabled(tc.m))
		})
	}
}

func TestIsGKEEnvironment(t *testing.T) {
	t.Parallel()
	var testCases = []struct {
		testName   string
		mountPoint string
		expected   bool
	}{
		{"non-GKE", "/usr/local/mount-folder", false},
		{"GKE mountpoint", "/dev/fd/", true},
		{"GKE /dev/fd/N", "/dev/fd/8", true},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, IsGKEEnvironment(tc.mountPoint))
		})
	}
}

func TestGetBucketType(t *testing.T) {
	tests := []struct {
		name         string
		hierarchical bool
		zonal        bool
		rcu          bool
		expected     BucketType
	}{
		{
			name:         "Zonal and Hierarchical (zonal takes priority)",
			hierarchical: true,
			zonal:        true,
			expected:     BucketTypeZonal,
		},
		{
			name:         "Rapid Cache Ultra and Hierarchical (Rapid Cache Ultra takes priority)",
			hierarchical: true,
			rcu:          true,
			expected:     BucketTypeRCU,
		},
		{
			name:         "Hierarchical bucket",
			hierarchical: true,
			zonal:        false,
			expected:     BucketTypeHierarchical,
		},
		{
			name:         "Flat bucket",
			hierarchical: false,
			zonal:        false,
			expected:     BucketTypeFlat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBucketType(tt.hierarchical, tt.zonal, tt.rcu)
			if result != tt.expected {
				t.Errorf("GetBucketType(%v, %v, %v) = %v; want %v",
					tt.hierarchical, tt.zonal, tt.rcu, result, tt.expected)
			}
		})
	}
}

func Test_MaxPagesForRequestSizeKb(t *testing.T) {
	tests := []struct {
		name          string
		requestSizeKb int
		expected      int
	}{
		{"1_KiB", 1, 1},
		{"exact_multiple_16_KiB", 16, ((16 * 1024) + kernelPageSize - 1) / kernelPageSize},
		{"non_exact_multiple", 5, ((5 * 1024) + kernelPageSize - 1) / kernelPageSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaxPagesForRequestSizeKb(tt.requestSizeKb))
		})
	}
}

func Test_StorageClass(t *testing.T) {
	assert.Equal(t, StorageClassRapid, BucketTypeZonal.StorageClass())
	assert.Equal(t, StorageClassRapid, BucketTypeRCU.StorageClass())
	assert.Equal(t, StorageClassStandard, BucketTypeFlat.StorageClass())
	assert.Equal(t, StorageClassStandard, BucketTypeHierarchical.StorageClass())
}
