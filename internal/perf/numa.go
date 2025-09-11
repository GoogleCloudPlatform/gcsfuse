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
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

func InitNuma() {
	if !numaLib.Available() {
		logger.Infof("NUMA not available on this system.")
		return
	}

	nodes := numaLib.NodeMask()
	firstNode := -1
	for i := 0; i < nodes.Len(); i++ {
		if nodes.Get(i) {
			firstNode = i
			break
		}
	}

	if firstNode != -1 {
		// Pin the process to the first available NUMA node.
		// For a real-world scenario, this should be a more intelligent decision,
		// for example, based on which node has more free memory or is closer to the
		// network card that handles the GCS traffic.
		err := numaLib.RunOnNode(firstNode)
		if err != nil {
			logger.Errorf("Failed to set NUMA affinity: %v", err)
		} else {
			logger.Infof("Process bound to NUMA node %d", firstNode)
		}
	}
}

// networkStats represents the network statistics for a NUMA node.
type networkStats struct {
	rxBytes uint64
	txBytes uint64
}

// getNetworkStatsPerNumaNode returns a map of NUMA node ID to network stats.
func getNetworkStatsPerNumaNode() (map[int]networkStats, error) {
	stats := make(map[int]networkStats)
	interfaces, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return nil, err
	}

	for _, intf := range interfaces {
		intfName := intf.Name()
		numaNodePath := filepath.Join("/sys/class/net", intfName, "device", "numa_node")
		content, err := os.ReadFile(numaNodePath)
		if err != nil {
			// Not all interfaces have a NUMA node file, so we skip them.
			continue
		}

		numaNode, err := strconv.Atoi(strings.TrimSpace(string(content)))
		if err != nil {
			continue
		}

		rxBytesPath := filepath.Join("/sys/class/net", intfName, "statistics", "rx_bytes")
		txBytesPath := filepath.Join("/sys/class/net", intfName, "statistics", "tx_bytes")

		rxBytes, err := readUint64FromFile(rxBytesPath)
		if err != nil {
			continue
		}

		txBytes, err := readUint64FromFile(txBytesPath)
		if err != nil {
			continue
		}

		s := stats[numaNode]
		s.rxBytes += rxBytes
		s.txBytes += txBytes
		stats[numaNode] = s
	}

	return stats, nil
}

func readUint64FromFile(path string) (uint64, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
}

func MonitorNuma(config *cfg.Config, metricHandle metrics.MetricHandle) {
	if !numaLib.Available() {
		return
	}

	const (
		STABLE = iota
		EXPERIMENTING
		ROLLING_BACK
	)

	state := STABLE
	var currentNode, previousNode int
	var currentBandwidth, experimentBandwidth float64
	var experimentStartTime, lastExperimentTime time.Time
	experimentFrequency := time.Minute // Start with a low frequency
	experimentCounter := 0

	// Get the initial node.
	_, currentNode = numaLib.GetCPUAndNode()
	previousNode = currentNode

	// Ticker for periodic checks.
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Last stats for bandwidth calculation.
	var lastReadBytes int64
	var lastTime time.Time

	for {
		select {
		case <-ticker.C:
			// Calculate bandwidth.
			readBytes := metricHandle.GcsReadBytesCountValue()
			if lastTime != (time.Time{}) {
				duration := time.Since(lastTime).Seconds()
				if duration > 0 {
					currentBandwidth = float64(readBytes-lastReadBytes) / duration
					logger.Infof("Current bandwidth: %f B/s", currentBandwidth)
				}
			}
			lastReadBytes = readBytes
			lastTime = time.Now()

			switch state {
			case STABLE:
				if time.Since(lastExperimentTime) > experimentFrequency {
					experimentCounter++
					experimentBandwidth = currentBandwidth
					previousNode = currentNode

					if experimentCounter%int(config.ExperimentalNumaUnbindingExperimentFrequencyMultiplier) == 0 {
						// Unbinding experiment.
						logger.Infof("Starting unbinding experiment.")
						err := numaLib.RunOnNode(-1)
						if err != nil {
							logger.Errorf("Failed to unbind NUMA affinity: %v", err)
						} else {
							currentNode = -1 // Unbound
							experimentStartTime = time.Now()
							state = EXPERIMENTING
						}
					} else {
						// Node switching experiment.
						nodesMask := numaLib.NodeMask()
						var nodes []int
						for i := 0; i < nodesMask.Len(); i++ {
							if nodesMask.Get(i) {
								nodes = append(nodes, i)
							}
						}
						if len(nodes) > 1 {
							// Find the next node to experiment with.
							var nextNode int
							if currentNode == -1 {
								// If unbound, pick a random node.
								nextNode = nodes[rand.Intn(len(nodes))]
							} else {
								// Pick a random node that is not the current one.
								for {
									nextNode = nodes[rand.Intn(len(nodes))]
									if nextNode != currentNode {
										break
									}
								}
							}
							logger.Infof("Starting experiment: switching from node %d to %d", currentNode, nextNode)
							err := numaLib.RunOnNode(nextNode)
							if err != nil {
								logger.Errorf("Failed to switch NUMA affinity to node %d: %v", nextNode, err)
							} else {
								currentNode = nextNode
								experimentStartTime = time.Now()
								state = EXPERIMENTING
							}
						}
					}
					lastExperimentTime = time.Now()
				}

			case EXPERIMENTING:
				if time.Since(experimentStartTime) > time.Duration(config.ExperimentalNumaMeasurementDurationSeconds)*time.Second {
					// Experiment is over, check the results.
					var improvement float64
					if experimentBandwidth > 0 {
						improvement = (currentBandwidth - experimentBandwidth) / experimentBandwidth * 100
					}
					if improvement >= float64(config.ExperimentalNumaImprovementThresholdPercent) {
						// Keep the new state.
						logger.Infof("Experiment successful: bandwidth improved by %f%%.", improvement)
						experimentFrequency *= 2 // Increase the time to the next experiment.
						state = STABLE
					} else {
						// Rollback.
						logger.Infof("Experiment failed: bandwidth did not improve enough. Rolling back.")
						state = ROLLING_BACK
					}
				}

			case ROLLING_BACK:
				err := numaLib.RunOnNode(previousNode)
				if err != nil {
					logger.Errorf("Failed to rollback NUMA affinity to node %d: %v", previousNode, err)
				} else {
					currentNode = previousNode
					experimentFrequency /= 2 // Decrease the time to the next experiment.
					if experimentFrequency < time.Minute {
						experimentFrequency = time.Minute
					}
					state = STABLE
				}
			}
		}
	}
}
