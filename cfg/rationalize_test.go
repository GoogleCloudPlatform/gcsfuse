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
	"bytes"
	"log"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRationalizeCustomEndpointSuccessful(t *testing.T) {
	testCases := []struct {
		name                   string
		config                 *Config
		expectedCustomEndpoint string
	}{
		{
			name: "Valid Config where input and expected custom endpoint match.",
			config: &Config{
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "https://bing.com/search?q=dotnet",
				},
			},
			expectedCustomEndpoint: "https://bing.com/search?q=dotnet",
		},
		{
			name: "Valid Config where input and expected custom endpoint differ.",
			config: &Config{
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "https://j@ne:password@google.com",
				},
			},
			expectedCustomEndpoint: "https://j%40ne:password@google.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := Rationalize(viper.New(), tc.config, []string{})

			require.NoError(t, actualErr)
			assert.Equal(t, tc.expectedCustomEndpoint, tc.config.GcsConnection.CustomEndpoint)
		})
	}
}

func TestRationalize_GcsRetriesConfig(t *testing.T) {
	testCases := []struct {
		name                     string
		config                   *Config
		expectedMaxRetryAttempts int
	}{
		{
			name: "max-retry-attempts is 0",
			config: &Config{
				GcsRetries: GcsRetriesConfig{
					MaxRetryAttempts: 0,
				},
			},
			expectedMaxRetryAttempts: math.MaxInt,
		},
		{
			name: "max-retry-attempts is not 0",
			config: &Config{
				GcsRetries: GcsRetriesConfig{
					MaxRetryAttempts: 10,
				},
			},
			expectedMaxRetryAttempts: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Rationalize(viper.New(), tc.config, []string{})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedMaxRetryAttempts, int(tc.config.GcsRetries.MaxRetryAttempts))
		})
	}
}

func TestRationalize_ReadConfig(t *testing.T) {
	testCases := []struct {
		name                    string
		config                  *Config
		expectedGlobalMaxBlocks int64
	}{
		{
			name: "global-max-blocks is -1",
			config: &Config{
				Read: ReadConfig{
					GlobalMaxBlocks: -1,
				},
			},
			expectedGlobalMaxBlocks: math.MaxInt32,
		},
		{
			name: "global-max-blocks is not -1",
			config: &Config{
				Read: ReadConfig{
					GlobalMaxBlocks: 100,
				},
			},
			expectedGlobalMaxBlocks: 100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Rationalize(viper.New(), tc.config, []string{})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedGlobalMaxBlocks, tc.config.Read.GlobalMaxBlocks)
		})
	}
}

func TestRationalizeCustomEndpointUnsuccessful(t *testing.T) {
	testCases := []struct {
		name   string
		config *Config
	}{
		{
			name: "Invalid Config",
			config: &Config{
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "a_b://abc",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, Rationalize(viper.New(), tc.config, []string{}))
		})
	}
}

func TestLoggingSeverityRationalization(t *testing.T) {
	testcases := []struct {
		name       string
		cfgSev     string
		debugFuse  bool
		debugGCS   bool
		debugMutex bool
		expected   LogSeverity
	}{
		{
			name:       "no debug flags",
			cfgSev:     "INFO",
			debugFuse:  false,
			debugGCS:   false,
			debugMutex: false,
			expected:   "INFO",
		},
		{
			name:       "debugFuse true",
			cfgSev:     "INFO",
			debugFuse:  true,
			debugGCS:   false,
			debugMutex: false,
			expected:   "TRACE",
		},
		{
			name:       "debugGCS true",
			cfgSev:     "INFO",
			debugFuse:  false,
			debugGCS:   true,
			debugMutex: false,
			expected:   "TRACE",
		},
		{
			name:       "debugMutex true",
			cfgSev:     "INFO",
			debugFuse:  false,
			debugGCS:   false,
			debugMutex: true,
			expected:   "TRACE",
		},
		{
			name:       "multiple debug flags true",
			cfgSev:     "INFO",
			debugFuse:  true,
			debugGCS:   false,
			debugMutex: true,
			expected:   "TRACE",
		},
	}

	for _, tc := range testcases {
		c := Config{
			Logging: LoggingConfig{
				Severity: LogSeverity(tc.cfgSev),
			},
			Debug: DebugConfig{
				Fuse:     tc.debugFuse,
				Gcs:      tc.debugGCS,
				LogMutex: tc.debugMutex,
			},
		}

		err := Rationalize(viper.New(), &c, []string{})

		require.NoError(t, err)
		assert.Equal(t, tc.expected, c.Logging.Severity)
	}
}

