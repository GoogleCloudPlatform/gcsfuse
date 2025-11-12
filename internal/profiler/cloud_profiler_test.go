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
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

func TestSetupCloudProfiler_Disabled(t *testing.T) {
	mockProfilerStartCalled := false
	profilerStart := func(_ cloudprofiler.Config, _ ...option.ClientOption) error {
		mockProfilerStartCalled = true
		return nil
	}
	cloudProfilerConfig := &cfg.CloudProfilerConfig{
		Enabled: false,
	}

	err := setupCloudProfiler(cloudProfilerConfig, profilerStart)

	require.NoError(t, err, "SetupCloudProfiler should not return an error")
	assert.False(t, mockProfilerStartCalled, "profilerStart should not be called when profiler is disabled")
}

func TestSetupCloudProfiler_EnabledSuccess(t *testing.T) {
	var capturedProfilerConfig cloudprofiler.Config
	mockProfilerStartCalled := false
	profilerStart := func(pcfg cloudprofiler.Config, _ ...option.ClientOption) error {
		mockProfilerStartCalled = true
		capturedProfilerConfig = pcfg
		return nil
	}
	cloudProfilerConfig := &cfg.CloudProfilerConfig{
		Enabled:       true,
		Label:         "v1.2.3",
		Mutex:         true,
		Cpu:           true,
		AllocatedHeap: false,
		Heap:          true,
		Goroutines:    false,
	}

	err := setupCloudProfiler(cloudProfilerConfig, profilerStart)

	require.NoError(t, err, "SetupCloudProfiler should not return an error")
	require.True(t, mockProfilerStartCalled, "profilerStart should be called")
	assert.Equal(t, "gcsfuse", capturedProfilerConfig.Service)
	assert.Equal(t, "v1.2.3", capturedProfilerConfig.ServiceVersion)
	assert.Equal(t, true, capturedProfilerConfig.MutexProfiling)
	assert.Equal(t, false, capturedProfilerConfig.NoCPUProfiling)
	assert.Equal(t, true, capturedProfilerConfig.NoAllocProfiling)
	assert.Equal(t, false, capturedProfilerConfig.NoHeapProfiling)
	assert.Equal(t, true, capturedProfilerConfig.NoGoroutineProfiling)
	assert.True(t, capturedProfilerConfig.AllocForceGC)
}

func TestSetupCloudProfiler_EnabledStartFails(t *testing.T) {
	expectedErr := errors.New("profiler failed to start")
	profilerStart := func(_ cloudprofiler.Config, _ ...option.ClientOption) error { return expectedErr }
	cloudProfilerConfig := &cfg.CloudProfilerConfig{
		Enabled: true,
	}

	err := setupCloudProfiler(cloudProfilerConfig, profilerStart)

	assert.EqualError(t, err, expectedErr.Error())
}
