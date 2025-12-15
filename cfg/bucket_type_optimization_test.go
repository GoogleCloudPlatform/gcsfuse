// Copyright 2025 Google LLC
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

package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBucketTypeString(t *testing.T) {
	tests := []struct {
		name         string
		hierarchical bool
		zonal        bool
		expected     string
	}{
		{
			name:         "Hierarchical bucket",
			hierarchical: true,
			zonal:        false,
			expected:     "hierarchical",
		},
		{
			name:         "Zonal bucket",
			hierarchical: false,
			zonal:        true,
			expected:     "zonal",
		},
		{
			name:         "Standard bucket",
			hierarchical: false,
			zonal:        false,
			expected:     "standard",
		},
		{
			name:         "Hierarchical and zonal (zonal takes precedence for FUSE optimization)",
			hierarchical: true,
			zonal:        true,
			expected:     "zonal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBucketTypeString(tt.hierarchical, tt.zonal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyBucketTypeOptimizations_ZonalBucket(t *testing.T) {
	config := Config{
		FileSystem: FileSystemConfig{
			MaxBackground:       0,
			CongestionThreshold: 0,
			AsyncRead:           false,
		},
		DisableAutoconfig: false,
		Profile:           "",
	}

	optimizations := config.ApplyBucketTypeOptimizations("zonal", []string{})

	// Verify optimizations were applied
	assert.Len(t, optimizations, 3, "Should have 3 optimizations for zonal bucket")

	// Check max-background
	assert.Equal(t, int64(128), config.FileSystem.MaxBackground)
	assert.Contains(t, optimizations, "file-system.max-background")
	assert.Equal(t, int64(128), optimizations["file-system.max-background"].FinalValue)
	assert.True(t, optimizations["file-system.max-background"].Optimized)

	// Check congestion-threshold
	assert.Equal(t, int64(96), config.FileSystem.CongestionThreshold)
	assert.Contains(t, optimizations, "file-system.congestion-threshold")
	assert.Equal(t, int64(96), optimizations["file-system.congestion-threshold"].FinalValue)

	// Check async-read
	assert.True(t, config.FileSystem.AsyncRead)
	assert.Contains(t, optimizations, "file-system.async-read")
	assert.Equal(t, true, optimizations["file-system.async-read"].FinalValue)
}

func TestApplyBucketTypeOptimizations_HierarchicalBucket(t *testing.T) {
	config := Config{
		FileSystem: FileSystemConfig{
			MaxBackground:       0,
			CongestionThreshold: 0,
			AsyncRead:           false,
		},
		DisableAutoconfig: false,
		Profile:           "",
	}

	optimizations := config.ApplyBucketTypeOptimizations("hierarchical", []string{})

	// No optimizations should be applied for hierarchical bucket (only zonal has values)
	assert.Empty(t, optimizations, "Should have no optimizations for hierarchical bucket")

	// Values should remain at defaults
	assert.Equal(t, int64(0), config.FileSystem.MaxBackground)
	assert.Equal(t, int64(0), config.FileSystem.CongestionThreshold)
	assert.False(t, config.FileSystem.AsyncRead)
}

func TestApplyBucketTypeOptimizations_StandardBucket(t *testing.T) {
	config := Config{
		FileSystem: FileSystemConfig{
			MaxBackground:       0,
			CongestionThreshold: 0,
			AsyncRead:           false,
		},
		DisableAutoconfig: false,
		Profile:           "",
	}

	optimizations := config.ApplyBucketTypeOptimizations("standard", []string{})

	// No optimizations should be applied for standard bucket
	assert.Empty(t, optimizations, "Should have no optimizations for standard bucket")

	// Values should remain at defaults
	assert.Equal(t, int64(0), config.FileSystem.MaxBackground)
	assert.Equal(t, int64(0), config.FileSystem.CongestionThreshold)
	assert.False(t, config.FileSystem.AsyncRead)
}

func TestApplyBucketTypeOptimizations_DisabledAutoconfig(t *testing.T) {
	config := Config{
		FileSystem: FileSystemConfig{
			MaxBackground:       0,
			CongestionThreshold: 0,
			AsyncRead:           false,
		},
		DisableAutoconfig: true,
		Profile:           "",
	}

	optimizations := config.ApplyBucketTypeOptimizations("zonal", []string{})

	// No optimizations should be applied when autoconfig is disabled
	assert.Empty(t, optimizations, "Should have no optimizations when autoconfig is disabled")
	assert.Equal(t, int64(0), config.FileSystem.MaxBackground)
	assert.Equal(t, int64(0), config.FileSystem.CongestionThreshold)
	assert.False(t, config.FileSystem.AsyncRead)
}

func TestApplyBucketTypeOptimizations_ProfileSet(t *testing.T) {
	config := Config{
		FileSystem: FileSystemConfig{
			MaxBackground:       0,
			CongestionThreshold: 0,
			AsyncRead:           false,
		},
		DisableAutoconfig: false,
		Profile:           "aiml-training",
	}

	optimizations := config.ApplyBucketTypeOptimizations("zonal", []string{})

	// No optimizations should be applied when profile is set (profile takes precedence)
	assert.Empty(t, optimizations, "Should have no optimizations when profile is set")
	assert.Equal(t, int64(0), config.FileSystem.MaxBackground)
	assert.Equal(t, int64(0), config.FileSystem.CongestionThreshold)
	assert.False(t, config.FileSystem.AsyncRead)
}

func TestApplyBucketTypeOptimizations_UserSetFlags(t *testing.T) {
	config := Config{
		FileSystem: FileSystemConfig{
			MaxBackground:       256, // User set this
			CongestionThreshold: 0,
			AsyncRead:           false,
		},
		DisableAutoconfig: false,
		Profile:           "",
	}

	// Simulate user setting max-background
	userSetFlags := []string{"file-system.max-background"}
	optimizations := config.ApplyBucketTypeOptimizations("zonal", userSetFlags)

	// Only 2 optimizations should be applied (max-background skipped)
	assert.Len(t, optimizations, 2, "Should skip user-set flag")

	// User value should be preserved
	assert.Equal(t, int64(256), config.FileSystem.MaxBackground)
	assert.NotContains(t, optimizations, "file-system.max-background")

	// Other values should be optimized
	assert.Equal(t, int64(96), config.FileSystem.CongestionThreshold)
	assert.True(t, config.FileSystem.AsyncRead)
}

func TestApplyBucketTypeOptimizations_PartialUserSetFlags(t *testing.T) {
	config := Config{
		FileSystem: FileSystemConfig{
			MaxBackground:       0,
			CongestionThreshold: 200,  // User set this
			AsyncRead:           true, // User set this
		},
		DisableAutoconfig: false,
		Profile:           "",
	}

	// Simulate user setting some flags
	userSetFlags := []string{
		"file-system.congestion-threshold",
		"file-system.async-read",
	}
	optimizations := config.ApplyBucketTypeOptimizations("zonal", userSetFlags)

	// Only 1 optimization should be applied (max-background)
	assert.Len(t, optimizations, 1, "Should only optimize non-user-set flags")

	// Only max-background should be optimized
	assert.Equal(t, int64(128), config.FileSystem.MaxBackground)
	assert.Contains(t, optimizations, "file-system.max-background")

	// User values should be preserved
	assert.Equal(t, int64(200), config.FileSystem.CongestionThreshold)
	assert.True(t, config.FileSystem.AsyncRead)
	assert.NotContains(t, optimizations, "file-system.congestion-threshold")
	assert.NotContains(t, optimizations, "file-system.async-read")
}

func TestApplyBucketTypeOptimizations_EmptyBucketType(t *testing.T) {
	config := Config{
		FileSystem: FileSystemConfig{
			MaxBackground:       0,
			CongestionThreshold: 0,
			AsyncRead:           false,
		},
		DisableAutoconfig: false,
		Profile:           "",
	}

	optimizations := config.ApplyBucketTypeOptimizations("", []string{})

	// No optimizations for empty bucket type
	assert.Empty(t, optimizations, "Should have no optimizations for empty bucket type")
	assert.Equal(t, int64(0), config.FileSystem.MaxBackground)
	assert.Equal(t, int64(0), config.FileSystem.CongestionThreshold)
	assert.False(t, config.FileSystem.AsyncRead)
}
