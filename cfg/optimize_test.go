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
		w.Header().Set("Metadata-Flavor", "Google")
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/n1-standard-1")
	})
	defer closeTestServer(t, server)

	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	machineType, err := getMachineType(&mockIsValueSet{})
	if err != nil {
		t.Fatalf("getMachineType failed: %v", err)
	}
	if machineType != "n1-standard-1" {
		t.Errorf("getMachineType returned unexpected machine type: %s", machineType)
	}
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
	if err == nil {
		t.Fatalf("getMachineType should have failed")
	}
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
	if err != nil {
		t.Fatalf("getMachineType failed: %v", err)
	}
	if machineType != "test-machine-type" {
		t.Errorf("getMachineType returned unexpected machine type: %s", machineType)
	}
}

func TestGetMachineType_QuotaError(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a quota error.
	retryCount := 0
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		if retryCount <= maxRetries {
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.Header().Set("Metadata-Flavor", "Google")
			fmt.Fprint(w, "zones/us-central1-a/machineTypes/n1-standard-1")
		}
	})
	defer closeTestServer(t, server)

	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	machineType, err := getMachineType(&mockIsValueSet{})
	if err != nil {
		t.Fatalf("getMachineType failed: %v", err)
	}
	if machineType != "n1-standard-1" {
		t.Errorf("getMachineType returned unexpected machine type: %s", machineType)
	}
}
func TestOptimize_DisableAutoConfig(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-4g")
	})
	defer closeTestServer(t, server)

	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{"disable-autoconfig": true}, boolFlags: map[string]bool{"disable-autoconfig": true}}

	err := Optimize(cfg, isSet)
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	if cfg.Write.EnableStreamingWrites {
		t.Errorf("Expected EnableStreamingWrites to be false")
	}
	if cfg.MetadataCache.NegativeTtlSecs != 0 {
		t.Errorf("Expected NegativeTTLSecs to be 0, got %d", cfg.MetadataCache.NegativeTtlSecs)
	}
	if cfg.MetadataCache.TtlSecs != 0 {
		t.Errorf("Expected TTLSecs to be 0, got %d", cfg.MetadataCache.TtlSecs)
	}
	if cfg.MetadataCache.StatCacheMaxSizeMb != 0 {
		t.Errorf("Expected StatCacheMaxSizeMb to be 0, got %d", cfg.MetadataCache.StatCacheMaxSizeMb)
	}
	if cfg.MetadataCache.TypeCacheMaxSizeMb != 0 {
		t.Errorf("Expected TypeCacheMaxSizeMb to be 0, got %d", cfg.MetadataCache.TypeCacheMaxSizeMb)
	}
	if cfg.ImplicitDirs {
		t.Errorf("Expected ImplicitDirs to be false")
	}
	if cfg.FileSystem.RenameDirLimit != 0 {
		t.Errorf("Expected RenameDirLimit to be 0, got %d", cfg.FileSystem.RenameDirLimit)
	}
}

func TestApplyMachineTypeOptimizations_MatchingMachineType(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-4g")
	})
	defer closeTestServer(t, server)

	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	config := DefaultOptimizationConfig
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	err := ApplyMachineTypeOptimizations(&config, cfg, isSet)
	if err != nil {
		t.Fatalf("ApplyMachineTypeOptimizations failed: %v", err)
	}

	if cfg.MetadataCache.NegativeTtlSecs != 0 {
		t.Errorf("Expected NegativeTTLSecs to be 0, got %d", cfg.MetadataCache.NegativeTtlSecs)
	}
	if cfg.MetadataCache.TtlSecs != -1 {
		t.Errorf("Expected TTLSecs to be -1, got %d", cfg.MetadataCache.TtlSecs)
	}
	if cfg.MetadataCache.StatCacheMaxSizeMb != 1024 {
		t.Errorf("Expected StatCacheMaxSizeMb to be 1024, got %d", cfg.MetadataCache.StatCacheMaxSizeMb)
	}
	if cfg.MetadataCache.TypeCacheMaxSizeMb != 128 {
		t.Errorf("Expected TypeCacheMaxSizeMb to be 128, got %d", cfg.MetadataCache.TypeCacheMaxSizeMb)
	}
	if !cfg.ImplicitDirs {
		t.Errorf("Expected ImplicitDirs to be true")
	}
	if cfg.FileSystem.RenameDirLimit != 200000 {
		t.Errorf("Expected RenameDirLimit to be 200000, got %d", cfg.FileSystem.RenameDirLimit)
	}
	if cfg.FileSystem.Gid != 1000 {
		t.Errorf("Expected Gid to be 1000, got %d", cfg.FileSystem.Gid)
	}
}

