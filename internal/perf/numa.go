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
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/lrita/numa"
)

func InitNuma() {
	if !numa.Available() {
		logger.Infof("NUMA not available on this system.")
		return
	}

	nodes := numa.NodeMask()
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
		err := numa.RunOnNode(firstNode)
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
	interfaces, err := ioutil.ReadDir("/sys/class/net")
	if err != nil {
		return nil, err
	}

	for _, intf := range interfaces {
		intfName := intf.Name()
		numaNodePath := filepath.Join("/sys/class/net", intfName, "device", "numa_node")
		content, err := ioutil.ReadFile(numaNodePath)
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
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
}

func MonitorNuma(config *cfg.Config) {
	if !numa.Available() {
		return
	}

	const (
		STABLE = iota
		EXPERIMENTING_NODE_SWITCH
		EXPERIMENTING_UNBOUND
		ROLLING_BACK
	)
	const (
		NODE_SWITCH_EXPERIMENT = iota
		UNBINDING_EXPERIMENT
	)

	state := STABLE
	var currentNode, previousNode int
	var currentBandwidth, experimentBandwidth float64
	var experimentStartTime time.Time
	nodeSwitchExperimentFrequency := time.Minute // Start with a low frequency
	unbindingExperimentFrequency := time.Duration(config.ExperimentalNumaUnbindingExperimentFrequencyMultiplier) * nodeSwitchExperimentFrequency
	var lastNodeSwitchExperimentTime, lastUnbindingExperimentTime time.Time
	var lastExperimentType int

	// Get the initial node.
	_, currentNode = numa.GetCPUAndNode()
	previousNode = currentNode

	// Ticker for periodic checks.
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Last stats for bandwidth calculation.
	var lastStats map[int]networkStats
	var lastTime time.Time

	for {
		select {
		case <-ticker.C:
			// Get current network stats.
			stats, err := getNetworkStatsPerNumaNode()
			if err != nil {
				logger.Errorf("Failed to get network stats per NUMA node: %v", err)
				continue
			}

			// Calculate bandwidth.
			if lastStats != nil {
				duration := time.Since(lastTime).Seconds()
				if duration > 0 {
					totalBw := float64(0)
					for node, s := range stats {
						lastS, ok := lastStats[node]
						if !ok {
							continue
						}
						bw := float64(s.rxBytes-lastS.rxBytes+s.txBytes-lastS.txBytes) / duration
						if node == currentNode {
							currentBandwidth = bw
						}
						totalBw += bw
						logger.Infof("NUMA node %d: bandwidth %f B/s", node, bw)
					}
					if currentNode == -1 { // Unbound
						currentBandwidth = totalBw
					}
				}
			}
			lastStats = stats
			lastTime = time.Now()

			switch state {
			case STABLE:
				if currentNode == -1 {
					// Currently unbound, let's see if binding to a node helps.
					if time.Since(lastNodeSwitchExperimentTime) > nodeSwitchExperimentFrequency {
						nodesMask := numa.NodeMask()
						var nodes []int
						for i := 0; i < nodesMask.Len(); i++ {
							if nodesMask.Get(i) {
								nodes = append(nodes, i)
							}
						}
						if len(nodes) > 0 {
							// Experiment with a random node.
							nextNode := nodes[rand.Intn(len(nodes))]
							logger.Infof("Starting experiment: binding from unbound to node %d", nextNode)
							experimentBandwidth = currentBandwidth
							err := numa.RunOnNode(nextNode)
							if err != nil {
								logger.Errorf("Failed to switch NUMA affinity to node %d: %v", nextNode, err)
							} else {
								previousNode = -1
								currentNode = nextNode
								experimentStartTime = time.Now()
								state = EXPERIMENTING_NODE_SWITCH
								lastExperimentType = NODE_SWITCH_EXPERIMENT
								lastNodeSwitchExperimentTime = time.Now()
							}
						}
					}
				} else {
					// Currently bound, let's see if unbinding or switching helps.
					if time.Since(lastUnbindingExperimentTime) > unbindingExperimentFrequency {
						logger.Infof("Starting unbinding experiment.")
						experimentBandwidth = currentBandwidth
						previousNode = currentNode
						currentNode = -1 // Unbound
						err := numa.RunOnNode(-1)
						if err != nil {
							logger.Errorf("Failed to unbind NUMA affinity: %v", err)
							currentNode = previousNode
						} else {
							experimentStartTime = time.Now()
							state = EXPERIMENTING_UNBOUND
							lastExperimentType = UNBINDING_EXPERIMENT
							lastUnbindingExperimentTime = time.Now()
						}
					} else if time.Since(lastNodeSwitchExperimentTime) > nodeSwitchExperimentFrequency {
						nodesMask := numa.NodeMask()
						var nodes []int
						for i := 0; i < nodesMask.Len(); i++ {
							if nodesMask.Get(i) {
								nodes = append(nodes, i)
							}
						}
						if len(nodes) > 1 {
							// Find the next node to experiment with.
							nextNode := -1
							for i, node := range nodes {
								if node == currentNode {
									nextNode = nodes[(i+1)%len(nodes)]
									break
								}
							}
							if nextNode != -1 {
								logger.Infof("Starting experiment: switching from node %d to %d", currentNode, nextNode)
								experimentBandwidth = currentBandwidth
								err := numa.RunOnNode(nextNode)
								if err != nil {
									logger.Errorf("Failed to switch NUMA affinity to node %d: %v", nextNode, err)
								} else {
									previousNode = currentNode
									currentNode = nextNode
									experimentStartTime = time.Now()
									state = EXPERIMENTING_NODE_SWITCH
									lastExperimentType = NODE_SWITCH_EXPERIMENT
									lastNodeSwitchExperimentTime = time.Now()
								}
							}
						}
					}
				}

			case EXPERIMENTING_NODE_SWITCH, EXPERIMENTING_UNBOUND:
				if time.Since(experimentStartTime) > time.Duration(config.ExperimentalNumaMeasurementDurationSeconds)*time.Second {
					// Experiment is over, check the results.
					improvement := (currentBandwidth - experimentBandwidth) / experimentBandwidth * 100
					if experimentBandwidth > 0 && improvement > float64(config.ExperimentalNumaImprovementThresholdPercent) {
						// Keep the new state.
						logger.Infof("Experiment successful: bandwidth improved by %f%%.", improvement)
						if state == EXPERIMENTING_NODE_SWITCH {
							nodeSwitchExperimentFrequency *= 2 // Double the experiment frequency.
						} else {
							unbindingExperimentFrequency *= 2
						}
						state = STABLE
					} else {
						// Rollback.
						logger.Infof("Experiment failed: bandwidth did not improve enough. Rolling back.")
						state = ROLLING_BACK
					}
				}

			case ROLLING_BACK:
				err := numa.RunOnNode(previousNode)
				if err != nil {
					logger.Errorf("Failed to rollback NUMA affinity to node %d: %v", previousNode, err)
				} else {
					currentNode = previousNode
					if lastExperimentType == NODE_SWITCH_EXPERIMENT {
						nodeSwitchExperimentFrequency /= 2
						if nodeSwitchExperimentFrequency < time.Minute {
							nodeSwitchExperimentFrequency = time.Minute
						}
					} else { // UNBINDING_EXPERIMENT
						unbindingExperimentFrequency /= 2
						if unbindingExperimentFrequency < time.Duration(config.ExperimentalNumaUnbindingExperimentFrequencyMultiplier)*time.Minute {
							unbindingExperimentFrequency = time.Duration(config.ExperimentalNumaUnbindingExperimentFrequencyMultiplier) * time.Minute
						}
					}
					state = STABLE
				}
			}
		}
	}
}