func TestRationalize_TokenURLSuccessful(t *testing.T) {
	testCases := []struct {
		name             string
		config           *Config
		expectedTokenURL string
	}{
		{
			name: "Valid Config where input and expected token url match.",
			config: &Config{
				GcsAuth: GcsAuthConfig{
					TokenUrl: "https://bing.com/search?q=dotnet",
				},
			},
			expectedTokenURL: "https://bing.com/search?q=dotnet",
		},
		{
			name: "Valid Config where input and expected token url differ.",
			config: &Config{
				GcsAuth: GcsAuthConfig{
					TokenUrl: "https://j@ne:password@google.com",
				},
			},
			expectedTokenURL: "https://j%40ne:password@google.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := Rationalize(viper.New(), tc.config, []string{})

			require.NoError(t, actualErr)
			assert.Equal(t, tc.expectedTokenURL, tc.config.GcsAuth.TokenUrl)
		})
	}
}

func TestRationalize_TokenURLUnsuccessful(t *testing.T) {
	testCases := []struct {
		name   string
		config *Config
	}{
		{
			name: "Invalid Config",
			config: &Config{
				GcsAuth: GcsAuthConfig{
					TokenUrl: "a_b://abc",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, Rationalize(viper.New(), tc.config, []string{}))
		})
	}
}

func TestRationalizeMetadataCache(t *testing.T) {
	testCases := []struct {
		name                    string
		userSetFlags            map[string]any
		config                  *Config
		expectedTTLSecs         int64
		expectedNegativeTTLSecs int64
		expectedStatCacheSize   int64
	}{
		{
			name:            "new_ttl_flag_set",
			userSetFlags:    map[string]any{"metadata-cache.ttl-secs": 30},
			config:          &Config{MetadataCache: MetadataCacheConfig{TtlSecs: 30}},
			expectedTTLSecs: 30,
		},
		{
			name:         "old_ttl_flags_set",
			userSetFlags: map[string]any{"metadata-cache.deprecated-stat-cache-ttl": 10 * time.Second, "metadata-cache.deprecated-type-cache-ttl": 5 * time.Second},
			config: &Config{
				MetadataCache: MetadataCacheConfig{
					DeprecatedStatCacheTtl: 10 * time.Second,
					DeprecatedTypeCacheTtl: 5 * time.Second,
				},
			},
			expectedTTLSecs: 5,
		},
		{
			name:                  "new_stat-cache-size-mb_flag_set",
			userSetFlags:          map[string]any{"metadata-cache.stat-cache-max-size-mb": 0},
			config:                &Config{MetadataCache: MetadataCacheConfig{StatCacheMaxSizeMb: 0}},
			expectedTTLSecs:       0, // Assuming no change to TtlSecs in this function
			expectedStatCacheSize: 0, // Should remain unchanged
		},
		{
			name:                  "old_stat-cache-capacity_flag_set",
			userSetFlags:          map[string]any{"metadata-cache.deprecated-stat-cache-capacity": 1000},
			config:                &Config{MetadataCache: MetadataCacheConfig{DeprecatedStatCacheCapacity: 1000}},
			expectedTTLSecs:       0,
			expectedStatCacheSize: 2,
		},
		{
			name:                  "no_relevant_flags_set",
			userSetFlags:          map[string]any{},
			config:                &Config{MetadataCache: MetadataCacheConfig{DeprecatedStatCacheCapacity: 50}},
			expectedTTLSecs:       0,
			expectedStatCacheSize: 1,
		},
		{
			name: "both_new_and_old_flags_set",
			userSetFlags: map[string]any{
				"metadata-cache.stat-cache-max-size-mb": 100,
				"stat-cache-capacity":                   50,
			},
			config: &Config{
				MetadataCache: MetadataCacheConfig{
					StatCacheMaxSizeMb:          100,
					DeprecatedStatCacheCapacity: 50,
				},
			},
			expectedTTLSecs:       0,
			expectedStatCacheSize: 100,
		},
		{
			name:         "ttl_and_stat_cache_size_set_to_-1",
			userSetFlags: map[string]any{"metadata-cache.ttl-secs": -1, "metadata-cache.stat-cache-max-size-mb": -1},
			config: &Config{
				MetadataCache: MetadataCacheConfig{
					TtlSecs:            -1,
					NegativeTtlSecs:    -1,
					StatCacheMaxSizeMb: -1,
				},
			},
			expectedTTLSecs:         math.MaxInt64 / int64(time.Second), // Max supported ttl in seconds.
			expectedNegativeTTLSecs: math.MaxInt64 / int64(time.Second), // Max supported ttl in seconds.
			expectedStatCacheSize:   math.MaxUint64 >> 20,               // Max supported cache size in MiB.
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := viper.New()
			for key, val := range tc.userSetFlags {
				v.Set(key, val)
			}

			err := Rationalize(v, tc.config, []string{})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedTTLSecs, tc.config.MetadataCache.TtlSecs)
			assert.Equal(t, tc.expectedStatCacheSize, tc.config.MetadataCache.StatCacheMaxSizeMb)
		})
	}
}

