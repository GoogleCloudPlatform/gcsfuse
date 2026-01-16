// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kernelparam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"golang.org/x/sys/unix"
)

// KernelParamsManager wraps KernelParamsConfig with a mutex to ensure thread safety.
type KernelParamsManager struct {
	*KernelParamsConfig
	mu sync.Mutex
}

// NewKernelParamsManager creates a new thread-safe configuration manager.
func NewKernelParamsManager() *KernelParamsManager {
	return &KernelParamsManager{
		KernelParamsConfig: newKernelParamsConfig(),
	}
}

// getDeviceMajorMinor returns the major and minor device numbers
// for the filesystem mounted at the given mountPoint.
func getDeviceMajorMinor(mountPoint string) (major uint32, minor uint32, err error) {
	if runtime.GOOS != "linux" {
		return 0, 0, fmt.Errorf("unsupported OS: %s, device major/minor lookup is linux-specific", runtime.GOOS)
	}

	fileInfo, err := os.Stat(mountPoint)
	if err != nil {
		err = fmt.Errorf("os.Stat: %w", err)
		return
	}

	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		err = fmt.Errorf("fileInfo.Sys() is not of type *syscall.Stat_t")
		return
	}

	devID := stat.Dev
	major = unix.Major(uint64(devID))
	minor = unix.Minor(uint64(devID))
	return
}

