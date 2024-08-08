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
	"testing"

	"github.com/stretchr/testify/assert"
)

func validLogRotateConfig() LogRotateLoggingConfig {
	return LogRotateLoggingConfig{
		BackupFileCount: 0,
		Compress:        false,
		MaxFileSizeMb:   1,
	}
}

func validFileCacheConfig(t *testing.T) FileCacheConfig {
	t.Helper()
	return FileCacheConfig{
		CacheFileForRangeRead:    false,
		DownloadChunkSizeMb:      50,
		EnableCrc:                false,
		EnableParallelDownloads:  false,
		MaxParallelDownloads:     4,
		MaxSizeMb:                -1,
		ParallelDownloadsPerFile: 16,
	}
}

func TestValidateConfigSuccessful(t *testing.T) {
	testCases := []struct {
		name   string
		config *Config
	}{
		{
			name: "Valid Config where input and expected custom endpoint match.",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "https://bing.com/search?q=dotnet",
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "disabled",
				},
			},
		},
		{
			name: "Valid Config where input and expected custom endpoint differ.",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "https://j@ne:password@google.com",
				},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "disabled",
				},
			},
		},
		{
			name: "experimental-metadata-prefetch-on-mount disabled",
			config: &Config{
				Logging: LoggingConfig{LogRotate: validLogRotateConfig()},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "disabled",
				},
			},
		},
		{
			name: "experimental-metadata-prefetch-on-mount async",
			config: &Config{
				Logging: LoggingConfig{LogRotate: validLogRotateConfig()},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "async",
				},
			},
		},
		{
			name: "experimental-metadata-prefetch-on-mount sync",
			config: &Config{
				Logging: LoggingConfig{LogRotate: validLogRotateConfig()},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "sync",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := ValidateConfig(tc.config)

			assert.NoError(t, actualErr)
		})
	}
}

func TestValidateConfigUnsuccessful(t *testing.T) {
	testCases := []struct {
		name   string
		config *Config
	}{
		{
			name: "Invalid Config due to invalid custom endpoint",
			config: &Config{
				Logging:   LoggingConfig{LogRotate: validLogRotateConfig()},
				FileCache: validFileCacheConfig(t),
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "a_b://abc",
				},
			},
		},
		{
			name: "Invalid experimental-metadata-prefetch-on-mount",
			config: &Config{
				Logging: LoggingConfig{LogRotate: validLogRotateConfig()},
				MetadataCache: MetadataCacheConfig{
					ExperimentalMetadataPrefetchOnMount: "a",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := ValidateConfig(tc.config)

			assert.Error(t, actualErr)
		})
	}
}
