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

func defaultConfig() Config {
	return Config{MetadataCache: MetadataCacheConfig{NegativeTtlSecs: 5, TtlSecs: 60, StatCacheMaxSizeMb: 33, TypeCacheMaxSizeMb: 4}, ImplicitDirs: false, FileSystem: FileSystemConfig{RenameDirLimit: 0}, Write: WriteConfig{EnableStreamingWrites: true}}
}

// Mock IsValueSet for testing.
type mockIsValueSet struct {
	setFlags    map[string]bool
	boolFlags   map[string]bool
	stringFlags map[string]string
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

// Helper function to detect if a given flag is present in the map of optimized flags.
func isFlagPresentInOptimizationResults(optimizationResults map[string]OptimizationResult, flag string) bool {
	_, ok := optimizationResults[flag]
	return ok
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
	isSet := &mockIsValueSet{}

	optimizedFlags := cfg.ApplyOptimizations(isSet)

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
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	optimizedFlags := cfg.ApplyOptimizations(isSet)

	assert.NotEmpty(t, optimizedFlags)
	assert.EqualValues(t, 0, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, -1, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 1024, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.EqualValues(t, 128, cfg.MetadataCache.TypeCacheMaxSizeMb)
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
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	optimizedFlags := cfg.ApplyOptimizations(isSet)

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
	isSet := &mockIsValueSet{setFlags: map[string]bool{"rename-dir-limit": true}}
	// Simulate setting config value by user
	cfg.FileSystem.RenameDirLimit = 10000

	optimizedFlags := cfg.ApplyOptimizations(isSet)

	assert.NotEmpty(t, optimizedFlags)
	assert.EqualValues(t, 0, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, -1, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 1024, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.EqualValues(t, 128, cfg.MetadataCache.TypeCacheMaxSizeMb)
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
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	optimizedFlags := cfg.ApplyOptimizations(isSet)

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
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	optimizedFlags := cfg.ApplyOptimizations(isSet)

	assert.NotEmpty(t, optimizedFlags)
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
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	optimizedFlags := cfg.ApplyOptimizations(isSet)

	assert.True(t, isFlagPresentInOptimizationResults(optimizedFlags, "write.global-max-blocks"))
	assert.EqualValues(t, 1600, cfg.Write.GlobalMaxBlocks)
	assert.True(t, isFlagPresentInOptimizationResults(optimizedFlags, "metadata-cache.negative-ttl-secs"))
	assert.EqualValues(t, 0, cfg.MetadataCache.NegativeTtlSecs)
	assert.EqualValues(t, -1, cfg.MetadataCache.TtlSecs)
	assert.EqualValues(t, 1024, cfg.MetadataCache.StatCacheMaxSizeMb)
	assert.EqualValues(t, 128, cfg.MetadataCache.TypeCacheMaxSizeMb)
	assert.True(t, cfg.ImplicitDirs)
	assert.EqualValues(t, 200000, cfg.FileSystem.RenameDirLimit)
}
