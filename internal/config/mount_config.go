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
	"math"
	"time"
)

const (
	// Default log rotation config values.
	defaultMaxFileSizeMB   = 512
	defaultBackupFileCount = 10
	defaultCompress        = true

	// TtlInSecsUnsetSentinel is the value internally
	// set for metada-cache:ttl-secs
	// when it is not set in the gcsfuse mount config file.
	// The constant value has been chosen deliberately
	// to be improbable for a user to explicitly set.
	TtlInSecsUnsetSentinel int64 = math.MinInt64

	// The maximum multiple of seconds representable by time.Duration.
	MaxSupportedTtlInSeconds int64 = int64(math.MaxInt64 / int64(time.Second))

	// DefaultTypeCacheMaxSizeMB is the default value of type-cache max-size for every directory in MiBs.
	// The value is set at the size needed for about 21k type-cache entries,
	// each of which is about 200 bytes in size.
	DefaultTypeCacheMaxSizeMB int = 4

	// StatCacheMaxSizeMBUnsetSentinel is the value internally
	// set for metada-cache:stat-cache-max-size-mb
	// when it is not set in the gcsfuse mount config file.
	StatCacheMaxSizeMBUnsetSentinel int64 = math.MinInt64

	DefaultFileCacheMaxSizeMB               int64 = -1
	DefaultEnableEmptyManagedFoldersListing       = false
	DefaultGrpcConnPoolSize                       = 1
	DefaultAnonymousAccess                        = false
	DefaultEnableHNS                              = false
)

type WriteConfig struct {
	CreateEmptyFile bool `yaml:"create-empty-file"`
}

type LogConfig struct {
	Severity        LogSeverity     `yaml:"severity"`
	Format          string          `yaml:"format"`
	FilePath        string          `yaml:"file-path"`
	LogRotateConfig LogRotateConfig `yaml:"log-rotate"`
}

type ListConfig struct {
	// This flag is specially added to handle the corner case in listing managed folders.
	// There are two corner cases (a) empty managed folder (b) nested managed folder which doesn't contain any descendent as object.
	// This flag always works in conjunction with ImplicitDirectories flag.
	//
	// (a) If only ImplicitDirectories is true, all managed folders are listed other than above two mentioned cases.
	// (b) If both ImplicitDirectories and EnableEmptyManagedFolders are true, then all the managed folders are listed including the above-mentioned corner case.
	// (c) If ImplicitDirectories is false then no managed folders are listed irrespective of EnableEmptyManagedFolders flag.
	EnableEmptyManagedFolders bool `yaml:"enable-empty-managed-folders"`
}

type GrpcClientConfig struct {
	// ConnPoolSize configures the number of gRPC channel in grpc client.
	ConnPoolSize int `yaml:"conn-pool-size,omitempty"`
}

type AuthConfig struct {
	// Authentication is enabled by default. The skip flag disables authentication. For users of the --custom-endpoint flag,
	// please pass anonymous-access flag explicitly if you do not want authentication enabled for your workflow.
	AnonymousAccess bool `yaml:"anonymous-access"`
}

// Enable the storage control client flow on HNS buckets to utilize new APIs.
type EnableHNS bool
type CacheDir string

type FileSystemConfig struct {
	IgnoreInterrupts      bool `yaml:"ignore-interrupts"`
	DisableParallelDirops bool `yaml:"disable-parallel-dirops"`
}

type FileCacheConfig struct {
	MaxSizeMB             int64 `yaml:"max-size-mb"`
	CacheFileForRangeRead bool  `yaml:"cache-file-for-range-read"`
}

type MetadataCacheConfig struct {
	// TtlInSeconds is the ttl
	// value in seconds, to be used for stat-cache and type-cache.
	// It can be set to -1 for no-ttl, 0 for
	// no cache and > 0 for ttl-controlled metadata-cache.
	// Any value set below -1 will throw an error.
	TtlInSeconds int64 `yaml:"ttl-secs,omitempty"`
	// TypeCacheMaxSizeMB is the upper limit
	// on the maximum size of type-cache maps,
	// which are currently
	// maintained at per-directory level.
	TypeCacheMaxSizeMB int `yaml:"type-cache-max-size-mb,omitempty"`

	// StatCacheMaxSizeMB is the maximum size of stat-cache
	// in MiBs.
	// It can also be set to -1 for no-size-limit, 0 for
	// no cache. Values below -1 are not supported.
	StatCacheMaxSizeMB int64 `yaml:"stat-cache-max-size-mb,omitempty"`
}

type MountConfig struct {
	WriteConfig         `yaml:"write"`
	LogConfig           `yaml:"logging"`
	FileCacheConfig     `yaml:"file-cache"`
	CacheDir            `yaml:"cache-dir"`
	MetadataCacheConfig `yaml:"metadata-cache"`
	ListConfig          `yaml:"list"`
	GrpcClientConfig    `yaml:"grpc"`
	AuthConfig          `yaml:"auth-config"`
	EnableHNS           `yaml:"enable-hns"`
	FileSystemConfig    `yaml:"file-system"`
}

// LogRotateConfig defines the parameters for log rotation. It consists of three
// configuration options:
// 1. max-file-size-mb: specifies the maximum size in megabytes that a log file
// can reach before it is rotated. The default value is 512 megabytes.
// 2. backup-file-count: determines the maximum number of backup log files to
// retain after they have been rotated. The default value is 10. When value is
// set to 0, all backup files are retained.
// 3. compress: indicates whether the rotated log files should be compressed
// using gzip. The default value is False.
type LogRotateConfig struct {
	MaxFileSizeMB   int  `yaml:"max-file-size-mb"`
	BackupFileCount int  `yaml:"backup-file-count"`
	Compress        bool `yaml:"compress"`
}

func DefaultLogRotateConfig() LogRotateConfig {
	return LogRotateConfig{
		MaxFileSizeMB:   defaultMaxFileSizeMB,
		BackupFileCount: defaultBackupFileCount,
		Compress:        defaultCompress,
	}
}

func NewMountConfig() *MountConfig {
	mountConfig := &MountConfig{}
	mountConfig.LogConfig = LogConfig{
		// Making the default severity as INFO.
		Severity: INFO,
		// Setting default values of log rotate config.
		LogRotateConfig: DefaultLogRotateConfig(),
	}
	mountConfig.FileCacheConfig = FileCacheConfig{
		MaxSizeMB: DefaultFileCacheMaxSizeMB,
	}
	mountConfig.MetadataCacheConfig = MetadataCacheConfig{
		TtlInSeconds:       TtlInSecsUnsetSentinel,
		TypeCacheMaxSizeMB: DefaultTypeCacheMaxSizeMB,
		StatCacheMaxSizeMB: StatCacheMaxSizeMBUnsetSentinel,
	}
	mountConfig.ListConfig = ListConfig{
		EnableEmptyManagedFolders: DefaultEnableEmptyManagedFoldersListing,
	}
	mountConfig.GrpcClientConfig = GrpcClientConfig{
		ConnPoolSize: DefaultGrpcConnPoolSize,
	}
	mountConfig.AuthConfig = AuthConfig{
		AnonymousAccess: DefaultAnonymousAccess,
	}
	mountConfig.EnableHNS = DefaultEnableHNS
	return mountConfig
}
