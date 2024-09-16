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

package main

import (
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cmd"
	"github.com/stretchr/testify/assert"
)

func TestMainArgParsing(t *testing.T) {
	originalArgs := os.Args
	originalEnvVar := os.Getenv("ENABLE_GCSFUSE_VIPER_CONFIG")
	originalExecuteLegacyMain := cmd.ExecuteLegacyMain
	originalExecuteNewMain := cmd.ExecuteNewMain
	defer func() {
		// Restore original os.Args after the test.
		os.Args = originalArgs
		// Reset the environment variable.
		_ = os.Setenv("ENABLE_GCSFUSE_VIPER_CONFIG", originalEnvVar)
		// Restore original execute functions.
		cmd.ExecuteLegacyMain = originalExecuteLegacyMain
		cmd.ExecuteNewMain = originalExecuteNewMain
	}()

	tests := []struct {
		name                  string
		inputArgs             []string
		inputEnvVariable      string
		expectedEnvVar        string
		expectedRemainingArgs []string
		expectLegacyCall      bool
		expectNewMainCall     bool
	}{
		{
			name:                  "disable_viper_config_with_short_flag",
			inputArgs:             []string{"gcsfuse", "-disable-viper-config", "bucket-name", "mount-point"},
			expectedEnvVar:        "false",
			expectedRemainingArgs: []string{"gcsfuse", "bucket-name", "mount-point"},
			expectLegacyCall:      true,
			expectNewMainCall:     false,
		},
		{
			name:                  "disable_viper_config_with_posix_flag",
			inputArgs:             []string{"gcsfuse", "--disable-viper-config", "bucket-name", "mount-point"},
			expectedEnvVar:        "false",
			expectedRemainingArgs: []string{"gcsfuse", "bucket-name", "mount-point"},
			expectLegacyCall:      true,
			expectNewMainCall:     false,
		},
		{
			name:                  "disable_viper_config_with_short_flag_and_value",
			inputArgs:             []string{"gcsfuse", "-disable-viper-config=true", "bucket-name", "mount-point"},
			expectedEnvVar:        "false",
			expectedRemainingArgs: []string{"gcsfuse", "bucket-name", "mount-point"},
			expectLegacyCall:      true,
			expectNewMainCall:     false,
		},
		{
			name:                  "disable_viper_config_with_posix_flag_and_value",
			inputArgs:             []string{"gcsfuse", "--disable-viper-config=true", "bucket-name", "mount-point"},
			expectedEnvVar:        "false",
			expectedRemainingArgs: []string{"gcsfuse", "bucket-name", "mount-point"},
			expectLegacyCall:      true,
			expectNewMainCall:     false,
		},
		{
			name:                  "no_disable_flag",
			inputArgs:             []string{"gcsfuse", "--implicit-dirs", "--debug_fuse", "bucket-name", "mount-point"},
			expectedEnvVar:        "", // No change expected
			expectedRemainingArgs: []string{"gcsfuse", "--implicit-dirs", "--debug_fuse", "bucket-name", "mount-point"},
			expectLegacyCall:      false,
			expectNewMainCall:     true,
		},
		{
			name:                  "disable_via_env_variable",
			inputArgs:             []string{"gcsfuse", "--implicit-dirs", "--debug_fuse", "bucket-name", "mount-point"},
			inputEnvVariable:      "false",
			expectedEnvVar:        "false",
			expectedRemainingArgs: []string{"gcsfuse", "--implicit-dirs", "--debug_fuse", "bucket-name", "mount-point"},
			expectLegacyCall:      true,
			expectNewMainCall:     false,
		},
		{
			name:                  "enable_via_env_variable",
			inputArgs:             []string{"gcsfuse", "--flag1", "bucket-name", "mount-point"},
			inputEnvVariable:      "true",
			expectedEnvVar:        "true",
			expectedRemainingArgs: []string{"gcsfuse", "--flag1", "bucket-name", "mount-point"},
			expectLegacyCall:      false,
			expectNewMainCall:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.inputArgs
			err := os.Setenv("ENABLE_GCSFUSE_VIPER_CONFIG", tt.inputEnvVariable)
			assert.NoError(t, err)
			// Mock cmd.ExecuteLegacyMain
			legacyMainCalled := false
			cmd.ExecuteLegacyMain = func() {
				legacyMainCalled = true
			}

			newMainCalled := false
			cmd.ExecuteNewMain = func() {
				newMainCalled = true
			}

			main()

			assert.EqualValues(t, tt.expectedEnvVar, os.Getenv("ENABLE_GCSFUSE_VIPER_CONFIG"))
			assert.EqualValues(t, tt.expectedRemainingArgs, os.Args)
			assert.Equal(t, tt.expectLegacyCall, legacyMainCalled)
			assert.Equal(t, tt.expectNewMainCall, newMainCalled)
		})
	}
}