func TestRationalizeMetadataCacheWithOptimization(t *testing.T) {
	testCases := []struct {
		name                    string
		userSetFlags            map[string]any
		config                  *Config
		expectedTTLSecs         int64
		expectedNegativeTTLSecs int64
		expectedStatCacheSize   int64
	}{
		{
			name:                    "negative_ttl_flag_set",
			userSetFlags:            map[string]any{"metadata-cache.negative-ttl-secs": 44},
			config:                  &Config{MetadataCache: MetadataCacheConfig{NegativeTtlSecs: 44}},
			expectedNegativeTTLSecs: 44,
		},
		{
			name:            "new_ttl_flag_set",
			userSetFlags:    map[string]any{"metadata-cache.ttl-secs": 30},
			config:          &Config{MetadataCache: MetadataCacheConfig{TtlSecs: 30}},
			expectedTTLSecs: 30,
		},
		{
			name:         "old_ttl_flags_set",
			userSetFlags: map[string]any{"metadata-cache.deprecated-stat-cache-ttl": 10 * time.Second, "metadata-cache.deprecated-type-cache-ttl": 5 * time.Second},
			config: &Config{
				MetadataCache: MetadataCacheConfig{
					DeprecatedStatCacheTtl: 10 * time.Second,
					DeprecatedTypeCacheTtl: 5 * time.Second,
				},
			},
			expectedTTLSecs: 5,
		},
		{
			name:         "new_and_old_ttl_flags_set",
			userSetFlags: map[string]any{"metadata-cache.ttl-secs": 30, "metadata-cache.deprecated-stat-cache-ttl": 10 * time.Second, "metadata-cache.deprecated-type-cache-ttl": 5 * time.Second},
			config: &Config{
				MetadataCache: MetadataCacheConfig{
					TtlSecs:                30,
					DeprecatedStatCacheTtl: 10 * time.Second,
					DeprecatedTypeCacheTtl: 5 * time.Second,
				},
			},
			expectedTTLSecs: 30,
		},
		{
			name:                  "new_stat-cache-size-mb_flag_set",
			userSetFlags:          map[string]any{"metadata-cache.stat-cache-max-size-mb": 100},
			config:                &Config{MetadataCache: MetadataCacheConfig{StatCacheMaxSizeMb: 100}},
			expectedTTLSecs:       0, // Assuming no change to TtlSecs in this function
			expectedStatCacheSize: 100,
		},
		{
			name:                  "old_stat-cache-capacity_flag_set",
			userSetFlags:          map[string]any{"metadata-cache.deprecated-stat-cache-capacity": 1000},
			config:                &Config{MetadataCache: MetadataCacheConfig{DeprecatedStatCacheCapacity: 1000}},
			expectedTTLSecs:       0,
			expectedStatCacheSize: 2,
		},
		{
			name:                  "new_and_old_stat-cache-capacity_flag_set",
			userSetFlags:          map[string]any{"metadata-cache.stat-cache-max-size-mb": 100, "metadata-cache.deprecated-stat-cache-capacity": 1000},
			config:                &Config{MetadataCache: MetadataCacheConfig{StatCacheMaxSizeMb: 100, DeprecatedStatCacheCapacity: 1000}},
			expectedTTLSecs:       0,
			expectedStatCacheSize: 100,
		},
		{
			name:         "ttl_and_stat_cache_size_set_to_-1",
			userSetFlags: map[string]any{"metadata-cache.ttl-secs": -1, "metadata-cache.stat-cache-max-size-mb": -1},
			config: &Config{
				MetadataCache: MetadataCacheConfig{
					TtlSecs:            -1,
					NegativeTtlSecs:    -1,
					StatCacheMaxSizeMb: -1,
				},
			},
			expectedTTLSecs:         math.MaxInt64 / int64(time.Second), // Max supported ttl in seconds.
			expectedNegativeTTLSecs: math.MaxInt64 / int64(time.Second), // Max supported ttl in seconds.
			expectedStatCacheSize:   math.MaxUint64 >> 20,               // Max supported cache size in MiB.
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := viper.New()
			for key, val := range tc.userSetFlags {
				v.Set(key, val)
			}

			err := Rationalize(v, tc.config, []string{"metadata-cache.negative-ttl-secs", "metadata-cache.ttl-secs", "metadata-cache.stat-cache-max-size-mb", "metadata-cache.deprecated-stat-cache-capacity", "metadata-cache.deprecated-stat-cache-ttl", "metadata-cache.deprecated-type-cache-ttl"})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedTTLSecs, tc.config.MetadataCache.TtlSecs)
			assert.Equal(t, tc.expectedNegativeTTLSecs, tc.config.MetadataCache.NegativeTtlSecs)
			assert.Equal(t, tc.expectedStatCacheSize, tc.config.MetadataCache.StatCacheMaxSizeMb)
		})
	}
}

