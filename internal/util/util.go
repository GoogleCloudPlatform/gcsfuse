// Copyright 2023 Google Inc. All Rights Reserved.
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
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
)

const GCSFUSE_PARENT_PROCESS_DIR = "gcsfuse-parent-process-dir"

// Constants for read types - Sequential/Random
const Sequential = "Sequential"
const Random = "Random"

// 1. Returns the same filepath in case of absolute path or empty filename.
// 2. For child process, it resolves relative path like, ./test.txt, test.txt
// ../test.txt etc, with respect to GCSFUSE_PARENT_PROCESS_DIR
// because we execute the child process from different directory and input
// files are provided with respect to GCSFUSE_PARENT_PROCESS_DIR.
// 3. For relative path starting with ~, it resolves with respect to home dir.
func GetResolvedPath(filePath string) (resolvedPath string, err error) {
	if filePath == "" || path.IsAbs(filePath) {
		resolvedPath = filePath
		return
	}

	// Relative path starting with tilda (~)
	if strings.HasPrefix(filePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("fetch home dir: %w", err)
		}
		return filepath.Join(homeDir, filePath[2:]), err
	}

	// We reach here, when relative path starts with . or .. or other than (/ or ~)
	gcsfuseParentProcessDir, _ := os.LookupEnv(GCSFUSE_PARENT_PROCESS_DIR)
	gcsfuseParentProcessDir = strings.TrimSpace(gcsfuseParentProcessDir)
	if gcsfuseParentProcessDir == "" {
		return filepath.Abs(filePath)
	} else {
		return filepath.Join(gcsfuseParentProcessDir, filePath), err
	}
}

func ResolveFilePath(filePath string, configKey string) (resolvedPath string, err error) {
	resolvedPath, err = GetResolvedPath(filePath)
	if filePath == resolvedPath || err != nil {
		return
	}

	logger.Infof("Value of [%s] resolved from [%s] to [%s]\n", configKey, filePath, resolvedPath)
	return resolvedPath, nil
}

// ResolveConfigFilePaths resolved the config file paths specified in the config file.
func ResolveConfigFilePaths(config *config.MountConfig) (err error) {
	config.LogConfig.FilePath, err = ResolveFilePath(config.LogConfig.FilePath, "logging: file")
	if err != nil {
		return
	}
	return
}

// RoundDurationToNextMultiple returns the next multiple of factor
// greater than or equal to the given input duration.
// It works even if duration is negative.
// If panics if factor is 0 or negative.
func RoundDurationToNextMultiple(duration time.Duration, factor time.Duration) time.Duration {
	if factor <= 0 {
		panic("factor <= 0 not supported")
	}
	return factor * time.Duration(int64(math.Ceil(float64(duration)/float64(factor))))
}

func minInt64(a, b int64) int64 {
	if b < a {
		return b
	}
	return a
}

// MinDuration returns the minimum of the two given durations.
func MinDuration(a, b time.Duration) time.Duration {
	return time.Duration(minInt64(a.Nanoseconds(), b.Nanoseconds()))
}
