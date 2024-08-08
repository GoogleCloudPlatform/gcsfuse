// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"gopkg.in/yaml.v3"
)

const (
	parseConfigFileErrMsgFormat = "error parsing config file: %v"

	MetadataCacheTtlSecsInvalidValueError = "the value of ttl-secs for metadata-cache can't be less than -1"
	MetadataCacheTtlSecsTooHighError      = "the value of ttl-secs in metadata-cache is too high to be supported. Max is 9223372036"
	TypeCacheMaxSizeMBInvalidValueError   = "the value of type-cache-max-size-mb for metadata-cache can't be less than -1"
	StatCacheMaxSizeMBInvalidValueError   = "the value of stat-cache-max-size-mb for metadata-cache can't be less than -1"
	StatCacheMaxSizeMBTooHighError        = "the value of stat-cache-max-size-mb for metadata-cache is too high! Max supported: 17592186044415"
)

func IsValidLogSeverity(severity string) bool {
	switch severity {
	case
		cfg.TRACE,
		cfg.DEBUG,
		cfg.INFO,
		cfg.WARNING,
		cfg.ERROR,
		cfg.OFF:
		return true
	}
	return false
}

func (metadataCacheConfig *MetadataCacheConfig) validate() error {
	if metadataCacheConfig.TtlInSeconds != cfg.TtlInSecsUnsetSentinel {
		if metadataCacheConfig.TtlInSeconds < -1 {
			return fmt.Errorf(MetadataCacheTtlSecsInvalidValueError)
		}
		if metadataCacheConfig.TtlInSeconds > MaxSupportedTtlInSeconds {
			return fmt.Errorf(MetadataCacheTtlSecsTooHighError)
		}
	}
	if metadataCacheConfig.TypeCacheMaxSizeMB < -1 {
		return fmt.Errorf(TypeCacheMaxSizeMBInvalidValueError)
	}

	if metadataCacheConfig.StatCacheMaxSizeMB != cfg.StatCacheMaxSizeMBUnsetSentinel {
		if metadataCacheConfig.StatCacheMaxSizeMB < -1 {
			return fmt.Errorf(StatCacheMaxSizeMBInvalidValueError)
		}
		if metadataCacheConfig.StatCacheMaxSizeMB > int64(cfg.MaxSupportedStatCacheMaxSizeMB) {
			return fmt.Errorf(StatCacheMaxSizeMBTooHighError)
		}
	}
	return nil
}

func (grpcClientConfig *GCSConnection) validate() error {
	if grpcClientConfig.GRPCConnPoolSize < 1 {
		return fmt.Errorf("the value of conn-pool-size can't be less than 1")
	}
	return nil
}

func (fileSystemConfig *FileSystemConfig) validate() error {
	err := IsTtlInSecsValid(fileSystemConfig.KernelListCacheTtlSeconds)
	if err != nil {
		return fmt.Errorf("invalid kernelListCacheTtlSecs: %w", err)
	}
	return nil
}

func ParseConfigFile(fileName string) (mountConfig *MountConfig, err error) {
	mountConfig = NewMountConfig()

	if fileName == "" {
		return
	}

	buf, err := os.ReadFile(fileName)
	if err != nil {
		err = fmt.Errorf("error reading config file: %w", err)
		return
	}

	// Ensure error is thrown when unexpected configs are passed in config file.
	// Ref: https://github.com/go-yaml/yaml/issues/602#issuecomment-623485602
	decoder := yaml.NewDecoder(bytes.NewReader(buf))
	decoder.KnownFields(true)
	if err = decoder.Decode(mountConfig); err != nil {
		// Decode returns EOF in case of empty config file.
		if err == io.EOF {
			return mountConfig, nil
		}
		return mountConfig, fmt.Errorf(parseConfigFileErrMsgFormat, err)
	}

	// convert log severity to upper-case
	mountConfig.LogConfig.Severity = strings.ToUpper(mountConfig.LogConfig.Severity)
	if !IsValidLogSeverity(mountConfig.LogConfig.Severity) {
		err = fmt.Errorf("error parsing config file: log severity should be one of [trace, debug, info, warning, error, off]")
		return
	}

	if err = mountConfig.MetadataCacheConfig.validate(); err != nil {
		return mountConfig, fmt.Errorf("error parsing metadata-cache configs: %w", err)
	}

	if err = mountConfig.GCSConnection.validate(); err != nil {
		return mountConfig, fmt.Errorf("error parsing gcs-connection configs: %w", err)
	}

	if err = mountConfig.FileSystemConfig.validate(); err != nil {
		return mountConfig, fmt.Errorf("error parsing file-system config: %w", err)
	}

	// The EnableEmptyManagedFolders flag must be set to true to enforce folder prefixes for Hierarchical buckets.
	if mountConfig.EnableHNS {
		mountConfig.ListConfig.EnableEmptyManagedFolders = true
	}

	return
}
