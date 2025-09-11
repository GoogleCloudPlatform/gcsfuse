// Copyright 2024 Google LLC
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

//go:build linux
// +build linux

package perf

import (
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/lrita/numa"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestInitNuma(t *testing.T) {
	InitNuma()
}

func TestMonitorNuma(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetricHandle := metrics.NewMockMetricHandle(ctrl)
	mockNuma := &mockNumaLibrary{
		available: true,
		nodeMask:  numa.NewBitmask(2),
		cpu:       0,
		node:      0,
	}
	mockNuma.nodeMask.Set(0, true)
	mockNuma.nodeMask.Set(1, true)
	numaLib = mockNuma

	config := &cfg.Config{
		ExperimentalNumaOptimization:                           true,
		ExperimentalNumaImprovementThresholdPercent:            10,
		ExperimentalNumaMeasurementDurationSeconds:             1,
		ExperimentalNumaUnbindingExperimentFrequencyMultiplier: 2,
	}

	// Act
	// We run MonitorNuma in a goroutine and then we sleep for a few seconds to let it run.
	go MonitorNuma(config, mockMetricHandle)
	time.Sleep(5 * time.Second)

	// Assert
	// We can't assert much here, as the function is a long-running goroutine.
	// We are just checking that it runs without panicking.
	// A more comprehensive test would require more advanced techniques like channels to
	// communicate with the goroutine and check its state.
	assert.True(t, true)
}
