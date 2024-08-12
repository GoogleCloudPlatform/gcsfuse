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
)

const (
	FileCacheMaxSizeMBInvalidValueError       = "the value of max-size-mb for file-cache can't be less than -1"
	MaxParallelDownloadsInvalidValueError     = "the value of max-parallel-downloads for file-cache can't be less than -1"
	ParallelDownloadsPerFileInvalidValueError = "the value of parallel-downloads-per-file for file-cache can't be less than 1"
	DownloadChunkSizeMBInvalidValueError      = "the value of download-chunk-size-mb for file-cache can't be less than 1"
	MaxParallelDownloadsCantBeZeroError       = "the value of max-parallel-downloads for file-cache must not be 0 when enable-parallel-downloads is true"
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

func isValidFileCacheConfig(config *FileCacheConfig) error {
	if config.MaxSizeMb < -1 {
		return fmt.Errorf(FileCacheMaxSizeMBInvalidValueError)
	}
	if config.MaxParallelDownloads < -1 {
		return fmt.Errorf(MaxParallelDownloadsInvalidValueError)
	}
	if config.EnableParallelDownloads && config.MaxParallelDownloads == 0 {
		return fmt.Errorf(MaxParallelDownloadsCantBeZeroError)
	}
	if config.ParallelDownloadsPerFile < 1 {
		return fmt.Errorf(ParallelDownloadsPerFileInvalidValueError)
	}
	if config.DownloadChunkSizeMb < 1 {
		return fmt.Errorf(DownloadChunkSizeMBInvalidValueError)
	}

	return nil
}

func IsValidExperimentalMetadataPrefetchOnMount(mode string) error {
	switch mode {
	case ExperimentalMetadataPrefetchOnMountDisabled,
		ExperimentalMetadataPrefetchOnMountSynchronous,
		ExperimentalMetadataPrefetchOnMountAsynchronous:
		return nil
	default:
		return fmt.Errorf("unsupported metadata-prefix-mode: \"%s\"; supported values: disabled, sync, async", mode)
	}
}

func isValidSequentialReadSizeMB(sequentialReadSizeMB int64) error {
	if sequentialReadSizeMB < 1 || sequentialReadSizeMB > maxSequentialReadSizeMB {
		return fmt.Errorf("sequential-read-size-mb should be between 1 and %d", maxSequentialReadSizeMB)
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

	if err = isValidFileCacheConfig(&config.FileCache); err != nil {
		return fmt.Errorf("error parsing file cache config: %w", err)
	}

	if err = IsValidExperimentalMetadataPrefetchOnMount(config.MetadataCache.ExperimentalMetadataPrefetchOnMount); err != nil {
		return fmt.Errorf("error parsing experimental-metadata-prefetch-on-mount: %w", err)
	}

	if err = isValidURL(config.GcsAuth.TokenUrl); err != nil {
		return fmt.Errorf("error parsing token-url config: %w", err)
	}

	if err = isValidSequentialReadSizeMB(config.GcsConnection.SequentialReadSizeMb); err != nil {
		return fmt.Errorf("error parsing gcs-connection config: %w", err)
	}

	return nil
}
