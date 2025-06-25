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
	"math"
	"os/exec"
	"regexp"
	"strings"
)

const (
	MiB                    = 1024 * 1024
	MaxMiBsInUint64 uint64 = math.MaxUint64 >> 20
	MaxMiBsInInt64  int64  = math.MaxInt64 >> 20
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

// BytesToHigherMiBs returns the MiBs equivalent
// to the given number of bytes.
// If bytes is not an exact number of MiBs,
// then it returns the next higher no. of MiBs.
// For reference, each MiB = 2^20 bytes.
func BytesToHigherMiBs(bytes uint64) uint64 {
	if bytes > (MaxMiBsInUint64 << 20) {
		return MaxMiBsInUint64 + 1
	}
	const bytesInOneMiB uint64 = 1 << 20
	return uint64(math.Ceil(float64(bytes) / float64(bytesInOneMiB)))
}

// kernelVersion is just a wrapper over GetKernelVersion. This
// allows us to mock it in the unit test of ShouldSkipKernelListCacheTest.
var kernelVersionToTest = func() (string, error) {
	return GetKernelVersion()
}

// IsKLCacheEvictionUnSupported returns true if Kernel List Cache Eviction is not supported
// for the current linux version.
// In case of any non-nil error it returns false.
func IsKLCacheEvictionUnSupported() (bool, error) {
	UnsupportedKernelVersions := []string{`^6\.9\.\d+`, `^6\.10\.\d+`, `^6\.11\.\d+`, `^6\.12\.\d+`}

	kernelVersion, err := kernelVersionToTest()
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
