// Copyright 2023 Google LLC
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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/common"
)

const (
	// TODO: Remove these constants in favor of common.* constants.
	// Constants for read types - Sequential/Random
	Sequential = common.ReadTypeSequential
	Random     = common.ReadTypeRandom
	Parallel   = common.ReadTypeParallel
)

const (
	GCSFUSE_PARENT_PROCESS_DIR = "gcsfuse-parent-process-dir"

	MaxMiBsInUint64 uint64 = common.MaxMiBsInUint64
	MaxMiBsInInt64  int64  = common.MaxMiBsInInt64
	MiB                    = common.MiB

	// HeapSizeToRssConversionFactor is a constant factor
	// which we multiply to the calculated heap-size
	// to get the corresponding resident set size.
	HeapSizeToRssConversionFactor float64 = 2

	MaxTimeDuration = time.Duration(math.MaxInt64)
)

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

// MiBsToBytes returns the bytes equivalent
// of given number of MiBs.
// For reference, each MiB = 2^20 bytes.
// It supports only upto 2^44-1 MiBs (~4 Tebi MiBs, or ~4 Ebi bytes)
// as inputs, and panics for higher inputs.
func MiBsToBytes(mibs uint64) uint64 {
	if mibs > MaxMiBsInUint64 {
		panic("Inputs above (2^44 - 1) not supported.")
	}
	return mibs << 20
}

// BytesToHigherMiBs returns the MiBs equivalent
// to the given number of bytes.
// If bytes is not an exact number of MiBs,
// then it returns the next higher no. of MiBs.
// For reference, each MiB = 2^20 bytes.
func BytesToHigherMiBs(bytes uint64) uint64 {
	return common.BytesToHigherMiBs(bytes)
}

func GenerateRandomID() string {
	// Create 4 random bytes
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}

	// Encode to an 8-character hex string
	fastID := hex.EncodeToString(bytes)
	return fastID
}
