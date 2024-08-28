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
	"net/url"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

// isSet interface is abstraction over the IsSet() method of viper, specially
// added to keep rationalize method simple. IsSet will be used to resolve
// conflicting deprecated flags and new configs.
type isSet interface {
	IsSet(string) bool
}

func decodeURL(u string) (string, error) {
	// TODO: check if we can replace url.Parse with url.ParseRequestURI.
	decodedURL, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return decodedURL.String(), nil
}

// resolveMetadataCacheTTL returns the ttl to be used for stat/type cache based
// on the user flags/configs.
func resolveMetadataCacheTTL(v isSet, c *MetadataCacheConfig) {
	// If metadata-cache:ttl-secs has been set, then it overrides both
	// stat-cache-ttl and type-cache-tll.
	if v.IsSet(MetadataCacheTTLConfigKey) {
		if c.TtlSecs == -1 {
			c.TtlSecs = maxSupportedTTLInSeconds
		}
		return
	}
	// Else, use deprecated stat/type cache ttl to resolve metadataCacheTTL.
	c.TtlSecs = int64(math.Ceil(math.Min(c.DeprecatedStatCacheTtl.Seconds(), c.DeprecatedTypeCacheTtl.Seconds())))
}

// resolveStatCacheMaxSizeMB returns the stat-cache size in MiBs based on the
// user old and new flags/configs.
func resolveStatCacheMaxSizeMB(v isSet, c *MetadataCacheConfig) {
	// If metadata-cache:stat-cache-size-mb has been set, then it overrides
	// stat-cache-capacity.
	if v.IsSet(StatCacheMaxSizeConfigKey) {
		if c.StatCacheMaxSizeMb == -1 {
			c.StatCacheMaxSizeMb = int64(maxSupportedStatCacheMaxSizeMB)
		}
		return
	}
	// Else, use deprecated stat-cache-capacity to resolve StatCacheMaxSizeMb.
	avgTotalStatCacheEntrySize := AverageSizeOfPositiveStatCacheEntry + AverageSizeOfNegativeStatCacheEntry
	c.StatCacheMaxSizeMb = int64(util.BytesToHigherMiBs(uint64(c.DeprecatedStatCacheCapacity) * avgTotalStatCacheEntrySize))
}

// Rationalize updates the config fields based on the values of other fields.
func Rationalize(v isSet, c *Config) error {
	var err error
	if c.GcsConnection.CustomEndpoint, err = decodeURL(c.GcsConnection.CustomEndpoint); err != nil {
		return err
	}

	if c.GcsAuth.TokenUrl, err = decodeURL(c.GcsAuth.TokenUrl); err != nil {
		return err
	}

	if c.Debug.Fuse || c.Debug.Gcs || c.Debug.LogMutex {
		c.Logging.Severity = "TRACE"
	}

	resolveMetadataCacheTTL(v, &c.MetadataCache)
	resolveStatCacheMaxSizeMB(v, &c.MetadataCache)

	return nil
}