func TestApplyMachineTypeOptimizations_NonMatchingMachineType(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a non-matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/n1-standard-1")
	})
	defer closeTestServer(t, server)

	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	config := DefaultOptimizationConfig
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	err := ApplyMachineTypeOptimizations(&config, cfg, isSet)
	if err != nil {
		t.Fatalf("ApplyMachineTypeOptimizations failed: %v", err)
	}

	// Check that no optimizations were applied.
	if cfg.Write.EnableStreamingWrites {
		t.Errorf("Expected EnableStreamingWrites to be false")
	}
}

func TestApplyMachineTypeOptimizations_UserSetFlag(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-4g")
	})
	defer closeTestServer(t, server)

	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	config := DefaultOptimizationConfig
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{"file-system.rename-dir-limit": true}}
	// Simulate setting config value by user
	cfg.FileSystem.RenameDirLimit = 10000

	err := ApplyMachineTypeOptimizations(&config, cfg, isSet)
	if err != nil {
		t.Fatalf("ApplyMachineTypeOptimizations failed: %v", err)
	}

	if cfg.MetadataCache.NegativeTtlSecs != 0 {
		t.Errorf("Expected NegativeTTLSecs to be 0, got %d", cfg.MetadataCache.NegativeTtlSecs)
	}
	if cfg.MetadataCache.TtlSecs != -1 {
		t.Errorf("Expected TTLSecs to be -1, got %d", cfg.MetadataCache.TtlSecs)
	}
	if cfg.MetadataCache.StatCacheMaxSizeMb != 1024 {
		t.Errorf("Expected StatCacheMaxSizeMb to be 1024, got %d", cfg.MetadataCache.StatCacheMaxSizeMb)
	}
	if cfg.MetadataCache.TypeCacheMaxSizeMb != 128 {
		t.Errorf("Expected TypeCacheMaxSizeMb to be 128, got %d", cfg.MetadataCache.TypeCacheMaxSizeMb)
	}
	if !cfg.ImplicitDirs {
		t.Errorf("Expected ImplicitDirs to be true")
	}
	if cfg.FileSystem.RenameDirLimit != 10000 {
		t.Errorf("Expected RenameDirLimit to be 10000, got %d", cfg.FileSystem.RenameDirLimit)
	}
}

func TestApplyMachineTypeOptimizations_MissingFlagOverrideSet(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a machine type with a missing FlagOverrideSet.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-4g")
	})
	defer closeTestServer(t, server)

	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	config := OptimizationConfig{
		FlagOverrideSets: []FlagOverrideSet{}, // Empty FlagOverrideSets.
		MachineTypes: []MachineType{
			{
				Names:               []string{"a3-highgpu-4g"},
				FlagOverrideSetName: "high-performance",
			},
		},
	}
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	err := ApplyMachineTypeOptimizations(&config, cfg, isSet)
	if err != nil {
		t.Fatalf("ApplyMachineTypeOptimizations failed: %v", err)
	}
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

	config := DefaultOptimizationConfig
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	err := ApplyMachineTypeOptimizations(&config, cfg, isSet)
	if err != nil {
		t.Fatalf("ApplyMachineTypeOptimizations failed: %v", err)
	}
}

func TestApplyMachineTypeOptimizations_NoError(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-4g")
	})
	defer closeTestServer(t, server)

	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	config := DefaultOptimizationConfig
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	err := ApplyMachineTypeOptimizations(&config, cfg, isSet)
	if err != nil {
		t.Fatalf("ApplyMachineTypeOptimizations failed: %v", err)
	}
}

func TestSetFlagValue_Bool(t *testing.T) {
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}
	err := setFlagValue(cfg, "implicit-dirs", FlagOverride{NewValue: true}, isSet)
	if err != nil {
		t.Fatalf("setFlagValue failed: %v", err)
	}
	if !cfg.ImplicitDirs {
		t.Errorf("Expected ImplicitDirs to be true")
	}
}

