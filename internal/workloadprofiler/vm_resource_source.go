package workloadprofiler

import (
	"runtime"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"golang.org/x/sys/unix"
)

type ResourceStats struct {
	CPUCount    int64
	DiskFreeGB  int64
	DiskTotalGB int64
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

	GIB := int64(1024 * 1024 * 1024) // Convert bytes to GiB
	data["vm_details_mb"] = ResourceStats{
		CPUCount:    int64(runtime.NumCPU()),
		DiskTotalGB: (int64(stat.Blocks*uint64(stat.Bsize)) + GIB - 1) / GIB, // Convert bytes to MB,
		DiskFreeGB:  (int64(stat.Bfree*uint64(stat.Bsize)) + GIB - 1) / GIB,  // Convert bytes to MB
	}

	return data
}
