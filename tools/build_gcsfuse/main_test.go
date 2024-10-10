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
	"os/exec"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "gcsfuse-test")
	if err != nil {
		t.Fatalf("Error while creating temporary directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	err = buildBinaries(dir, "../../", "99.88.77", nil)
	if err != nil {
		t.Fatalf("Error while building binary: %v", err)
	}
	testCases := []struct {
		name     string
		args     string
		expected string
	}{
		{
			name:     "Version Flag without Viper config",
			args:     "--version",
			expected: "gcsfuse version 99.88.77",
		},
		{
			name:     "Version Flag with Viper config",
			args:     "--version",
			expected: "gcsfuse version 99.88.77",
		},
		{
			name:     "Version Shorthand with Viper config",
			args:     "-v",
			expected: "gcsfuse version 99.88.77",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(path.Join(dir, "bin/gcsfuse"), tc.args)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Error running gcsfuse with args %v: %v", tc.args, err)
			}
			assert.Contains(t, string(output), tc.expected)
		})
	}
}
