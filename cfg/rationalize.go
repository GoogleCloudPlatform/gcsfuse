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
	"net/url"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

func decodeURL(u string) (string, error) {
	// TODO: check if we can replace url.Parse with url.ParseRequestURI.
	decodedURL, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return decodedURL.String(), nil
}

// Rationalize updates the config fields based on the values of other fields.
func Rationalize(c *Config) error {
	// The EnableEmptyManagedFolders flag must be set to true to enforce folder prefixes for Hierarchical buckets.
	if c.EnableHns {
		c.List.EnableEmptyManagedFolders = true
	}

	var err error
	c.GcsConnection.CustomEndpoint, err = decodeURL(c.GcsConnection.CustomEndpoint)
	if err != nil {
		return err
	}

	if c.Debug.Fuse || c.Debug.Gcs || c.Debug.LogMutex {
		c.Logging.Severity = "TRACE"
	}

	c.MetadataCache.StatCacheMaxSizeMb = resolveStatCacheMaxSizeMB(&c.MetadataCache)
	c.MetadataCache.TtlSecs = int64(resolveMetadataCacheTTL(&c.MetadataCache).Seconds())

	return nil
}

// resolveMetadataCacheTTL returns the ttl to be used for stat/type cache based on the user flags/configs.
func resolveMetadataCacheTTL(c *MetadataCacheConfig) time.Duration {
	if !isMetadataCacheTtlSet(c) {
		return time.Second * time.Duration(uint64(math.Ceil(math.Min(c.DeprecatedStatCacheTtl.Seconds(), c.DeprecatedTypeCacheTtl.Seconds()))))
	}
	if c.TtlSecs == -1 {
		return time.Duration(math.MaxInt64)
	}
	return time.Second * time.Duration(c.TtlSecs)
}

// resolveStatCacheMaxSizeMB returns the stat-cache size in MiBs based on the user old and new flags/configs.
func resolveStatCacheMaxSizeMB(c *MetadataCacheConfig) int64 {
	if isStatCacheMaxSizeMbSet(c) {
		if c.StatCacheMaxSizeMb == -1 {
			return int64(MaxSupportedStatCacheMaxSizeMB)
		}
		return c.StatCacheMaxSizeMb
	}
	if isStatCacheCapacitySet(c) {
		avgTotalStatCacheEntrySize := AverageSizeOfPositiveStatCacheEntry + AverageSizeOfNegativeStatCacheEntry
		return int64(util.BytesToHigherMiBs(
			uint64(c.DeprecatedStatCacheCapacity) * avgTotalStatCacheEntrySize))
	}
	return DefaultStatCacheMaxSizeMB
}
