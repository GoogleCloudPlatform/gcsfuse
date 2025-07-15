package workloadprofiler

import (
	"runtime"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"golang.org/x/sys/unix"
)

type ResourceStats struct {
	CPUCount  int64
	DiskFree  int64
	DiskTotal int64
}

type VMResourceSource struct {
	ProfilerSource
	mu sync.RWMutex
}

func NewVMResourceSource() *VMResourceSource {
	return &VMResourceSource{}
}

func (vms *VMResourceSource) GetProfileData() map[string]interface{} {
	vms.mu.RLock()
	defer vms.mu.RUnlock()

	logger.Debugf("Collecting VM resource profile data")
	data := make(map[string]interface{})
	runtimeStats := &runtime.MemStats{}
	runtime.ReadMemStats(runtimeStats)
	var stat unix.Statfs_t
	unix.Statfs("/", &stat)

	data["vm_details_mb"] = ResourceStats{
		CPUCount:  int64(runtime.NumCPU()),
		DiskTotal: int64(stat.Blocks*uint64(stat.Bsize)) / (1024 * 1024), // Convert bytes to MB,
		DiskFree:  int64(stat.Bfree*uint64(stat.Bsize)) / (1024 * 1024),  // Convert bytes to MB
	}

	return data
}
