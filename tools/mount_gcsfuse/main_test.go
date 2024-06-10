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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeGcsfuseArgs(t *testing.T) {
	testCases := []struct {
		name       string
		device     string
		mountPoint string
		opts       map[string]string
		expected   []string
		expectErr  bool
	}{
		{
			name:       "TestMakeGcsfuseArgs with NoOptions",
			device:     "gcsfuse",
			mountPoint: "/mnt/gcs",
			opts:       map[string]string{},
			expected:   []string{"gcsfuse", "/mnt/gcs"},
		},

		{
			name:       "TestMakeGcsfuseArgs for BooleanFlags with underscore",
			device:     "gcsfuse",
			mountPoint: "/mnt/gcs",
			opts: map[string]string{"implicit_dirs": "",
				"foreground":                    "true",
				"experimental_local_file_cache": "",
				"reuse_token_from_url":          "false",
				"enable_nonexistent_type_cache": "",
				"experimental_enable_json_read": "true",
				"enable_hns":                    "",
				"ignore_interrupts":             "",
				"anonymous_access":              "false",
				"log_rotate_compress":           "false"},
			expected: []string{"--implicit-dirs=true",
				"--foreground=true",
				"--experimental-local-file-cache=true",
				"--reuse-token-from-url=false",
				"--enable-nonexistent-type-cache=true",
				"--experimental-enable-json-read=true",
				"--enable-hns=true",
				"--ignore-interrupts=true",
				"--anonymous-access=false",
				"--log-rotate-compress=false",
				"gcsfuse", "/mnt/gcs"},
		},

		{
			name:       "TestMakeGcsfuseArgs for BooleanFlags with hyphens",
			device:     "gcsfuse",
			mountPoint: "/mnt/gcs",
			opts: map[string]string{"implicit_dirs": "",
				"foreground":                    "true",
				"experimental-local-file-cache": "",
				"reuse-token-from-url":          "false",
				"enable-nonexistent-type-cache": "",
				"experimental-enable-json-read": "true",
				"enable-hns":                    "",
				"ignore-interrupts":             "",
				"anonymous-access":              "false",
				"log_rotate-compress":           "false"},
			expected: []string{"--implicit-dirs=true",
				"--foreground=true",
				"--experimental-local-file-cache=true",
				"--reuse-token-from-url=false",
				"--enable-nonexistent-type-cache=true",
				"--experimental-enable-json-read=true",
				"--enable-hns=true",
				"--ignore-interrupts=true",
				"--anonymous-access=false",
				"--log-rotate-compress=false",
				"gcsfuse", "/mnt/gcs"},
		},

		{
			name:       "TestMakeGcsfuseArgs for StringFlags with underscore",
			device:     "gcsfuse",
			mountPoint: "/mnt/gcs",
			opts: map[string]string{
				"dir_mode": "0755", "key_file": "/path/to/key"},
			expected: []string{"--dir-mode", "0755", "--key-file", "/path/to/key", "gcsfuse", "/mnt/gcs"},
		},

		{
			name:       "TestMakeGcsfuseArgs for StringFlags with hyphen",
			device:     "gcsfuse",
			mountPoint: "/mnt/gcs",
			opts: map[string]string{
				"dir-mode": "0755", "key-file": "/path/to/key"},
			expected: []string{"--dir-mode", "0755", "--key-file", "/path/to/key", "gcsfuse", "/mnt/gcs"},
		},

		{
			name:       "TestMakeGcsfuseArgs with DebugFlags",
			device:     "gcsfuse",
			mountPoint: "/mnt/gcs",
			opts:       map[string]string{"debug_fuse": "", "debug_gcs": ""},
			expected:   []string{"--debug_fuse", "--debug_gcs", "gcsfuse", "/mnt/gcs"},
		},

		// Test ignored options
		{
			name:       "TestMakeGcsfuseArgs with IgnoredOptions",
			device:     "gcsfuse",
			mountPoint: "/mnt/gcs",
			opts:       map[string]string{"user": "nobody", "_netdev": ""},
			expected:   []string{"gcsfuse", "/mnt/gcs"},
		},

		{
			name:       "TestMakeGcsfuseArgs with RegularOptions",
			device:     "gcsfuse",
			mountPoint: "/mnt/gcs",
			opts:       map[string]string{"allow_other": "", "ro": ""},
			expected:   []string{"-o", "allow_other", "-o", "ro", "gcsfuse", "/mnt/gcs"},
		},

		{
			name:       "TestMakeGcsfuseArgs with MixedOptions",
			device:     "gcsfuse",
			mountPoint: "/mnt/gcs",
			opts: map[string]string{
				"implicit_dirs": "", "file_mode": "0644", "debug_fuse": "", "allow_other": "",
			},
			expected: []string{"--implicit-dirs=true", "--file-mode", "0644", "--debug_fuse", "-o", "allow_other", "gcsfuse", "/mnt/gcs"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := makeGcsfuseArgs(tc.device, tc.mountPoint, tc.opts)
			if tc.expectErr && err == nil {
				t.Errorf("Expected error, but got none")
			} else if !tc.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Assert that all flags are present (no matter the order).
			assert.ElementsMatch(t, args[:len(args)-2], tc.expected[:len(tc.expected)-2])
			// Assert that device and mount-point are present at correct position.
			assert.Equal(t, args[len(args)-2:], tc.expected[len(tc.expected)-2:])
		})
	}
}