func TestRationalize_WriteConfig(t *testing.T) {
	testCases := []struct {
		name                     string
		config                   *Config
		expectedCreateEmptyFile  bool
		expectedMaxBlocksPerFile int64
		expectedBlockSizeMB      float64
	}{
		{
			name: "valid_config_streaming_writes_enabled",
			config: &Config{
				Write: WriteConfig{
					BlockSizeMb:           10,
					CreateEmptyFile:       true,
					EnableStreamingWrites: true,
					GlobalMaxBlocks:       -1,
					MaxBlocksPerFile:      -1,
				},
			},
			expectedCreateEmptyFile:  false,
			expectedMaxBlocksPerFile: math.MaxInt16,
			expectedBlockSizeMB:      10,
		},
		{
			name: "valid_config_global_max_blocks_less_than_blocks_per_file",
			config: &Config{
				Write: WriteConfig{
					BlockSizeMb:           5,
					CreateEmptyFile:       true,
					EnableStreamingWrites: true,
					GlobalMaxBlocks:       10,
					MaxBlocksPerFile:      20,
				},
			},
			expectedCreateEmptyFile:  false,
			expectedMaxBlocksPerFile: 20,
			expectedBlockSizeMB:      5,
		},
		{
			name: "valid_config_global_max_blocks_more_than_blocks_per_file",
			config: &Config{
				Write: WriteConfig{
					BlockSizeMb:           64,
					CreateEmptyFile:       true,
					EnableStreamingWrites: true,
					GlobalMaxBlocks:       20,
					MaxBlocksPerFile:      10,
				},
			},
			expectedCreateEmptyFile:  false,
			expectedMaxBlocksPerFile: 10,
			expectedBlockSizeMB:      64,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := Rationalize(viper.New(), tc.config, []string{})

			require.NoError(t, actualErr)
			assert.Equal(t, tc.expectedCreateEmptyFile, tc.config.Write.CreateEmptyFile)
			assert.Equal(t, tc.expectedMaxBlocksPerFile, tc.config.Write.MaxBlocksPerFile)
			assert.Equal(t, tc.expectedBlockSizeMB, tc.config.Write.BlockSizeMb)
		})
	}
}

func TestRationalizeMetricsConfig(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		config   *Config
		expected int64
	}{
		{
			name: "both_0",
			config: &Config{
				Metrics: MetricsConfig{
					StackdriverExportInterval:      0,
					CloudMetricsExportIntervalSecs: 0,
				},
			},
			expected: 0,
		},
		{
			name: "stackdriver_set",
			config: &Config{
				Metrics: MetricsConfig{
					StackdriverExportInterval:      2 * time.Hour,
					CloudMetricsExportIntervalSecs: 0,
				},
			},
			expected: 7200,
		},
		{
			name: "cloud_metrics_set",
			config: &Config{
				Metrics: MetricsConfig{
					StackdriverExportInterval:      0,
					CloudMetricsExportIntervalSecs: 10,
				},
			},
			expected: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := Rationalize(viper.New(), tc.config, []string{})

			require.NoError(t, err)
			assert.Equal(t, tc.expected, tc.config.Metrics.CloudMetricsExportIntervalSecs)
		})
	}
}

func TestRationalizeTraceConfig(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		config   *Config
		expected []string
	}{
		{
			name: "all_params_filled",
			config: &Config{
				Trace: TraceConfig{
					Exporters:     []string{"stdout"},
					ProjectId:     "test-gcp-project",
					SamplingRatio: 0.2,
				},
			},
			expected: []string{"stdout"},
		},
		{
			name: "missing_project_id",
			config: &Config{
				Trace: TraceConfig{
					Exporters:     []string{"stdout"},
					ProjectId:     "test-gcp-project",
					SamplingRatio: 0.2,
				},
			},
			expected: []string{"stdout"},
		},
		{
			name: "multiple_tracing_modes",
			config: &Config{
				Trace: TraceConfig{
					Exporters:     []string{"stdout ", " gcpexporter "},
					ProjectId:     "test-gcp-project",
					SamplingRatio: 0.2,
				},
			},
			expected: []string{"stdout", "gcpexporter"},
		},
		{
			name: "multiple_tracing_modes",
			config: &Config{
				Trace: TraceConfig{
					Exporters:     []string{"STDout ", " GcPExpoRter "},
					ProjectId:     "test-gcp-project",
					SamplingRatio: 0.2,
				},
			},
			expected: []string{"stdout", "gcpexporter"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := Rationalize(viper.New(), tc.config, []string{})

			require.NoError(t, err)
			assert.Equal(t, tc.expected, tc.config.Trace.Exporters)
		})
	}
}

