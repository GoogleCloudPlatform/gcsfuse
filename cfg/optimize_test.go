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
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultConfig() Config {
	return Config{MetadataCache: MetadataCacheConfig{NegativeTtlSecs: 5, TtlSecs: 60, StatCacheMaxSizeMb: 33, TypeCacheMaxSizeMb: 4}, ImplicitDirs: false, FileSystem: FileSystemConfig{RenameDirLimit: 0}, Write: WriteConfig{EnableStreamingWrites: true}}
}

// Helper function to create a test server.
func createTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	return server
}

// Helper function to close a test server.
func closeTestServer(t *testing.T, server *httptest.Server) {
	t.Helper()
	server.Close()
}

// Helper function to reset metadataEndpoints.
func resetMetadataEndpoints(t *testing.T) {
	t.Helper()
	metadataEndpoints = []string{
		"http://metadata.google.internal/computeMetadata/v1/instance/machine-type",
	}
}

// Helper function to detect if a given flag is present in the map of optimized flags.
func isFlagPresentInOptimizationResults(optimizationResults map[string]OptimizationResult, flag string) bool {
	_, ok := optimizationResults[flag]
	return ok
}

func TestGetMachineType_Failure(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a non-200 status code.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	_, err := getMachineType(viper.New())

	assert.Error(t, err)
}

func TestGetMachineType_InputPrecedenceOrder(t *testing.T) {
	tests := []struct {
		name                string
		userSetFlags        map[string]string
		expectedMachineType string
	}{
		{
			name: "User_config_set_overrides_metadata",
			userSetFlags: map[string]string{
				"machine-type": "test-machine-type",
			},
			expectedMachineType: "test-machine-type",
		},
		{
			name:                "User_config_not_set_falls_back_to_metadata",
			userSetFlags:        map[string]string{},
			expectedMachineType: "n1-standard-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetMetadataEndpoints(t)
			// Create a test server that returns a machine type.
			server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "zones/us-central1-a/machineTypes/n1-standard-1")
			})
			defer closeTestServer(t, server)
			// Override metadataEndpoints for testing.
			metadataEndpoints = []string{server.URL}
			v := viper.New()
			for key, val := range tc.userSetFlags {
				v.Set(key, val)
			}

			machineType, err := getMachineType(v)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedMachineType, machineType)
		})
	}
}

func TestGetMachineType_QuotaError(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a quota error.
	retryCount := 0
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		if retryCount < maxRetries {
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			fmt.Fprint(w, "zones/us-central1-a/machineTypes/n1-standard-1")
		}
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	machineType, err := getMachineType(viper.New())

	require.NoError(t, err)
	assert.Equal(t, "n1-standard-1", machineType)
}

