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

package cmd

import (
	"fmt"
	"math"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getConfigObject(t *testing.T, args []string) (*cfg.Config, error) {
	t.Helper()
	var c cfg.Config
	cmd, err := NewRootCmd(func(config cfg.Config) error {
		c = config
		return nil
	})
	require.Nil(t, err)
	cmdArgs := append([]string{"gcsfuse"}, args...)
	cmdArgs = append(cmdArgs, "a")
	cmd.SetArgs(cmdArgs)
	if err = cmd.Execute(); err != nil {
		return nil, err
	}

	return &c, nil
}

func TestMetadataCacheTTLResolution(t *testing.T) {
	testcases := []struct {
		name            string
		args            []string
		expectedTTLSecs int64
	}{
		{
			name:            "Most common scenario, when user doesn't set any of the TTL config parameters.",
			args:            []string{},
			expectedTTLSecs: 60,
		},
		{
			name:            "user sets only metadata-cache:ttl-secs and sets it to -1",
			args:            []string{"--metadata-cache-ttl=-1"},
			expectedTTLSecs: math.MaxInt64,
		},
		{
			name:            "user sets only metadata-cache:ttl-secs and sets it to 0.",
			args:            []string{"--metadata-cache-ttl=0"},
			expectedTTLSecs: 0,
		},
		{
			name:            "user sets only metadata-cache:ttl-secs and sets it to a positive value.",
			args:            []string{"--metadata-cache-ttl=30"},
			expectedTTLSecs: 30,
		},
		{
			name:            "user sets only metadata-cache:ttl-secs and sets it to its highest supported value.",
			args:            []string{fmt.Sprintf("--metadata-cache-ttl=%d", config.MaxSupportedTtlInSeconds)},
			expectedTTLSecs: config.MaxSupportedTtlInSeconds,
		},
		{
			name:            "user sets both the old flags and the metadata-cache:ttl-secs. Here ttl-secs overrides both flags.",
			args:            []string{"--stat-cache-ttl=5m", "--type-cache-ttl=1h", "--metadata-cache-ttl=10800"},
			expectedTTLSecs: 10800,
		},
		{
			name:            "user sets only stat/type-cache-ttl flag(s), and not metadata-cache:ttl-secs.",
			args:            []string{"--stat-cache-ttl=0s", "--type-cache-ttl=0s"},
			expectedTTLSecs: 0,
		},
		{
			name:            "stat-cache enabled, but not type-cache.",
			args:            []string{"--stat-cache-ttl=1h", "--type-cache-ttl=0s"},
			expectedTTLSecs: 0,
		},
		{
			name:            "type-cache enabled, but not stat-cache.",
			args:            []string{"--type-cache-ttl=1h", "--stat-cache-ttl=0s"},
			expectedTTLSecs: 0,
		},
		{
			name:            "both type-cache and stat-cache enabled.",
			args:            []string{"--type-cache-ttl=1h", "--stat-cache-ttl=30s"},
			expectedTTLSecs: 30,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := getConfigObject(t, tc.args)

			if assert.Nil(t, err) {
				config.MetadataCache.TtlSecs = tc.expectedTTLSecs
			}
		})
	}
}

func TestEnableEmptyManagedFoldersResolution(t *testing.T) {
	testcases := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "enable-hns set to true",
			args:     []string{"--enable-hns"},
			expected: true,
		},
		{
			name:     "enable-hns set to true but enable-empty-managed-folders set to false",
			args:     []string{"--enable-hns", "--enable-empty-managed-folders=false"},
			expected: true,
		},
		{
			name:     "enable-hns not true but enable-empty-managed-folders set to true",
			args:     []string{"--enable-hns=false", "--enable-empty-managed-folders=true"},
			expected: true,
		},
		{
			name:     "both enable-hns and enable-empty-managed-folders not true",
			args:     []string{"--enable-hns=false", "--enable-empty-managed-folders=false"},
			expected: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := getConfigObject(t, tc.args)

			if assert.Nil(t, err) {
				config.List.EnableEmptyManagedFolders = tc.expected
			}
		})
	}
}
