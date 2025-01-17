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

package common

import (
	"os/exec"
	"strings"
	"regexp"
)

// GetKernelVersion returns the kernel version.
func GetKernelVersion() (string, error) {
	cmd := exec.Command("uname", "-r")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	kernelVersion := strings.TrimSpace(string(out))
	return kernelVersion, nil
}

// kernelVersion is just a wrapper over GetKernelVersion. This 
// allows us to mock it in the unit test of ShouldSkipKernelListCacheTest.
var kernelVersionForTest = func() (string, error) {
	return GetKernelVersion()
}

// IsKLCacheEvictionUnSupported returns true if Kernel List Cache Eviction is not supported
// for the current linux version.
// In case of any non-nil error it returns false.
func IsKLCacheEvictionUnSupported() (bool, error) {
	UnsupportedKernelVersions := []string{`^6\.9\.\d+`, `^6\.10\.\d+`, `^6\.11\.\d+`, `^6\.12\.\d+`}

	kernelVersion, err := kernelVersionForTest()
	if err != nil {
		return false, err
	}

	for i := 0; i < len(UnsupportedKernelVersions); i++ {
		matched, err := regexp.MatchString(UnsupportedKernelVersions[i], kernelVersion)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}

	return false, nil
}
