// Copyright 2026 Google LLC
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

package parallel

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/persistent_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/require"
)

// RunConfiguration defines the pre-compiled execution plan for a single worker task.
type RunConfiguration struct {
	MountType       setup.MountType
	AuthMode        setup.AuthMode
	Flags           []string
	MntDir          string
	LogFile         string
	OnlyDir         string // Populated only if MountType == OnlyDirMount
	IsolatedTaskDir string // Tracked for cleanup
	TestBucket      string
	FlagSetIndex    int
	Env             map[string]string
}

// RunConfigurations is a collection of execution plans.
type RunConfigurations []RunConfiguration

// BuildExecutionPlan pre-compiles the execution plan for parallel tests.
func BuildExecutionPlan(
	ctx context.Context,
	t *testing.T,
	pkgConfigs []test_suite.TestConfig,
	filterMounts []setup.MountType,
	filterAuthModes []setup.AuthMode,
	keyFilePath string,
) RunConfigurations {
	if len(pkgConfigs) == 0 {
		t.Fatalf("No configuration provided for tests in config.")
	}

	var allConfigs RunConfigurations

	for _, cfg := range pkgConfigs {
		// Use BucketType directly to avoid global state side-effects of TestEnvironment in planner
		bucketType, err := setup.BucketType(ctx, cfg.TestBucket)
		if err != nil {
			t.Fatalf("BucketType failed for %s: %v", cfg.TestBucket, err)
		}
		flagsSet := setup.BuildFlagSets(cfg, bucketType, "")

		allMountingTypes := []setup.MountType{setup.StaticMount, setup.OnlyDirMount, setup.PersistentMount, setup.DynamicMount}

		var mountingTypesToRun []setup.MountType
		if len(filterMounts) > 0 {
			mountingTypesToRun = filterMounts
		} else {
			mountingTypesToRun = allMountingTypes
		}

		allAuthModes := []setup.AuthMode{setup.DefaultAuth}
		var authModesToRun []setup.AuthMode
		if len(filterAuthModes) > 0 {
			authModesToRun = filterAuthModes
		} else {
			authModesToRun = allAuthModes
		}

		for _, mt := range mountingTypesToRun {
			for _, authMode := range authModesToRun {
				for i, flags := range flagsSet {
					runUniqueId := path.Base(setup.TestDir())
					
					dirSuffix := fmt.Sprintf("%s_flagSet%d", mt, i)
					if authMode != "" && authMode != setup.DefaultAuth {
						dirSuffix = fmt.Sprintf("%s_%s_flagSet%d", mt, authMode, i)
					}

					isolatedTaskDir := path.Join(setup.TestDir(), dirSuffix)
					isolatedMntDir := path.Join(isolatedTaskDir, "mnt")
					isolatedLogFile := isolatedMntDir + ".log"

					var onlyDir string
					if mt == setup.OnlyDirMount {
						onlyDir = fmt.Sprintf("onlyDirMounted_%s_%s_flagSet%d", runUniqueId, authMode, i)
					}

					// Override paths in flags for THIS task
					taskFlags := make([]string, len(flags))
					for j, flag := range flags {
						taskFlags[j] = strings.ReplaceAll(flag, "/gcsfuse-tmp", path.Join(isolatedTaskDir, "gcsfuse-tmp"))
					}

					taskEnv := make(map[string]string)
					switch authMode {
					case setup.EnvVar:
						taskEnv["GOOGLE_APPLICATION_CREDENTIALS"] = keyFilePath
					case setup.KeyFileOnly:
						taskFlags = append(taskFlags, "--key-file="+keyFilePath)
					case setup.EnvVarAndKeyFile:
						taskEnv["GOOGLE_APPLICATION_CREDENTIALS"] = keyFilePath
						taskFlags = append(taskFlags, "--key-file="+keyFilePath)
					}

					allConfigs = append(allConfigs, RunConfiguration{
						MountType:       mt,
						AuthMode:        authMode,
						Flags:           taskFlags,
						MntDir:          isolatedMntDir,
						LogFile:         isolatedLogFile,
						OnlyDir:         onlyDir,
						IsolatedTaskDir: isolatedTaskDir,
						TestBucket:      cfg.TestBucket,
						FlagSetIndex:    i,
						Env:             taskEnv,
					})
				}
			}
		}
	}

	return allConfigs
}

// RunParallelTestDriver is the main entry point for test packages to run tests in parallel.
func RunParallelTestDriver(
	t *testing.T,
	ctx context.Context,
	storageClient *storage.Client,
	runCfgs RunConfigurations,
	testRunner func(t *testing.T, runCfg RunConfiguration),
) {
	mountFuncs := map[setup.MountType]func(*test_suite.TestConfig, []string) error{
		setup.StaticMount:     static_mounting.MountGcsfuseWithStaticMountingWithConfigFile,
		setup.OnlyDirMount:    only_dir_mounting.MountGcsfuseWithOnlyDirWithConfigFile,
		setup.PersistentMount: persistent_mounting.MountGcsfuseWithPersistentMountingWithConfigFile,
		setup.DynamicMount:    dynamic_mounting.MountGcsfuseWithDynamicMountingWithConfig,
	}

	var wg sync.WaitGroup

	for _, runCfg := range runCfgs {
		wg.Add(1)

		runCfg := runCfg // capture range variable

		go func() {
			defer wg.Done()

			taskName := fmt.Sprintf("%s/FlagSet_%d", runCfg.MountType, runCfg.FlagSetIndex)
			if runCfg.AuthMode != "" && runCfg.AuthMode != setup.DefaultAuth {
				taskName = fmt.Sprintf("%s/%s/FlagSet_%d", runCfg.MountType, runCfg.AuthMode, runCfg.FlagSetIndex)
			}

			t.Run(taskName, func(t *testing.T) {
				t.Parallel()
				// 1. Isolated Setup
				err := os.MkdirAll(runCfg.IsolatedTaskDir, 0755)
				require.NoError(t, err)
				defer os.RemoveAll(runCfg.IsolatedTaskDir)

				err = os.MkdirAll(runCfg.MntDir, 0755)
				require.NoError(t, err)

				// Create isolated config for mounting
				isolatedCfg := test_suite.TestConfig{
					TestBucket:              runCfg.TestBucket,
					GCSFuseMountedDirectory: runCfg.MntDir,
					LogFile:                 runCfg.LogFile,
					OnlyDir:                 runCfg.OnlyDir,
					Env:                     runCfg.Env,
				}

				if runCfg.MountType == setup.OnlyDirMount {
					client.SetupTestDirectory(ctx, storageClient, runCfg.OnlyDir)
					defer func() {
						err := client.DeleteAllObjectsWithPrefix(ctx, storageClient, runCfg.OnlyDir)
						if err != nil {
							log.Printf("Error deleting isolated only-dir %s on GCS: %v", runCfg.OnlyDir, err)
						}
					}()
				}

				mountFunc, ok := mountFuncs[runCfg.MountType]
				require.True(t, ok, "Unknown mount type: %s", runCfg.MountType)

				log.Printf("Mounting GCSFuse for %s with flags: %v", taskName, runCfg.Flags)

				// 2. Mount
				err = mountFunc(&isolatedCfg, runCfg.Flags)
				require.NoError(t, err)
				t.Cleanup(func() {
					log.Printf("Unmounting GCSFuse for %s/FlagSet_%d", runCfg.MountType, runCfg.FlagSetIndex)
					setup.UnmountGCSFuseWithConfig(&isolatedCfg)
				})

				// 3. Run Tests
				testRunner(t, runCfg)
			})
		}()
	}
	wg.Wait()
}


