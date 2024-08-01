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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getConfigObject(t *testing.T, args []string) (*cfg.Config, error) {
	t.Helper()
	var c *cfg.Config
	cmd, err := NewRootCmd(func(config *cfg.Config, _, _ string) error {
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

	return c, nil
}

func getConfigObjectWithConfigFile(t *testing.T, configFilePath string) (*cfg.Config, error) {
	t.Helper()
	return getConfigObject(t, []string{fmt.Sprintf("--config-file=%s", configFilePath)})
}

func TestValidateConfigFile(t *testing.T) {
	testCases := []struct {
		name       string
		configFile string
		wantErr    bool
	}{
		{
			name:       "empty file",
			configFile: "testdata/empty_file.yaml",
			wantErr:    false,
		},
		{
			name:       "non-existent file",
			configFile: "testdata/nofile.yml",
			wantErr:    true,
		},
		{
			name:       "invalid config file",
			configFile: "testdata/invalid_config.yaml",
			wantErr:    true,
		},
		{
			name:       "logrotate with 0 backup file count",
			configFile: "testdata/valid_config_with_0_backup-file-count.yaml",
			wantErr:    false,
		},
		{
			name:       "unexpected field in config",
			configFile: "testdata/invalid_unexpectedfield_config.yaml",
			wantErr:    true,
		},
		{
			name:       "valid config",
			configFile: "testdata/valid_config.yaml",
			wantErr:    false,
		},
		{
			name:       "invalid log config",
			configFile: "testdata/invalid_log_config.yaml",
			wantErr:    true,
		},
		{
			name:       "invalid logrotate config: test #1",
			configFile: "testdata/invalid_log_rotate_config_1.yaml",
			wantErr:    true,
		},
		{
			name:       "invalid logrotate config: test #1",
			configFile: "testdata/invalid_log_rotate_config_2.yaml",
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCliFlag(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "empty file",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "logrotate with 0 backup file count",
			args:    []string{"--log-rotate-backup-file-count=0"},
			wantErr: false,
		},
		{
			name:    "invalid log severity",
			args:    []string{"--log-severity=critical"},
			wantErr: true,
		},
		{
			name:    "invalid log-rotate-max-log-file-size-mb",
			args:    []string{"--log-rotate-max-log-file-size-mb=-1"},
			wantErr: true,
		},
		{
			name:    "invalid log-rotate-backup-file-count",
			args:    []string{"--log-rotate-backup-file-count=-1"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := getConfigObject(t, tc.args)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfigFile_WriteConfig(t *testing.T) {
	testCases := []struct {
		name           string
		configFile     string
		expectedConfig *cfg.Config
	}{
		{
			name:       "Empty config file [default values].",
			configFile: "testdata/empty_file.yaml",
			expectedConfig: &cfg.Config{
				Write: cfg.WriteConfig{CreateEmptyFile: false},
			},
		},
		{
			name:       "Valid config file.",
			configFile: "testdata/valid_config.yaml",
			expectedConfig: &cfg.Config{
				Write: cfg.WriteConfig{CreateEmptyFile: true},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConfig, err := getConfigObjectWithConfigFile(t, tc.configFile)

			if assert.NoError(t, err) {
				assert.EqualValues(t, tc.expectedConfig.Write, gotConfig.Write)
			}
		})
	}
}
