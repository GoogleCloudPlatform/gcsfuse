// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not...
//
// ... (copyright header truncated for brevity)
//
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package perf

import (
	"context"
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
		ExperimentalNumaOptimization:                 true,
		ExperimentalNumaImprovementThresholdPercent:  10,
		ExperimentalNumaMeasurementDurationSeconds:   1,
		ExperimentalNumaExperimentInterval:           1 * time.Second,
		ExperimentalNumaExperimentIntervalMultiplier: 2,
	}

	// Expectations
	// We expect GcsReadBytesCountValue to be called multiple times.
	// We don't care about the order or the return values in this test.
	mockMetricHandle.EXPECT().GcsReadBytesCountValue().AnyTimes().Return(int64(100))

	// Act
	// We run MonitorNuma in a goroutine and then we sleep for a few seconds to let it run.
	ctx, cancel := context.WithCancel(context.Background())
	go MonitorNuma(ctx, config, mockMetricHandle)
	time.Sleep(3 * time.Second)
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Assert
	// We are just checking that it runs without panicking.
	assert.True(t, true)
}

func TestRunExperimentRound_NoImprovement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetricHandle := metrics.NewMockMetricHandle(ctrl)
	config := &cfg.Config{
		ExperimentalNumaImprovementThresholdPercent: 10,
		ExperimentalNumaMeasurementDurationSeconds:  1,
	}
	currentBestNode := 0
	currentBestBandwidth := 100.0

	gomock.InOrder(
		// Unbound experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(1000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(1100)), // bandwidth 100
		// Node 0 experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(2000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(2100)), // bandwidth 100
		// Node 1 experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(3000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(3100)), // bandwidth 100
	)

	mockNuma := &mockNumaLibrary{
		available: true,
		nodeMask:  numa.NewBitmask(2),
	}
	mockNuma.nodeMask.Set(0, true)
	mockNuma.nodeMask.Set(1, true)
	numaLib = mockNuma

	bestNode, bestBandwidth := runExperimentRound(config, mockMetricHandle, currentBestNode, currentBestBandwidth)

	assert.Equal(t, currentBestNode, bestNode)
	assert.Equal(t, currentBestBandwidth, bestBandwidth)
}

func TestRunExperimentRound_UnboundImproves(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetricHandle := metrics.NewMockMetricHandle(ctrl)
	config := &cfg.Config{
		ExperimentalNumaImprovementThresholdPercent: 10,
		ExperimentalNumaMeasurementDurationSeconds:  1,
	}
	currentBestNode := 0
	currentBestBandwidth := 100.0

	gomock.InOrder(
		// Unbound experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(1000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(1120)), // bandwidth 120
		// Node 0 experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(2000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(2100)), // bandwidth 100
		// Node 1 experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(3000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(3100)), // bandwidth 100
	)

	mockNuma := &mockNumaLibrary{
		available: true,
		nodeMask:  numa.NewBitmask(2),
	}
	mockNuma.nodeMask.Set(0, true)
	mockNuma.nodeMask.Set(1, true)
	numaLib = mockNuma

	bestNode, bestBandwidth := runExperimentRound(config, mockMetricHandle, currentBestNode, currentBestBandwidth)

	assert.Equal(t, -1, bestNode)
	assert.Equal(t, 120.0, bestBandwidth)
}

func TestRunExperimentRound_NodeImproves(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetricHandle := metrics.NewMockMetricHandle(ctrl)
	config := &cfg.Config{
		ExperimentalNumaImprovementThresholdPercent: 10,
		ExperimentalNumaMeasurementDurationSeconds:  1,
	}
	currentBestNode := 0
	currentBestBandwidth := 100.0

	gomock.InOrder(
		// Unbound experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(1000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(1100)), // bandwidth 100
		// Node 0 experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(2000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(2100)), // bandwidth 100
		// Node 1 experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(3000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(3150)), // bandwidth 150
	)

	mockNuma := &mockNumaLibrary{
		available: true,
		nodeMask:  numa.NewBitmask(2),
	}
	mockNuma.nodeMask.Set(0, true)
	mockNuma.nodeMask.Set(1, true)
	numaLib = mockNuma

	bestNode, bestBandwidth := runExperimentRound(config, mockMetricHandle, currentBestNode, currentBestBandwidth)

	assert.Equal(t, 1, bestNode)
	assert.Equal(t, 150.0, bestBandwidth)
}

func TestRunExperimentRound_InitialZeroBandwidth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetricHandle := metrics.NewMockMetricHandle(ctrl)
	config := &cfg.Config{
		ExperimentalNumaImprovementThresholdPercent: 10,
		ExperimentalNumaMeasurementDurationSeconds:  1,
	}
	currentBestNode := -1
	currentBestBandwidth := 0.0

	gomock.InOrder(
		// Unbound experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(1000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(1000)), // bandwidth 0
		// Node 0 experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(2000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(2100)), // bandwidth 100
		// Node 1 experiment
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(3000)),
		mockMetricHandle.EXPECT().GcsReadBytesCountValue().Return(int64(3050)), // bandwidth 50
	)

	mockNuma := &mockNumaLibrary{
		available: true,
		nodeMask:  numa.NewBitmask(2),
	}
	mockNuma.nodeMask.Set(0, true)
	mockNuma.nodeMask.Set(1, true)
	numaLib = mockNuma

	bestNode, bestBandwidth := runExperimentRound(config, mockMetricHandle, currentBestNode, currentBestBandwidth)

	assert.Equal(t, 0, bestNode)
	assert.Equal(t, 100.0, bestBandwidth)
}

func TestMeasureBandwidth_ZeroDuration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetricHandle := metrics.NewMockMetricHandle(ctrl)
	config := &cfg.Config{
		ExperimentalNumaMeasurementDurationSeconds: 0,
	}

	bandwidth := measureBandwidth(config, mockMetricHandle)

	assert.Equal(t, 0.0, bandwidth)
}
