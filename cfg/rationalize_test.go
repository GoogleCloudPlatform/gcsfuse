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
	"math"
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
			actualErr := Rationalize(&mockIsSet{}, tc.config)

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
			assert.Error(t, Rationalize(&mockIsSet{}, tc.config))
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

		err := Rationalize(&mockIsSet{}, &c)

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
			actualErr := Rationalize(&mockIsSet{}, tc.config)

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
			assert.Error(t, Rationalize(&mockIsSet{}, tc.config))
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
		name                  string
		flags                 flagSet
		config                *Config
		expectedTTLSecs       int64
		expectedStatCacheSize int64
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
					StatCacheMaxSizeMb: -1,
				},
			},
			expectedTTLSecs:       math.MaxInt64 / int64(time.Second), // Max supported ttl in seconds.
			expectedStatCacheSize: math.MaxUint64 >> 20,               // Max supported cache size in MiB.
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if assert.NoError(t, Rationalize(tc.flags, tc.config)) {
				assert.Equal(t, tc.expectedTTLSecs, tc.config.MetadataCache.TtlSecs)
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
	}{
		{
			name: "valid_config_streaming_writes_enabled",
			config: &Config{
				Write: WriteConfig{
					BlockSizeMb:                       10,
					CreateEmptyFile:                   true,
					ExperimentalEnableStreamingWrites: true,
					GlobalMaxBlocks:                   -1,
					MaxBlocksPerFile:                  -1,
				},
			},
			expectedCreateEmptyFile:  false,
			expectedMaxBlocksPerFile: math.MaxInt64,
		},
		{
			name: "valid_config_global_max_blocks_less_than_blocks_per_file",
			config: &Config{
				Write: WriteConfig{
					BlockSizeMb:                       10,
					CreateEmptyFile:                   true,
					ExperimentalEnableStreamingWrites: true,
					GlobalMaxBlocks:                   10,
					MaxBlocksPerFile:                  20,
				},
			},
			expectedCreateEmptyFile:  false,
			expectedMaxBlocksPerFile: 10,
		},
		{
			name: "valid_config_global_max_blocks_more_than_blocks_per_file",
			config: &Config{
				Write: WriteConfig{
					BlockSizeMb:                       10,
					CreateEmptyFile:                   true,
					ExperimentalEnableStreamingWrites: true,
					GlobalMaxBlocks:                   20,
					MaxBlocksPerFile:                  10,
				},
			},
			expectedCreateEmptyFile:  false,
			expectedMaxBlocksPerFile: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := Rationalize(&mockIsSet{}, tc.config)

			if assert.NoError(t, actualErr) {
				assert.Equal(t, tc.expectedCreateEmptyFile, tc.config.Write.CreateEmptyFile)
				assert.Equal(t, tc.expectedMaxBlocksPerFile, tc.config.Write.MaxBlocksPerFile)
			}
		})
	}
}
