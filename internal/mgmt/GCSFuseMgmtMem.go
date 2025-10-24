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
	"log"
	"path/filepath"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/shirou/gopsutil/mem"
)

func (mgmt *GCSFuseMgmtService) fetchMemory() {

	mgmt.svc.CgroupVersion = 0
	mgmt.svc.PodMemLimitBytes = -1

	//Node memory
	mgmt.getNodeMemory()

	if !mgmt.svc.IsContainer {
		return
	}

	memLimit, err := mgmt.getCgroupV2MemoryLimit()
	if err != nil {
		memLimit, err = mgmt.getCgroupV1MemoryLimit()
		mgmt.svc.CgroupVersion = 1
		if err != nil {
			logger.Warnf("[cgroup v2/v1] Not available: %v\n", err)
			return
		}
	} else {
		mgmt.svc.CgroupVersion = 2
	}
	mgmt.svc.PodMemLimitBytes = memLimit

}

func (mgmt *GCSFuseMgmtService) getCgroupV2MemoryLimit() (int64, error) {
	filePath := filepath.Join(cgroupV2Base, "memory.max")
	content, err := mgmt.readFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("cgroup v2: could not read %s: %w", filePath, err)
	}

	if content == "max" {
		return -1, nil // No limit
	}

	limit, err := mgmt.parseInt64(content)
	if err != nil {
		return -1, fmt.Errorf("cgroup v2: could not parse memory.max value '%s': %w", content, err)
	}
	return limit, nil
}

func (mgmt *GCSFuseMgmtService) getCgroupV1MemoryLimit() (int64, error) {
	filePath := filepath.Join(cgroupV1Base, "memory", "memory.limit_in_bytes")
	content, err := mgmt.readFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("cgroup v1: could not read %s: %w", filePath, err)
	}

	limit, err := mgmt.parseInt64(content)
	if err != nil {
		return 0, fmt.Errorf("cgroup v1: could not parse memory.limit_in_bytes value '%s': %w", content, err)
	}

	// In cgroup v1, an extremely large value often indicates no effective limit.
	// This threshold is roughly 8 EiB.
	if limit > 9000000000000000000 {
		return -1, nil // Effectively no limit
	}
	return limit, nil
}

func (mgmt *GCSFuseMgmtService) getNodeMemory() {
	// Get virtual memory statistics (includes RAM and swap)
	mgmt.svc.NodeMemTotalBytes = -1
	mgmt.svc.NodeMemUtilBytes = -1
	mgmt.svc.NodeSwapTotalBytes = -1
	mgmt.svc.NodeSwapUtilBytes = -1

	v, err := mem.VirtualMemory()
	if err != nil {
		log.Fatalf("Error fetching memory stats: %v", err)
	} else {
		mgmt.svc.NodeMemTotalBytes = int64(v.Total)
		mgmt.svc.NodeMemUtilBytes = int64(v.Used)
	}

	s, err := mem.SwapMemory()
	if err != nil {
		log.Fatalf("Error fetching swap stats: %v", err)
	} else {
		mgmt.svc.NodeSwapTotalBytes = int64(s.Total)
		mgmt.svc.NodeSwapUtilBytes = int64(s.Used)

	}

}
