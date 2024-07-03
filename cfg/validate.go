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

	oldconfig "github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
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

func isValidMetadataCacheConfig(config *MetadataCacheConfig) error {
	if config.TtlSecs < -1 {
		return fmt.Errorf("the value of ttl-secs for metadata-cache can't be less than -1")
	}
	if config.TtlSecs > oldconfig.MaxSupportedTtlInSeconds {
		return fmt.Errorf("the value of ttl-secs in metadata-cache is too high to be supported. Max is 9223372036")
	}
	if config.TypeCacheMaxSizeMb < -1 {
		return fmt.Errorf("the value of type-cache-max-size-mb for metadata-cache can't be less than -1")
	}

	if config.StatCacheMaxSizeMb < -1 {
		return fmt.Errorf("the value of stat-cache-max-size-mb for metadata-cache can't be less than -1")
	}
	if config.StatCacheMaxSizeMb > int64(util.MaxMiBsInUint64) {
		return fmt.Errorf("the value of stat-cache-max-size-mb for metadata-cache is too high! Max supported: %d", int64(util.MaxMiBsInUint64))
	}

	return nil
}

func isValidGCSConnectionConfig(config *GcsConnectionConfig) error {
	if config.GrpcConnPoolSize < 1 {
		return fmt.Errorf("the value of grpc-conn-pool-size can't be less than 1")
	}
	return nil
}

func isValidFileSystemConfig(config *FileSystemConfig) error {
	err := oldconfig.IsTtlInSecsValid(config.KernelListCacheTtlSecs)
	if err != nil {
		return fmt.Errorf("invalid kernelListCacheTtlSecs: %w", err)
	}
	return nil
}

func IsValidConfig(config *Config) error {
	if err := isValidLogRotateConfig(&config.Logging.LogRotate); err != nil {
		return fmt.Errorf("error parsing log-rotate config: %w", err)
	}
	if err := isValidFileCacheConfig(&config.FileCache); err != nil {
		return fmt.Errorf("error parsing file-cache config: %w", err)
	}

	if err := isValidMetadataCacheConfig(&config.MetadataCache); err != nil {
		return fmt.Errorf("error parsing metadata-cache config: %w", err)
	}

	if err := isValidGCSConnectionConfig(&config.GcsConnection); err != nil {
		return fmt.Errorf("error parsing gcs-connection configs: %w", err)
	}

	if err := isValidFileSystemConfig(&config.FileSystem); err != nil {
		return fmt.Errorf("error parsing file-system config: %w", err)
	}

	return nil
}

func isValidFileCacheConfig(config *FileCacheConfig) error {
	if config.MaxSizeMb < -1 {
		return fmt.Errorf("the value of max-size-mb for file-cache can't be less than -1")
	}
	if config.MaxParallelDownloads < -1 {
		return fmt.Errorf("the value of max-parallel-downloads for file-cache can't be less than -1")
	}
	if config.EnableParallelDownloads && config.MaxParallelDownloads == 0 {
		return fmt.Errorf("the value of max-parallel-downloads for file-cache must not be 0 when enable-parallel-downloads is true")
	}
	if config.ParallelDownloadsPerFile < 1 {
		return fmt.Errorf("the value of parallel-downloads-per-file for file-cache can't be less than 1")
	}
	if config.DownloadChunkSizeMb < 1 {
		return fmt.Errorf("the value of download-chunk-size-mb for file-cache can't be less than 1")
	}
	return nil
}

func VetConfig(config *Config) {
	// The EnableEmptyManagedFolders flag must be set to true to enforce folder prefixes for Hierarchical buckets.
	if config.EnableHns {
		config.List.EnableEmptyManagedFolders = true
	}
}
