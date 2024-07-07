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

package cmd

import (
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/mitchellh/mapstructure"
)

// cliContext is abstraction over the IsSet() method of cli.Context, specially
// added to keep OverrideWithIgnoreInterruptsFlag method's unit test simple.
type cliContext interface {
	IsSet(string) bool
}

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
)

// PopulateNewConfigFromLegacyFlagsAndConfig takes cliContext, legacy flags and legacy MountConfig and resolves it into new cfg.Config Object.
func PopulateNewConfigFromLegacyFlagsAndConfig(c cliContext, flags *flagStorage, legacyConfig *config.MountConfig) (*cfg.Config, error) {
	if flags == nil || legacyConfig == nil {
		return nil, fmt.Errorf("PopulateNewConfigFromLegacyFlagsAndConfig: unexpected nil flags or mount config")
	}

	resolvedConfig := &cfg.Config{}

	structuredFlags := &map[string]interface{}{
		"app-name": flags.AppName,
		"debug": &map[string]interface{}{
			"exit-on-invariant-violation": flags.DebugInvariants,
			"gcs":                         flags.DebugGCS,
			"log-mutex":                   flags.DebugMutex,
			"fuse":                        flags.DebugFuse,
		},
		"file-system": map[string]interface{}{
			"dir-mode":  flags.DirMode,
			"file-mode": flags.FileMode,
			// Todo: "fuse-options":      nil,
			"gid":                        flags.Gid,
			"ignore-interrupts":          flags.IgnoreInterrupts,
			"rename-dir-limit":           flags.RenameDirLimit,
			"temp-dir":                   flags.TempDir,
			"uid":                        flags.Uid,
			"kernel-list-cache-ttl-secs": flags.KernelListCacheTtlSeconds,
		},
		"foreground": flags.Foreground,
		"gcs-auth": map[string]interface{}{
			"anonymous-access":     flags.AnonymousAccess,
			"key-file":             flags.KeyFile,
			"reuse-token-from-url": flags.ReuseTokenFromUrl,
			"token-url":            flags.TokenUrl,
		},
		"gcs-connection": map[string]interface{}{
			"billing-project":               flags.BillingProject,
			"client-protocol":               string(flags.ClientProtocol),
			"custom-endpoint":               flags.CustomEndpoint,
			"experimental-enable-json-read": flags.ExperimentalEnableJsonRead,
			"http-client-timeout":           flags.HttpClientTimeout,
			"limit-bytes-per-sec":           flags.EgressBandwidthLimitBytesPerSecond,
			"limit-ops-per-sec":             flags.OpRateLimitHz,
			"max-conns-per-host":            flags.MaxConnsPerHost,
			"max-idle-conns-per-host":       flags.MaxIdleConnsPerHost,
			"sequential-read-size-mb":       flags.SequentialReadSizeMb,
		},
		"gcs-retries": map[string]interface{}{
			"max-retry-sleep":    flags.MaxRetrySleep,
			"multiplier":         flags.RetryMultiplier,
			"max-retry-attempts": flags.MaxRetryAttempts,
		},
		"implicit-dirs": flags.ImplicitDirs,
		"logging": map[string]interface{}{
			"file-path": flags.LogFile,
			"format":    flags.LogFormat,
		},
		"metadata-cache": map[string]interface{}{
			"deprecated-stat-cache-capacity":          flags.StatCacheCapacity,
			"deprecated-stat-cache-ttl":               flags.StatCacheTTL,
			"deprecated-type-cache-ttl":               flags.TypeCacheTTL,
			"enable-nonexistent-type-cache":           flags.EnableNonexistentTypeCache,
			"experimental-metadata-prefetch-on-mount": flags.ExperimentalMetadataPrefetchOnMount,
		},
		"metrics": map[string]interface{}{
			"stackdriver-export-interval": flags.StackdriverExportInterval,
		},
		"monitoring": map[string]interface{}{
			"experimental-opentelemetry-collector-address": flags.OtelCollectorAddress,
		},
		"only-dir": flags.OnlyDir,
	}

	// Use decoder to convert flagStorage to cfg.Config.
	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook: cfg.DecodeHook(),
		Result:     resolvedConfig,
		TagName:    "yaml",
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, fmt.Errorf("mapstructure.NewDecoder: %w", err)
	}
	// Decoding flags.
	if err = decoder.Decode(structuredFlags); err != nil {
		return nil, fmt.Errorf("decoder.Decode(structuredFlags): %w", err)
	}

	// If config file does not have any values, no need to decode it.
	if reflect.ValueOf(*legacyConfig).IsZero() {
		return resolvedConfig, nil
	}

	// Save overlapping flags in a map to override the config value later.
	var (
		logFile                = resolvedConfig.Logging.FilePath
		logFormat              = resolvedConfig.Logging.Format
		ignoreInterrupts       = resolvedConfig.FileSystem.IgnoreInterrupts
		anonymousAccess        = resolvedConfig.GcsAuth.AnonymousAccess
		kernelListCacheTTLSecs = resolvedConfig.FileSystem.KernelListCacheTtlSecs
		maxRetryAttempts       = resolvedConfig.GcsRetries.MaxRetryAttempts
	)

	// Decoding config to the same config structure (resolvedConfig).
	if err = decoder.Decode(legacyConfig); err != nil {
		return nil, fmt.Errorf("decoder.Decode(config): %w", err)
	}

	// Override/Give priority to flags in case of overlap in flags and config.
	overrideWithFlag(c, "log-file", &resolvedConfig.Logging.FilePath, logFile)
	overrideWithFlag(c, "log-format", &resolvedConfig.Logging.Format, logFormat)
	overrideWithFlag(c, "ignore-interrupts", &resolvedConfig.FileSystem.IgnoreInterrupts, ignoreInterrupts)
	overrideWithFlag(c, "anonymous-access", &resolvedConfig.GcsAuth.AnonymousAccess, anonymousAccess)
	overrideWithFlag(c, "kernel-list-cache-ttl-secs", &resolvedConfig.FileSystem.KernelListCacheTtlSecs, kernelListCacheTTLSecs)
	overrideWithFlag(c, "max-retry-attempts", &resolvedConfig.GcsRetries.MaxRetryAttempts, maxRetryAttempts)

	maxStatCacheSizeMb, err := resolveStatCacheMaxSizeMB(
		legacyConfig.StatCacheMaxSizeMB, flags.StatCacheCapacity)
	if err != nil {
		return nil, err
	}
	resolvedConfig.MetadataCache.StatCacheMaxSizeMb = int64(maxStatCacheSizeMb)
	resolvedConfig.MetadataCache.TtlSecs = int64(resolveMetadataCacheTTL(flags.StatCacheTTL, flags.TypeCacheTTL, legacyConfig.TtlInSeconds).Seconds())
	return resolvedConfig, nil
}

// overrideWithFlag function overrides the toUpdate value with updateValue if
// the flag is set in cliCOntext.
func overrideWithFlag[T any](c cliContext, flag string, toUpdate *T, updateValue T) {
	if !c.IsSet(flag) {
		return
	}
	*toUpdate = updateValue
}

// resolveMetadataCacheTTL returns the ttl to be used for stat/type cache based on the user flags/configs.
func resolveMetadataCacheTTL(statCacheTTL, typeCacheTTL time.Duration, ttlInSeconds int64) (metadataCacheTTL time.Duration) {
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

// resolveStatCacheMaxSizeMB returns the stat-cache size in MiBs based on the user old and new flags/configs.
func resolveStatCacheMaxSizeMB(mountConfigStatCacheMaxSizeMB int64, flagStatCacheCapacity int) (statCacheMaxSizeMB uint64, err error) {
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
