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
	"log"

	"github.com/shirou/gopsutil/disk"
)

func (mgmt *GCSFuseMgmtService) fetchDisks() {

	partitions, err := disk.Partitions(false)
	if err != nil {
		log.Fatalf("Error fetching disk partitions: %v", err)
	}

	for _, p := range partitions {
		usage, usageErr := disk.Usage(p.Mountpoint)

		if usageErr == nil {
			mgmt.addDiskUsage(p.Device, p.Mountpoint, p.Fstype, usage.Total, usage.Used, usage.Free, usage.UsedPercent)
		} else {
			log.Printf("Error fetching disk [%s: %s] usage: %v", p.Device, p.Mountpoint, usageErr)
			mgmt.addDiskUsage(p.Device, p.Mountpoint, p.Fstype, 0, 0, 0, 0)
		}

	}
}
