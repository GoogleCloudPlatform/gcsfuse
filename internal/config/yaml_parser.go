// Copyright 2021 Google Inc. All Rights Reserved.
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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"gopkg.in/yaml.v3"
)

const (
	TRACE   string = "TRACE"
	DEBUG   string = "DEBUG"
	INFO    string = "INFO"
	WARNING string = "WARNING"
	ERROR   string = "ERROR"
	OFF     string = "OFF"

	parseConfigFileErrMsgFormat = "error parsing config file: %v"

	MetadataCacheTtlSecsInvalidValueError     = "the value of ttl-secs for metadata-cache can't be less than -1"
	MetadataCacheTtlSecsTooHighError          = "the value of ttl-secs in metadata-cache is too high to be supported. Max is 9223372036"
	TypeCacheMaxSizeMBInvalidValueError       = "the value of type-cache-max-size-mb for metadata-cache can't be less than -1"
	StatCacheMaxSizeMBInvalidValueError       = "the value of stat-cache-max-size-mb for metadata-cache can't be less than -1"
	StatCacheMaxSizeMBTooHighError            = "the value of stat-cache-max-size-mb for metadata-cache is too high! Max supported: 17592186044415"
	MaxSupportedStatCacheMaxSizeMB            = util.MaxMiBsInUint64
	UnsupportedMetadataPrefixModeError        = "unsupported metadata-prefix-mode: \"%s\"; supported values: disabled, sync, async"
	FileCacheMaxSizeMBInvalidValueError       = "the value of max-size-mb for file-cache can't be less than -1"
	MaxParallelDownloadsInvalidValueError     = "the value of max-parallel-downloads for file-cache can't be less than -1"
	ParallelDownloadsPerFileInvalidValueError = "the value of parallel-downloads-per-file for file-cache can't be less than 1"
	DownloadChunkSizeMBInvalidValueError      = "the value of download-chunk-size-mb for file-cache can't be less than 1"
)

func IsValidLogSeverity(severity string) bool {
	switch severity {
	case
		TRACE,
		DEBUG,
		INFO,
		WARNING,
		ERROR,
		OFF:
		return true
	}
	return false
}

func IsValidLogRotateConfig(config LogRotateConfig) error {
	if config.MaxFileSizeMB <= 0 {
		return fmt.Errorf("max-file-size-mb should be atleast 1")
	}
	if config.BackupFileCount < 0 {
		return fmt.Errorf("backup-file-count should be 0 (to retain all backup files) or a positive value")
	}
	return nil
}

func (fileCacheConfig *FileCacheConfig) validate() error {
	if fileCacheConfig.MaxSizeMB < -1 {
		return fmt.Errorf(FileCacheMaxSizeMBInvalidValueError)
	}
	if fileCacheConfig.MaxParallelDownloads < -1 {
		return fmt.Errorf(MaxParallelDownloadsInvalidValueError)
	}
	if fileCacheConfig.EnableParallelDownloads && fileCacheConfig.MaxParallelDownloads == 0 {
		return fmt.Errorf("the value of max-parallel-downloads for file-cache must not be 0 when enable-parallel-downloads is true")
	}
	if fileCacheConfig.ParallelDownloadsPerFile < 1 {
		return fmt.Errorf(ParallelDownloadsPerFileInvalidValueError)
	}
	if fileCacheConfig.DownloadChunkSizeMB < 1 {
		return fmt.Errorf(DownloadChunkSizeMBInvalidValueError)
	}

	return nil
}

func (metadataCacheConfig *MetadataCacheConfig) validate() error {
	if metadataCacheConfig.TtlInSeconds != TtlInSecsUnsetSentinel {
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

	if metadataCacheConfig.StatCacheMaxSizeMB != StatCacheMaxSizeMBUnsetSentinel {
		if metadataCacheConfig.StatCacheMaxSizeMB < -1 {
			return fmt.Errorf(StatCacheMaxSizeMBInvalidValueError)
		}
		if metadataCacheConfig.StatCacheMaxSizeMB > int64(MaxSupportedStatCacheMaxSizeMB) {
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

	if err = IsValidLogRotateConfig(mountConfig.LogConfig.LogRotateConfig); err != nil {
		err = fmt.Errorf(parseConfigFileErrMsgFormat, err)
		return
	}

	if err = mountConfig.FileCacheConfig.validate(); err != nil {
		return mountConfig, fmt.Errorf("error parsing file-cache configs: %w", err)
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
