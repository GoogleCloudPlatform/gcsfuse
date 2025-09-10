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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/lrita/numa"
)

func InitNuma() {
	if !numa.IsAvailable() {
		logger.Infof("NUMA not available on this system.")
		return
	}

	nodes := numa.AvailableNodes()
	if len(nodes) > 0 {
// Pin the process to the first available NUMA node.
// For a real-world scenario, this should be a more intelligent decision,
// for example, based on which node has more free memory or is closer to the
// network card that handles the GCS traffic.
		err := numa.Affinity(nodes[0])
		if err != nil {
			logger.Errorf("Failed to set NUMA affinity: %v", err)
		} else {
			logger.Infof("Process bound to NUMA node %d", nodes[0])
		}
	}
}

func MonitorNuma() {
	if !numa.IsAvailable() {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// This is a placeholder for the actual monitoring logic.
		// In a real implementation, we would monitor CPU and bandwidth usage
		// and decide whether to switch NUMA nodes or remove the affinity.
		// For this PoC, we just log a message.
		logger.Infof("Checking NUMA performance...")

		nodes := numa.AvailableNodes()
		if len(nodes) <= 1 {
			continue
		}

		currentNode, err := numa.CurrentNode()
		if err != nil {
			logger.Errorf("Failed to get current NUMA node: %v", err)
			continue
		}

		// Placeholder logic to switch to the next available node.
		// A real implementation would have a more sophisticated algorithm.
		nextNode := -1
		for i, node := range nodes {
			if node == currentNode {
				nextNode = nodes[(i+1)%len(nodes)]
				break
			}
		}

		if nextNode != -1 && nextNode != currentNode {
			err := numa.Affinity(nextNode)
			if err != nil {
				logger.Errorf("Failed to switch NUMA affinity to node %d: %v", nextNode, err)
			} else {
				logger.Infof("Switched NUMA affinity from node %d to %d", currentNode, nextNode)
			}
		}
	}
}
