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
	"runtime"
	"time"
)

const (
	ttlInSecsInvalidValueError = "the value of ttl-secs can't be less than -1"
	ttlInSecsTooHighError      = "the value of ttl-secs is too high to be supported. Max is 9223372036"
)

func DefaultMaxParallelDownloads() int {
	return max(16, 2*runtime.NumCPU())
}

func IsFileCacheEnabled(mountConfig *Config) bool {
	return mountConfig.FileCache.MaxSizeMb != 0 && string(mountConfig.CacheDir) != ""
}

// isTTLInSecsValid return nil error if ttlInSecs is valid.
func isTTLInSecsValid(TTLInSecs int64) error {
	if TTLInSecs < -1 {
		return fmt.Errorf(ttlInSecsInvalidValueError)
	}

	if TTLInSecs > MaxSupportedTTLInSeconds {
		return fmt.Errorf(ttlInSecsTooHighError)
	}

	return nil
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