func TestApplyOptimizations_DisableAutoConfig(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	cfg := defaultConfig()
	cfg.DisableAutoconfig = true

	optimizedFlags := cfg.ApplyOptimizations(viper.New(), nil)

	require.Empty(t, optimizedFlags)
	assert.EqualValues(t, 5, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, 60, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 33, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.EqualValues(t, 4, cfg.MetadataCache.TypeCacheMaxSizeMb)
	assert.False(t, cfg.ImplicitDirs)
	assert.EqualValues(t, 0, cfg.FileSystem.RenameDirLimit)
}

func TestApplyOptimizations_MatchingMachineType(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	cfg := defaultConfig()

	optimizedFlags := cfg.ApplyOptimizations(viper.New(), nil)

	assert.NotEmpty(t, optimizedFlags)
	assert.EqualValues(t, 0, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, -1, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 1024, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.True(t, cfg.ImplicitDirs)
	assert.EqualValues(t, 200000, cfg.FileSystem.RenameDirLimit)
}

func TestApplyOptimizations_NonMatchingMachineType(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a non-matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/n1-standard-1")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	cfg := defaultConfig()

	optimizedFlags := cfg.ApplyOptimizations(viper.New(), nil)

	assert.Empty(t, optimizedFlags)
	assert.EqualValues(t, 5, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, 60, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 33, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.EqualValues(t, 4, cfg.MetadataCache.TypeCacheMaxSizeMb)
	assert.False(t, cfg.ImplicitDirs)
	assert.EqualValues(t, 0, cfg.FileSystem.RenameDirLimit)
}

func TestApplyOptimizations_UserSetFlag(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	cfg := defaultConfig()
	v := viper.New()
	v.Set("file-system.rename-dir-limit", 10000)
	// Simulate setting config value by user
	cfg.FileSystem.RenameDirLimit = 10000

	optimizedFlags := cfg.ApplyOptimizations(v, nil)

	assert.NotEmpty(t, optimizedFlags)
	assert.EqualValues(t, 0, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, -1, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 1024, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.True(t, cfg.ImplicitDirs)
	assert.EqualValues(t, 10000, cfg.FileSystem.RenameDirLimit)
}

func TestApplyOptimizations_GetMachineTypeError(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns an error.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	cfg := defaultConfig()

	optimizedFlags := cfg.ApplyOptimizations(viper.New(), nil)

	assert.Empty(t, optimizedFlags)
	assert.EqualValues(t, 5, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, 60, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 33, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.EqualValues(t, 4, cfg.MetadataCache.TypeCacheMaxSizeMb)
	assert.False(t, cfg.ImplicitDirs)
	assert.EqualValues(t, 0, cfg.FileSystem.RenameDirLimit)
}

func TestApplyOptimizations_NoError(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	cfg := defaultConfig()

	optimizedFlags := cfg.ApplyOptimizations(viper.New(), nil)

	assert.NotEmpty(t, optimizedFlags)
}

func TestApplyOptimizations_Success(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	cfg := defaultConfig()

	optimizedFlags := cfg.ApplyOptimizations(viper.New(), nil)

	assert.True(t, isFlagPresentInOptimizationResults(optimizedFlags, "write.global-max-blocks"))
	assert.EqualValues(t, 1600, cfg.Write.GlobalMaxBlocks)
	assert.True(t, isFlagPresentInOptimizationResults(optimizedFlags, "metadata-cache.negative-ttl-secs"))
	assert.EqualValues(t, 0, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, -1, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 1024, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.True(t, cfg.ImplicitDirs)
	assert.EqualValues(t, 200000, cfg.FileSystem.RenameDirLimit)
}

func TestApplyOptimizations_ConditionalBucketType_PirloWithConditions(t *testing.T) {
	t.Run("Conditions_Met_Optimizes_Write_Params", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.Write.EnableRapidWrites = true
		cfg.Write.EnableRapidAppends = false
		cfg.Write.BlockSizeMb = 32
		cfg.Write.MaxBlocksPerFile = 1
		cfg.Write.GlobalMaxBlocks = 4

		v := viper.New()
		optimizedFlags := cfg.ApplyOptimizations(v, &OptimizationInput{BucketType: BucketTypePirlo})

		assert.Contains(t, optimizedFlags, "write.block-size-mb")
		assert.EqualValues(t, 1.0, cfg.Write.BlockSizeMb)
		assert.Equal(t, `bucket-type "pirlo" with matching conditions`, optimizedFlags["write.block-size-mb"].OptimizationReason)

		assert.Contains(t, optimizedFlags, "write.max-blocks-per-file")
		assert.EqualValues(t, 4, cfg.Write.MaxBlocksPerFile)
		assert.Equal(t, `bucket-type "pirlo" with matching conditions`, optimizedFlags["write.max-blocks-per-file"].OptimizationReason)

		assert.Contains(t, optimizedFlags, "write.global-max-blocks")
		assert.EqualValues(t, 16, cfg.Write.GlobalMaxBlocks)
		assert.Equal(t, `bucket-type "pirlo" with matching conditions`, optimizedFlags["write.global-max-blocks"].OptimizationReason)
	})

	t.Run("Conditions_Not_Met_RapidWrites_False", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.Write.EnableRapidWrites = false
		cfg.Write.EnableRapidAppends = false
		cfg.Write.BlockSizeMb = 32
		cfg.Write.MaxBlocksPerFile = 1
		cfg.Write.GlobalMaxBlocks = 4

		v := viper.New()
		optimizedFlags := cfg.ApplyOptimizations(v, &OptimizationInput{BucketType: BucketTypePirlo})

		assert.NotContains(t, optimizedFlags, "write.block-size-mb")
		assert.EqualValues(t, 32, cfg.Write.BlockSizeMb)
		assert.NotContains(t, optimizedFlags, "write.max-blocks-per-file")
		assert.EqualValues(t, 1, cfg.Write.MaxBlocksPerFile)
		assert.NotContains(t, optimizedFlags, "write.global-max-blocks")
		assert.EqualValues(t, 4, cfg.Write.GlobalMaxBlocks)
	})

	t.Run("Conditions_Not_Met_RapidAppends_True", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.Write.EnableRapidWrites = true
		cfg.Write.EnableRapidAppends = true
		cfg.Write.BlockSizeMb = 32
		cfg.Write.MaxBlocksPerFile = 1
		cfg.Write.GlobalMaxBlocks = 4

		v := viper.New()
		optimizedFlags := cfg.ApplyOptimizations(v, &OptimizationInput{BucketType: BucketTypePirlo})

		assert.NotContains(t, optimizedFlags, "write.block-size-mb")
		assert.EqualValues(t, 32, cfg.Write.BlockSizeMb)
		assert.NotContains(t, optimizedFlags, "write.max-blocks-per-file")
		assert.EqualValues(t, 1, cfg.Write.MaxBlocksPerFile)
		assert.NotContains(t, optimizedFlags, "write.global-max-blocks")
		assert.EqualValues(t, 4, cfg.Write.GlobalMaxBlocks)
	})

	t.Run("Non_Pirlo_Bucket_No_Optimization", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.Write.EnableRapidWrites = true
		cfg.Write.EnableRapidAppends = false
		cfg.Write.BlockSizeMb = 32
		cfg.Write.MaxBlocksPerFile = 1
		cfg.Write.GlobalMaxBlocks = 4

		v := viper.New()
		optimizedFlags := cfg.ApplyOptimizations(v, &OptimizationInput{BucketType: BucketTypeZonal})

		assert.NotContains(t, optimizedFlags, "write.block-size-mb")
		assert.EqualValues(t, 32, cfg.Write.BlockSizeMb)
		assert.NotContains(t, optimizedFlags, "write.max-blocks-per-file")
		assert.EqualValues(t, 1, cfg.Write.MaxBlocksPerFile)
		assert.NotContains(t, optimizedFlags, "write.global-max-blocks")
		assert.EqualValues(t, 4, cfg.Write.GlobalMaxBlocks)
	})

	t.Run("User_Set_Flag_Takes_Precedence", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.Write.EnableRapidWrites = true
		cfg.Write.EnableRapidAppends = false
		cfg.Write.BlockSizeMb = 8.0
		cfg.Write.MaxBlocksPerFile = 1
		cfg.Write.GlobalMaxBlocks = 4

		v := viper.New()
		v.Set("write.block-size-mb", 8.0)

		optimizedFlags := cfg.ApplyOptimizations(v, &OptimizationInput{BucketType: BucketTypePirlo})

		assert.NotContains(t, optimizedFlags, "write.block-size-mb")
		assert.EqualValues(t, 8.0, cfg.Write.BlockSizeMb)
		// Other flags should still be optimized
		assert.Contains(t, optimizedFlags, "write.max-blocks-per-file")
		assert.EqualValues(t, 4, cfg.Write.MaxBlocksPerFile)
		assert.Contains(t, optimizedFlags, "write.global-max-blocks")
		assert.EqualValues(t, 16, cfg.Write.GlobalMaxBlocks)
	})

	t.Run("DisableAutoconfig_Skips_Optimization", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.DisableAutoconfig = true
		cfg.Write.EnableRapidWrites = true
		cfg.Write.EnableRapidAppends = false
		cfg.Write.BlockSizeMb = 32
		cfg.Write.MaxBlocksPerFile = 1
		cfg.Write.GlobalMaxBlocks = 4

		v := viper.New()
		optimizedFlags := cfg.ApplyOptimizations(v, &OptimizationInput{BucketType: BucketTypePirlo})

		assert.Empty(t, optimizedFlags)
		assert.EqualValues(t, 32, cfg.Write.BlockSizeMb)
		assert.EqualValues(t, 1, cfg.Write.MaxBlocksPerFile)
		assert.EqualValues(t, 4, cfg.Write.GlobalMaxBlocks)
	})

	t.Run("High_Performance_Machine_Precedence_For_GlobalMaxBlocks", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.Write.EnableRapidWrites = true
		cfg.Write.EnableRapidAppends = false
		cfg.Write.BlockSizeMb = 32
		cfg.Write.MaxBlocksPerFile = 1
		cfg.Write.GlobalMaxBlocks = 4

		v := viper.New()
		v.Set("machine-type", "a3-highgpu-8g")

		optimizedFlags := cfg.ApplyOptimizations(v, &OptimizationInput{BucketType: BucketTypePirlo})

		// Machine-type optimization takes precedence for global-max-blocks: 1600 instead of 16
		assert.Contains(t, optimizedFlags, "write.global-max-blocks")
		assert.EqualValues(t, 1600, cfg.Write.GlobalMaxBlocks)
		assert.Equal(t, `machine-type group "high-performance"`, optimizedFlags["write.global-max-blocks"].OptimizationReason)

		// Bucket-type optimizations still apply to the other write parameters
		assert.Contains(t, optimizedFlags, "write.block-size-mb")
		assert.EqualValues(t, 1.0, cfg.Write.BlockSizeMb)
		assert.Contains(t, optimizedFlags, "write.max-blocks-per-file")
		assert.EqualValues(t, 4, cfg.Write.MaxBlocksPerFile)
	})
}

