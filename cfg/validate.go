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
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/spf13/viper"
)

const (
	FileCacheMaxSizeMBInvalidValueError       = "the value of max-size-mb for file-cache can't be less than -1"
	MaxParallelDownloadsInvalidValueError     = "the value of max-parallel-downloads for file-cache can't be less than -1"
	ParallelDownloadsPerFileInvalidValueError = "the value of parallel-downloads-per-file for file-cache can't be less than 1"
	DownloadChunkSizeMBInvalidValueError      = "the value of download-chunk-size-mb for file-cache can't be less than 1"
	MaxParallelDownloadsCantBeZeroError       = "the value of max-parallel-downloads for file-cache must not be 0 when enable-parallel-downloads is true"
	ProfileAIMLTraining                       = "aiml-training"
	ProfileAIMLServing                        = "aiml-serving"
	ProfileAIMLCheckpointing                  = "aiml-checkpointing"
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

func isValidParallelDownloadConfig(config *Config) error {
	if config.FileCache.EnableParallelDownloads {
		if !IsFileCacheEnabled(config) {
			return errors.New("file cache should be enabled for parallel download support")
		}
		if config.FileCache.MaxParallelDownloads == 0 {
			return errors.New("the value of max-parallel-downloads for file-cache must not be 0 when enable-parallel-downloads is true")
		}
		if config.FileCache.WriteBufferSize < CacheUtilMinimumAlignSizeForWriting {
			return errors.New("the value of write-buffer-size for file-cache can't be less than 4096")
		}
		if (config.FileCache.WriteBufferSize % CacheUtilMinimumAlignSizeForWriting) != 0 {
			return errors.New("the value of write-buffer-size for file-cache should be in multiple of 4096")
		}
	}

	return nil
}

func isValidFileCacheConfig(config *FileCacheConfig) error {
	if config.MaxSizeMb < -1 {
		return errors.New(FileCacheMaxSizeMBInvalidValueError)
	}
	if config.MaxParallelDownloads < -1 {
		return errors.New(MaxParallelDownloadsInvalidValueError)
	}
	if config.ParallelDownloadsPerFile < 1 {
		return errors.New(ParallelDownloadsPerFileInvalidValueError)
	}
	if config.DownloadChunkSizeMb < 1 {
		return errors.New(DownloadChunkSizeMBInvalidValueError)
	}
	if _, err := regexp.Compile(config.ExcludeRegex); err != nil {
		return fmt.Errorf("invalid regex value %q provided for exclude-regex", config.ExcludeRegex)
	}

	if _, err := regexp.Compile(config.IncludeRegex); err != nil {
		return fmt.Errorf("invalid regex value %q provided for include-regex", config.IncludeRegex)
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

func isValidMetadataCache(v *viper.Viper, c *MetadataCacheConfig) error {
	// Validate ttl-secs.
	if v.IsSet(MetadataCacheTTLConfigKey) {
		if c.TtlSecs < -1 {
			return fmt.Errorf("the value of ttl-secs for metadata-cache can't be less than -1")
		}
		if c.TtlSecs > maxSupportedTTLInSeconds {
			return fmt.Errorf("the value of ttl-secs in metadata-cache is too high to be supported. Max is 9223372036")
		}
	}

	// Validate negative-ttl-secs.
	if v.IsSet(MetadataNegativeCacheTTLConfigKey) {
		if c.NegativeTtlSecs < -1 {
			return fmt.Errorf("the value of negative-ttl-secs for metadata-cache can't be less than -1")
		}
		if c.NegativeTtlSecs > maxSupportedTTLInSeconds {
			return fmt.Errorf("the value of negative-ttl-secs in metadata-cache is too high to be supported. Max is 9223372036")
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

	// Validate prefetch configs.
	if c.MetadataPrefetchMaxWorkers < -1 {
		return fmt.Errorf("invalid value of metadata-cache.metadata-prefetch-max-workers: %d; should be >=0 or -1 (for infinite)", c.MetadataPrefetchMaxWorkers)
	}

	if c.MetadataPrefetchEntriesLimit < -1 {
		return fmt.Errorf("invalid value of metadata-cache.metadata-prefetch-entries-limit: %d; should be >=0 or -1 (for infinite)", c.MetadataPrefetchEntriesLimit)
	}

	return nil
}

func isValidWriteStreamingConfig(wc *WriteConfig) error {
	if !wc.EnableStreamingWrites {
		return nil
	}

	if wc.BlockSizeMb <= 0 || wc.BlockSizeMb > util.MaxMiBsInInt64 {
		return fmt.Errorf("invalid value of write-block-size-mb; can't be less than 1 or more than %d", util.MaxMiBsInInt64)
	}
	if !(wc.MaxBlocksPerFile == -1 || wc.MaxBlocksPerFile >= 1) {
		return fmt.Errorf("invalid value of write-max-blocks-per-file: %d; should be >=1 or -1 (for infinite)", wc.MaxBlocksPerFile)
	}
	if wc.GlobalMaxBlocks < -1 {
		return fmt.Errorf("invalid value of write-global-max-blocks: %d; should be >=0 or -1 (for infinite)", wc.GlobalMaxBlocks)
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

func isValidMetricsConfig(m *MetricsConfig) error {
	if m.StackdriverExportInterval != 0 && m.CloudMetricsExportIntervalSecs != 0 {
		return fmt.Errorf("exactly one of stackdriver-export-interval and cloud-metrics-export-interval-secs must be specified")
	}
	const maxPortNumber = math.MaxUint16
	if m.PrometheusPort > maxPortNumber {
		return fmt.Errorf("prometheus-port must not be higher than the maximum allowed port number: %d but received: %d instead", maxPortNumber, m.PrometheusPort)
	}
	if m.Workers < 1 {
		return fmt.Errorf("number of metrics workers cannot be less than 1")
	}
	if m.BufferSize < 1 {
		return fmt.Errorf("metrics buffer size cannot be less than 1")
	}
	return nil
}

func isValidMonitoringConfig(m *MonitoringConfig) error {
	validExporters := []string{"stdout", "gcptrace"}

	if m.ExperimentalTracingSamplingRatio > 1 || m.ExperimentalTracingSamplingRatio < 0 {
		return fmt.Errorf("invalid tracing sampling ratio: %f, tracing sampling ratio should be in the range [0.0, 1.0]", m.ExperimentalTracingSamplingRatio)
	}

	if len(m.ExperimentalTracingMode) == 0 {
		return nil
	}

	for _, e := range m.ExperimentalTracingMode {
		if !slices.Contains(validExporters, strings.TrimSpace(strings.ToLower(e))) {
			return fmt.Errorf("encountered invalid/unsupported tracing mode: %s", e)
		}
	}

	return nil
}

func isValidChunkTransferTimeoutForRetriesConfig(chunkTransferTimeoutSecs int64) error {
	if chunkTransferTimeoutSecs < 0 || chunkTransferTimeoutSecs > maxSupportedTTLInSeconds {
		return fmt.Errorf("invalid value of ChunkTransferTimeout: %d; should be > 0 or 0 (for infinite)", chunkTransferTimeoutSecs)
	}
	return nil
}

func isValidBufferedReadConfig(rc *ReadConfig) error {
	if !rc.EnableBufferedRead {
		return nil
	}

	if rc.BlockSizeMb <= 0 || rc.BlockSizeMb > util.MaxMiBsInInt64 {
		return fmt.Errorf("invalid value of read-block-size-mb; can't be less than 1 or more than %d", util.MaxMiBsInInt64)
	}

	if rc.GlobalMaxBlocks < -1 {
		return fmt.Errorf("invalid value of read-global-max-blocks: %d; should be >=0 or -1 (for infinite)", rc.GlobalMaxBlocks)
	}

	if rc.StartBlocksPerHandle < 1 && rc.StartBlocksPerHandle != -1 {
		return fmt.Errorf("invalid value of read-start-blocks-per-handle: %d; should be >=1 or -1 (for infinite)", rc.StartBlocksPerHandle)
	}

	if rc.MaxBlocksPerHandle < 1 && rc.MaxBlocksPerHandle != -1 {
		return fmt.Errorf("invalid value of read-max-blocks-per-handle: %d; should be >=1 or -1 (for infinite)", rc.MaxBlocksPerHandle)
	}

	if rc.MinBlocksPerHandle < 1 || (rc.MaxBlocksPerHandle != -1 && rc.MinBlocksPerHandle > rc.MaxBlocksPerHandle) {
		return fmt.Errorf("invalid value of read-min-blocks-per-handle: %d; should be >=1 or less than or equal to read-max-blocks-per-handle: %d", rc.MinBlocksPerHandle, rc.MaxBlocksPerHandle)
	}

	return nil
}

func isValidMRDConfig(mrdConfig *MrdConfig) error {
	if mrdConfig.PoolSize < 1 {
		return fmt.Errorf("invalid value of mrd-pool-size: %d; should be >=1", mrdConfig.PoolSize)
	}
	return nil
}

func isValidOptimizationProfile(config *Config) error {
	if config.Profile == "" {
		return nil
	}

	switch config.Profile {
	case ProfileAIMLServing, ProfileAIMLCheckpointing, ProfileAIMLTraining:
		// Supported profiles.
	default:
		return fmt.Errorf("Unknown profile: %q", config.Profile)
	}

	return nil
}

// ValidateConfig returns a non-nil error if the config is invalid.
func ValidateConfig(v *viper.Viper, config *Config) error {
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

	if err = isValidChunkTransferTimeoutForRetriesConfig(config.GcsRetries.ChunkTransferTimeoutSecs); err != nil {
		return fmt.Errorf("error parsing chunk-transfer-timeout-secs config: %w", err)
	}

	if err = isValidMetricsConfig(&config.Metrics); err != nil {
		return fmt.Errorf("error parsing metrics config: %w", err)
	}

	if err = isValidMonitoringConfig(&config.Monitoring); err != nil {
		return fmt.Errorf("error parsing monitoring config: %w", err)
	}

	if err = isValidParallelDownloadConfig(config); err != nil {
		return fmt.Errorf("error parsing parallel download config: %w", err)
	}

	if err = isValidBufferedReadConfig(&config.Read); err != nil {
		return fmt.Errorf("error parsing buffered read config: %w", err)
	}

	if err = isValidMRDConfig(&config.Mrd); err != nil {
		return fmt.Errorf("error parsing mrd config: %w", err)
	}

	if err = isValidOptimizationProfile(config); err != nil {
		return fmt.Errorf("error parsing optimize profile config: %w", err)
	}

	return nil
}
