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
// See the License for the specific language governing permissions and
// limitations under the License.

package profiler

import (
	"errors"
	"testing"

	cloudprofiler "cloud.google.com/go/profiler"
	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

// defaultProfilerSettings provides a base configuration for profiler tests.
var defaultProfilerSettings = cfg.ProfilerConfig{
	Enabled:       false, // Explicitly set per test
	VersionTag:    "test-version",
	Mutex:         true,
	Cpu:           true,
	AllocatedHeap: true,
	Heap:          true,
	Goroutines:    true,
}

func TestSetupCloudProfiler_Disabled(t *testing.T) {
	// Save and restore the original profilerStart function.
	originalProfilerStart := profilerStart
	t.Cleanup(func() {
		profilerStart = originalProfilerStart
	})
	mockProfilerStartCalled := false
	profilerStart = func(cfg cloudprofiler.Config, options ...option.ClientOption) error {
		mockProfilerStartCalled = true
		return nil
	}
	profilerConfig := &defaultProfilerSettings // Uses the package-level var
	profilerConfig.Enabled = false

	err := SetupCloudProfiler(profilerConfig)

	require.NoError(t, err, "SetupCloudProfiler should not return an error")
	assert.False(t, mockProfilerStartCalled, "profilerStart should not be called when profiler is disabled")
}

func TestSetupCloudProfiler_EnabledSuccess(t *testing.T) {
	// Save and restore the original profilerStart function.
	originalProfilerStart := profilerStart
	t.Cleanup(func() {
		profilerStart = originalProfilerStart
	})

	var capturedProfilerConfig cloudprofiler.Config
	mockProfilerStartCalled := false
	profilerStart = func(pcfg cloudprofiler.Config, options ...option.ClientOption) error {
		mockProfilerStartCalled = true
		capturedProfilerConfig = pcfg
		return nil
	}

	profilerConfig := &defaultProfilerSettings // Uses the package-level var
	profilerConfig.Enabled = true
	profilerConfig.VersionTag = "v1.2.3"
	profilerConfig.Mutex = true
	profilerConfig.Cpu = true            // NoCPUProfiling = false
	profilerConfig.AllocatedHeap = false // NoAllocProfiling = true
	profilerConfig.Heap = true           // NoHeapProfiling = false
	profilerConfig.Goroutines = false    // NoGoroutineProfiling = true

	err := SetupCloudProfiler(profilerConfig)

	require.NoError(t, err, "SetupCloudProfiler should not return an error")
	require.True(t, mockProfilerStartCalled, "profilerStart should be called")
	assert.Equal(t, "GCSFuse", capturedProfilerConfig.Service)
	assert.Equal(t, "v1.2.3", capturedProfilerConfig.ServiceVersion)
	assert.Equal(t, true, capturedProfilerConfig.MutexProfiling)
	assert.Equal(t, false, capturedProfilerConfig.NoCPUProfiling)
	assert.Equal(t, true, capturedProfilerConfig.NoAllocProfiling)
	assert.Equal(t, false, capturedProfilerConfig.NoHeapProfiling)
	assert.Equal(t, true, capturedProfilerConfig.NoGoroutineProfiling)
	assert.True(t, capturedProfilerConfig.AllocForceGC)
}

func TestSetupCloudProfiler_EnabledStartFails(t *testing.T) {
	// Save and restore the original profilerStart function.
	originalProfilerStart := profilerStart
	t.Cleanup(func() {
		profilerStart = originalProfilerStart
	})
	expectedErr := errors.New("profiler failed to start")
	profilerStart = func(pcfg cloudprofiler.Config, options ...option.ClientOption) error { return expectedErr }
	profilerConfig := &defaultProfilerSettings // Uses the package-level var
	profilerConfig.Enabled = true

	err := SetupCloudProfiler(profilerConfig)

	assert.EqualError(t, err, expectedErr.Error())
}
