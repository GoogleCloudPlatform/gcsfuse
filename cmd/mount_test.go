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
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	for _, tc := range testCases {
		newConfig := &cfg.Config{
			FileSystem: cfg.FileSystemConfig{
				FuseOptions: tc.inputFuseOptions,
			},
		}

		fuseMountCfg := getFuseMountConfig(fsName, newConfig)

		assert.Equal(t, fsName, fuseMountCfg.FSName)
		assert.Equal(t, "gcsfuse", fuseMountCfg.Subtype)
		assert.Equal(t, "gcsfuse", fuseMountCfg.VolumeName)
		assert.Equal(t, tc.expectedFuseOptions, fuseMountCfg.Options)
		assert.True(t, fuseMountCfg.EnableParallelDirOps) // Default true unless explicitly disabled
	}
}

func TestGetFuseMountConfig_LoggerInitializationInFuse(t *testing.T) {
	testCases := []struct {
		name                  string
		gcsFuseLogLevel       string
		shouldInitializeTrace bool
		shouldInitializeError bool
	}{
		{
			name:                  "GcsFuseOffLogLevelShouldNotInitializeAnyLogger",
			gcsFuseLogLevel:       "OFF",
			shouldInitializeTrace: false,
			shouldInitializeError: false,
		},
		{
			name:                  "GcsFuseErrorLogLevelShouldInitializeErrorLoggerOnly",
			gcsFuseLogLevel:       "ERROR",
			shouldInitializeTrace: false,
			shouldInitializeError: true,
		},
		{
			name:                  "GcsFuseDebugLogLevelShouldInitializeErrorLoggerOnly",
			gcsFuseLogLevel:       "DEBUG",
			shouldInitializeTrace: false,
			shouldInitializeError: true,
		},
		{
			name:                  "GcsFuseTraceLogLevelShouldInitializeBothLogger",
			gcsFuseLogLevel:       "TRACE",
			shouldInitializeTrace: true,
			shouldInitializeError: true,
		},
	}

	fsName := "mybucket"
	for _, tc := range testCases {
		newConfig := &cfg.Config{
			Logging: cfg.LoggingConfig{
				Severity: cfg.LogSeverity(tc.gcsFuseLogLevel),
			},
		}

		fuseMountCfg := getFuseMountConfig(fsName, newConfig)

		assert.Equal(t, tc.shouldInitializeError, fuseMountCfg.ErrorLogger != nil)
		assert.Equal(t, tc.shouldInitializeTrace, fuseMountCfg.DebugLogger != nil)
	}
}

func TestGetFuseMountConfig_EnableReaddirplus(t *testing.T) {
	testCases := []struct {
		name              string
		enableReaddirplus bool
		expectedValue     bool
	}{
		{
			name:              "ExperimentalEnableReaddirplusFlagFalse",
			enableReaddirplus: false,
			expectedValue:     false,
		},
		{
			name:              "ExperimentalEnableReaddirplusFlagTrue",
			enableReaddirplus: true,
			expectedValue:     true,
		},
	}

	fsName := "mybucket"
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newConfig := &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					ExperimentalEnableReaddirplus: tc.enableReaddirplus,
				},
			}

			fuseMountCfg := getFuseMountConfig(fsName, newConfig)

			assert.Equal(t, tc.expectedValue, fuseMountCfg.EnableReaddirplus)
		})
	}
}

func TestGetFuseMountConfig_MaxWriteAndMaxPages(t *testing.T) {
	pageSize := os.Getpagesize()
	testCases := []struct {
		name             string
		maxWriteSizeKb   int64
		maxRequestSizeKb int64
		expectedMaxWrite uint32
		expectedMaxPages uint16
	}{
		{
			name:             "only_max_write_set",
			maxWriteSizeKb:   16384,
			maxRequestSizeKb: 0,
			expectedMaxWrite: 16 * 1024 * 1024,
			expectedMaxPages: uint16((16384 * 1024) / pageSize),
		},
		{
			name:             "max_write_and_request_size_set_write_dominant",
			maxWriteSizeKb:   16384,
			maxRequestSizeKb: 1024,
			expectedMaxWrite: 16 * 1024 * 1024,
			expectedMaxPages: uint16((16384 * 1024) / pageSize),
		},
		{
			name:             "max_write_and_request_size_set_request_dominant",
			maxWriteSizeKb:   1024,
			maxRequestSizeKb: 16384,
			expectedMaxWrite: 1 * 1024 * 1024,
			expectedMaxPages: uint16((16384 * 1024) / pageSize),
		},
		{
			name:             "neither_set",
			maxWriteSizeKb:   0,
			maxRequestSizeKb: 0,
			expectedMaxWrite: 0,
			expectedMaxPages: 0,
		},
	}

	fsName := "mybucket"
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newConfig := &cfg.Config{
				FileSystem: cfg.FileSystemConfig{
					FuseMaxWriteSizeKb:   tc.maxWriteSizeKb,
					FuseMaxRequestSizeKb: tc.maxRequestSizeKb,
				},
			}

			err := cfg.Rationalize(viper.New(), newConfig, []string{})
			require.NoError(t, err)

			fuseMountCfg := getFuseMountConfig(fsName, newConfig)

			assert.Equal(t, tc.expectedMaxWrite, fuseMountCfg.MaxWrite)
			assert.Equal(t, tc.expectedMaxPages, fuseMountCfg.MaxPages)
		})
	}
}
