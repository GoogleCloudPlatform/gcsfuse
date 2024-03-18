// Copyright 2024 Google Inc. All Rights Reserved.
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

package mount

import (
	"math"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	. "github.com/jacobsa/ogletest"
)

func TestFlag(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type FlagTest struct {
}

func init() {
	RegisterTestSuite(&FlagTest{})
}

func (t *FlagTest) SetUp(ti *TestInfo) {
}

////////////////////////////////////////////////////////////////////////
// Tests for FlagTest
////////////////////////////////////////////////////////////////////////

func (t *FlagTest) TestResolveMetadataCacheTTL() {
	inputs := []struct {
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
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             config.TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: DefaultStatOrTypeCacheTTL,
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to -1.
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             -1,
			expectedMetadataCacheTTL: time.Duration(math.MaxInt64),
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to 0.
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             0,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to a positive value.
			statCacheTTL:             DefaultStatOrTypeCacheTTL,
			typeCacheTTL:             DefaultStatOrTypeCacheTTL,
			ttlInSeconds:             30,
			expectedMetadataCacheTTL: 30 * time.Second,
		},
		{
			// Scenario where user sets only metadata-cache:ttl-secs and sets it to its highest supported value.
			statCacheTTL: DefaultStatOrTypeCacheTTL,
			typeCacheTTL: DefaultStatOrTypeCacheTTL,
			ttlInSeconds: config.MaxSupportedTtlInSeconds,

			expectedMetadataCacheTTL: time.Second * time.Duration(config.MaxSupportedTtlInSeconds),
		},
		{
			// Scenario where user sets both the old flags and the metadata-cache:ttl-secs. Here ttl-secs overrides both flags. case 1.
			statCacheTTL:             5 * time.Minute,
			typeCacheTTL:             time.Hour,
			ttlInSeconds:             10800,
			expectedMetadataCacheTTL: 10800 * time.Second,
		},
		{
			// Scenario where user sets both the old flags and the metadata-cache:ttl-secs. Here ttl-secs overrides both flags. case 2.
			statCacheTTL:             5 * time.Minute,
			typeCacheTTL:             time.Hour,
			ttlInSeconds:             1800,
			expectedMetadataCacheTTL: 1800 * time.Second,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not metadata-cache:ttl-secs. Case 1.
			statCacheTTL:             0,
			typeCacheTTL:             0,
			ttlInSeconds:             config.TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not metadata-cache:ttl-secs. Case 2. Stat-cache enabled, but not type-cache.
			statCacheTTL:             time.Hour,
			typeCacheTTL:             0,
			ttlInSeconds:             config.TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not metadata-cache:ttl-secs. Case 3. Type-cache enabled, but not stat-cache.
			statCacheTTL:             0,
			typeCacheTTL:             time.Hour,
			ttlInSeconds:             config.TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: 0,
		},
		{
			// Old-scenario where user sets only stat/type-cache-ttl flag(s), and not metadata-cache:ttl-secs. Case 4. Both Type-cache and stat-cache enabled. The lower of the two TTLs is taken.
			statCacheTTL:             time.Second,
			typeCacheTTL:             time.Minute,
			ttlInSeconds:             config.TtlInSecsUnsetSentinel,
			expectedMetadataCacheTTL: time.Second,
		},
	}
	for _, input := range inputs {
		AssertEq(input.expectedMetadataCacheTTL, ResolveMetadataCacheTTL(input.statCacheTTL, input.typeCacheTTL, input.ttlInSeconds))
	}
}

func (t *FlagTest) TestResolveStatCacheMaxSizeMB() {
	for _, input := range []struct {
		// Equivalent of user-setting of flag --stat-cache-capacity.
		flagStatCacheCapacity int

		// Equivalent of user-setting of metadata-cache:stat-cache-max-size-mb in --config-file.
		mountConfigStatCacheMaxSizeMB int64

		// Expected output
		expectedStatCacheMaxSizeMB uint64
	}{
		{
			// Most common scenario, when user doesn't set either the flag or the config.
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: config.StatCacheMaxSizeMBUnsetSentinel,
			expectedStatCacheMaxSizeMB:    DefaultStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and sets it to -1.
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: -1,
			expectedStatCacheMaxSizeMB:    config.MaxSupportedStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and sets it to 0.
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: 0,
			expectedStatCacheMaxSizeMB:    0,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and sets it to a positive value.
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: 100,
			expectedStatCacheMaxSizeMB:    100,
		},
		{
			// Scenario where user sets only metadata-cache:stat-cache-max-size-mb and sets it to its highest user-input value.
			flagStatCacheCapacity:         DefaultStatCacheCapacity,
			mountConfigStatCacheMaxSizeMB: int64(config.MaxSupportedStatCacheMaxSizeMB),
			expectedStatCacheMaxSizeMB:    config.MaxSupportedStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets both stat-cache-capacity and the metadata-cache:stat-cache-max-size-mb. Here stat-cache-max-size-mb overrides stat-cache-capacity. case 1.
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: 100,
			expectedStatCacheMaxSizeMB:    100,
		},
		{
			// Scenario where user sets both stat-cache-capacity and the metadata-cache:stat-cache-max-size-mb. Here stat-cache-max-size-mb overrides stat-cache-capacity. case 2.
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: -1,
			expectedStatCacheMaxSizeMB:    config.MaxSupportedStatCacheMaxSizeMB,
		},
		{
			// Scenario where user sets both stat-cache-capacity and the metadata-cache:stat-cache-max-size-mb. Here stat-cache-max-size-mb overrides stat-cache-capacity. case 3.
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: 0,
			expectedStatCacheMaxSizeMB:    0,
		},
		{
			// Old-scenario where user sets only stat-cache-capacity flag(s), and not metadata-cache:stat-cache-max-size-mb. Case 1: stat-cache-capacity is 0.
			flagStatCacheCapacity:         0,
			mountConfigStatCacheMaxSizeMB: config.StatCacheMaxSizeMBUnsetSentinel,
			expectedStatCacheMaxSizeMB:    0,
		},
		{
			// Old-scenario where user sets only stat-cache-capacity flag(s), and not metadata-cache:stat-cache-max-size-mb. Case 2: stat-cache-capacity is non-zero.
			flagStatCacheCapacity:         10000,
			mountConfigStatCacheMaxSizeMB: config.StatCacheMaxSizeMBUnsetSentinel,
			expectedStatCacheMaxSizeMB:    16, // 16 MiB = MiB ceiling (10k entries * 1640 bytes (AssumedSizeOfPositiveStatCacheEntry + AssumedSizeOfNegativeStatCacheEntry))
		},
	} {
		statCacheMaxSizeMB, err := ResolveStatCacheMaxSizeMB(input.mountConfigStatCacheMaxSizeMB, input.flagStatCacheCapacity)
		AssertEq(nil, err)
		AssertEq(input.expectedStatCacheMaxSizeMB, statCacheMaxSizeMB)
	}
}
