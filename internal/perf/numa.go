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

package perf

import (
	"context"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

func InitNuma() {
	// Start unbound.
	err := numaLib.RunOnNode(-1)
	if err != nil {
		logger.Errorf("Failed to unbind NUMA affinity: %v", err)
	}
}

func MonitorNuma(ctx context.Context, config *cfg.Config, metricHandle metrics.MetricHandle) {
	if !numaLib.Available() {
		return
	}

	bestNode := -1 // -1 means unbound
	var bestBandwidth float64
	experimentInterval := config.ExperimentalNumaExperimentInterval

	ticker := time.NewTicker(experimentInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if bestBandwidth < 10 {
				experimentInterval = config.ExperimentalNumaExperimentInterval
				ticker.Reset(experimentInterval)
				continue
			}
			// Run a round of experiments.
			logger.Infof("Starting a round of NUMA experiments.")
			newBestNode, newBestBandwidth := runExperimentRound(config, metricHandle, bestNode, bestBandwidth)

			if newBestNode == bestNode {
				experimentInterval *= time.Duration(config.ExperimentalNumaExperimentIntervalMultiplier)
				ticker.Reset(experimentInterval)
				logger.Infof("Configuration is stable. New experiment interval: %v", experimentInterval)
			} else {
				experimentInterval = config.ExperimentalNumaExperimentInterval
				ticker.Reset(experimentInterval)
				logger.Infof("Configuration changed. New experiment interval: %v", experimentInterval)
			}
			bestNode = newBestNode
			bestBandwidth = newBestBandwidth

			// Bind to the best configuration.
			err := numaLib.RunOnNode(bestNode)
			if err != nil {
				logger.Errorf("Failed to bind to best node %d: %v", bestNode, err)
			} else {
				logger.Infof("Best configuration is node %d with bandwidth %f B/s", bestNode, bestBandwidth)
			}
		case <-ctx.Done():
			logger.Infof("Stopping NUMA monitoring.")
			return
		}
	}
}

func runExperimentRound(config *cfg.Config, metricHandle metrics.MetricHandle, currentBestNode int, currentBestBandwidth float64) (bestNode int, bestBandwidth float64) {
	bestNode = currentBestNode
	bestBandwidth = currentBestBandwidth

	// Measure unbound bandwidth.
	err := numaLib.RunOnNode(-1)
	if err != nil {
		logger.Errorf("Failed to unbind NUMA affinity: %v", err)
	} else {
		unboundBandwidth := measureBandwidth(config, metricHandle)
		logger.Infof("Unbound bandwidth: %f B/s", unboundBandwidth)
		var improvement float64
		if bestBandwidth > 0 {
			improvement = (unboundBandwidth - bestBandwidth) / bestBandwidth * 100
		}
		if (bestBandwidth == 0 && unboundBandwidth > 0) || (bestBandwidth > 0 && improvement > float64(config.ExperimentalNumaImprovementThresholdPercent)) {
			bestBandwidth = unboundBandwidth
			bestNode = -1
		}
	}

	// Experiment with each NUMA node.
	nodesMask := numaLib.NodeMask()
	var nodes []int
	for i := 0; i < nodesMask.Len(); i++ {
		if nodesMask.Get(i) {
			nodes = append(nodes, i)
		}
	}

	for _, node := range nodes {
		// Bind to the node.
		err := numaLib.RunOnNode(node)
		if err != nil {
			logger.Errorf("Failed to bind to node %d: %v", node, err)
			continue
		}
		logger.Infof("Experimenting with node %d", node)

		// Measure bandwidth.
		bandwidth := measureBandwidth(config, metricHandle)
		logger.Infof("Bandwidth on node %d: %f B/s", node, bandwidth)

		// Check for improvement.
		var improvement float64
		if bestBandwidth > 0 {
			improvement = (bandwidth - bestBandwidth) / bestBandwidth * 100
		}
		if (bestBandwidth == 0 && bandwidth > 0) || (bestBandwidth > 0 && improvement > float64(config.ExperimentalNumaImprovementThresholdPercent)) {
			bestBandwidth = bandwidth
			bestNode = node
		}
	}
	return
}

func measureBandwidth(config *cfg.Config, metricHandle metrics.MetricHandle) float64 {
	if config.ExperimentalNumaMeasurementDurationSeconds == 0 {
		return 0
	}
	initialReadBytes := metricHandle.GcsReadBytesCountValue()
	time.Sleep(time.Duration(config.ExperimentalNumaMeasurementDurationSeconds) * time.Second)
	finalReadBytes := metricHandle.GcsReadBytesCountValue()
	return float64(finalReadBytes-initialReadBytes) / float64(config.ExperimentalNumaMeasurementDurationSeconds)
}
