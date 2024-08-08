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

func Test_DefaultMaxParallelDownloads(t *testing.T) {
	assert.GreaterOrEqual(t, DefaultMaxParallelDownloads(), 16)
}

func TestIsFileCacheEnabled(t *testing.T) {
	mountConfig := &Config{
		CacheDir: "/tmp/folder/",
		FileCache: FileCacheConfig{
			MaxSizeMb: -1,
		},
	}
	assert.True(t, IsFileCacheEnabled(mountConfig))

	mountConfig1 := &Config{}
	assert.False(t, IsFileCacheEnabled(mountConfig1))

	mountConfig2 := &Config{
		CacheDir: "",
		FileCache: FileCacheConfig{
			MaxSizeMb: -1,
		},
	}
	assert.False(t, IsFileCacheEnabled(mountConfig2))

	mountConfig3 := &Config{
		CacheDir: "//tmp//folder//",
		FileCache: FileCacheConfig{
			MaxSizeMb: 0,
		},
	}
	assert.False(t, IsFileCacheEnabled(mountConfig3))
}
