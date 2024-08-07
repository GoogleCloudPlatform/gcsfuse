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

package cmd

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestGetFuseMountConfig_MountOptionsFormattedCorrectly(t *testing.T) {
	testCases := []struct {
		name                string
		inputFuseOptions    []string
		expectedFuseOptions map[string]string
	}{
		{
			name:             "Fuse options input with comma [legacy flag format].",
			inputFuseOptions: []string{"rw,nodev", "user=jacobsa,noauto"},
			expectedFuseOptions: map[string]string{
				"noauto": "",
				"nodev":  "",
				"rw":     "",
				"user":   "jacobsa",
			},
		},
		{
			name:             "Fuse options input without comma [new config format].",
			inputFuseOptions: []string{"rw", "nodev", "user=jacobsa", "noauto"},
			expectedFuseOptions: map[string]string{
				"noauto": "",
				"nodev":  "",
				"rw":     "",
				"user":   "jacobsa",
			},
		},
	}

	fsName := "mybucket"
	mountConfig := &config.MountConfig{}
	for _, tc := range testCases {
		newConfig := &cfg.Config{
			FileSystem: cfg.FileSystemConfig{
				FuseOptions: tc.inputFuseOptions,
			},
		}

		fuseMountCfg := getFuseMountConfig(fsName, newConfig, mountConfig)

		assert.Equal(t, fsName, fuseMountCfg.FSName)
		assert.Equal(t, "gcsfuse", fuseMountCfg.Subtype)
		assert.Equal(t, "gcsfuse", fuseMountCfg.VolumeName)
		assert.Equal(t, tc.expectedFuseOptions, fuseMountCfg.Options)
		assert.True(t, fuseMountCfg.EnableParallelDirOps) // Default true unless explicitly disabled
	}
}
