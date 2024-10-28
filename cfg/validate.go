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

	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
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
	if config.EnableParallelDownloads {
		if config.MaxParallelDownloads == 0 {
			return fmt.Errorf("the value of max-parallel-downloads for file-cache must not be 0 when enable-parallel-downloads is true")
		}
		if config.WriteBufferSize < cacheutil.MinimumAlignSizeForWriting {
			return fmt.Errorf("the value of write-buffer-size for file-cache can't be less than 4096")
		}
		if (config.WriteBufferSize % cacheutil.MinimumAlignSizeForWriting) != 0 {
			return fmt.Errorf("the value of write-buffer-size for file-cache should be in multiple of 4096")
		}
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

func isValidSequentialReadSizeMB(size int64) error {
	if size < 1 || size > maxSequentialReadSizeMB {
		return fmt.Errorf("sequential-read-size-mb should be between 1 and %d", maxSequentialReadSizeMB)
	}
	return nil
}

// isTTLInSecsValid return nil error if ttlInSecs is valid.
func isTTLInSecsValid(secs int64) error {
	if secs < -1 {
		return fmt.Errorf("the value of ttl-secs can't be less than -1")
	}
	if secs > maxSupportedTTLInSeconds {
		return fmt.Errorf("the value of ttl-secs is too high to be supported. Max is 9223372036")
	}
	return nil
}

func isValidKernelListCacheTTL(TTLSecs int64) error {
	if err := isTTLInSecsValid(TTLSecs); err != nil {
		return fmt.Errorf("invalid kernelListCacheTtlSecs: %w", err)
	}
	return nil
}

func isValidMetadataCache(v isSet, c *MetadataCacheConfig) error {
	// Validate ttl-secs.
	if v.IsSet(MetadataCacheTTLConfigKey) {
		if c.TtlSecs < -1 {
			return fmt.Errorf("the value of ttl-secs for metadata-cache can't be less than -1")
		}
		if c.TtlSecs > maxSupportedTTLInSeconds {
			return fmt.Errorf("the value of ttl-secs in metadata-cache is too high to be supported. Max is 9223372036")
		}
	}

	// Validate type-cache-max-size-mb.
	if c.TypeCacheMaxSizeMb < -1 {
		return fmt.Errorf("the value of type-cache-max-size-mb for metadata-cache can't be less than -1")
	}

	// Validate stat-cache-max-size-mb.
	if v.IsSet(StatCacheMaxSizeConfigKey) {
		if c.StatCacheMaxSizeMb < -1 {
			return fmt.Errorf("the value of stat-cache-max-size-mb for metadata-cache can't be less than -1")
		}
		if c.StatCacheMaxSizeMb > int64(maxSupportedStatCacheMaxSizeMB) {
			return fmt.Errorf("the value of stat-cache-max-size-mb for metadata-cache is too high! Max supported: 17592186044415")
		}
	}

	// [Deprecated] Validate stat-cache-capacity.
	if c.DeprecatedStatCacheCapacity < 0 {
		return fmt.Errorf("invalid value of stat-cache-capacity (%v), can't be less than 0", c.DeprecatedStatCacheCapacity)
	}

	return nil
}

func isValidWriteStreamingConfig(wc *WriteConfig) error {
	if !wc.ExperimentalEnableStreamingWrites {
		return nil
	}

	if wc.BlockSizeMb <= 0 {
		return fmt.Errorf("invalid value of write-block-size-mb; can't be less than 1")
	}
	if !(wc.MaxBlocksPerFile == -1 || wc.MaxBlocksPerFile >= 2) {
		return fmt.Errorf("invalid value of write-max-blocks-per-file: %d; should be >=2 or -1 (for infinite)", wc.MaxBlocksPerFile)
	}
	if !(wc.GlobalMaxBlocks == -1 || wc.GlobalMaxBlocks >= 2) {
		return fmt.Errorf("invalid value of write-global-max-blocks: %d; should be >=2 or -1 (for infinite)", wc.GlobalMaxBlocks)
	}
	return nil
}

func isValidReadStallGcsRetriesConfig(rsrc *ReadStallGcsRetriesConfig) error {
	if rsrc == nil {
		return nil
	}
	if rsrc.Enable {
		if rsrc.ReqIncreaseRate <= 0 {
			return fmt.Errorf("invalid value of increaseRate: %f, can't be 0 or negative", rsrc.ReqIncreaseRate)
		}
		if rsrc.ReqTargetPercentile <= 0 || rsrc.ReqTargetPercentile >= 1 {
			return fmt.Errorf("invalid value of targetPercentile: %f, should be in the range (0, 1)", rsrc.ReqTargetPercentile)
		}
	}
	return nil
}

// ValidateConfig returns a non-nil error if the config is invalid.
func ValidateConfig(v isSet, config *Config) error {
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

	if err = isValidKernelListCacheTTL(config.FileSystem.KernelListCacheTtlSecs); err != nil {
		return fmt.Errorf("error parsing kernel-list-cache-ttl-secs config: %w", err)
	}

	if err = isValidMetadataCache(v, &config.MetadataCache); err != nil {
		return fmt.Errorf("error parsing metadata-cache config: %w", err)
	}

	if err = isValidWriteStreamingConfig(&config.Write); err != nil {
		return fmt.Errorf("error parsing write config: %w", err)
	}

	if err = isValidReadStallGcsRetriesConfig(&config.GcsRetries.ReadStall); err != nil {
		return fmt.Errorf("error parsing read-stall-gcs-retries config: %w", err)
	}

	return nil
}
