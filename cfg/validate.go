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
	"fmt"
	"math"
	"time"
)

const (
	MetadataCacheTtlSecsInvalidValueError = "the value of ttl-secs for metadata-cache can't be less than -1"
	MetadataCacheTtlSecsTooHighError      = "the value of ttl-secs in metadata-cache is too high to be supported. Max is 9223372036"
	TypeCacheMaxSizeMBInvalidValueError   = "the value of type-cache-max-size-mb for metadata-cache can't be less than -1"
	StatCacheMaxSizeMBInvalidValueError   = "the value of stat-cache-max-size-mb for metadata-cache can't be less than -1"
	StatCacheMaxSizeMBTooHighError        = "the value of stat-cache-max-size-mb for metadata-cache is too high! Max supported: 17592186044415"
	// MaxSupportedTtlInSeconds represents maximum multiple of seconds representable by time.Duration.
	MaxSupportedTtlInSeconds = math.MaxInt64 / int64(time.Second)
)

func isValidLogRotateConfig(config *LogRotateLoggingConfig) error {
	if config.MaxFileSizeMb <= 0 {
		return fmt.Errorf("max-file-size-mb should be atleast 1")
	}
	if config.BackupFileCount < 0 {
		return fmt.Errorf("backup-file-count should be 0 (to retain all backup files) or a positive value")
	}
	return nil
}

func isValidURL(u string) error {
	_, err := decodeURL(u)
	return err
}

func isValidMetadataConfig(c *MetadataCacheConfig) error {
	if isMetadataCacheTtlSet(c) {
		if c.TtlSecs < -1 {
			return fmt.Errorf(MetadataCacheTtlSecsInvalidValueError)
		}
		if c.TtlSecs > MaxSupportedTtlInSeconds {
			return fmt.Errorf(MetadataCacheTtlSecsTooHighError)
		}
	}
	if c.TypeCacheMaxSizeMb < -1 {
		return fmt.Errorf(TypeCacheMaxSizeMBInvalidValueError)
	}
	if isStatCacheMaxSizeMbSet(c) {
		if c.StatCacheMaxSizeMb < -1 {
			return fmt.Errorf(StatCacheMaxSizeMBInvalidValueError)
		}
		if c.StatCacheMaxSizeMb > int64(MaxSupportedStatCacheMaxSizeMB) {
			return fmt.Errorf(StatCacheMaxSizeMBTooHighError)
		}
	}
	return nil
}

// ValidateConfig returns a non-nil error if the config is invalid.
func ValidateConfig(config *Config) error {
	var err error

	if err = isValidLogRotateConfig(&config.Logging.LogRotate); err != nil {
		return fmt.Errorf("error parsing log-rotate config: %w", err)
	}

	if err = isValidURL(config.GcsConnection.CustomEndpoint); err != nil {
		return fmt.Errorf("error parsing custom-endpoint config: %w", err)
	}

	if err = isValidMetadataConfig(&config.MetadataCache); err != nil {
		return fmt.Errorf("error parsing metadata-cache config: %w", err)
	}

	return nil
}
