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
	cloudprofiler "cloud.google.com/go/profiler"
	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
)

// profilerStart is a variable to enable mocking of profiler.Start in tests.
//
//nolint:gochecknoglobals
var profilerStart = cloudprofiler.Start

// SetupCloudProfiler initializes and starts the Cloud Profiler based on the application configuration.
func SetupCloudProfiler(mpc *cfg.ProfilingConfig) error {
	if !mpc.Enabled {
		return nil
	}

	pConfig := cloudprofiler.Config{
		Service:              "GCSFuse",
		ServiceVersion:       mpc.VersionTag,
		MutexProfiling:       mpc.Mutex,
		NoCPUProfiling:       !mpc.Cpu,
		NoAllocProfiling:     !mpc.AllocatedHeap,
		NoHeapProfiling:      !mpc.Heap,
		NoGoroutineProfiling: !mpc.Goroutines,
		AllocForceGC:         true,
	}

	if err := profilerStart(pConfig); err != nil {
		logger.Warnf("Unable to start the profiler: %v", err)
		return err
	}

	return nil
}