func TestGetAndSetConfigValueByPath(t *testing.T) {
	cfg := defaultConfig()

	// Test getConfigValueByPath
	val, ok := getConfigValueByPath(&cfg, "write.enable-streaming-writes")
	assert.True(t, ok)
	assert.Equal(t, true, val)

	val, ok = getConfigValueByPath(&cfg, "non.existent.path")
	assert.False(t, ok)
	assert.Nil(t, val)

	// Test setConfigValueByPath
	err := setConfigValueByPath(&cfg, "write.block-size-mb", 64.0)
	assert.NoError(t, err)
	assert.Equal(t, 64.0, cfg.Write.BlockSizeMb)

	err = setConfigValueByPath(&cfg, "write.global-max-blocks", int64(100))
	assert.NoError(t, err)
	assert.EqualValues(t, 100, cfg.Write.GlobalMaxBlocks)

	err = setConfigValueByPath(&cfg, "non.existent.path", true)
	assert.Error(t, err)
}

func TestValuesMatch(t *testing.T) {
	assert.True(t, valuesMatch(true, true))
	assert.True(t, valuesMatch(false, false))
	assert.False(t, valuesMatch(true, false))

	assert.True(t, valuesMatch("hello", "hello"))
	assert.False(t, valuesMatch("hello", "world"))

	// Cross-type numeric comparisons
	assert.True(t, valuesMatch(int(10), int64(10)))
	assert.True(t, valuesMatch(int64(10), int(10)))
	assert.True(t, valuesMatch(float64(1.0), int(1)))
	assert.True(t, valuesMatch(int(1), float64(1.0)))
	assert.True(t, valuesMatch(uint(5), int64(5)))
	assert.True(t, valuesMatch(int64(5), uint(5)))
	assert.False(t, valuesMatch(int(-5), uint(5)))
	assert.False(t, valuesMatch(int(10), int(20)))
}

