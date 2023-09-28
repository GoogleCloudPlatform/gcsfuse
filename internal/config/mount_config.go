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

type WriteConfig struct {
	CreateEmptyFile bool `yaml:"create-empty-file"`
}

type LogConfig struct {
	Severity LogSeverity `yaml:"severity"`
	Format   string      `yaml:"format"`
	FilePath string      `yaml:"file-path"`
}

type FileCacheConfig struct {
	Path                    string `yaml:"path"`
	Size                    int64  `yaml:"size-mb"`
	TTL                     int64  `yaml:"ttl-sec"`
	CacheFileForRandomReads bool   `yaml:"cache-file-for-random-reads"`
}

type MetadataCacheConfig struct {
	Capacity int64 `yaml:"capacity"`
	TTL      int64 `yaml:"ttl-sec"`
}

type TypeCacheConfig struct {
	TTL              int64 `yaml:"ttl-sec"`
	CacheNonexistent bool  `yaml:"cache-nonexistent"`
}

type MountConfig struct {
	WriteConfig         `yaml:"write"`
	LogConfig           `yaml:"logging"`
	FileCacheConfig     `yaml:"file-cache"`
	MetadataCacheConfig `yaml:"metadata-cache"`
	TypeCacheConfig     `yaml:"type-cache"`
}

func NewMountConfig() *MountConfig {
	mountConfig := &MountConfig{}
	mountConfig.LogConfig = LogConfig{
		// Making the default severity as INFO.
		Severity: INFO,
	}
	mountConfig.MetadataCacheConfig = MetadataCacheConfig{
		Capacity: 4096,
		TTL:      60,
	}
	mountConfig.TypeCacheConfig = TypeCacheConfig{
		TTL: 60,
	}
	return mountConfig
}
