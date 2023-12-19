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
	// TtlInSecsUnset is set when metada-cache:ttl-secs is not set
	// in the gcsfuse mount config file.
	TtlInSecsUnset int64 = math.MinInt64
)

type WriteConfig struct {
	CreateEmptyFile bool `yaml:"create-empty-file"`
}

type LogConfig struct {
	Severity LogSeverity `yaml:"severity"`
	Format   string      `yaml:"format"`
	FilePath string      `yaml:"file-path"`
}

type CacheLocation string

type FileCacheConfig struct {
	MaxSizeInMB               int64 `yaml:"max-size-in-mb"`
	DownloadFileForRandomRead bool  `yaml:"download-file-for-random-read"`
}

type MetadataCacheConfig struct {
	// TtlInSeconds is the ttl
	// value in seconds, to be used for stat-cache and type-cache.
	// It can be -1 for no-ttl, 0 for
	// no cache and > 0 for ttl-controlled metadata-cache.
	// Any value set below -1 will throw an error.
	// If it is not set in the yaml config file,
	// its default value is set to TtlInSecsUnset.
	TtlInSeconds int64 `yaml:"ttl-secs,omitempty"`
}

type MountConfig struct {
	WriteConfig         `yaml:"write"`
	LogConfig           `yaml:"logging"`
	FileCacheConfig     `yaml:"file-cache"`
	CacheLocation       `yaml:"cache-location"`
	MetadataCacheConfig `yaml:"metadata-cache"`
}

func NewMountConfig() *MountConfig {
	mountConfig := &MountConfig{}
	mountConfig.LogConfig = LogConfig{
		// Making the default severity as INFO.
		Severity: INFO,
	}
	mountConfig.FileCacheConfig = FileCacheConfig{
		MaxSizeInMB: 0,
	}
	mountConfig.MetadataCacheConfig = MetadataCacheConfig{
		TtlInSeconds: TtlInSecsUnset,
	}
	return mountConfig
}