func TestCreateHierarchicalOptimizedFlags_Positive(t *testing.T) {
	testCases := []struct {
		name     string
		inputMap map[string]OptimizationResult
		expected map[string]any
	}{
		{
			name:     "Empty map",
			inputMap: map[string]OptimizationResult{},
			expected: map[string]any{},
		},
		{
			name: "Flat keys",
			inputMap: map[string]OptimizationResult{
				"key1": {FinalValue: "value1", OptimizationReason: "reason1"},
				"key2": {FinalValue: 123, OptimizationReason: "reason2"},
			},
			expected: map[string]any{
				"key1": OptimizationResult{FinalValue: "value1", OptimizationReason: "reason1"},
				"key2": OptimizationResult{FinalValue: 123, OptimizationReason: "reason2"},
			},
		},
		{
			name: "Single level nesting",
			inputMap: map[string]OptimizationResult{
				"a.b": {FinalValue: "valueAB", OptimizationReason: "reasonAB"},
				"a.c": {FinalValue: "valueAC", OptimizationReason: "reasonAC"},
			},
			expected: map[string]any{
				"a": map[string]any{
					"b": OptimizationResult{FinalValue: "valueAB", OptimizationReason: "reasonAB"},
					"c": OptimizationResult{FinalValue: "valueAC", OptimizationReason: "reasonAC"},
				},
			},
		},
		{
			name: "Multi-level nesting",
			inputMap: map[string]OptimizationResult{
				"a.b.c": {FinalValue: "valueABC", OptimizationReason: "reasonABC"},
				"a.b.d": {FinalValue: "valueABD", OptimizationReason: "reasonABD"},
				"x.y.z": {FinalValue: true, OptimizationReason: "reasonXYZ"},
			},
			expected: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": OptimizationResult{FinalValue: "valueABC", OptimizationReason: "reasonABC"},
						"d": OptimizationResult{FinalValue: "valueABD", OptimizationReason: "reasonABD"},
					},
				},
				"x": map[string]any{
					"y": map[string]any{
						"z": OptimizationResult{FinalValue: true, OptimizationReason: "reasonXYZ"},
					},
				},
			},
		},
		{
			name: "No conflict complex keys",
			inputMap: map[string]OptimizationResult{
				"metadata-cache.ttl-secs":               {FinalValue: int64(-1), OptimizationReason: "reasonTTL"},
				"metadata-cache.stat-cache-max-size-mb": {FinalValue: int64(1024), OptimizationReason: "reasonStat"},
				"file-cache.cache-file-for-range-read":  {FinalValue: true, OptimizationReason: "reasonFileCache"},
			},
			expected: map[string]any{
				"metadata-cache": map[string]any{
					"ttl-secs":               OptimizationResult{FinalValue: int64(-1), OptimizationReason: "reasonTTL"},
					"stat-cache-max-size-mb": OptimizationResult{FinalValue: int64(1024), OptimizationReason: "reasonStat"},
				},
				"file-cache": map[string]any{
					"cache-file-for-range-read": OptimizationResult{FinalValue: true, OptimizationReason: "reasonFileCache"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CreateHierarchicalOptimizedFlags(tc.inputMap)

			assert.NoError(t, err)
			if !reflect.DeepEqual(tc.expected, got) {
				t.Errorf("CreateHierarchicalOptimizedFlags() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestCreateHierarchicalOptimizedFlags_Negative(t *testing.T) {
	testCases := []struct {
		name     string
		inputMap map[string]OptimizationResult
	}{
		{
			name: "Conflict: Prefix as terminal key first",
			inputMap: map[string]OptimizationResult{
				"a.b":   {FinalValue: "valAB", OptimizationReason: "rAB"},
				"a.b.d": {FinalValue: "valABD", OptimizationReason: "rABD"},
			},
		},
		{
			name: "Conflict: Path key first",
			inputMap: map[string]OptimizationResult{
				"a.b.d": {FinalValue: "valABD", OptimizationReason: "rABD"},
				"a.b":   {FinalValue: "valAB", OptimizationReason: "rAB"},
			},
		},
		{
			name: "Conflict: Deeper nesting",
			inputMap: map[string]OptimizationResult{
				"a.b.c":   {FinalValue: "valABC", OptimizationReason: "rABC"},
				"a.b.c.d": {FinalValue: "valABCD", OptimizationReason: "rABCD"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CreateHierarchicalOptimizedFlags(tc.inputMap)

			assert.Error(t, err)
			assert.Nil(t, got)
			assert.Contains(t, err.Error(), "key conflict")
		})
	}
}
