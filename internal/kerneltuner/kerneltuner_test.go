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

package kerneltuner

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestPathForParam(t *testing.T) {
	tests := []struct {
		name      ParamName
		major     uint32
		minor     uint32
		expected  string
		expectErr bool
	}{
		{ReadAheadKb, 1, 2, "/sys/class/bdi/1:2/read_ahead_kb", false},
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

func TestKernelParamsConfig_Setters(t *testing.T) {
	cfg := NewKernelParamsConfig()

	cfg.SetMaxPagesLimit(123)
	assert.Contains(t, cfg.Parameters, KernelParam{Name: MaxPagesLimit, Value: "123"})

	cfg.SetTransparentHugePages("always")
	assert.Contains(t, cfg.Parameters, KernelParam{Name: TransparentHugePages, Value: "always"})

	cfg.SetReadAheadKb(456)
	assert.Contains(t, cfg.Parameters, KernelParam{Name: ReadAheadKb, Value: "456"})

	cfg.SetMaxBackgroundRequests(789)
	assert.Contains(t, cfg.Parameters, KernelParam{Name: MaxBackgroundRequests, Value: "789"})

	cfg.SetCongestionWindowThreshold(101)
	assert.Contains(t, cfg.Parameters, KernelParam{Name: CongestionWindowThreshold, Value: "101"})
}

func TestApplyGKE(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "kernel_params.json")
	cfg := NewKernelParamsConfig()
	cfg.SetReadAheadKb(1024)

	cfg.ApplyGKE(filePath)

	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)

	var actualCfg KernelParamsConfig
	err = json.Unmarshal(content, &actualCfg)
	assert.NoError(t, err)

	assert.Equal(t, CurrentContractVersion, actualCfg.Version)
	assert.NotEmpty(t, actualCfg.RequestID)
	assert.NotEmpty(t, actualCfg.Timestamp)
	assert.Len(t, actualCfg.Parameters, 1)
	assert.Equal(t, ReadAheadKb, actualCfg.Parameters[0].Name)
	assert.Equal(t, "1024", actualCfg.Parameters[0].Value)
}
