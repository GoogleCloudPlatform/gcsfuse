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

package flag_optimizations

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/kernelparams"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

func tearDownOptimizationTest(t *testing.T) {
	setup.SaveGCSFuseLogFileInCaseOfFailure(t)
	setup.UnmountGCSFuseWithConfig(&testEnv.cfg)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func validateConfigValues(t *testing.T, logFile string, requiredLogKey string, expectedConfig map[string]interface{}) {
	// Open log file to verify config
	file, err := os.Open(logFile)
	require.NoError(t, err)
	defer file.Close()

	var configFound bool
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			// Skip non-JSON lines (e.g. command invocation)
			continue
		}
		// Check if the log entry contains the required key
		if _, ok := entry[requiredLogKey]; !ok {
			continue
		}

		if fullConfigMap, ok := entry["Full Config"].(map[string]interface{}); ok {
			configFound = true
			for key, expectedVal := range expectedConfig {
				var actualVal interface{}
				if valMap, ok := fullConfigMap[key].(map[string]interface{}); ok {
					actualVal = valMap["final_value"]
				}
				assert.EqualValues(t, expectedVal, actualVal, "Config %q mismatch", key)
			}
			break
		}
	}
	assert.True(t, configFound, "GCSFuse Config log not found")
}

////////////////////////////////////////////////////////////////////////
// Test Functions
////////////////////////////////////////////////////////////////////////

func TestImplicitDirsNotEnabled(t *testing.T) {
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			mustMountGCSFuseAndSetupTestDir(flags, testEnv.ctx, testEnv.storageClient)
			defer tearDownOptimizationTest(t)

			// Arrange
			implicitDirPath := filepath.Join(testDirName, "implicitDir"+setup.GenerateRandomString(5))
			mountedImplicitDirPath := filepath.Join(setup.MntDir(), implicitDirPath)
			client.CreateImplicitDir(testEnv.ctx, testEnv.storageClient, implicitDirPath, t)
			defer func() {
				err := client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, implicitDirPath)
				require.NoError(t, err)
			}()

			// Act
			_, err := os.Stat(mountedImplicitDirPath)

			// Assert
			require.Error(t, err, "Found unexpected implicit directory %q", mountedImplicitDirPath)
		})
	}
}

func TestZonalBucketOptimizations_LogVerification(t *testing.T) {
	if setup.IsDynamicMount(testEnv.mountDir, testEnv.rootDir) {
		t.Skip("Skipping test for dynamic mounting")
	}

	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			mustMountGCSFuseAndSetupTestDir(flags, testEnv.ctx, testEnv.storageClient)
			defer tearDownOptimizationTest(t)

			expectedConfig := map[string]interface{}{
				"file-system.enable-kernel-reader": true,
				"file-system.max-read-ahead-kb":    16384,
				"file-system.max-background":       cfg.DefaultMaxBackground(),
				"file-system.congestion-threshold": cfg.DefaultCongestionThreshold(),
			}
			validateConfigValues(t, testEnv.cfg.LogFile, "Applied optimizations for bucket-type: ", expectedConfig)
		})
	}
}

func TestZonalBucketOptimizations_KernelParamVerification(t *testing.T) {
	if setup.IsDynamicMount(testEnv.mountDir, testEnv.rootDir) {
		t.Skip("Skipping test for dynamic mounting")
	}

	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			mustMountGCSFuseAndSetupTestDir(flags, testEnv.ctx, testEnv.storageClient)
			defer tearDownOptimizationTest(t)

			// Verify kernel parameters in /sys
			var stat unix.Stat_t
			err := unix.Stat(setup.MntDir(), &stat)
			require.NoError(t, err)
			devMajor := unix.Major(stat.Dev)
			devMinor := unix.Minor(stat.Dev)
			readAheadPath, err := kernelparams.PathForParam(kernelparams.MaxReadAheadKb, devMajor, devMinor)
			require.NoError(t, err)
			maxBackgroundPath, err := kernelparams.PathForParam(kernelparams.MaxBackgroundRequests, devMajor, devMinor)
			require.NoError(t, err)
			congestionThresholdPath, err := kernelparams.PathForParam(kernelparams.CongestionWindowThreshold, devMajor, devMinor)
			require.NoError(t, err)
			expected := map[string]string{
				readAheadPath:           "16384",
				maxBackgroundPath:       fmt.Sprintf("%d", cfg.DefaultMaxBackground()),
				congestionThresholdPath: fmt.Sprintf("%d", cfg.DefaultCongestionThreshold()),
			}

			for path, val := range expected {
				content, err := os.ReadFile(path)

				require.NoError(t, err)
				assert.Equal(t, val, strings.TrimSpace(string(content)))
			}
		})
	}
}

