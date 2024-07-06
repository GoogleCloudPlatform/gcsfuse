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

package cfg

import (
	"fmt"
	"math"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/spf13/viper"
)

func Validate(c *Config) error {
	if c.MetadataCache.DeprecatedStatCacheCapacity < 0 {
		return fmt.Errorf("invalid value of stat-cache-capacity (%v), can't be less than 0", c.MetadataCache.DeprecatedStatCacheCapacity)
	}
	return nil
}

func VetConfig(v *viper.Viper, c *Config) {
	// The EnableEmptyManagedFolders flag must be set to true to enforce folder prefixes for Hierarchical buckets.
	if c.EnableHns {
		c.List.EnableEmptyManagedFolders = true
	}
	// Handle metadatacache ttl
	resolveMetadataCacheTTL(v, c)
	resolveMetadataCacheCapacity(v, c)
}

// RresolveMetadataCacheTTL returns the ttl to be used for stat/type cache based on the user flags/configs.
func resolveMetadataCacheTTL(v *viper.Viper, config *Config) {
	// If metadata-cache:ttl-secs has been set in config-file, then
	// it overrides both stat-cache-ttl and type-cache-tll.
	if v.IsSet("metadata-cache.ttl-secs") {
		// if ttl-secs is set to -1, set StatOrTypeCacheTTL to the max possible duration.
		if config.MetadataCache.TtlSecs == -1 {
			config.MetadataCache.TtlSecs = math.MaxInt64
		}
		return
	}
	config.MetadataCache.TtlSecs = int64(math.Ceil(math.Min(config.MetadataCache.DeprecatedStatCacheTtl.Seconds(),
		config.MetadataCache.DeprecatedTypeCacheTtl.Seconds())))
}

func resolveMetadataCacheCapacity(v *viper.Viper, c *Config) {
	if v.IsSet("metadata-cache.stat-cache-max-size-mb") {
		if c.MetadataCache.StatCacheMaxSizeMb != -1 {
			return
		}
		c.MetadataCache.StatCacheMaxSizeMb = int64(config.MaxSupportedStatCacheMaxSizeMB)
		return
	}
	if v.IsSet("metadata-cache.deprecated-stat-cache-capacity") {
		avgTotalStatCacheEntrySize := mount.AverageSizeOfPositiveStatCacheEntry + mount.AverageSizeOfNegativeStatCacheEntry
		c.MetadataCache.StatCacheMaxSizeMb = int64(util.BytesToHigherMiBs(
			uint64(c.MetadataCache.DeprecatedStatCacheCapacity) * avgTotalStatCacheEntrySize))
	}
}
