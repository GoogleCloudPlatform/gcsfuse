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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
)

const (
	// Logging-level constants

	TRACE   string = "TRACE"
	DEBUG   string = "DEBUG"
	INFO    string = "INFO"
	WARNING string = "WARNING"
	ERROR   string = "ERROR"
	OFF     string = "OFF"
)

const (
	// Logging-format constants
	logFormatText    = "text"
	logFormatJSON    = "json"
	defaultLogFormat = logFormatJSON
)

const (
	// ExperimentalMetadataPrefetchOnMountDisabled is the mode without metadata-prefetch.
	ExperimentalMetadataPrefetchOnMountDisabled = "disabled"
	// ExperimentalMetadataPrefetchOnMountSynchronous is the prefetch-mode where mounting is not marked complete until prefetch is complete.
	ExperimentalMetadataPrefetchOnMountSynchronous = "sync"
	// ExperimentalMetadataPrefetchOnMountAsynchronous is the prefetch-mode where mounting is marked complete once prefetch has started.
	ExperimentalMetadataPrefetchOnMountAsynchronous = "async"
)

const (
	// maxSequentialReadSizeMb is the max value supported by sequential-read-size-mb flag.
	maxSequentialReadSizeMB = 1024
)

const (
	// maxSupportedTTLInSeconds represents maximum multiple of seconds representable by time.Duration.
	maxSupportedTTLInSeconds = math.MaxInt64 / int64(time.Second)
	maxSupportedTTL          = time.Duration(maxSupportedTTLInSeconds * int64(time.Second))
)

const (
	// TtlInSecsUnsetSentinel is the value internally
	// set for metada-cache:ttl-secs
	// when it is not set in the gcsfuse mount config file.
	// The constant value has been chosen deliberately
	// to be improbable for a user to explicitly set.
	TtlInSecsUnsetSentinel = math.MinInt64
	// StatCacheMaxSizeMBUnsetSentinel is the value internally set for
	// metadata-cache:stat-cache-max-size-mb when it is not set in the gcsfuse
	// mount config file.
	StatCacheMaxSizeMBUnsetSentinel = math.MinInt64
	// AverageSizeOfPositiveStatCacheEntry is the assumed size of each positive stat-cache-entry,
	// meant for two purposes.
	// 1. for conversion from stat-cache-capacity to stat-cache-max-size-mb.
	// 2. internal testing.
	// Note: When adding 'X' bytes to the heap,it is expected the Resident Set Size
	// which is closer to the actual memory usage increases by roughly '2X'
	// due to overheads like fragmentation and alignment.
	// Thus, incase an attribute of size x has been added to stat cache entry, update
	// the average size here by 2x.
	AverageSizeOfPositiveStatCacheEntry uint64 = 1448
	// AverageSizeOfNegativeStatCacheEntry is the assumed size of each negative stat-cache-entry,
	// meant for two purposes.
	// 1. for conversion from stat-cache-capacity to stat-cache-max-size-mb.
	// 2. internal testing.
	AverageSizeOfNegativeStatCacheEntry uint64 = 240
	// MetadataCacheTTLConfigKey is the Viper configuration key for the metadata
	//cache's time-to-live (TTL) in seconds.
	MetadataCacheTTLConfigKey               = "metadata-cache.ttl-secs"
	MetadataCacheStatCacheTTLConfigKey      = "metadata-cache.deprecated-stat-cache-ttl"
	MetadataCacheTypeCacheTTLConfigKey      = "metadata-cache.deprecated-type-cache-ttl"
	MetadataCacheStatCacheCapacityConfigKey = "metadata-cache.deprecated-stat-cache-capacity"
	MetadataNegativeCacheTTLConfigKey       = "metadata-cache.negative-ttl-secs"
	// StatCacheMaxSizeConfigKey is the Viper configuration key for the maximum
	//size of the metadata stat cache in megabytes.
	StatCacheMaxSizeConfigKey = "metadata-cache.stat-cache-max-size-mb"
	// FileCacheParallelDownloadsConfigKey is the Viper configuration key for the
	//parallel-downloads enablement.
	FileCacheParallelDownloadsConfigKey = "file-cache.enable-parallel-downloads"
	maxSupportedStatCacheMaxSizeMB      = util.MaxMiBsInUint64
)

// CacheUtilMinimumAlignSizeForWriting is the minimum buffer size used for memory-aligned
// writes, ensuring all writes are a multiple of this value to optimize for
// underlying storage. This may result in padding with null data if the content
// size is not a multiple of CacheUtilMinimumAlignSizeForWriting.
const CacheUtilMinimumAlignSizeForWriting = 4096
const ConfigFileFlagName = "config-file"
