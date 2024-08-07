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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DefaultMaxParallelDownloads(t *testing.T) {
	assert.GreaterOrEqual(t, DefaultMaxParallelDownloads(), 16)
}

func TestIsFileCacheEnabled(t *testing.T) {
	testCases := []struct {
		name                       string
		config                     *Config
		expectedIsFileCacheEnabled bool
	}{
		{
			name: "Config with CacheDir set and cache size non zero.",
			config: &Config{
				CacheDir: "/tmp/folder/",
				FileCache: FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedIsFileCacheEnabled: true,
		},
		{
			name:                       "Empty Config.",
			config:                     &Config{},
			expectedIsFileCacheEnabled: false,
		},
		{
			name: "Config with CacheDir unset",
			config: &Config{
				CacheDir: "",
				FileCache: FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedIsFileCacheEnabled: false,
		},
		{
			name: "Config with CacheDir set and cache size zero.",
			config: &Config{
				CacheDir: "//tmp//folder//",
				FileCache: FileCacheConfig{
					MaxSizeMb: 0,
				},
			},
			expectedIsFileCacheEnabled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedIsFileCacheEnabled, IsFileCacheEnabled(tc.config))
		})
	}

}
