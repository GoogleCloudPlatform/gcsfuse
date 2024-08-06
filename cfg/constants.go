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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
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

// Metadata-cache related constants.
const (
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

	// TtlInSecsUnsetSentinel is the value internally
	// set for metada-cache:ttl-secs
	// when it is not set in the gcsfuse mount config file.
	// The constant value has been chosen deliberately
	// to be improbable for a user to explicitly set.
	TtlInSecsUnsetSentinel int64 = math.MinInt64

	// DefaultTypeCacheMaxSizeMB is the default value of type-cache max-size for every directory in MiBs.
	// The value is set at the size needed for about 21k type-cache entries,
	// each of which is about 200 bytes in size.
	DefaultTypeCacheMaxSizeMB int = 4

	// StatCacheMaxSizeMBUnsetSentinel is the value internally
	// set for metadata-cache:stat-cache-max-size-mb
	// when it is not set in the gcsfuse mount config file.
	StatCacheMaxSizeMBUnsetSentinel int64 = math.MinInt64

	MaxSupportedStatCacheMaxSizeMB = util.MaxMiBsInUInt64

	MaxSupportedTtl = time.Duration(MaxSupportedTtlInSeconds * int64(time.Second))
)
