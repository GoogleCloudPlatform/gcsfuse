// Copyright 2015 Google Inc. All Rights Reserved.
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
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

type ClientProtocol string

const (
	HTTP1 ClientProtocol = "http1"
	HTTP2 ClientProtocol = "http2"
	// DefaultStatOrTypeCacheTTL is the default value used for
	// stat-cache-ttl or type-cache-ttl if they have not been set
	// by the user.
	DefaultStatOrTypeCacheTTL time.Duration = time.Minute
	// DefaultStatCacheCapacity is the default value for stat-cache-capacity.
	// This is equivalent of setting metadata-cache: stat-cache-max-size-mb.
	DefaultStatCacheCapacity = 20460
	// DefaultStatCacheMaxSizeMB is the default for stat-cache-max-size-mb
	// and is to be used when neither stat-cache-max-size-mb nor
	// stat-cache-capacity is set.
	DefaultStatCacheMaxSizeMB = 32
	// AverageSizeOfPositiveStatCacheEntry is the assumed size of each positive stat-cache-entry,
	// meant for two purposes.
	// 1. for conversion from stat-cache-capacity to stat-cache-max-size-mb.
	// 2. internal testing.
	AverageSizeOfPositiveStatCacheEntry uint64 = 1400
	// AverageSizeOfNegativeStatCacheEntry is the assumed size of each negative stat-cache-entry,
	// meant for two purposes..
	// 1. for conversion from stat-cache-capacity to stat-cache-max-size-mb.
	// 2. internal testing.
	AverageSizeOfNegativeStatCacheEntry uint64 = 240
)

func (cp ClientProtocol) IsValid() bool {
	switch cp {
	case HTTP1, HTTP2:
		return true
	}
	return false
}

// ParseOptions parse an option string in the format accepted by mount(8) and
// generated for its external mount helpers.
//
// It is assumed that option name and values do not contain commas, and that
// the first equals sign in an option is the name/value separator. There is no
// support for escaping.
//
// For example, if the input is
//
//	user,foo=bar=baz,qux
//
// then the following will be inserted into the map.
//
//	"user": "",
//	"foo": "bar=baz",
//	"qux": "",
func ParseOptions(m map[string]string, s string) {
	// NOTE: The man pages don't define how escaping works, and as far
	// as I can tell there is no way to properly escape or quote a comma in the
	// options list for an fstab entry. So put our fingers in our ears and hope
	// that nobody needs a comma.
	for _, p := range strings.Split(s, ",") {
		var name string
		var value string

		// Split on the first equals sign.
		if equalsIndex := strings.IndexByte(p, '='); equalsIndex != -1 {
			name = p[:equalsIndex]
			value = p[equalsIndex+1:]
		} else {
			name = p
		}

		m[name] = value
	}

}

// ResolveMetadataCacheTTL returns the ttl to be used for stat/type cache based on the user flags/configs.
func ResolveMetadataCacheTTL(statCacheTTL, typeCacheTTL time.Duration, ttlInSeconds int64) (metadataCacheTTL time.Duration) {
	// If metadata-cache:ttl-secs has been set in config-file, then
	// it overrides both stat-cache-ttl and type-cache-tll.
	if ttlInSeconds != config.TtlInSecsUnsetSentinel {
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
	if mountConfigStatCacheMaxSizeMB != config.StatCacheMaxSizeMBUnsetSentinel {
		if mountConfigStatCacheMaxSizeMB == -1 {
			statCacheMaxSizeMB = config.MaxSupportedStatCacheMaxSizeMB
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
			return DefaultStatCacheMaxSizeMB, nil
		}
	}
	return
}
