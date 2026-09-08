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
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRationalizeMetadataCache(t *testing.T) {
	testCases := []struct {
		name                  string
		args                  []string
		expectedTTLSecs       int64
		expectedStatCacheSize int64
	}{
		{
			name:                  "new_ttl_flag_set",
			args:                  []string{"--metadata-cache-ttl-secs=30"},
			expectedTTLSecs:       30,
			expectedStatCacheSize: 34, // default.
		},
		{
			name:                  "old_ttl_flags_set",
			args:                  []string{"--stat-cache-ttl=10s", "--type-cache-ttl=5s"},
			expectedTTLSecs:       5,
			expectedStatCacheSize: 34, // default.
		},
		{
			name:                  "new_stat-cache-size-mb_flag_set",
			args:                  []string{"--stat-cache-max-size-mb=20"},
			expectedTTLSecs:       60, // default.
			expectedStatCacheSize: 20,
		},
		{
			name:                  "old_stat-cache-capacity_flag_set",
			args:                  []string{"--stat-cache-capacity=1000"},
			expectedTTLSecs:       60, // default.
			expectedStatCacheSize: 2,
		},
		{
			name:                  "no_relevant_flags_set",
			args:                  []string{""},
			expectedTTLSecs:       60, // default.
			expectedStatCacheSize: 34, //default.
		},
		{
			name:                  "both_new_and_old_flags_set",
			args:                  []string{"--metadata-cache-ttl-secs=30", "--stat-cache-ttl=10s", "--type-cache-ttl=5s", "--stat-cache-capacity=1000", "--stat-cache-max-size-mb=20"},
			expectedTTLSecs:       30,
			expectedStatCacheSize: 20,
		},
		{
			name:                  "ttl_and_stat_cache_size_set_to_-1",
			args:                  []string{"--metadata-cache-ttl-secs=-1", "--stat-cache-max-size-mb=-1"},
			expectedTTLSecs:       math.MaxInt64 / int64(time.Second), // Max supported ttl in seconds.
			expectedStatCacheSize: math.MaxUint64 >> 20,               // Max supported cache size in MiB.
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := getConfigObject(t, tc.args)

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedTTLSecs, c.MetadataCache.TtlSecs)
				assert.Equal(t, tc.expectedStatCacheSize, c.MetadataCache.StatCacheMaxSizeMb)
			}
		})
	}
}

func TestRationalizeCloudMetricsExportIntervalSecs(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected int64
	}{
		{
			name:     "stackdriver-export-interval-set",
			args:     []string{"--stackdriver-export-interval=30h"},
			expected: 30 * 3600,
		},
		{
			name:     "cloud-metrics-export-interval-set",
			args:     []string{"--cloud-metrics-export-interval-secs=3200"},
			expected: 3200,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := getConfigObject(t, tc.args)

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, c.Metrics.CloudMetricsExportIntervalSecs)
			}
		})
	}
}

func getMountInfoForTest(t *testing.T, args []string) (*mountInfo, error) {
	t.Helper()
	var info mountInfo
	cmd, err := newRootCmd(func(mi *mountInfo, _, _ string) error {
		info = *mi
		return nil
	})
	require.NoError(t, err)
	cmdArgs := append([]string{"gcsfuse"}, args...)
	cmdArgs = append(cmdArgs, "a")
	cmd.SetArgs(convertToPosixArgs(cmdArgs, cmd))
	if err = cmd.Execute(); err != nil {
		return nil, err
	}
	return &info, nil
}

