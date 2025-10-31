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

package mgmt

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

func (mgmt *GCSFuseMgmtService) fetchCPU() {

	mgmt.svc.NodeCPUAvail = -1
	mgmt.svc.PodCPULimit = -1

	mgmt.svc.NodeCPUAvail = runtime.NumCPU()
	mgmt.svc.ProcCPUAvail = runtime.GOMAXPROCS(0)

	if !mgmt.svc.IsContainer {
		return
	}

	cpuQuota, cpuPeriod, err := mgmt.getCgroupV2CPULimit()

	if err != nil {
		cpuQuota, cpuPeriod, err = mgmt.getCgroupV1CPULimit()
		if err != nil {
			logger.Warnf("[cgroup v2/v1] Not available: %v\n", err)
			return
		}

	}

	if cpuPeriod > 0 && cpuQuota > 0 {
		mgmt.svc.PodCPULimit = float64(cpuQuota) / float64(cpuPeriod)
	}

}

func (mgmt *GCSFuseMgmtService) getCgroupV2CPULimit() (quota int64, period int64, err error) {
	filePath := filepath.Join(cgroupV2Base, "cpu.max")
	content, err := mgmt.readFile(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("cgroup v2: could not read %s: %w", filePath, err)
	}

	parts := strings.Fields(content)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("cgroup v2: invalid format in cpu.max: expected '$QUOTA $PERIOD', got '%s'", content)
	}

	periodVal, err := mgmt.parseInt64(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("cgroup v2: could not parse period from cpu.max '%s': %w", content, err)
	}

	if parts[0] == "max" {
		return -1, periodVal, nil // No limit
	}

	quotaVal, err := mgmt.parseInt64(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("cgroup v2: could not parse quota from cpu.max '%s': %w", content, err)
	}

	return quotaVal, periodVal, nil
}

func (mgmt *GCSFuseMgmtService) getCgroupV1CPULimit() (quota int64, period int64, err error) {
	quotaPath := filepath.Join(cgroupV1Base, "cpu", "cpu.cfs_quota_us")
	periodPath := filepath.Join(cgroupV1Base, "cpu", "cpu.cfs_period_us")

	quotaStr, err := mgmt.readFile(quotaPath)
	if err != nil {
		return 0, 0, fmt.Errorf("cgroup v1: could not read %s: %w", quotaPath, err)
	}
	quotaVal, err := mgmt.parseInt64(quotaStr)
	if err != nil {
		return 0, 0, fmt.Errorf("cgroup v1: could not parse cpu.cfs_quota_us value '%s': %w", quotaStr, err)
	}

	periodStr, err := mgmt.readFile(periodPath)
	if err != nil {
		return 0, 0, fmt.Errorf("cgroup v1: could not read %s: %w", periodPath, err)
	}
	periodVal, err := mgmt.parseInt64(periodStr)
	if err != nil {
		return 0, 0, fmt.Errorf("cgroup v1: could not parse cpu.cfs_period_us value '%s': %w", periodStr, err)
	}

	return quotaVal, periodVal, nil
}
