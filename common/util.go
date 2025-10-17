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
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type ShutdownFn func(ctx context.Context) error

// JoinShutdownFunc combines the provided shutdown functions into a single function.
func JoinShutdownFunc(shutdownFns ...ShutdownFn) ShutdownFn {
	return func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFns {
			if fn == nil {
				continue
			}
			err = errors.Join(err, fn(ctx))
		}
		return err
	}
}

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

	for i := range UnsupportedKernelVersions {
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

func CloseFile(file *os.File) {
	if err := file.Close(); err != nil {
		log.Fatalf("error in closing: %v", err)
	}
}

func WriteFile(fileName string, content string) (err error) {
	f, err := os.OpenFile(fileName, os.O_RDWR, 0600)
	if err != nil {
		err = fmt.Errorf("open file for write at start: %v", err)
		return
	}

	// Closing file at the end.
	defer CloseFile(f)

	_, err = f.WriteAt([]byte(content), 0)

	return
}

func ReadFile(filePath string) (content []byte, err error) {
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0600)
	if err != nil {
		err = fmt.Errorf("error in the opening the file %v", err)
		return
	}

	// Closing file at the end.
	defer CloseFile(f)

	content, err = os.ReadFile(f.Name())
	if err != nil {
		err = fmt.Errorf("ReadAll: %v", err)
		return
	}
	return
}
