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
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util/diskutil"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type FsInternalTestSuite struct {
	suite.Suite
	cacheDir string
}

func TestFsInternalTestSuite(t *testing.T) {
	suite.Run(t, new(FsInternalTestSuite))
}

func (suite *FsInternalTestSuite) SetupTest() {
	var err error
	suite.cacheDir, err = os.MkdirTemp("", "fs_internal_test")
	require.NoError(suite.T(), err)
}

func (suite *FsInternalTestSuite) TearDownTest() {
	if suite.cacheDir != "" {
		_ = os.RemoveAll(suite.cacheDir)
	}
}

func (suite *FsInternalTestSuite) TestCacheDirVolumeBlockSize_SizeCalcFixEnabled_NotSparse() {
	serverCfg := &ServerConfig{
		NewConfig: &cfg.Config{
			FileCache: cfg.FileCacheConfig{
				ExperimentalDisableSizeCalculationFix: false,
				ExperimentalEnableChunkCache:          false,
			},
		},
	}

	actualBlockSize := diskutil.GetVolumeBlockSize(suite.cacheDir)
	blockSize := cacheDirVolumeBlockSize(serverCfg, suite.cacheDir)

	assert.Equal(suite.T(), actualBlockSize, blockSize)
}

func (suite *FsInternalTestSuite) TestCacheDirVolumeBlockSize_SizeCalcFixDisabled() {
	serverCfg := &ServerConfig{
		NewConfig: &cfg.Config{
			FileCache: cfg.FileCacheConfig{
				ExperimentalDisableSizeCalculationFix: true,
				ExperimentalEnableChunkCache:          false,
			},
		},
	}

	// Because the size calculation fix is completely disabled, the block size returned should be 1
	blockSize := cacheDirVolumeBlockSize(serverCfg, suite.cacheDir)

	assert.Equal(suite.T(), uint64(1), blockSize)
}

func (suite *FsInternalTestSuite) TestCacheDirVolumeBlockSize_SparseModeEnabled() {
	serverCfg := &ServerConfig{
		NewConfig: &cfg.Config{
			FileCache: cfg.FileCacheConfig{
				ExperimentalDisableSizeCalculationFix: false,
				ExperimentalEnableChunkCache:          true,
			},
		},
	}

	// Sparse mode overrides and explicitly disables block size up-rounding
	blockSize := cacheDirVolumeBlockSize(serverCfg, suite.cacheDir)

	assert.Equal(suite.T(), uint64(1), blockSize)
}