func TestRationalizeKernelReadAheadCliFlags(t *testing.T) {
	testCases := []struct {
		name                string
		args                []string
		expectedKernelRd    bool
		expectedRequestKb   int64
		expectedReadAheadKb int64
	}{
		{
			name:                "default_request_size",
			args:                []string{"--enable-kernel-reader"},
			expectedKernelRd:    true,
			expectedRequestKb:   1024,
			expectedReadAheadKb: 1024,
		},
		{
			name:                "large_request_size",
			args:                []string{"--enable-kernel-reader", "--fuse-max-request-size-kb=261120"},
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 261120,
		},
		{
			name:                "explicit_read_ahead_preserved",
			args:                []string{"--enable-kernel-reader", "--fuse-max-request-size-kb=261120", "--max-read-ahead-kb=4096"},
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 4096,
		},
		{
			name:                "explicit_zero_preserved",
			args:                []string{"--enable-kernel-reader", "--fuse-max-request-size-kb=261120", "--max-read-ahead-kb=0"},
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 0,
		},
		{
			name:                "explicit_one_preserved",
			args:                []string{"--enable-kernel-reader", "--fuse-max-request-size-kb=261120", "--max-read-ahead-kb=1"},
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 1,
		},
		{
			name:                "single_hyphen_auto_adjusts",
			args:                []string{"-enable-kernel-reader", "-fuse-max-request-size-kb=261120"},
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 261120,
		},
		{
			name:                "single_hyphen_preserves_zero",
			args:                []string{"-enable-kernel-reader", "-fuse-max-request-size-kb=261120", "-max-read-ahead-kb=0"},
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := getMountInfoForTest(t, tc.args)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedKernelRd, info.config.FileSystem.EnableKernelReader)
			assert.Equal(t, tc.expectedRequestKb, info.config.FileSystem.FuseMaxRequestSizeKb)
			assert.Equal(t, tc.expectedReadAheadKb, info.config.FileSystem.MaxReadAheadKb)
		})
	}
}

func TestRationalizeKernelReadAheadDisabled(t *testing.T) {
	testCases := []struct {
		name                string
		args                []string
		expectedKernelRd    bool
		expectedRequestKb   int64
		expectedReadAheadKb int64
	}{
		{
			name:                "implicit_disabled_large_request_size",
			args:                []string{"--fuse-max-request-size-kb=261120"},
			expectedKernelRd:    false,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 0,
		},
		{
			name:                "explicit_disabled_large_request_size",
			args:                []string{"--enable-kernel-reader=false", "--fuse-max-request-size-kb=261120"},
			expectedKernelRd:    false,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 0,
		},
		{
			name:                "explicit_disabled_preserves_read_ahead",
			args:                []string{"--enable-kernel-reader=false", "--fuse-max-request-size-kb=261120", "--max-read-ahead-kb=2048"},
			expectedKernelRd:    false,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 2048,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := getMountInfoForTest(t, tc.args)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedKernelRd, info.config.FileSystem.EnableKernelReader)
			assert.Equal(t, tc.expectedRequestKb, info.config.FileSystem.FuseMaxRequestSizeKb)
			assert.Equal(t, tc.expectedReadAheadKb, info.config.FileSystem.MaxReadAheadKb)
		})
	}
}

func TestRationalizeKernelReadAheadConfigFile(t *testing.T) {
	testCases := []struct {
		name                string
		yamlContent         string
		expectedKernelRd    bool
		expectedRequestKb   int64
		expectedReadAheadKb int64
	}{
		{
			name: "config_file_large_request_size",
			yamlContent: `file-system:
  enable-kernel-reader: true
  fuse-max-request-size-kb: 261120
`,
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 261120,
		},
		{
			name: "config_file_explicit_zero_preserved",
			yamlContent: `file-system:
  enable-kernel-reader: true
  fuse-max-request-size-kb: 261120
  max-read-ahead-kb: 0
`,
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 0,
		},
		{
			name: "config_file_explicit_one_preserved",
			yamlContent: `file-system:
  enable-kernel-reader: true
  fuse-max-request-size-kb: 261120
  max-read-ahead-kb: 1
`,
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 1,
		},
		{
			name: "config_file_explicit_read_ahead_preserved",
			yamlContent: `file-system:
  enable-kernel-reader: true
  fuse-max-request-size-kb: 261120
  max-read-ahead-kb: 4096
`,
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 4096,
		},
		{
			name: "config_file_disabled_remains_zero",
			yamlContent: `file-system:
  enable-kernel-reader: false
  fuse-max-request-size-kb: 261120
`,
			expectedKernelRd:    false,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfgPath := createTempConfigFile(t, tc.yamlContent)
			info, err := getMountInfoForTest(t, []string{fmt.Sprintf("--config-file=%s", cfgPath)})
			require.NoError(t, err)
			assert.Equal(t, tc.expectedKernelRd, info.config.FileSystem.EnableKernelReader)
			assert.Equal(t, tc.expectedRequestKb, info.config.FileSystem.FuseMaxRequestSizeKb)
			assert.Equal(t, tc.expectedReadAheadKb, info.config.FileSystem.MaxReadAheadKb)
		})
	}
}