func TestRationalize_ParallelDownloadsConfig(t *testing.T) {
	testCases := []struct {
		name                      string
		userSetFlags              map[string]any
		config                    *Config
		expectedParallelDownloads bool
	}{
		{
			name: "valid_config_file_cache_enabled",
			config: &Config{
				CacheDir: ResolvedPath("/some-path"),
				FileCache: FileCacheConfig{
					MaxSizeMb: 500,
				},
			},
			expectedParallelDownloads: true,
		},
		{
			name:                      "valid_config_file_cache_disabled",
			config:                    &Config{},
			expectedParallelDownloads: false,
		},
		{
			name: "valid_config_cache_dir_not_set_and_max_size_mb_set",
			config: &Config{
				FileCache: FileCacheConfig{
					MaxSizeMb: 500,
				},
			},
			expectedParallelDownloads: false,
		},
		{
			name: "valid_config_parallel_download_explicit_false",
			// flagset here is representing viper config, value true is not actual value of the flag
			// it just means flag is SET by the user.
			userSetFlags: map[string]any{"file-cache.enable-parallel-downloads": true},
			config: &Config{
				CacheDir: ResolvedPath("/some-path"),
				FileCache: FileCacheConfig{
					MaxSizeMb: 500,
				},
			},
			expectedParallelDownloads: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := viper.New()
			for key, val := range tc.userSetFlags {
				v.Set(key, val)
			}

			err := Rationalize(v, tc.config, []string{})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedParallelDownloads, tc.config.FileCache.EnableParallelDownloads)
		})
	}
}

func TestRationalize_FileCacheAndBufferedReadConflict(t *testing.T) {
	testCases := []struct {
		name                       string
		userSetFlags               map[string]any
		config                     *Config
		expectedEnableBufferedRead bool
		expectWarning              bool
	}{
		{
			name:         "file cache and buffered read enabled (user set)",
			userSetFlags: map[string]any{"read.enable-buffered-read": true},
			config: &Config{
				CacheDir: "/some/path",
				FileCache: FileCacheConfig{
					MaxSizeMb: -1,
				},
				Read: ReadConfig{
					EnableBufferedRead: true,
				},
			},
			expectedEnableBufferedRead: false,
			expectWarning:              true,
		},
		{
			name:         "file cache enabled, buffered read enabled (default)",
			userSetFlags: map[string]any{},
			config: &Config{
				CacheDir: "/some/path",
				FileCache: FileCacheConfig{
					MaxSizeMb: -1,
				},
				Read: ReadConfig{
					EnableBufferedRead: true,
				},
			},
			expectedEnableBufferedRead: false,
			expectWarning:              false,
		},
		{
			name:         "file cache disabled, buffered read enabled",
			userSetFlags: map[string]any{"read.enable-buffered-read": true},
			config: &Config{
				Read: ReadConfig{
					EnableBufferedRead: true,
				},
			},
			expectedEnableBufferedRead: true,
			expectWarning:              false,
		},
		{
			name:                       "both disabled",
			userSetFlags:               map[string]any{},
			config:                     &Config{},
			expectedEnableBufferedRead: false,
			expectWarning:              false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture log output.
			var buf bytes.Buffer
			log.SetOutput(&buf)
			// Restore original logger output after test.
			defer log.SetOutput(os.Stderr)

			v := viper.New()
			for key, val := range tc.userSetFlags {
				v.Set(key, val)
			}

			err := Rationalize(v, tc.config, []string{})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedEnableBufferedRead, tc.config.Read.EnableBufferedRead)
			logOutput := buf.String()
			if tc.expectWarning {
				assert.True(t, strings.Contains(logOutput, "Warning: File Cache and Buffered Read features are mutually exclusive. Disabling Buffered Read in favor of File Cache."))
			} else {
				assert.False(t, strings.Contains(logOutput, "Warning: File Cache and Buffered Read features are mutually exclusive. Disabling Buffered Read in favor of File Cache."))
			}
		})
	}
}

