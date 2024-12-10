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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
		testName  string
		traceMode string
		expected  bool
	}{
		{"empty", "", false},
		{"not_empty", "gcptrace", true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, IsTracingEnabled(&Config{Monitoring: MonitoringConfig{
				ExperimentalTracingMode: tc.traceMode,
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
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.enabled, IsMetricsEnabled(tc.m))
		})
	}
}
