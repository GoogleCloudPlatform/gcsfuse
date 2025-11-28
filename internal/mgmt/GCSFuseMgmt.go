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
	"encoding/json"
	"fmt"
	"sync"

	"os"

	"strconv"
	"strings"
)

const (
	cgroupV2Base  = "/sys/fs/cgroup"
	cgroupV1Base  = "/sys/fs/cgroup"
	dockerProcess = "/proc/1/cgroup"
)

type DiskPartitions struct {
	Name        string               `json:"name"`
	MountPoints map[string]DiskMount `json:"mountPoints"`
}

type DiskMount struct {
	Name       string  `json:"name"`
	FileSystem string  `json:"fileSystem"`
	TotUsage   uint64  `json:"totUsage"`
	UsedUsage  uint64  `json:"usedUsage"`
	FreeUsage  uint64  `json:"freeUsage"`
	UsagePct   float64 `json:"usagePct"`
}

type GCSFuseMgmt struct {
	IsContainer        bool                      `json:"isContainer"`
	CgroupVersion      int8                      `json:"cgroupVersion"`
	NodeMemTotalBytes  int64                     `json:"nodeMemTotalBytes"`
	NodeMemUtilBytes   int64                     `json:"nodeMemUtilBytes"`
	NodeSwapTotalBytes int64                     `json:"nodeSwapTotalBytes"`
	NodeSwapUtilBytes  int64                     `json:"nodeSwapUtilBytes"`
	PodMemLimitBytes   int64                     `json:"podMemLimitBytes"`
	NodeCPUAvail       int                       `json:"nodeCPUAvail"`
	ProcCPUAvail       int                       `json:"procCPUAvail"`
	PodCPULimit        float64                   `json:"podCPULimit"`
	Disks              map[string]DiskPartitions `json:"disks"`
}

type GCSFuseMgmtService struct {
	mgmtLock *sync.RWMutex
	svc      GCSFuseMgmt
}

func NewGCSFuseMgmtService() *GCSFuseMgmtService {
	mgmt := new(GCSFuseMgmtService)
	mgmt.mgmtLock = &sync.RWMutex{}
	mgmt.svc.Disks = make(map[string]DiskPartitions)
	return mgmt
}

func (mgmt *GCSFuseMgmtService) addDiskUsage(partName string, diskName string, fileSystem string,
	totUsage uint64, usedUsage uint64, freeUsage uint64, usagePct float64) {

	mp := DiskMount{
		Name:       diskName,
		FileSystem: fileSystem,
		TotUsage:   totUsage,
		UsedUsage:  usedUsage,
		FreeUsage:  freeUsage,
		UsagePct:   usagePct,
	}

	if _, ok := mgmt.svc.Disks[partName]; !ok {
		mgmt.svc.Disks[partName] = DiskPartitions{
			Name:        partName,
			MountPoints: make(map[string]DiskMount),
		}
	}

	mgmt.svc.Disks[partName].MountPoints[diskName] = mp

}

func (mgmt *GCSFuseMgmtService) readFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (mgmt *GCSFuseMgmtService) parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func (mgmt *GCSFuseMgmtService) getMgmt() GCSFuseMgmtService {
	mgmt.mgmtLock.RLock()
	defer mgmt.mgmtLock.RUnlock()

	// Create a shallow copy of the main struct.
	mgmt_new := *mgmt

	// Deep copy the Disks map to ensure thread safety.
	mgmt_new.svc.Disks = make(map[string]DiskPartitions, len(mgmt.svc.Disks))
	for k, v := range mgmt.svc.Disks {
		// Deep copy the inner map as well.
		newMountPoints := make(map[string]DiskMount, len(v.MountPoints))
		for mk, mv := range v.MountPoints {
			newMountPoints[mk] = mv
		}
		v.MountPoints = newMountPoints
		mgmt_new.svc.Disks[k] = v
	}

	return mgmt_new
}

func (mgmt *GCSFuseMgmtService) PrettyString() string {
	safeCopy := mgmt.getMgmt().svc
	jsonData, err := json.MarshalIndent(safeCopy, "", "  ")
	if err != nil {
		return fmt.Sprintf("failed to marshal to JSON: %+v", mgmt.svc)
	}
	return string(jsonData)
}

func (mgmt *GCSFuseMgmtService) String() string {
	safeCopy := mgmt.getMgmt().svc
	jsonData, err := json.Marshal(safeCopy)
	if err != nil {
		return fmt.Sprintf("failed to marshal to JSON: %+v", mgmt.svc)
	}
	return string(jsonData)
}

func (mgmt *GCSFuseMgmtService) GetConfig() GCSFuseMgmt {
	safeCopy := mgmt.getMgmt().svc
	return safeCopy
}

func (mgmt *GCSFuseMgmtService) Refresh() {

	mgmt.mgmtLock.Lock()
	defer mgmt.mgmtLock.Unlock()

	mgmt.detectEnv()
	mgmt.fetchMemory()
	mgmt.fetchCPU()
	mgmt.fetchDisks()

}
