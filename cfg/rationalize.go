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
	"log"
	"math"
	"net/url"
	"slices"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
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

// resolveMetadataCacheConfig calculates the ttl to be used for stat/type cache based
// on the user flags/configs or machine type based optimizations.
func resolveMetadataCacheConfig(v isSet, c *MetadataCacheConfig, optimizedFlags []string) {
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

	if c.ExperimentalMaxParallelPrefetches == -1 {
		c.ExperimentalMaxParallelPrefetches = math.MaxInt64
	}

	if c.ExperimentalMetadataPrefetchLimit == -1 {
		c.ExperimentalMetadataPrefetchLimit = math.MaxInt64
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

func resolveFileCacheAndBufferedReadConflict(v isSet, c *Config) {
	if IsFileCacheEnabled(c) && c.Read.EnableBufferedRead {
		// Log a warning only if the user has explicitly enabled buffered-read.
		// The default value for enable-buffered-read is true, so we don't want to
		// log a warning for the default case.
		if v.IsSet("read.enable-buffered-read") {
			log.Printf("Warning: File Cache and Buffered Read features are mutually exclusive. Disabling Buffered Read in favor of File Cache.")
		}
		c.Read.EnableBufferedRead = false
	}
}

func resolveReadConfig(r *ReadConfig) {
	if r.GlobalMaxBlocks == -1 {
		r.GlobalMaxBlocks = math.MaxInt32
	}
}

func resolveLoggingConfig(config *Config) {
	if config.Debug.Fuse || config.Debug.Gcs || config.Debug.LogMutex {
		config.Logging.Severity = "TRACE"
	}

	configLogFormat := config.Logging.Format // capture initial value for error reporting
	config.Logging.Format = strings.ToLower(config.Logging.Format)
	if !slices.Contains([]string{logFormatText, logFormatJSON}, config.Logging.Format) {
		log.Printf("Unsupported log format provided: %s. Defaulting to %s log format.", configLogFormat, defaultLogFormat)
		config.Logging.Format = defaultLogFormat // defaulting to json format
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

	resolveLoggingConfig(c)
	resolveReadConfig(&c.Read)
	resolveStreamingWriteConfig(&c.Write)
	resolveMetadataCacheConfig(v, &c.MetadataCache, optimizedFlags)
	resolveStatCacheMaxSizeMB(v, &c.MetadataCache, optimizedFlags)
	resolveCloudMetricsUploadIntervalSecs(&c.Metrics)
	resolveParallelDownloadsValue(v, &c.FileCache, c)
	resolveFileCacheAndBufferedReadConflict(v, c)

	return nil
}
