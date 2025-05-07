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

// resolveMetadataCacheTTL calculates the ttl to be used for stat/type cache based
// on the user flags/configs or machine type based optimizations.
func resolveMetadataCacheTTL(v isSet, c *MetadataCacheConfig, optimizedFlags []string) {
	optimizationAppliedToNegativeCacheTTL := isFlagPresent(optimizedFlags, MetadataNegativeCacheTTLConfigKey)

	if v.IsSet(MetadataNegativeCacheTTLConfigKey) || optimizationAppliedToNegativeCacheTTL {
		if c.NegativeTtlSecs == -1 {
			c.NegativeTtlSecs = maxSupportedTTLInSeconds
		}
	}

	// Order of precedence for setting TTL seconds
	// 1. If metadata-cache:ttl-secs has been set, then it has highest precedence
	// 2. If metadata-cache:stat-cache-ttl or metadata-cache:type-cache-ttl has been set or no optimization applied, then it has second highest precedence
	// 3. Optimization is applied (implicit) and take care of special case of -1 which can occur even in defaults
	optimizationAppliedToMetadataCacheTTL := isFlagPresent(optimizedFlags, MetadataCacheTTLConfigKey)
	if v.IsSet(MetadataCacheTTLConfigKey) {
		if c.TtlSecs == -1 {
			c.TtlSecs = maxSupportedTTLInSeconds
		}
	} else if (v.IsSet(MetadataCacheStatCacheTTLConfigKey) || v.IsSet(MetadataCacheTypeCacheTTLConfigKey)) || (!optimizationAppliedToMetadataCacheTTL) {
		c.TtlSecs = int64(math.Ceil(math.Min(c.DeprecatedStatCacheTtl.Seconds(), c.DeprecatedTypeCacheTtl.Seconds())))
	} else if c.TtlSecs == -1 {
		c.TtlSecs = maxSupportedTTLInSeconds
	}
}

// resolveStatCacheMaxSizeMB calculates the stat-cache size in MiBs based on the
// machine-type default override, user's old and new flags/configs.
func resolveStatCacheMaxSizeMB(v isSet, c *MetadataCacheConfig, optimizedFlags []string) {
	// Local function to calculate size based on deprecated capacity.
	calculateSizeFromCapacity := func(capacity int64) int64 {
		avgTotalStatCacheEntrySize := AverageSizeOfPositiveStatCacheEntry + AverageSizeOfNegativeStatCacheEntry
		return int64(util.BytesToHigherMiBs(uint64(capacity) * avgTotalStatCacheEntrySize))
	}

	// Order of precedence for setting stat cache size
	// 1. If metadata-cache:stat-cache-size-mb is set it has the highest precedence
	// 2. If stat-cache-capacity is set or optimization is not applied then use it to calculate stat cache size
	// 3. Else handle special case of -1 for both optimized or possible default value
	optimizationAppliedToStatCacheMaxSize := isFlagPresent(optimizedFlags, StatCacheMaxSizeConfigKey)
	if v.IsSet(StatCacheMaxSizeConfigKey) {
		if c.StatCacheMaxSizeMb == -1 {
			c.StatCacheMaxSizeMb = int64(maxSupportedStatCacheMaxSizeMB)
		}
	} else if v.IsSet(MetadataCacheStatCacheCapacityConfigKey) || (!optimizationAppliedToStatCacheMaxSize) {
		c.StatCacheMaxSizeMb = calculateSizeFromCapacity(c.DeprecatedStatCacheCapacity)
	} else if c.StatCacheMaxSizeMb == -1 {
		c.StatCacheMaxSizeMb = int64(maxSupportedStatCacheMaxSizeMB)
	}
}

func resolveStreamingWriteConfig(w *WriteConfig) {
	if w.EnableStreamingWrites {
		w.CreateEmptyFile = false
	}

	w.BlockSizeMb *= util.MiB

	if w.GlobalMaxBlocks == -1 {
		w.GlobalMaxBlocks = math.MaxInt64
	}

	if w.MaxBlocksPerFile == -1 {
		// Setting a reasonable value here because if enough heap space is not
		// available, make channel results in panic.
		w.MaxBlocksPerFile = math.MaxInt16
	}
}

func resolveCloudMetricsUploadIntervalSecs(m *MetricsConfig) {
	if m.CloudMetricsExportIntervalSecs == 0 {
		m.CloudMetricsExportIntervalSecs = int64(m.StackdriverExportInterval.Seconds())
	}
}

func resolveParallelDownloadsValue(v isSet, fc *FileCacheConfig, c *Config) {
	// Parallel downloads should be default ON when file cache is enabled, in case
	// it is explicitly set by the user, use that value.
	if IsFileCacheEnabled(c) && !v.IsSet(FileCacheParallelDownloadsConfigKey) {
		fc.EnableParallelDownloads = true
	}
}

func resolveReadConfig(c *Config, r *ReadConfig) {
	// Only enable for GRPC client protocol.
	if c.GcsConnection.ClientProtocol != GRPC {
		r.InactiveStreamTimeout = 0
	}
}

// Rationalize updates the config fields based on the values of other fields.
func Rationalize(v isSet, c *Config, optimizedFlags []string) error {
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

	resolveStreamingWriteConfig(&c.Write)
	resolveMetadataCacheTTL(v, &c.MetadataCache, optimizedFlags)
	resolveStatCacheMaxSizeMB(v, &c.MetadataCache, optimizedFlags)
	resolveCloudMetricsUploadIntervalSecs(&c.Metrics)
	resolveParallelDownloadsValue(v, &c.FileCache, c)
	resolveReadConfig(c, &c.Read)

	return nil
}
