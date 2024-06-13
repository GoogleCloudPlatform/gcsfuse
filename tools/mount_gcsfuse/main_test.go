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
		name          string
		opts          map[string]string
		expectedFlags []string
	}{
		{
			name:          "TestMakeGcsfuseArgs with NoOptions",
			opts:          map[string]string{},
			expectedFlags: []string{},
		},

		{
			name: "TestMakeGcsfuseArgs for BooleanFlags with underscore",
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
			expectedFlags: []string{"--implicit-dirs=true",
				"--foreground=true",
				"--experimental-local-file-cache=true",
				"--reuse-token-from-url=false",
				"--enable-nonexistent-type-cache=true",
				"--experimental-enable-json-read=true",
				"--enable-hns=true",
				"--ignore-interrupts=true",
				"--anonymous-access=false",
				"--log-rotate-compress=false"},
		},

		{
			name: "TestMakeGcsfuseArgs for BooleanFlags with hyphens",
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
			expectedFlags: []string{"--implicit-dirs=true",
				"--foreground=true",
				"--experimental-local-file-cache=true",
				"--reuse-token-from-url=false",
				"--enable-nonexistent-type-cache=true",
				"--experimental-enable-json-read=true",
				"--enable-hns=true",
				"--ignore-interrupts=true",
				"--anonymous-access=false",
				"--log-rotate-compress=false"},
		},

		{
			name: "TestMakeGcsfuseArgs for StringFlags with underscore",
			opts: map[string]string{
				"dir_mode":                     "0755",
				"key_file":                     "/path/to/key",
				"log_rotate_backup_file_count": "2",
			},
			expectedFlags: []string{"--dir-mode", "0755", "--key-file", "/path/to/key", "--log-rotate-backup-file-count", "2"},
		},

		{
			name: "TestMakeGcsfuseArgs for StringFlags with hyphen",
			opts: map[string]string{
				"dir-mode":                     "0755",
				"key-file":                     "/path/to/key",
				"log-rotate-backup-file-count": "2",
			},
			expectedFlags: []string{"--dir-mode", "0755", "--key-file", "/path/to/key", "--log-rotate-backup-file-count", "2"},
		},

		{
			name:          "TestMakeGcsfuseArgs with DebugFlags",
			opts:          map[string]string{"debug_fuse": "", "debug_gcs": ""},
			expectedFlags: []string{"--debug_fuse", "--debug_gcs"},
		},

		// Test ignored options
		{
			name:          "TestMakeGcsfuseArgs with IgnoredOptions",
			opts:          map[string]string{"user": "nobody", "_netdev": ""},
			expectedFlags: []string{},
		},

		{
			name:          "TestMakeGcsfuseArgs with RegularOptions",
			opts:          map[string]string{"allow_other": "", "ro": ""},
			expectedFlags: []string{"-o", "allow_other", "-o", "ro"},
		},

		{
			name: "TestMakeGcsfuseArgs with MixedOptions",
			opts: map[string]string{
				"implicit_dirs": "", "file_mode": "0644", "debug_fuse": "", "allow_other": "",
			},
			expectedFlags: []string{"--implicit-dirs=true", "--file-mode", "0644", "--debug_fuse", "-o", "allow_other"},
		},
	}
	device := "gcsfuse"
	mountPoint := "/mnt/gcs"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := makeGcsfuseArgs(device, mountPoint, tc.opts)

			if assert.Nil(t, err) {
				assert.ElementsMatch(t, args[:len(args)-2], tc.expectedFlags)
				assert.Equal(t, args[len(args)-2:], []string{device, mountPoint})
			}
		})
	}
}
