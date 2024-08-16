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
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

func DefaultMaxParallelDownloads() int {
	return max(16, 2*runtime.NumCPU())
}

func IsFileCacheEnabled(mountConfig *Config) bool {
	return mountConfig.FileCache.MaxSizeMb != 0 && string(mountConfig.CacheDir) != ""
}

// ListCacheTTLSecsToDuration converts TTL in seconds to time.Duration.
func ListCacheTTLSecsToDuration(secs int64) time.Duration {
	err := isTTLInSecsValid(secs)
	if err != nil {
		panic(fmt.Sprintf("invalid argument: %d, %v", secs, err))
	}

	if secs == -1 {
		return maxSupportedTTL
	}

	return time.Duration(secs * int64(time.Second))
}

// ResolveMetadataCacheTTL returns the ttl to be used for stat/type cache based on the user flags/configs.
func ResolveMetadataCacheTTL(statCacheTTL, typeCacheTTL time.Duration, ttlInSeconds int64) (metadataCacheTTL time.Duration) {
	// If metadata-cache:ttl-secs has been set in config-file, then
	// it overrides both stat-cache-ttl and type-cache-tll.
	if ttlInSeconds != TtlInSecsUnsetSentinel {
		// if ttl-secs is set to -1, set StatOrTypeCacheTTL to the max possible duration.
		if ttlInSeconds == -1 {
			metadataCacheTTL = time.Duration(math.MaxInt64)
		} else {
			metadataCacheTTL = time.Second * time.Duration(ttlInSeconds)
		}
	} else {
		metadataCacheTTL = time.Second * time.Duration(uint64(math.Ceil(math.Min(statCacheTTL.Seconds(), typeCacheTTL.Seconds()))))
	}
	return
}

// ResolveStatCacheMaxSizeMB returns the stat-cache size in MiBs based on the user old and new flags/configs.
func ResolveStatCacheMaxSizeMB(mountConfigStatCacheMaxSizeMB int64, flagStatCacheCapacity int) (statCacheMaxSizeMB uint64, err error) {
	if mountConfigStatCacheMaxSizeMB != StatCacheMaxSizeMBUnsetSentinel {
		if mountConfigStatCacheMaxSizeMB == -1 {
			statCacheMaxSizeMB = maxSupportedStatCacheMaxSizeMB
		} else {
			statCacheMaxSizeMB = uint64(mountConfigStatCacheMaxSizeMB)
		}
	} else {
		if flagStatCacheCapacity != DefaultStatCacheCapacity {
			if flagStatCacheCapacity < 0 {
				return 0, fmt.Errorf("invalid value of stat-cache-capacity (%v), can't be less than 0", flagStatCacheCapacity)
			}
			avgTotalStatCacheEntrySize := AverageSizeOfPositiveStatCacheEntry + AverageSizeOfNegativeStatCacheEntry
			return util.BytesToHigherMiBs(
				uint64(flagStatCacheCapacity) * avgTotalStatCacheEntrySize), nil
		} else {
			return defaultStatCacheMaxSizeMB, nil
		}
	}
	return
}
