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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvalidConfig(t *testing.T) {
	cmd, err := NewRootCmd(func(config cfg.Config) error { return nil })
	if err != nil {
		t.Fatalf("Error while creating the root command: %v", err)
	}
	cmd.SetArgs([]string{"--config-file=testdata/invalid_config.yml", "abc", "pqr"})

	err = cmd.Execute()

	if assert.NotNil(t, err) {
		expectedErr := &mapstructure.Error{}
		assert.ErrorAs(t, err, &expectedErr)
	}
}

func TestValidConfig(t *testing.T) {
	cmd, err := NewRootCmd(func(config cfg.Config) error { return nil })
	if err != nil {
		t.Fatalf("Error while creating the root command: %v", err)
	}
	cmd.SetArgs([]string{"--config-file=testdata/valid_config.yml", "abc", "pqr"})

	assert.Nil(t, cmd.Execute())
}

func TestDefaultMaxParallelDownloads(t *testing.T) {
	var actual cfg.Config
	cmd, err := NewRootCmd(func(c cfg.Config) error {
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
			cmd, err := NewRootCmd(func(config cfg.Config) error { return nil })
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
