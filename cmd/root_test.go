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
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMaxParallelDownloads(t *testing.T) {
	var actual *cfg.Config
	cmd, err := NewRootCmd(func(c *cfg.Config, _, _ string) error {
		actual = c
		return nil
	})
	require.Nil(t, err)
	cmd.SetArgs([]string{"abc", "pqr"})

	if assert.Nil(t, cmd.Execute()) {
		assert.LessOrEqual(t, int64(16), actual.FileCache.MaxParallelDownloads)
	}
}

func TestCobraArgsNumInRange(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "Too many args",
			args:        []string{"gcsfuse", "abc", "pqr", "xyz"},
			expectError: true,
		},
		{
			name:        "Too few args",
			args:        []string{"gcsfuse"},
			expectError: true,
		},
		{
			name:        "Two args is okay",
			args:        []string{"gcsfuse", "abc"},
			expectError: false,
		},
		{
			name:        "Three args is okay",
			args:        []string{"gcsfuse", "abc", "pqr"},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := NewRootCmd(func(*cfg.Config, string, string) error { return nil })
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if tc.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestArgsParsing(t *testing.T) {
	wd, err := os.Getwd()
	require.Nil(t, err)
	hd, err := os.UserHomeDir()
	require.Nil(t, err)
	tests := []struct {
		name               string
		args               []string
		expectedBucket     string
		expectedMountpoint string
	}{
		{
			name:               "Both bucket and mountpoint specified.",
			args:               []string{"gcsfuse", "abc", "pqr"},
			expectedBucket:     "abc",
			expectedMountpoint: path.Join(wd, "pqr"),
		},
		{
			name:               "Only mountpoint specified",
			args:               []string{"gcsfuse", "pqr"},
			expectedBucket:     "",
			expectedMountpoint: path.Join(wd, "pqr"),
		},
		{
			name:               "Absolute path for mountpoint specified",
			args:               []string{"gcsfuse", "/pqr"},
			expectedBucket:     "",
			expectedMountpoint: "/pqr",
		},
		{
			name:               "Relative path from user's home specified as mountpoint",
			args:               []string{"gcsfuse", "~/pqr"},
			expectedBucket:     "",
			expectedMountpoint: path.Join(hd, "pqr"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bucketName, mountPoint string
			cmd, err := NewRootCmd(func(_ *cfg.Config, b string, m string) error {
				bucketName = b
				mountPoint = m
				return nil
			})
			require.Nil(t, err)
			cmd.SetArgs(tc.args)

			err = cmd.Execute()

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedBucket, bucketName)
				assert.Equal(t, tc.expectedMountpoint, mountPoint)
			}
		})
	}
}
