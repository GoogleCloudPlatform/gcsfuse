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

func TestResolveMetadataCacheTTL(t *testing.T) {
	testcases := []struct {
		name string
		// Equivalent of user-setting of --stat-cache-ttl.
		statCacheTTL time.Duration
		// Equivalent of user-setting of --type-cache-ttl.
		typeCacheTTL time.Duration
		// Equivalent of user-setting of metadata-cache:ttl-secs in --config-file.
		ttlInSeconds             int64
		expectedMetadataCacheTTL time.Duration
	}{
		{
			// Most common scenario, when user doesn't set any of the TTL config parameters.
			name:                     "no_flag_or_config_set",
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: DefaultStatOrTypeCacheTTL,
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to -1.
			name:                     "metadata_cache_ttl_-1",
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             -1,
			expectedMetadataCacheTTL: time.Duration(math.MaxInt64),
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to 0.
			name:                     "metadata_cache_ttl_0",
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             0,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to a
			// positive value.
			name:                     "metadata_cache_ttl_positive",
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             30,
			expectedMetadataCacheTTL: 30 * time.Second,
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to
			// its highest supported value.
			name:         "metadata_cache_ttl_maximum",
			statCacheTTL: DefaultStatOrTypeCacheTTL,
			typeCacheTTL: DefaultStatOrTypeCacheTTL,
			ttlInSeconds: maxSupportedTTLInSeconds,

			expectedMetadataCacheTTL: time.Second * time.Duration(maxSupportedTTLInSeconds),
		},
		{
			// Scenario where user sets both the old flags and the
			// metadata-cache:ttl-secs. Here ttl-secs overrides both flags. case 1.
			name:                     "both_config_and_flags_set_1",
			statCacheTTL:             5 * time.Minute,
			typeCacheTTL:             time.Hour,
			ttlInSeconds:             10800,
			expectedMetadataCacheTTL: 10800 * time.Second,
		},
		{
			// Scenario where user sets both the old flags and the
			// metadata-cache:ttl-secs. Here ttl-secs overrides both flags. case 2.
			name:                     "both_config_and_flags_set_2",
			statCacheTTL:             5 * time.Minute,
			typeCacheTTL:             time.Hour,
			ttlInSeconds:             1800,
			expectedMetadataCacheTTL: 1800 * time.Second,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not
			// metadata-cache:ttl-secs. Case 1.
			name:                     "only_flags_set_1",
			statCacheTTL:             0,
			typeCacheTTL:             0,
			ttlInSeconds:             TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not
			// metadata-cache:ttl-secs. Case 2. Stat-cache enabled, but not type-cache.
			name:                     "only_flags_set_2",
			statCacheTTL:             time.Hour,
			typeCacheTTL:             0,
			ttlInSeconds:             TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not
			// metadata-cache:ttl-secs. Case 3. Type-cache enabled, but not stat-cache.
			name:                     "only_flags_set_3",
			statCacheTTL:             0,
			typeCacheTTL:             time.Hour,
			ttlInSeconds:             TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not
			// metadata-cache:ttl-secs. Case 4. Both Type-cache and stat-cache enabled.
			// The lower of the two TTLs is taken.
			name:                     "only_flags_set_4",
			statCacheTTL:             time.Second,
			typeCacheTTL:             time.Minute,
			ttlInSeconds:             TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: time.Second,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedMetadataCacheTTL, ResolveMetadataCacheTTL(tc.statCacheTTL, tc.typeCacheTTL, tc.ttlInSeconds))
		})
	}
}

func TestResolveStatCacheMaxSizeMB(t *testing.T) {
	testcases := []struct {
		name string
		// Equivalent of user-setting of flag --stat-cache-capacity.
		flagStatCacheCapacity int
		// Equivalent of user-setting of metadata-cache:stat-cache-max-size-mb in
		// --config-file.
		mountConfigStatCacheMaxSizeMB int64
		// Expected output
		expectedStatCacheMaxSizeMB uint64
	}{
		{
			// Most common scenario, when user doesn't set either the flag or the
			// config.
			name:                          "no_flag_or_config_set",
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: StatCacheMaxSizeMBUnsetSentinel,
			expectedStatCacheMaxSizeMB:    defaultStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and
			// sets it to -1.
			name:                          "stat_cache_size_mb_-1",
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: -1,
			expectedStatCacheMaxSizeMB:    maxSupportedStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and
			// sets it to 0.
			name:                          "stat_cache_size_mb_0",
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: 0,
			expectedStatCacheMaxSizeMB:    0,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and
			// sets it to a positive value.
			name:                          "stat_cache_size_mb_positive",
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: 100,
			expectedStatCacheMaxSizeMB:    100,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and
			// sets it to its highest user-input value.
			name:                          "stat_cache_size_mb_maximum",
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: int64(maxSupportedStatCacheMaxSizeMB),
			expectedStatCacheMaxSizeMB:    maxSupportedStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets both stat-cache-capacity and the
			// metadata-cache:stat-cache-max-size-mb. Here stat-cache-max-size-mb
			// overrides stat-cache-capacity. case 1.
			name:                          "both_stat_cache_size_mb_and_capacity_set_1",
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: 100,
			expectedStatCacheMaxSizeMB:    100,
		},
		{
			// Scenario where user sets both stat-cache-capacity and the
			// metadata-cache:stat-cache-max-size-mb. Here stat-cache-max-size-mb
			// overrides stat-cache-capacity. case 2.
			name:                          "both_stat_cache_size_mb_and_capacity_set_2",
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: -1,
			expectedStatCacheMaxSizeMB:    maxSupportedStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets both stat-cache-capacity and the
			// metadata-cache:stat-cache-max-size-mb. Here stat-cache-max-size-mb
			// overrides stat-cache-capacity. case 3.
			name:                          "both_stat_cache_size_mb_and_capacity_set_3",
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: 0,
			expectedStatCacheMaxSizeMB:    0,
		},
		{
			// Old-scenario where user sets only stat-cache-capacity flag(s), and not
			// metadata-cache:stat-cache-max-size-mb. Case 1: stat-cache-capacity is 0.
			name:                          "stat_cache_capacity_0",
			flagStatCacheCapacity:         0,
			mountConfigStatCacheMaxSizeMB: StatCacheMaxSizeMBUnsetSentinel,
			expectedStatCacheMaxSizeMB:    0,
		},
		{
			// Old-scenario where user sets only stat-cache-capacity flag(s), and not
			// metadata-cache:stat-cache-max-size-mb. Case 2: stat-cache-capacity is
			// non-zero.
			name:                          "stat_cache_capacity_non_zero",
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: StatCacheMaxSizeMBUnsetSentinel,
			expectedStatCacheMaxSizeMB:    16, // 16 MiB = MiB ceiling (10k entries * 1640 bytes (AssumedSizeOfPositiveStatCacheEntry + AssumedSizeOfNegativeStatCacheEntry))
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			statCacheMaxSizeMB, err := ResolveStatCacheMaxSizeMB(tc.mountConfigStatCacheMaxSizeMB, tc.flagStatCacheCapacity)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatCacheMaxSizeMB, statCacheMaxSizeMB)
		})
	}
}
