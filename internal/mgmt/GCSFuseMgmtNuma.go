package mgmt

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// GetNICNUMANode reads the NUMA node ID for a given network interface name (e.g., "eth0").
// It works by reading the 'numa_node' file in the device's sysfs entry on Linux.
func GetNICNUMANode(nicName string) (int, error) {
	// 1. Construct the path to the device symlink in sysfs.
	// Example: /sys/class/net/eth0/device
	deviceSymlinkPath := filepath.Join("/sys/class/net", nicName, "device")

	// 2. Resolve the symlink to get the actual PCI device path.
	// This ensures we have the correct path for the numa_node file.
	devicePath, err := os.Readlink(deviceSymlinkPath)
	if err != nil {
		return -1, fmt.Errorf("could not resolve device symlink for %s: %w", nicName, err)
	}

	// The resolved path is relative, so we get the absolute path to the PCI device directory.
	// Example: /sys/devices/pci0000:00/0000:00:1c.0/0000:01:00.0
	pciDeviceDir := filepath.Join(filepath.Dir(deviceSymlinkPath), devicePath)

	// 3. Read the numa_node file content.
	// Example: /sys/devices/.../numa_node
	numaNodePath := filepath.Join(pciDeviceDir, "numa_node")

	content, err := ioutil.ReadFile(numaNodePath)
	if err != nil {
		return -1, fmt.Errorf("could not read numa_node file at %s: %w", numaNodePath, err)
	}

	// 4. Parse the content (which is a string like "0\n" or "-1\n").
	numaNodeStr := strings.TrimSpace(string(content))

	// The numa_node file contains the NUMA node ID.
	// A value of '-1' typically means NUMA is not supported, or the device's NUMA node is unknown/irrelevant (e.g., on older kernels).
	numaNode, err := strconv.Atoi(numaNodeStr)
	if err != nil {
		return -1, fmt.Errorf("failed to convert numa_node value '%s' to integer: %w", numaNodeStr, err)
	}

	return numaNode, nil
}
