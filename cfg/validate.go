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
	"math"

	"github.com/spf13/viper"
)

func VetConfig(v *viper.Viper, config *Config) {
	// The EnableEmptyManagedFolders flag must be set to true to enforce folder prefixes for Hierarchical buckets.
	if config.EnableHns {
		config.List.EnableEmptyManagedFolders = true
	}
	// Handle metadatacache ttl
	resolveMetadataCacheTTL(v, config)
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