func TestResolveLoggingConfig(t *testing.T) {
	testCases := []struct {
		name              string
		config            *Config
		expectedLogFormat string
	}{
		{
			name: "valid_log_format_json",
			config: &Config{
				Logging: LoggingConfig{
					Format: "json",
				},
			},
			expectedLogFormat: "json",
		},
		{
			name: "valid_log_format_text",
			config: &Config{
				Logging: LoggingConfig{
					Format: "text",
				},
			},
			expectedLogFormat: "text",
		},
		{
			name: "valid_case_insensitive_log_format",
			config: &Config{
				Logging: LoggingConfig{
					Format: "TEXT",
				},
			},
			expectedLogFormat: "text",
		},
		{
			name: "invalid_log_format",
			config: &Config{
				Logging: LoggingConfig{
					Format: "INVALID",
				},
			},
			expectedLogFormat: "json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolveLoggingConfig(tc.config)

			assert.Equal(t, tc.expectedLogFormat, tc.config.Logging.Format)
		})
	}
}

func TestRationalize_MetadataCacheConfig(t *testing.T) {
	testCases := []struct {
		name                                 string
		config                               *Config
		expectedMetadataPrefetchCount        int64
		expectedConcurrentMetadataPrefetches int64
	}{
		{
			name: "valid_config_metadata_prefetch_count_set_to_-1",
			config: &Config{
				MetadataCache: MetadataCacheConfig{
					MetadataPrefetchEntriesLimit: -1,
					MetadataPrefetchMaxWorkers:   5,
				},
			},
			expectedMetadataPrefetchCount:        math.MaxInt64,
			expectedConcurrentMetadataPrefetches: 5,
		},
		{
			name: "valid_config_concurrent_prefetches_set_to_-1",
			config: &Config{
				MetadataCache: MetadataCacheConfig{
					MetadataPrefetchEntriesLimit: 8,
					MetadataPrefetchMaxWorkers:   -1,
				},
			},
			expectedMetadataPrefetchCount:        8,
			expectedConcurrentMetadataPrefetches: math.MaxInt64,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := Rationalize(viper.New(), tc.config, []string{})

			require.NoError(t, actualErr)
			assert.Equal(t, tc.expectedConcurrentMetadataPrefetches, tc.config.MetadataCache.MetadataPrefetchMaxWorkers)
			assert.Equal(t, tc.expectedMetadataPrefetchCount, tc.config.MetadataCache.MetadataPrefetchEntriesLimit)
		})
	}
}

func TestResolveOnlyDir(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Clean relative path", "foo/bar", "foo/bar"},
		{"Trailing slash", "foo/bar/", "foo/bar"},
		{"Leading slash", "/foo/bar", "foo/bar"},
		{"Leading and trailing slashes", "/foo/bar/", "foo/bar"},
		{"Relative parent segment", "foo/../bar/", "bar"},
		{"Root slash", "/", ""},
		{"Root dot", "/.", ""},
		{"Parent directory", "..", ""},
		{"Root parent directory", "/..", ""},
		{"Current directory prefix", "./foo", "foo"},
		{"Parent directory prefix", "../foo", "foo"},
		{"Empty string", "", ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{OnlyDir: tc.input}
			resolveOnlyDir(c)
			assert.Equal(t, tc.expected, c.OnlyDir)
		})
	}
}

