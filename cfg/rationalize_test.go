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

	"github.com/stretchr/testify/assert"
)

type mockIsSet struct{}

func (*mockIsSet) IsSet(flag string) bool {
	return false
}

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
			actualErr := Rationalize(&mockIsSet{}, tc.config, []string{})

			if assert.NoError(t, actualErr) {
				assert.Equal(t, tc.expectedCustomEndpoint, tc.config.GcsConnection.CustomEndpoint)
			}
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
			assert.Error(t, Rationalize(&mockIsSet{}, tc.config, []string{}))
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

		err := Rationalize(&mockIsSet{}, &c, []string{})

		if assert.NoError(t, err) {
			assert.Equal(t, tc.expected, c.Logging.Severity)
		}
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
			actualErr := Rationalize(&mockIsSet{}, tc.config, []string{})

			if assert.NoError(t, actualErr) {
				assert.Equal(t, tc.expectedTokenURL, tc.config.GcsAuth.TokenUrl)
			}
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
			assert.Error(t, Rationalize(&mockIsSet{}, tc.config, []string{}))
		})
	}
}

// Implement the isSet interface
type flagSet map[string]bool

func (f flagSet) IsSet(key string) bool {
	return f[key]
}

func TestRationalizeMetadataCache(t *testing.T) {
	testCases := []struct {
		name                    string
		flags                   flagSet
		config                  *Config
		expectedTTLSecs         int64
		expectedNegativeTTLSecs int64
		expectedStatCacheSize   int64
	}{
		{
			name:            "new_ttl_flag_set",
			flags:           flagSet{"metadata-cache.ttl-secs": true},
			config:          &Config{MetadataCache: MetadataCacheConfig{TtlSecs: 30}},
			expectedTTLSecs: 30,
		},
		{
			name:  "old_ttl_flags_set",
			flags: flagSet{"metadata-cache.deprecated-stat-cache-ttl": true, "metadata-cache.deprecated-type-cache-ttl": true},
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
			flags:                 flagSet{"metadata-cache.stat-cache-max-size-mb": true},
			config:                &Config{MetadataCache: MetadataCacheConfig{StatCacheMaxSizeMb: 0}},
			expectedTTLSecs:       0, // Assuming no change to TtlSecs in this function
			expectedStatCacheSize: 0, // Should remain unchanged
		},
		{
			name:                  "old_stat-cache-capacity_flag_set",
			flags:                 flagSet{"metadata-cache.deprecated-stat-cache-capacity": true},
			config:                &Config{MetadataCache: MetadataCacheConfig{DeprecatedStatCacheCapacity: 1000}},
			expectedTTLSecs:       0,
			expectedStatCacheSize: 2,
		},
		{
			name:                  "no_relevant_flags_set",
			flags:                 flagSet{},
			config:                &Config{MetadataCache: MetadataCacheConfig{DeprecatedStatCacheCapacity: 50}},
			expectedTTLSecs:       0,
			expectedStatCacheSize: 1,
		},
		{
			name: "both_new_and_old_flags_set",
			flags: flagSet{
				"metadata-cache.stat-cache-max-size-mb": true,
				"stat-cache-capacity":                   true,
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
			name:  "ttl_and_stat_cache_size_set_to_-1",
			flags: flagSet{"metadata-cache.ttl-secs": true, "metadata-cache.stat-cache-max-size-mb": true},
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
			if assert.NoError(t, Rationalize(tc.flags, tc.config, []string{})) {
				assert.Equal(t, tc.expectedTTLSecs, tc.config.MetadataCache.TtlSecs)
				assert.Equal(t, tc.expectedStatCacheSize, tc.config.MetadataCache.StatCacheMaxSizeMb)
			}
		})
	}
}

