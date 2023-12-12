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

import "math"

const (
	// TtlInSecsUnsetSentinel is the value internally
	// set for metada-cache:ttl-secs
	// when it is not set in the gcsfuse mount config file.
	// The constant value has been chosen deliberately
	// to be improbable for a user to explicitly set.
	TtlInSecsUnsetSentinel int64 = math.MinInt64

	// Default log rotation config values.
	defaultMaxFileSizeMB   = 512
	defaultBackupFileCount = 10
	defaultCompress        = true

	// TtlInSecsUnset is set when
	// metadata-cache:type-cache-max-size-mb-per-dir
	// is not set
	// in the gcsfuse mount config file.
	TypeCacheMaxSizeInMbPerDirectoryUnset int = math.MinInt
	// DefaultTypeCacheMaxSizeInMbPerDirectory is maximum size of
	// type-cache per directory in MiBs.
	// This is the value to be used if the user
	// did not the value of metadata-cache:type-cache-max-size-mb-per-dir
	// in config file.
	DefaultTypeCacheMaxSizeInMbPerDirectory int = 16
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

type CacheLocation string

type FileCacheConfig struct {
	MaxSizeInMB           int64 `yaml:"max-size-in-mb"`
	CacheFileForRangeRead bool  `yaml:"cache-file-for-range-read"`
}

type MetadataCacheConfig struct {
	// TtlInSeconds is the ttl
	// value in seconds, to be used for stat-cache and type-cache.
	// It can be set to -1 for no-ttl, 0 for
	// no cache and > 0 for ttl-controlled metadata-cache.
	// Any value set below -1 will throw an error.
	TtlInSeconds int64 `yaml:"ttl-secs,omitempty"`
	// // TypeCacheMaxEntriesPerDirectory is the upper limit on the number of
	// // entries of type-cache maps, which are currently
	// // maintained at per-directory level.
	// // If this is not set, a default value of
	// // DefaultTypeCacheMaxEntriesPerDirectory is taken.
	// // TODO: Delete it.
	// // This is to be deleted in favour of TypeCacheMaxSizeMbPerDirectory.
	// TypeCacheMaxEntriesPerDirectory int `yaml:"type-cache-max-entries-per-dir" default:"1048576"`
	// TypeCacheMaxEntriesPerDirectory is the upper limit
	// on the maximum size of type-cache maps,
	// which are currently
	// maintained at per-directory level.
	// If this is not set, a default value of
	// 16 is taken.
	TypeCacheMaxSizeMbPerDirectory int `yaml:"type-cache-max-size-mb-per-dir,omitempty"`
}

type MountConfig struct {
	WriteConfig         `yaml:"write"`
	LogConfig           `yaml:"logging"`
	FileCacheConfig     `yaml:"file-cache"`
	CacheLocation       `yaml:"cache-location"`
	MetadataCacheConfig `yaml:"metadata-cache"`
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
		MaxSizeInMB: 0,
	}
	mountConfig.MetadataCacheConfig = MetadataCacheConfig{
		TtlInSeconds: TtlInSecsUnsetSentinel,
		TypeCacheMaxSizeMbPerDirectory: TypeCacheMaxSizeInMbPerDirectoryUnset,
	}
	return mountConfig
}
