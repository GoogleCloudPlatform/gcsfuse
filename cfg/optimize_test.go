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

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/shared"
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

func TestConditionalBucketTypeOptimizations(t *testing.T) {
	blockSizeRules := &shared.OptimizationRules{
		BucketTypeOptimization: []shared.BucketTypeOptimization{
			{
				BucketTypes: shared.BucketTypeList{"rcu"},
				Conditions: map[string]any{
					"write.rapid-write": "enabled",
				},
				Value: float64(4.0),
			},
		},
	}
	maxBlocksPerFileRules := &shared.OptimizationRules{
		BucketTypeOptimization: []shared.BucketTypeOptimization{
			{
				BucketTypes: shared.BucketTypeList{"rcu"},
				Conditions: map[string]any{
					"write.rapid-write": "enabled",
				},
				Value: int64(4),
			},
		},
	}

	testCases := []struct {
		name                     string
		rapidWrite               RapidWriteStrategy
		bucketType               BucketType
		expectedOptimized        bool
		expectedBlockSize        float64
		expectedMaxBlocksPerFile int64
		expectedReason           string
	}{
		{
			name:                     "Conditions_Met_Optimizes_Write_Params",
			rapidWrite:               RapidWriteStrategyEnabled,
			bucketType:               BucketTypeRCU,
			expectedOptimized:        true,
			expectedBlockSize:        4.0,
			expectedMaxBlocksPerFile: 4,
			expectedReason:           `bucket-type "rcu" with matching conditions`,
		},
		{
			name:                     "Conditions_Not_Met_RapidWrite_Disabled",
			rapidWrite:               RapidWriteStrategyDisabled,
			bucketType:               BucketTypeRCU,
			expectedOptimized:        false,
			expectedBlockSize:        32,
			expectedMaxBlocksPerFile: 1,
		},
		{
			name:                     "Non_RCU_Bucket_No_Optimization",
			rapidWrite:               RapidWriteStrategyEnabled,
			bucketType:               BucketTypeZonal,
			expectedOptimized:        false,
			expectedBlockSize:        32,
			expectedMaxBlocksPerFile: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := defaultConfig()
			cfg.Write.RapidWrite = tc.rapidWrite
			cfg.Write.BlockSizeMb = 32
			cfg.Write.MaxBlocksPerFile = 1
			input := &OptimizationInput{BucketType: tc.bucketType}

			blockSizeRes := getOptimizedValue(blockSizeRules, cfg.Write.BlockSizeMb, "", "", input, machineTypeToGroupMap, &cfg)
			maxBlocksRes := getOptimizedValue(maxBlocksPerFileRules, cfg.Write.MaxBlocksPerFile, "", "", input, machineTypeToGroupMap, &cfg)

			assert.Equal(t, tc.expectedOptimized, blockSizeRes.Optimized)
			assert.EqualValues(t, tc.expectedBlockSize, blockSizeRes.FinalValue)
			assert.Equal(t, tc.expectedReason, blockSizeRes.OptimizationReason)
			assert.Equal(t, tc.expectedOptimized, maxBlocksRes.Optimized)
			assert.EqualValues(t, tc.expectedMaxBlocksPerFile, maxBlocksRes.FinalValue)
			assert.Equal(t, tc.expectedReason, maxBlocksRes.OptimizationReason)
		})
	}
}

func TestMatchesConditions(t *testing.T) {
	cfg := defaultConfig()
	cfg.Write.RapidWrite = RapidWriteStrategyEnabled

	assert.True(t, matchesConditions(&cfg, nil))
	assert.True(t, matchesConditions(&cfg, map[string]any{}))
	assert.False(t, matchesConditions(nil, map[string]any{"write.rapid-write": "enabled"}))
	assert.True(t, matchesConditions(&cfg, map[string]any{"write.rapid-write": "enabled"}))
	assert.False(t, matchesConditions(&cfg, map[string]any{"write.rapid-write": "disabled"}))
	assert.False(t, matchesConditions(&cfg, map[string]any{"non.existent": true}))
}

func TestGetConfigFieldByPath(t *testing.T) {
	cfg := defaultConfig()
	cfg.Write.RapidWrite = RapidWriteStrategyEnabled
	var nilCfg *Config

	testCases := []struct {
		name        string
		config      any
		path        string
		expectedOk  bool
		expectedVal any
	}{
		{
			name:        "pointer_config_bool_field",
			config:      &cfg,
			path:        "write.enable-streaming-writes",
			expectedOk:  true,
			expectedVal: true,
		},
		{
			name:        "value_config_string_field",
			config:      cfg,
			path:        "write.rapid-write",
			expectedOk:  true,
			expectedVal: RapidWriteStrategyEnabled,
		},
		{
			name:   "typed_nil_config",
			config: nilCfg,
			path:   "write.rapid-write",
		},
		{
			name:   "untyped_nil_config",
			config: nil,
			path:   "write.rapid-write",
		},
		{
			name:   "empty_path",
			config: &cfg,
			path:   "",
		},
		{
			name:   "non_struct_intermediate_field",
			config: &cfg,
			path:   "write.rapid-write.invalid-subpath",
		},
		{
			name:   "non_existent_path",
			config: &cfg,
			path:   "non.existent.path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val, ok := getConfigFieldByPath(tc.config, tc.path)

			assert.Equal(t, tc.expectedOk, ok)
			if tc.expectedOk {
				assert.Equal(t, tc.expectedVal, val.Interface())
			} else {
				assert.False(t, val.IsValid())
			}
		})
	}
}

func TestValuesMatch(t *testing.T) {
	assert.True(t, valuesMatch(true, true))
	assert.False(t, valuesMatch(true, false))

	// Custom string type vs string literal
	assert.True(t, valuesMatch(RapidWriteStrategyEnabled, "enabled"))
	assert.False(t, valuesMatch(RapidWriteStrategyDisabled, "enabled"))

	// Cross-type numeric comparisons
	assert.True(t, valuesMatch(int64(10), int(10)))
	assert.True(t, valuesMatch(float64(4.0), int(4)))

	// Nil and incompatible types
	assert.True(t, valuesMatch(nil, nil))
	assert.False(t, valuesMatch(nil, "enabled"))
	assert.False(t, valuesMatch(10, "enabled"))
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
