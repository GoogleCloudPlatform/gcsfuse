// Copyright 2026 Google LLC
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

package fs

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util/diskutil"
	"github.com/stretchr/testify/assert"
)

func TestCacheDirVolumeBlockSize(t *testing.T) {
	cacheDir := t.TempDir()
	actualBlockSize := diskutil.GetVolumeBlockSize(cacheDir)

	for _, tc := range []struct {
		name                         string
		disableSizeCalculationFix    bool
		enableExperimentalChunkCache bool
		expectedBlockSize            uint64
	}{
		{
			name:                         "SizeCalcFixEnabled_NotSparse",
			disableSizeCalculationFix:    false,
			enableExperimentalChunkCache: false,
			expectedBlockSize:            actualBlockSize,
		},
		{
			name:                         "SizeCalcFixDisabled",
			disableSizeCalculationFix:    true,
			enableExperimentalChunkCache: false,
			expectedBlockSize:            1,
		},
		{
			name:                         "SparseModeEnabled",
			disableSizeCalculationFix:    false,
			enableExperimentalChunkCache: true,
			expectedBlockSize:            1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serverCfg := &ServerConfig{
				NewConfig: &cfg.Config{
					FileCache: cfg.FileCacheConfig{
						ExperimentalDisableSizeCalculationFix: tc.disableSizeCalculationFix,
						ExperimentalEnableChunkCache:          tc.enableExperimentalChunkCache,
					},
				},
			}

			blockSize := cacheDirVolumeBlockSize(serverCfg, cacheDir)

			assert.Equal(t, tc.expectedBlockSize, blockSize)
		})
	}
}