func TestSetFlagValue_String(t *testing.T) {
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}
	err := setFlagValue(cfg, "app-name", FlagOverride{NewValue: "optimal_gcsfuse"}, isSet)
	if err != nil {
		t.Fatalf("setFlagValue failed: %v", err)
	}
	if cfg.AppName != "optimal_gcsfuse" {
		t.Errorf("Expected AppName to be optimal_gcsfuse, got %s", cfg.AppName)
	}
}

func TestSetFlagValue_Int(t *testing.T) {
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}
	err := setFlagValue(cfg, "file-system.gid", FlagOverride{NewValue: 1000}, isSet)
	if err != nil {
		t.Fatalf("setFlagValue failed: %v", err)
	}
	if cfg.FileSystem.Gid != 1000 {
		t.Errorf("Expected Gid to be 1000, got %d", cfg.FileSystem.Gid)
	}
}

func TestSetFlagValue_InvalidFlagName(t *testing.T) {
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}
	err := setFlagValue(cfg, "invalid-flag", FlagOverride{NewValue: true}, isSet)
	if err == nil {
		t.Fatalf("setFlagValue should have failed")
	}
}

func TestApplyMachineTypeOptimizations_NoMachineTypes(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-4g")
	})
	defer closeTestServer(t, server)

	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	config := OptimizationConfig{
		FlagOverrideSets: []FlagOverrideSet{
			{
				Name: "high-performance",
				Overrides: map[string]FlagOverride{
					"write.enable-streaming-writes":         {NewValue: true},
					"write.max-concurrency":                 {NewValue: 128},
					"metadata-cache.negative-ttl-secs":      {NewValue: 0},
					"metadata-cache.ttl-secs":               {NewValue: -1},
					"metadata-cache.stat-cache-max-size-mb": {NewValue: 1024},
					"metadata-cache.type-cache-max-size-mb": {NewValue: 128},
					"implicit-dirs":                         {NewValue: true},
					"file-system.rename-dir-limit":          {NewValue: 200000},
					"file-system.gid":                       {NewValue: "1000"},
				},
			},
		},
		MachineTypes: []MachineType{},
	}
	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	err := ApplyMachineTypeOptimizations(&config, cfg, isSet)
	if err != nil {
		t.Fatalf("ApplyMachineTypeOptimizations failed: %v", err)
	}
	// Check that no optimizations were applied as no machine mapping is set.
	if cfg.Write.EnableStreamingWrites {
		t.Errorf("Expected EnableStreamingWrites to be false")
	}
}

func TestOptimize_Success(t *testing.T) {
	resetMetadataEndpoints(t)
	// Create a test server that returns a matching machine type.
	server := createTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		fmt.Fprint(w, "zones/us-central1-a/machineTypes/a3-highgpu-4g")
	})
	defer closeTestServer(t, server)

	// Override metadataEndpoints for testing.
	metadataEndpoints = []string{server.URL}

	cfg := &Config{}
	isSet := &mockIsValueSet{setFlags: map[string]bool{}}

	err := Optimize(cfg, isSet)
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	if !cfg.Write.EnableStreamingWrites {
		t.Errorf("Expected EnableStreamingWrites to be true")
	}
	if cfg.MetadataCache.NegativeTtlSecs != 0 {
		t.Errorf("Expected NegativeTTLSecs to be 0, got %d", cfg.MetadataCache.NegativeTtlSecs)
	}
	if cfg.MetadataCache.TtlSecs != -1 {
		t.Errorf("Expected TTLSecs to be -1, got %d", cfg.MetadataCache.TtlSecs)
	}
	if cfg.MetadataCache.StatCacheMaxSizeMb != 1024 {
		t.Errorf("Expected StatCacheMaxSizeMb to be 1024, got %d", cfg.MetadataCache.StatCacheMaxSizeMb)
	}
	if cfg.MetadataCache.TypeCacheMaxSizeMb != 128 {
		t.Errorf("Expected TypeCacheMaxSizeMb to be 128, got %d", cfg.MetadataCache.TypeCacheMaxSizeMb)
	}
	if !cfg.ImplicitDirs {
		t.Errorf("Expected ImplicitDirs to be true")
	}
	if cfg.FileSystem.RenameDirLimit != 200000 {
		t.Errorf("Expected RenameDirLimit to be 200000, got %d", cfg.FileSystem.RenameDirLimit)
	}
	if cfg.FileSystem.Gid != 1000 {
		t.Errorf("Expected Gid to be 1000, got %d", cfg.FileSystem.Gid)
	}
}