func TestRationalizeMetadataCacheWithOptimization(t *testing.T) {
	testCases := []struct {
		name                    string
		flags                   flagSet
		config                  *Config
		expectedTTLSecs         int64
		expectedNegativeTTLSecs int64
		expectedStatCacheSize   int64
	}{
		{
			name:                    "negative_ttl_flag_set",
			flags:                   flagSet{"metadata-cache.negative-ttl-secs": true},
			config:                  &Config{MetadataCache: MetadataCacheConfig{NegativeTtlSecs: 44}},
			expectedNegativeTTLSecs: 44,
		},
		{
			name:            "new_ttl_flag_set",
			flags:           flagSet{"metadata-cache.ttl-secs": true},
			config:          &Config{MetadataCache: MetadataCacheConfig{TtlSecs: 30}},
			expectedTTLSecs: 30,
		},
		{
			name:  "old_ttl_flags_set",
			flags: flagSet{"metadata-cache.deprecated-stat-cache-ttl": true, "metadata-cache.deprecated-type-cache-ttl": true},
			config: &Config{
				MetadataCache: MetadataCacheConfig{
					DeprecatedStatCacheTtl: 10 * time.Second,
					DeprecatedTypeCacheTtl: 5 * time.Second,
				},
			},
			expectedTTLSecs: 5,
		},
		{
			name:  "new_and_old_ttl_flags_set",
			flags: flagSet{"metadata-cache.ttl-secs": true, "metadata-cache.deprecated-stat-cache-ttl": true, "metadata-cache.deprecated-type-cache-ttl": true},
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
			flags:                 flagSet{"metadata-cache.stat-cache-max-size-mb": true},
			config:                &Config{MetadataCache: MetadataCacheConfig{StatCacheMaxSizeMb: 100}},
			expectedTTLSecs:       0, // Assuming no change to TtlSecs in this function
			expectedStatCacheSize: 100,
		},
		{
			name:                  "old_stat-cache-capacity_flag_set",
			flags:                 flagSet{"metadata-cache.deprecated-stat-cache-capacity": true},
			config:                &Config{MetadataCache: MetadataCacheConfig{DeprecatedStatCacheCapacity: 1000}},
			expectedTTLSecs:       0,
			expectedStatCacheSize: 2,
		},
		{
			name:                  "new_and_old_stat-cache-capacity_flag_set",
			flags:                 flagSet{"metadata-cache.stat-cache-max-size-mb": true, "metadata-cache.deprecated-stat-cache-capacity": true},
			config:                &Config{MetadataCache: MetadataCacheConfig{StatCacheMaxSizeMb: 100, DeprecatedStatCacheCapacity: 1000}},
			expectedTTLSecs:       0,
			expectedStatCacheSize: 100,
		},
		{
			name:  "ttl_and_stat_cache_size_set_to_-1",
			flags: flagSet{"metadata-cache.ttl-secs": true, "metadata-cache.stat-cache-max-size-mb": true},
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
			if assert.NoError(t, Rationalize(tc.flags, tc.config, []string{"metadata-cache.negative-ttl-secs", "metadata-cache.ttl-secs", "metadata-cache.stat-cache-max-size-mb", "metadata-cache.deprecated-stat-cache-capacity", "metadata-cache.deprecated-stat-cache-ttl", "metadata-cache.deprecated-type-cache-ttl"})) {
				assert.Equal(t, tc.expectedTTLSecs, tc.config.MetadataCache.TtlSecs)
				assert.Equal(t, tc.expectedNegativeTTLSecs, tc.config.MetadataCache.NegativeTtlSecs)
				assert.Equal(t, tc.expectedStatCacheSize, tc.config.MetadataCache.StatCacheMaxSizeMb)
			}
		})
	}
}

func TestRationalize_WriteConfig(t *testing.T) {
	testCases := []struct {
		name                     string
		config                   *Config
		expectedCreateEmptyFile  bool
		expectedMaxBlocksPerFile int64
		expectedBlockSizeMB      int64
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
			actualErr := Rationalize(&mockIsSet{}, tc.config, []string{})

			if assert.NoError(t, actualErr) {
				assert.Equal(t, tc.expectedCreateEmptyFile, tc.config.Write.CreateEmptyFile)
				assert.Equal(t, tc.expectedMaxBlocksPerFile, tc.config.Write.MaxBlocksPerFile)
				assert.Equal(t, tc.expectedBlockSizeMB, tc.config.Write.BlockSizeMb)
			}
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if assert.NoError(t, Rationalize(&mockIsSet{}, tc.config, []string{})) {
				assert.Equal(t, tc.expected, tc.config.Metrics.CloudMetricsExportIntervalSecs)
			}
		})
	}
}

func TestRationalize_ParallelDownloadsConfig(t *testing.T) {
	testCases := []struct {
		name                      string
		flags                     flagSet
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
			// it just means flag is SET by the user
			flags: flagSet{"file-cache.enable-parallel-downloads": true},
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
			err := Rationalize(tc.flags, tc.config, []string{})

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedParallelDownloads, tc.config.FileCache.EnableParallelDownloads)
			}
		})
	}
}

func TestRationalize_FileCacheAndBufferedReadConflict(t *testing.T) {
	testCases := []struct {
		name                       string
		flags                      flagSet
		config                     *Config
		expectedEnableBufferedRead bool
		expectWarning              bool
	}{
		{
			name:  "file cache and buffered read enabled (user set)",
			flags: flagSet{"read.enable-buffered-read": true},
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
			name:  "file cache enabled, buffered read enabled (default)",
			flags: flagSet{},
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
			name:  "file cache disabled, buffered read enabled",
			flags: flagSet{"read.enable-buffered-read": true},
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
			flags:                      flagSet{},
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

			err := Rationalize(tc.flags, tc.config, []string{})

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedEnableBufferedRead, tc.config.Read.EnableBufferedRead)
				logOutput := buf.String()
				if tc.expectWarning {
					assert.True(t, strings.Contains(logOutput, "Warning: File Cache and Buffered Read features are mutually exclusive. Disabling Buffered Read in favor of File Cache."))
				} else {
					assert.False(t, strings.Contains(logOutput, "Warning: File Cache and Buffered Read features are mutually exclusive. Disabling Buffered Read in favor of File Cache."))
				}
			}
		})
	}
}
