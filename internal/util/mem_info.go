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

package util

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	cgroupV1MemLimitFile = "/sys/fs/cgroup/memory/memory.limit_in_bytes"
	cgroupV2MountPoint   = "/sys/fs/cgroup"
	procSelfCgroup       = "/proc/self/cgroup"
)

// GetTotalMemory returns the total available memory in bytes.
// It tries to read the container memory limit (cgroup v1 or v2).
// If the limit is not set or is unlimited, it falls back to the total system memory.
func GetTotalMemory() (uint64, error) {
	memLimit, err := getContainerMemoryLimit()
	if err != nil {
		// Fallback to system memory if we can't determine container limit
		// or if we are not in a container.
		return getSystemTotalMemory()
	}

	sysMem, err := getSystemTotalMemory()
	if err != nil {
		return 0, err
	}

	// Return the minimum of container limit and system memory.
	// Container limit can be higher than physical memory (swap, or just set high),
	// but we can't use more than physical memory + swap (though Sysinfo.Totalram is physical).
	// Typically, if limit > sysMem, we are bound by sysMem.
	if memLimit < sysMem {
		return memLimit, nil
	}
	return sysMem, nil
}

func getSystemTotalMemory() (uint64, error) {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return 0, err
	}
	return uint64(info.Totalram) * uint64(info.Unit), nil
}

func getContainerMemoryLimit() (uint64, error) {
	// Try Cgroup V2 first
	// Check if cgroup.controllers exists in mount point
	if _, err := os.Stat(filepath.Join(cgroupV2MountPoint, "cgroup.controllers")); err == nil {
		return getCgroupV2MemoryLimit()
	}

	// Try Cgroup V1
	return getCgroupV1MemoryLimit()
}

func getCgroupV1MemoryLimit() (uint64, error) {
	data, err := os.ReadFile(cgroupV1MemLimitFile)
	if err != nil {
		return 0, err
	}
	return parseMemoryLimit(strings.TrimSpace(string(data)))
}

func getCgroupV2MemoryLimit() (uint64, error) {
	// Need to find the cgroup path for the current process
	cgroupPath, err := getCurrentCgroupPathV2()
	if err != nil {
		return 0, err
	}

	limitFile := filepath.Join(cgroupV2MountPoint, cgroupPath, "memory.max")
	data, err := os.ReadFile(limitFile)
	if err != nil {
		return 0, err
	}

	s := strings.TrimSpace(string(data))
	if s == "max" {
		return 0, fmt.Errorf("memory limit is max") // Treat as unlimited
	}

	return parseMemoryLimit(s)
}

func getCurrentCgroupPathV2() (string, error) {
	f, err := os.Open(procSelfCgroup)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// format: hierarchy-ID:controller-list:cgroup-path
		// For cgroup v2, hierarchy-ID is 0 and controller-list is empty (usually)
		// e.g. "0::/user.slice/user-1000.slice/..."
		parts := strings.SplitN(line, ":", 3)
		if len(parts) == 3 && parts[0] == "0" && parts[1] == "" {
			return parts[2], nil
		}
	}
	return "", fmt.Errorf("cgroup v2 path not found in %s", procSelfCgroup)
}

func parseMemoryLimit(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
