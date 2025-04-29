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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock IsValueSet for testing.
type mockIsValueSet struct {
	setFlags    map[string]bool
	boolFlags   map[string]bool
	stringFlags map[string]string
}

func (m *mockIsValueSet) IsValueSet(flag string) bool {
	return m.setFlags[flag]
}

func (m *mockIsValueSet) IsSet(flag string) bool {
	return m.setFlags[flag]
}

func (m *mockIsValueSet) GetBool(flag string) bool {
	return m.boolFlags[flag]
}

func (m *mockIsValueSet) GetString(flag string) string {
	return m.stringFlags[flag]
}

func (m *mockIsValueSet) Set(flag string) {
	m.setFlags[flag] = true
}

func (m *mockIsValueSet) SetString(flag string, value string) {
	m.stringFlags[flag] = value
}

func (m *mockIsValueSet) SetBool(flag string, value bool) {
	m.boolFlags[flag] = value
}

func (m *mockIsValueSet) Unset(flag string) {
	delete(m.setFlags, flag)
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

func TestGetMachineType_Success(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/n1-standard-1")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	machineType, err := getMachineType(&mockIsValueSet{})

	require.NoError(t, err)
	assert.Equal(t, "n1-standard-1", machineType)
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

	_, err := getMachineType(&mockIsValueSet{})

	assert.Error(t, err)
}

// Add a test wherein machine-type is set by the flag
// and getMachineType returns the same value
func TestGetMachineType_FlagIsSet(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a mockIsValueSet where machine-type is set.
	isSet := &mockIsValueSet{
		setFlags:    map[string]bool{"machine-type": true},
		stringFlags: map[string]string{"machine-type": "test-machine-type"},
	}

	machineType, err := getMachineType(isSet)

	require.NoError(t, err)
	assert.Equal(t, "test-machine-type", machineType)
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

	machineType, err := getMachineType(&mockIsValueSet{})

	require.NoError(t, err)
	assert.Equal(t, "n1-standard-1", machineType)
}
func TestOptimize_DisableAutoConfig(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{"disable-autoconfig": true}, boolFlags: map[string]bool{"disable-autoconfig": true}}

	_, err := Optimize(cfg, isSet)

	require.NoError(t, err)
	assert.False(t, cfg.Write.EnableStreamingWrites)
	assert.EqualValues(t, 0, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, 0, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 0, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.EqualValues(t, 0, cfg.MetadataCache.TypeCacheMaxSizeMb)
	assert.False(t, cfg.ImplicitDirs)
	assert.EqualValues(t, 0, cfg.FileSystem.RenameDirLimit)
}

func TestApplyMachineTypeOptimizations_MatchingMachineType(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	config := defaultOptimizationConfig
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	optimizedFlags, err := applyMachineTypeOptimizations(&config, cfg, isSet)

	require.NoError(t, err)
	assert.NotEmpty(t, optimizedFlags)
	assert.EqualValues(t, 0, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, -1, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 1024, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.EqualValues(t, 128, cfg.MetadataCache.TypeCacheMaxSizeMb)
	assert.True(t, cfg.ImplicitDirs)
	assert.EqualValues(t, 200000, cfg.FileSystem.RenameDirLimit)
}

func TestApplyMachineTypeOptimizations_NonMatchingMachineType(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a non-matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/n1-standard-1")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	config := defaultOptimizationConfig
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	optimizedFlags, err := applyMachineTypeOptimizations(&config, cfg, isSet)

	require.NoError(t, err)
	assert.Empty(t, optimizedFlags)
	assert.False(t, cfg.Write.EnableStreamingWrites)
}

func TestApplyMachineTypeOptimizations_UserSetFlag(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	config := defaultOptimizationConfig
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{"file-system.rename-dir-limit": true}}
	// Simulate setting config value by user
	cfg.FileSystem.RenameDirLimit = 10000

	optimizedFlags, err := applyMachineTypeOptimizations(&config, cfg, isSet)

	require.NoError(t, err)
	assert.NotEmpty(t, optimizedFlags)
	assert.EqualValues(t, 0, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, -1, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 1024, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.EqualValues(t, 128, cfg.MetadataCache.TypeCacheMaxSizeMb)
	assert.True(t, cfg.ImplicitDirs)
	assert.EqualValues(t, 10000, cfg.FileSystem.RenameDirLimit)
}

func TestApplyMachineTypeOptimizations_MissingFlagOverrideSet(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a machine type with a missing FlagOverrideSet.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	config := optimizationConfig{
		flagOverrideSets: []flagOverrideSet{}, // Empty FlagOverrideSets.
		machineTypes: []machineType{
			{
				names:               []string{"a3-highgpu-8g"},
				flagOverrideSetName: "high-performance",
			},
		},
	}
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	_, err := applyMachineTypeOptimizations(&config, cfg, isSet)

	require.NoError(t, err)
}

func TestApplyMachineTypeOptimizations_GetMachineTypeError(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns an error.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	config := defaultOptimizationConfig
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	_, err := applyMachineTypeOptimizations(&config, cfg, isSet)

	assert.NoError(t, err)
}

func TestApplyMachineTypeOptimizations_NoError(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	config := defaultOptimizationConfig
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	_, err := applyMachineTypeOptimizations(&config, cfg, isSet)

	assert.NoError(t, err)
}

func TestSetFlagValue_Bool(t *testing.T) {
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	err := setFlagValue(cfg, "implicit-dirs", flagOverride{newValue: true}, isSet)

	require.NoError(t, err)
	assert.True(t, cfg.ImplicitDirs)
}

func TestSetFlagValue_String(t *testing.T) {
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	err := setFlagValue(cfg, "app-name", flagOverride{newValue: "optimal_gcsfuse"}, isSet)

	require.NoError(t, err)
	assert.Equal(t, "optimal_gcsfuse", cfg.AppName)
}

func TestSetFlagValue_Int(t *testing.T) {
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	err := setFlagValue(cfg, "metadata-cache.stat-cache-max-size-mb", flagOverride{newValue: 1024}, isSet)

	require.NoError(t, err)
	assert.EqualValues(t, 1024, cfg.MetadataCache.StatCacheMaxSizeMb)
}

func TestSetFlagValue_InvalidFlagName(t *testing.T) {
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	err := setFlagValue(cfg, "invalid-flag", flagOverride{newValue: true}, isSet)

	assert.Error(t, err)
}

func TestApplyMachineTypeOptimizations_NoMachineTypes(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	config := optimizationConfig{
		flagOverrideSets: []flagOverrideSet{
			{
				name: "high-performance",
				overrides: map[string]flagOverride{
					"write.enable-streaming-writes": {newValue: true},
				},
			},
		},
		machineTypes: []machineType{},
	}
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	_, err := applyMachineTypeOptimizations(&config, cfg, isSet)

	require.NoError(t, err)
	// Check that no optimizations were applied as no machine mapping is set.
	assert.False(t, cfg.Write.EnableStreamingWrites)
}

func TestOptimize_Success(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-8g")
	})
	defer closeTestServer(t, server)
	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	optimizedFlags, err := Optimize(cfg, isSet)

	require.NoError(t, err)
	assert.True(t, isFlagPresent(optimizedFlags, "metadata-cache.negative-ttl-secs"))
	assert.EqualValues(t, 0, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, -1, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 1024, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.EqualValues(t, 128, cfg.MetadataCache.TypeCacheMaxSizeMb)
	assert.True(t, cfg.ImplicitDirs)
	assert.EqualValues(t, 200000, cfg.FileSystem.RenameDirLimit)
}
