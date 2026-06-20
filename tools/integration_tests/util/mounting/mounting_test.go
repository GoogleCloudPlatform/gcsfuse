// Copyright 2026 Google LLC
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

package mounting

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMountDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mounting_test_dir")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	tempBucketDir, err := os.MkdirTemp("", "bucket_mock_dir")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempBucketDir) }()

	testCases := []struct {
		name     string
		flags    []string
		expected string
	}{
		{
			name:     "standard mount flags with bucket and mount point",
			flags:    []string{"--log-severity=trace", "--log-file=/tmp/gcsfuse.log", "my-bucket", tempDir},
			expected: tempDir,
		},
		{
			name:     "dynamic mount flags",
			flags:    []string{"--log-severity=trace", "--log-file=/tmp/gcsfuse.log", tempDir},
			expected: tempDir,
		},
		{
			name:     "persistent mount flags",
			flags:    []string{"my-bucket", tempDir, "-o", "log_severity=trace", "-o", "log_file=/tmp/gcsfuse.log"},
			expected: tempDir,
		},
		{
			name:     "both bucket and mount point exist as directories",
			flags:    []string{"--log-severity=trace", tempBucketDir, tempDir},
			expected: tempDir,
		},
		{
			name:     "no directories in flags",
			flags:    []string{"--log-severity=trace", "my-bucket", "not-a-real-directory-12345"},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := getMountDir(tc.flags)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
