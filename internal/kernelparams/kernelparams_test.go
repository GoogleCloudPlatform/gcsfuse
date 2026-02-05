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

package kernelparams

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAtomicFileWrite(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	data := []byte("hello world")

	err := atomicFileWrite(filePath, data)
	assert.NoError(t, err)

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, data, content)
}

func TestGetDeviceMajorMinor_ValidPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping test on non-linux OS")
	}
	tempDir := t.TempDir()

	major, minor, err := getDeviceMajorMinor(tempDir)

	assert.NoError(t, err)
	t.Logf("Device major: %d, minor: %d for %s", major, minor, tempDir)
}

func TestGetDeviceMajorMinor_NonExistentPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping test on non-linux OS")
	}
	path := "/path/that/does/not/exist"

	_, _, err := getDeviceMajorMinor(path)

	assert.Error(t, err)
}

func TestPathForParam(t *testing.T) {
	tests := []struct {
		name      ParamName
		major     uint32
		minor     uint32
		expected  string
		expectErr bool
	}{
		{MaxReadAheadKb, 1, 2, "/sys/class/bdi/1:2/read_ahead_kb", false},
		{MaxBackgroundRequests, 1, 2, "/sys/fs/fuse/connections/2/max_background", false},
		{CongestionWindowThreshold, 1, 2, "/sys/fs/fuse/connections/2/congestion_threshold", false},
		{MaxPagesLimit, 1, 2, "/sys/module/fuse/parameters/max_pages_limit", false},
		{TransparentHugePages, 1, 2, "/sys/kernel/mm/transparent_hugepage/enabled", false},
		{"unknown", 1, 2, "", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {

			path, err := pathForParam(tt.name, tt.major, tt.minor)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, path)
			}
		})
	}
}

func TestSetMaxPagesLimit_NewValue(t *testing.T) {
	cfg := NewKernelParamsManager()

	cfg.SetMaxPagesLimit(123)

	assert.Len(t, cfg.Parameters, 1)
	assert.Equal(t, MaxPagesLimit, cfg.Parameters[0].Name)
	assert.Equal(t, "123", cfg.Parameters[0].Value)
}

func TestSetMaxPagesLimit_UpdateValue(t *testing.T) {
	cfg := NewKernelParamsManager()
	cfg.SetMaxPagesLimit(123)

	cfg.SetMaxPagesLimit(456)

	assert.Len(t, cfg.Parameters, 1)
	assert.Equal(t, MaxPagesLimit, cfg.Parameters[0].Name)
	assert.Equal(t, "456", cfg.Parameters[0].Value)
}

func TestSetTransparentHugePages_NewValue(t *testing.T) {
	cfg := NewKernelParamsManager()

	cfg.SetTransparentHugePages("always")

	assert.Len(t, cfg.Parameters, 1)
	assert.Equal(t, TransparentHugePages, cfg.Parameters[0].Name)
	assert.Equal(t, "always", cfg.Parameters[0].Value)
}

func TestSetTransparentHugePages_UpdateValue(t *testing.T) {
	cfg := NewKernelParamsManager()
	cfg.SetTransparentHugePages("always")

	cfg.SetTransparentHugePages("never")

	assert.Len(t, cfg.Parameters, 1)
	assert.Equal(t, TransparentHugePages, cfg.Parameters[0].Name)
	assert.Equal(t, "never", cfg.Parameters[0].Value)
}

func TestSetReadAheadKb(t *testing.T) {
	cfg := NewKernelParamsManager()

	cfg.SetReadAheadKb(1024)

	assert.Len(t, cfg.Parameters, 1)
	assert.Equal(t, MaxReadAheadKb, cfg.Parameters[0].Name)
	assert.Equal(t, "1024", cfg.Parameters[0].Value)
}

func TestSetMaxBackgroundRequests(t *testing.T) {
	cfg := NewKernelParamsManager()

	cfg.SetMaxBackgroundRequests(12)

	assert.Len(t, cfg.Parameters, 1)
	assert.Equal(t, MaxBackgroundRequests, cfg.Parameters[0].Name)
	assert.Equal(t, "12", cfg.Parameters[0].Value)
}

func TestSetCongestionWindowThreshold(t *testing.T) {
	cfg := NewKernelParamsManager()

	cfg.SetCongestionWindowThreshold(9)

	assert.Len(t, cfg.Parameters, 1)
	assert.Equal(t, CongestionWindowThreshold, cfg.Parameters[0].Name)
	assert.Equal(t, "9", cfg.Parameters[0].Value)
}

func TestSetMultipleKernelParams(t *testing.T) {
	cfg := NewKernelParamsManager()

	cfg.SetMaxPagesLimit(123)
	cfg.SetTransparentHugePages("always")
	cfg.SetReadAheadKb(456)

	assert.Len(t, cfg.Parameters, 3)
	// Verify values
	expected := map[ParamName]string{
		MaxPagesLimit:        "123",
		TransparentHugePages: "always",
		MaxReadAheadKb:       "456",
	}
	for _, p := range cfg.Parameters {
		val, ok := expected[p.Name]
		assert.True(t, ok)
		assert.Equal(t, val, p.Value)
	}
}

func TestApplyGKE(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "kernel_params.json")
	cfg := NewKernelParamsManager()
	cfg.SetReadAheadKb(1024)

	cfg.ApplyGKE(filePath)

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	var actualCfg KernelParamsConfig
	err = json.Unmarshal(content, &actualCfg)
	assert.NoError(t, err)
	assert.NotEmpty(t, actualCfg.RequestID)
	assert.NotEmpty(t, actualCfg.Timestamp)
	assert.Len(t, actualCfg.Parameters, 1)
	assert.Equal(t, MaxReadAheadKb, actualCfg.Parameters[0].Name)
	assert.Equal(t, "1024", actualCfg.Parameters[0].Value)
}

func TestApplyGKE_EmptyParams(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "kernel_params.json")
	cfg := NewKernelParamsManager()

	cfg.ApplyGKE(filePath)

	_, err := os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))
}

func TestWriteValue_DirectWriteSuccess(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping test on non-linux OS")
	}
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "test_file")
	value := "100"

	err := writeValue(path, value)

	assert.NoError(t, err)
	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "100\n", string(content))
}

func TestWriteValue_DirectWriteFailure_NoSudo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping test on non-linux OS")
	}
	// Use a path in a non-existent directory to trigger a non-permission error
	path := filepath.Join(t.TempDir(), "missing_dir", "test_file")
	value := "100"

	err := writeValue(path, value)

	assert.Error(t, err)
	// Should not be a sudo error, but the original fs error
	assert.NotContains(t, err.Error(), "sudo error")
}

func TestWriteValue_PermissionDenied_SudoFallback(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping test on non-linux OS")
	}
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "readonly_file")
	// Create file and make it read-only to trigger permission error
	err := os.WriteFile(path, []byte("initial"), 0444)
	assert.NoError(t, err)

	err = writeValue(path, "new_value")

	// This depends on the environment.
	// If sudo works, err is nil. If not, err contains "sudo error".
	if err == nil {
		content, _ := os.ReadFile(path)
		assert.Equal(t, "new_value\n", string(content))
	} else {
		assert.Contains(t, err.Error(), "sudo error")
	}
}