func TestRationalizeKernelReadAheadConfigVsCli(t *testing.T) {
	testCases := []struct {
		name                string
		yamlContent         string
		cliArgs             []string
		expectedKernelRd    bool
		expectedRequestKb   int64
		expectedReadAheadKb int64
	}{
		{
			name: "config_explicit_read_ahead_cli_large_request",
			yamlContent: `file-system:
  enable-kernel-reader: true
  max-read-ahead-kb: 2048
`,
			cliArgs:             []string{"--fuse-max-request-size-kb=261120"},
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 2048,
		},
		{
			name: "config_large_request_cli_explicit_read_ahead",
			yamlContent: `file-system:
  enable-kernel-reader: true
  fuse-max-request-size-kb: 261120
`,
			cliArgs:             []string{"--max-read-ahead-kb=8192"},
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 8192,
		},
		{
			name: "config_explicit_read_ahead_cli_override_zero",
			yamlContent: `file-system:
  enable-kernel-reader: true
  fuse-max-request-size-kb: 261120
  max-read-ahead-kb: 4096
`,
			cliArgs:             []string{"--max-read-ahead-kb=0"},
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 0,
		},
		{
			name: "config_explicit_zero_cli_override_read_ahead",
			yamlContent: `file-system:
  enable-kernel-reader: true
  fuse-max-request-size-kb: 261120
  max-read-ahead-kb: 0
`,
			cliArgs:             []string{"--max-read-ahead-kb=8192"},
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 8192,
		},
		{
			name: "config_disabled_cli_enabled",
			yamlContent: `file-system:
  enable-kernel-reader: false
  fuse-max-request-size-kb: 261120
`,
			cliArgs:             []string{"--enable-kernel-reader"},
			expectedKernelRd:    true,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 261120,
		},
		{
			name: "config_enabled_cli_disabled",
			yamlContent: `file-system:
  enable-kernel-reader: true
  fuse-max-request-size-kb: 261120
`,
			cliArgs:             []string{"--enable-kernel-reader=false"},
			expectedKernelRd:    false,
			expectedRequestKb:   261120,
			expectedReadAheadKb: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfgPath := createTempConfigFile(t, tc.yamlContent)
			args := append([]string{fmt.Sprintf("--config-file=%s", cfgPath)}, tc.cliArgs...)
			info, err := getMountInfoForTest(t, args)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedKernelRd, info.config.FileSystem.EnableKernelReader)
			assert.Equal(t, tc.expectedRequestKb, info.config.FileSystem.FuseMaxRequestSizeKb)
			assert.Equal(t, tc.expectedReadAheadKb, info.config.FileSystem.MaxReadAheadKb)
		})
	}
}

func TestRationalizeKernelReadAheadViperIsSet(t *testing.T) {
	testCases := []struct {
		name          string
		yamlContent   string
		cliArgs       []string
		expectedIsSet bool
	}{
		{
			name:          "cli_flag_is_set_with_value_zero",
			cliArgs:       []string{"--enable-kernel-reader", "--max-read-ahead-kb=0"},
			expectedIsSet: true,
		},
		{
			name:          "cli_flag_is_set_with_value_one",
			cliArgs:       []string{"--enable-kernel-reader", "--max-read-ahead-kb=1"},
			expectedIsSet: true,
		},
		{
			name: "config_file_is_set_with_value_zero",
			yamlContent: `file-system:
  enable-kernel-reader: true
  max-read-ahead-kb: 0
`,
			expectedIsSet: true,
		},
		{
			name: "config_file_is_set_with_value_one",
			yamlContent: `file-system:
  enable-kernel-reader: true
  max-read-ahead-kb: 1
`,
			expectedIsSet: true,
		},
		{
			name:          "cli_flag_is_not_set_with_only_enable_kernel_reader",
			cliArgs:       []string{"--enable-kernel-reader"},
			expectedIsSet: false,
		},
		{
			name:          "cli_flag_is_not_set_with_only_fuse_max_request_size",
			cliArgs:       []string{"--fuse-max-request-size-kb=261120"},
			expectedIsSet: false,
		},
		{
			name:          "no_flags_set",
			cliArgs:       []string{},
			expectedIsSet: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var args []string
			if tc.yamlContent != "" {
				cfgPath := createTempConfigFile(t, tc.yamlContent)
				args = append(args, fmt.Sprintf("--config-file=%s", cfgPath))
			}
			args = append(args, tc.cliArgs...)
			info, err := getMountInfoForTest(t, args)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedIsSet, info.viperConfig.IsSet("file-system.max-read-ahead-kb"))
		})
	}
}