func TestResolveKernelReadAhead(t *testing.T) {
	testCases := []struct {
		name                string
		userSetFlags        map[string]any
		nilViper            bool
		enableKernelReader  bool
		initialReadAheadKb  int64
		requestSizeKb       int64
		expectedReadAheadKb int64
	}{
		{
			name:                "enabled_unset_large_request_size",
			enableKernelReader:  true,
			initialReadAheadKb:  131072, // 128 MiB (regional default)
			requestSizeKb:       261120, // 255 MiB
			expectedReadAheadKb: 261120,
		},
		{
			name:                "enabled_startup_zero_adjusted",
			enableKernelReader:  true,
			initialReadAheadKb:  0,
			requestSizeKb:       261120,
			expectedReadAheadKb: 261120,
		},
		{
			name:                "enabled_startup_default_request_size",
			enableKernelReader:  true,
			initialReadAheadKb:  0,
			requestSizeKb:       1024,
			expectedReadAheadKb: 1024,
		},
		{
			name:                "enabled_explicit_read_ahead_lower",
			userSetFlags:        map[string]any{"file-system.max-read-ahead-kb": 1024},
			enableKernelReader:  true,
			initialReadAheadKb:  1024,
			requestSizeKb:       261120,
			expectedReadAheadKb: 1024,
		},
		{
			name:                "enabled_explicit_zero_preserved",
			userSetFlags:        map[string]any{"file-system.max-read-ahead-kb": 0},
			enableKernelReader:  true,
			initialReadAheadKb:  0,
			requestSizeKb:       261120,
			expectedReadAheadKb: 0,
		},
		{
			name:                "enabled_explicit_read_ahead_higher",
			userSetFlags:        map[string]any{"file-system.max-read-ahead-kb": 524288},
			enableKernelReader:  true,
			initialReadAheadKb:  524288,
			requestSizeKb:       16384,
			expectedReadAheadKb: 524288,
		},
		{
			name:                "enabled_request_size_equal_read_ahead",
			enableKernelReader:  true,
			initialReadAheadKb:  131072,
			requestSizeKb:       131072,
			expectedReadAheadKb: 131072,
		},
		{
			name:                "enabled_request_size_smaller_read_ahead",
			enableKernelReader:  true,
			initialReadAheadKb:  131072, // 128 MiB
			requestSizeKb:       16384,  // 16 MiB
			expectedReadAheadKb: 131072,
		},
		{
			name:                "enabled_explicit_small_request_size",
			enableKernelReader:  true,
			initialReadAheadKb:  131072,
			requestSizeKb:       8192,
			expectedReadAheadKb: 131072,
		},
		{
			name:                "enabled_request_size_zero",
			enableKernelReader:  true,
			initialReadAheadKb:  131072,
			requestSizeKb:       0,
			expectedReadAheadKb: 131072,
		},
		{
			name:                "disabled_large_request_size",
			enableKernelReader:  false,
			initialReadAheadKb:  131072,
			requestSizeKb:       261120,
			expectedReadAheadKb: 131072,
		},
		{
			name:                "disabled_startup_zero",
			enableKernelReader:  false,
			initialReadAheadKb:  0,
			requestSizeKb:       261120,
			expectedReadAheadKb: 0,
		},
		{
			name:                "rapid_defaults_preserved",
			enableKernelReader:  true,
			initialReadAheadKb:  16384, // 16 MiB rapid default
			requestSizeKb:       1024,  // 1 MiB rapid default
			expectedReadAheadKb: 16384,
		},
		{
			name:                "rapid_explicit_read_ahead_preserved",
			userSetFlags:        map[string]any{"file-system.max-read-ahead-kb": 4096},
			enableKernelReader:  true,
			initialReadAheadKb:  4096,
			requestSizeKb:       1024,
			expectedReadAheadKb: 4096,
		},
		{
			name:                "rapid_explicit_lower_preserved",
			userSetFlags:        map[string]any{"file-system.max-read-ahead-kb": 512},
			enableKernelReader:  true,
			initialReadAheadKb:  512,
			requestSizeKb:       1024,
			expectedReadAheadKb: 512,
		},
		{
			name:                "rapid_explicit_higher_preserved",
			userSetFlags:        map[string]any{"file-system.max-read-ahead-kb": 32768},
			enableKernelReader:  true,
			initialReadAheadKb:  32768,
			requestSizeKb:       1024,
			expectedReadAheadKb: 32768,
		},
		{
			name:                "nil_viper_auto_adjustment",
			nilViper:            true,
			enableKernelReader:  true,
			initialReadAheadKb:  131072,
			requestSizeKb:       261120,
			expectedReadAheadKb: 261120,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var v *viper.Viper
			if !tc.nilViper {
				v = viper.New()
				for key, val := range tc.userSetFlags {
					v.Set(key, val)
				}
			}
			c := &Config{
				FileSystem: FileSystemConfig{
					EnableKernelReader:   tc.enableKernelReader,
					MaxReadAheadKb:       tc.initialReadAheadKb,
					FuseMaxRequestSizeKb: tc.requestSizeKb,
				},
			}
			resolveKernelReadAhead(v, c)

			assert.Equal(t, tc.expectedReadAheadKb, c.FileSystem.MaxReadAheadKb)
		})
	}
}

func TestRationalizeKernelReadAhead(t *testing.T) {
	testCases := []struct {
		name                string
		userSetFlags        map[string]any
		enableKernelReader  bool
		initialReadAheadKb  int64
		requestSizeKb       int64
		expectedReadAheadKb int64
	}{
		{
			name:                "adjusts_large_request_size",
			enableKernelReader:  true,
			initialReadAheadKb:  131072,
			requestSizeKb:       261120,
			expectedReadAheadKb: 261120,
		},
		{
			name:                "preserves_explicit_read_ahead",
			userSetFlags:        map[string]any{"file-system.max-read-ahead-kb": 2048},
			enableKernelReader:  true,
			initialReadAheadKb:  2048,
			requestSizeKb:       261120,
			expectedReadAheadKb: 2048,
		},
		{
			name:                "preserves_default_read_ahead",
			enableKernelReader:  true,
			initialReadAheadKb:  131072,
			requestSizeKb:       16384,
			expectedReadAheadKb: 131072,
		},
		{
			name:                "disabled_reader_preserves_read_ahead",
			enableKernelReader:  false,
			initialReadAheadKb:  131072,
			requestSizeKb:       261120,
			expectedReadAheadKb: 131072,
		},
		{
			name:                "preserves_rapid_defaults",
			enableKernelReader:  true,
			initialReadAheadKb:  16384,
			requestSizeKb:       1024,
			expectedReadAheadKb: 16384,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := viper.New()
			for key, val := range tc.userSetFlags {
				v.Set(key, val)
			}
			c := &Config{
				FileSystem: FileSystemConfig{
					EnableKernelReader:   tc.enableKernelReader,
					MaxReadAheadKb:       tc.initialReadAheadKb,
					FuseMaxRequestSizeKb: tc.requestSizeKb,
				},
			}
			err := Rationalize(v, c, []string{})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedReadAheadKb, c.FileSystem.MaxReadAheadKb)
		})
	}
}