func TestRenameDirLimitNotSet(t *testing.T) {
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			mustMountGCSFuseAndSetupTestDir(flags, testEnv.ctx, testEnv.storageClient)
			defer tearDownOptimizationTest(t)

			// Arrange
			srcDirPath := filepath.Join(testDirName, "srcDirContainingFiles"+setup.GenerateRandomString(5))
			mountedSrcDirPath := filepath.Join(setup.MntDir(), srcDirPath)
			dstDirPath := filepath.Join(testDirName, "dstDirContainingFiles"+setup.GenerateRandomString(5))
			mountedDstDirPath := filepath.Join(setup.MntDir(), dstDirPath)
			require.NoError(t, client.CreateGcsDir(testEnv.ctx, testEnv.storageClient, srcDirPath, setup.TestBucket(), ""))
			client.CreateNFilesInDir(testEnv.ctx, testEnv.storageClient, 1, "file", 1024, srcDirPath, t)
			defer func() {
				err := client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, srcDirPath)
				require.NoError(t, err)
				err = client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, dstDirPath)
				require.NoError(t, err)
			}()

			// Act
			err := os.Rename(mountedSrcDirPath, mountedDstDirPath)

			// Assert
			require.Error(t, err, "Unexpectedly succeeded in renaming directory %q to %q", mountedSrcDirPath, mountedDstDirPath)
		})
	}
}

func TestImplicitDirsEnabled(t *testing.T) {
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			mustMountGCSFuseAndSetupTestDir(flags, testEnv.ctx, testEnv.storageClient)
			defer tearDownOptimizationTest(t)

			// Arrange
			implicitDirPath := filepath.Join(testDirName, "implicitDir"+setup.GenerateRandomString(5))
			mountedImplicitDirPath := filepath.Join(setup.MntDir(), implicitDirPath)
			client.CreateImplicitDir(testEnv.ctx, testEnv.storageClient, implicitDirPath, t)
			defer func() {
				err := client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, implicitDirPath)
				require.NoError(t, err)
			}()

			// Act
			fi, err := os.Stat(mountedImplicitDirPath)

			// Assert
			require.NoError(t, err, "Got error statting %q: %v", mountedImplicitDirPath, err)
			require.NotNil(t, fi, "Expected directory %q", mountedImplicitDirPath)
			assert.True(t, fi.IsDir(), "Expected %q to be a directory, but got not-dir", mountedImplicitDirPath)
		})
	}
}

func TestRenameDirLimitSet(t *testing.T) {
	flagsSet := setup.BuildFlagSets(testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			mustMountGCSFuseAndSetupTestDir(flags, testEnv.ctx, testEnv.storageClient)
			defer tearDownOptimizationTest(t)

			// Arrange
			srcDirPath := filepath.Join(testDirName, "srcDirContainingFiles"+setup.GenerateRandomString(5))
			mountedSrcDirPath := filepath.Join(setup.MntDir(), srcDirPath)
			dstDirPath := filepath.Join(testDirName, "dstDirContainingFiles"+setup.GenerateRandomString(5))
			mountedDstDirPath := filepath.Join(setup.MntDir(), dstDirPath)
			require.NoError(t, client.CreateGcsDir(testEnv.ctx, testEnv.storageClient, srcDirPath, setup.TestBucket(), ""))
			client.CreateNFilesInDir(testEnv.ctx, testEnv.storageClient, 1, "file", 1024, srcDirPath, t)
			defer func() {
				err := client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, srcDirPath)
				require.NoError(t, err)
				err = client.DeleteAllObjectsWithPrefix(testEnv.ctx, testEnv.storageClient, dstDirPath)
				require.NoError(t, err)
			}()

			// Act
			err := os.Rename(mountedSrcDirPath, mountedDstDirPath)

			// Assert
			require.NoError(t, err, "Failed to rename directory %q to %q: %v", mountedSrcDirPath, mountedDstDirPath, err)
		})
	}
}
