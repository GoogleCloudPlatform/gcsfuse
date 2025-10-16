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
