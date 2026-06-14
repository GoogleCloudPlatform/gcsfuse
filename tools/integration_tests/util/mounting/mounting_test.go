// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mounting

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestGetMountDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mounting_test_dir")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tempBucketDir, err := os.MkdirTemp("", "bucket_mock_dir")
	assert.NoError(t, err)
	defer os.RemoveAll(tempBucketDir)

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

func TestConfigureReadAheadBypass(t *testing.T) {
	// If readAheadKB <= 0, should return nil immediately.
	err := ConfigureReadAhead("/non-existent-dir", -1)
	assert.NoError(t, err)

	err = ConfigureReadAhead("/non-existent-dir", 0)
	assert.NoError(t, err)

	// If runtime is not Linux, should return nil immediately.
	if runtime.GOOS != "linux" {
		err = ConfigureReadAhead("/non-existent-dir", 128)
		assert.NoError(t, err)
	}
}

func TestVerifyReadAheadBypass(t *testing.T) {
	// If runtime is not Linux, should return nil immediately.
	if runtime.GOOS != "linux" {
		err := VerifyReadAhead("/non-existent-dir", 128)
		assert.NoError(t, err)
	}
}

func TestConfigureAndVerifyReadAheadMock(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping test on non-Linux OS")
	}

	// Create temp directories
	tempBdiPrefix, err := os.MkdirTemp("", "mock_sysfs_bdi")
	assert.NoError(t, err)
	defer os.RemoveAll(tempBdiPrefix)

	tempMountDir, err := os.MkdirTemp("", "mock_mount_dir")
	assert.NoError(t, err)
	defer os.RemoveAll(tempMountDir)

	// Stat mount directory to get major and minor numbers
	var stat unix.Stat_t
	err = unix.Stat(tempMountDir, &stat)
	assert.NoError(t, err)

	major := unix.Major(stat.Dev)
	minor := unix.Minor(stat.Dev)

	// Create BDI subdirectory and the read_ahead_kb file
	mockBdiDir := fmt.Sprintf("%s/%d:%d", tempBdiPrefix, major, minor)
	err = os.MkdirAll(mockBdiDir, 0755)
	assert.NoError(t, err)

	// Override package-level sysfsBdiPrefix
	oldPrefix := sysfsBdiPrefix
	sysfsBdiPrefix = tempBdiPrefix
	defer func() {
		sysfsBdiPrefix = oldPrefix
	}()

	// Test ConfigureReadAhead (write path)
	expectedKB := 128
	err = ConfigureReadAhead(tempMountDir, expectedKB)
	assert.NoError(t, err)

	// Verify the mock file content directly
	mockBdiFile := mockBdiDir + "/read_ahead_kb"
	content, err := os.ReadFile(mockBdiFile)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%d\n", expectedKB), string(content))

	// Test VerifyReadAhead (read path)
	err = VerifyReadAhead(tempMountDir, expectedKB)
	assert.NoError(t, err)

	// VerifyReadAhead mismatch path
	err = VerifyReadAhead(tempMountDir, expectedKB+1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read-ahead setting mismatch")
}