func TestRationalizeWithBucketOptimization(t *testing.T) {
	testCases := []struct {
		name                string
		bucketType          BucketType
		userSetFlags        map[string]any
		setupConfig         func(c *Config)
		expectedReadAheadKb int64
		expectedRequestKb   int64
		expectedKernelRd    bool
	}{
		{
			name:       "flat_bucket_large_request_size",
			bucketType: BucketTypeFlat,
			userSetFlags: map[string]any{
				"file-system.enable-kernel-reader":     true,
				"file-system.fuse-max-request-size-kb": 261120,
			},
			setupConfig: func(c *Config) {
				c.FileSystem.EnableKernelReader = true
				c.FileSystem.FuseMaxRequestSizeKb = 261120
			},
			expectedReadAheadKb: 261120,
			expectedRequestKb:   261120,
			expectedKernelRd:    true,
		},
		{
			name:       "flat_bucket_default_request_size",
			bucketType: BucketTypeFlat,
			userSetFlags: map[string]any{
				"file-system.enable-kernel-reader": true,
			},
			setupConfig: func(c *Config) {
				c.FileSystem.EnableKernelReader = true
			},
			expectedReadAheadKb: 131072, // 128 MiB
			expectedRequestKb:   16384,  // 16 MiB
			expectedKernelRd:    true,
		},
		{
			name:       "flat_bucket_explicit_read_ahead",
			bucketType: BucketTypeFlat,
			userSetFlags: map[string]any{
				"file-system.enable-kernel-reader":     true,
				"file-system.fuse-max-request-size-kb": 261120,
				"file-system.max-read-ahead-kb":        4096,
			},
			setupConfig: func(c *Config) {
				c.FileSystem.EnableKernelReader = true
				c.FileSystem.FuseMaxRequestSizeKb = 261120
				c.FileSystem.MaxReadAheadKb = 4096
			},
			expectedReadAheadKb: 4096,
			expectedRequestKb:   261120,
			expectedKernelRd:    true,
		},
		{
			name:         "rapid_bucket_default_optimization",
			bucketType:   BucketTypeZonal,
			userSetFlags: map[string]any{},
			setupConfig: func(c *Config) {
				c.FileSystem.FuseMaxRequestSizeKb = int64(StorageClassRapid.DefaultFuseMaxRequestSizeKb())
			},
			expectedReadAheadKb: 16384, // 16 MiB
			expectedRequestKb:   1024,  // 1 MiB
			expectedKernelRd:    true,
		},
		{
			name:       "rapid_bucket_explicit_read_ahead",
			bucketType: BucketTypeZonal,
			userSetFlags: map[string]any{
				"file-system.max-read-ahead-kb": 8192,
			},
			setupConfig: func(c *Config) {
				c.FileSystem.MaxReadAheadKb = 8192
				c.FileSystem.FuseMaxRequestSizeKb = int64(StorageClassRapid.DefaultFuseMaxRequestSizeKb())
			},
			expectedReadAheadKb: 8192,
			expectedRequestKb:   1024,
			expectedKernelRd:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := viper.New()
			for key, val := range tc.userSetFlags {
				v.Set(key, val)
			}
			c := &Config{}
			tc.setupConfig(c)

			// Step 1: Apply bucket-type optimizations
			optimizedFlags := c.ApplyOptimizations(v, &OptimizationInput{BucketType: tc.bucketType})

			// Step 2: Rationalize
			var optFlagNames []string
			for k := range optimizedFlags {
				optFlagNames = append(optFlagNames, k)
			}
			err := Rationalize(v, c, optFlagNames)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedKernelRd, c.FileSystem.EnableKernelReader)
			assert.Equal(t, tc.expectedRequestKb, c.FileSystem.FuseMaxRequestSizeKb)
			assert.Equal(t, tc.expectedReadAheadKb, c.FileSystem.MaxReadAheadKb)
		})
	}
}