// atomicFileWrite performs a safe write by creating a temporary file and
// renaming it to the target destination. This ensures the config file is
// never left in a partially written state.
func atomicFileWrite(kernelParamsFile string, data []byte) error {
	dir := filepath.Dir(kernelParamsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tempFile, err := os.CreateTemp(dir, filepath.Base(kernelParamsFile)+"-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	return os.Rename(tempFile.Name(), kernelParamsFile)
}

// pathForParam returns the sysfs path for a given parameter.
func pathForParam(name ParamName, major, minor uint32) (string, error) {
	switch name {
	case ReadAheadKb:
		return fmt.Sprintf("/sys/class/bdi/%d:%d/read_ahead_kb", major, minor), nil

	case MaxBackgroundRequests:
		return fmt.Sprintf("/sys/fs/fuse/connections/%d/max_background", minor), nil

	case CongestionWindowThreshold:
		return fmt.Sprintf("/sys/fs/fuse/connections/%d/congestion_threshold", minor), nil

	case MaxPagesLimit:
		return "/sys/module/fuse/parameters/max_pages_limit", nil

	case TransparentHugePages:
		return "/sys/kernel/mm/transparent_hugepage/enabled", nil

	default:
		logger.Warnf("Unknown parameter name %q found in kernel parameters config. Skipping...", name)
		return "", fmt.Errorf("unknown parameter name: %q", name)
	}
}

// writeValue attempts to write a value to a sysfs path. It first tries a direct
// filesystem write (effective if running as root) and falls back to 'sudo tee'
// if a permission error occurs.
// Note: Fallback attempt succeeds only if passwordless sudo is available.
func writeValue(path, value string) error {
	data := []byte(value + "\n")

	// Attempt a direct write first it succeeds if.
	// 1. GCSFuse is running as root
	// 2. GCSFuse has required permissions to modify files.
	err := os.WriteFile(path, data, 0644)

	// If direct write fails with a Permission Denied, attempt sudo fallback.
	if err != nil && os.IsPermission(err) {
		logger.Warnf("Direct write to file path %q failed with error: %v, Attempting to write using sudo..", path, err)
		// -n: non-interactive mode.
		cmd := exec.Command("sudo", "-n", "tee", path)
		cmd.Stdin = strings.NewReader(value + "\n")

		// Capture Stderr to see why sudo/tee failed
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if sudoErr := cmd.Run(); sudoErr != nil {
			// Combine the system error with the actual message from stderr
			return fmt.Errorf("sudo error: %v, stderr: %q", sudoErr, strings.TrimSpace(stderr.String()))
		}
		return nil
	}
	return err
}

// applyDirectly iterates through all parameters in the config, resolves their
// system paths, and attempts to apply them to the current host using writeValue helper.
func (c *KernelParamsConfig) applyDirectly(mountPoint string) {
	major, minor, err := getDeviceMajorMinor(mountPoint)
	if err != nil {
		logger.Warnf("Failed to apply kernel parameters directly on mount point %q due to err %v", mountPoint, err)
		return
	}
	for _, p := range c.Parameters {
		path, err := pathForParam(p.Name, major, minor)
		if err != nil {
			logger.Warnf("Unable to update setting %q to value %q for the mount point %q due to err: %v", p.Name, p.Value, mountPoint, err)
			continue
		}

		if err := writeValue(path, p.Value); err != nil {
			logger.Warnf("Unable to update setting %q to value %q for the mount point %q due to err: %v", p.Name, p.Value, mountPoint, err)
			continue
		}
		logger.Infof("Setting %q updated successfully to value %q for the mount point %q", p.Name, p.Value, mountPoint)
	}
}

// addParam adds a new parameter to the config or updates the value if the
// parameter already exists.
func (m *KernelParamsManager) addParam(name ParamName, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.Parameters {
		if p.Name == name {
			m.Parameters[i].Value = value
			return
		}
	}
	m.Parameters = append(m.Parameters, KernelParam{
		Name:  name,
		Value: value,
	})
}

// SetMaxPagesLimit adds the max_pages_limit parameter to the config.
func (m *KernelParamsManager) SetMaxPagesLimit(limit int) {
	if limit > 0 {
		m.addParam(MaxPagesLimit, fmt.Sprintf("%d", limit))
	}
}

// SetTransparentHugePages adds the THP enabled mode to the config.
func (m *KernelParamsManager) SetTransparentHugePages(mode string) {
	if mode != "" {
		m.addParam(TransparentHugePages, mode)
	}
}

// SetReadAheadKb adds the BDI read_ahead_kb parameter to the config.
func (m *KernelParamsManager) SetReadAheadKb(kb int) {
	if kb > 0 {
		m.addParam(ReadAheadKb, fmt.Sprintf("%d", kb))
	}
}

// SetMaxBackgroundRequests adds the FUSE connection max_background parameter to the config.
func (m *KernelParamsManager) SetMaxBackgroundRequests(limit int) {
	if limit > 0 {
		m.addParam(MaxBackgroundRequests, fmt.Sprintf("%d", limit))
	}
}

// SetCongestionWindowThreshold adds the FUSE connection congestion_threshold parameter to the config.
func (m *KernelParamsManager) SetCongestionWindowThreshold(threshold int) {
	if threshold > 0 {
		m.addParam(CongestionWindowThreshold, fmt.Sprintf("%d", threshold))
	}
}

// ApplyGKE atomically writes the KernelParamsConfig to a JSON file at the specified path.
// This is used in GKE environments where CSI Driver (privileged) reads the file
// to apply settings.
func (m *KernelParamsManager) ApplyGKE(kernelParamsFile string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	kernelConfigJson, err := json.Marshal(m.KernelParamsConfig)
	if err != nil {
		logger.Warnf("Failed to marshal kernel parameters config: %v", err)
		return
	}
	logger.Info("Writing kernel parameters to file for GKE environment", "file", kernelParamsFile, "kernel config", string(kernelConfigJson))
	if err := atomicFileWrite(kernelParamsFile, kernelConfigJson); err != nil {
		logger.Warnf("Failed to write kernel parameters to file %q: %v", kernelParamsFile, err)
		return
	}
	logger.Info("Successfully wrote kernel parameters to file", "file", kernelParamsFile)
}

// ApplyNonGKE applies the kernel settings directly to the host's sysfs entries.
func (m *KernelParamsManager) ApplyNonGKE(mountPoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	kernelConfigJson, err := json.Marshal(m.KernelParamsConfig)
	if err != nil {
		logger.Warnf("Failed to marshal kernel parameters config: %v", err)
		return
	}
	logger.Info("Applying kernel parameters directly for non-GKE environment", "mountPoint", mountPoint, "kernel config", string(kernelConfigJson))
	m.applyDirectly(mountPoint)
}
