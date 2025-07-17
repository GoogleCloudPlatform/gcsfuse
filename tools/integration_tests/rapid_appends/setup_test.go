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

package rapid_appends

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	testDirName                 = "RapidAppendsTest"
	fileNamePrefix              = "rapid-append-file-"
	metadataCacheEnableFlag     = "--metadata-cache-ttl-secs=60"
	metadataCacheDisableFlag    = "--metadata-cache-ttl-secs=0"
	fileCacheMaxSizeFlag        = "--file-cache-max-size-mb=-1"
	cacheDirFlagPrefix          = "--cache-dir="
	writeRapidAppendsEnableFlag = "--write-experimental-enable-rapid-appends=true"
)

type scenarioConfig struct {
	enableMetadataCache bool
	enableFileCache     bool
}

// Struct to store the details of a mount point
type mountPoint struct {
	rootDir     string
	testDirPath string
	logFilePath string
}

var (
	// Flags for mount options for primary mount.
	flags []string
	// Mount function to be used for the mounting.
	mountFunc func([]string) error

	// Structs for primary and secondary mounts to store their details
	primaryMount   mountPoint
	secondaryMount mountPoint

	// Clients to create the object in GCS.
	storageClient *storage.Client
	ctx           context.Context

	// Scenario being run by the current test.
	scenario scenarioConfig
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func scenariosToBeRun() []scenarioConfig {
	return []scenarioConfig{
		{ // Default scenario with no caches enabled.
		},
		{ // Metadata cache enabled.
			enableMetadataCache: true,
		},
		{ // Both metadata and file cache enabled.
			enableMetadataCache: true,
			enableFileCache:     true,
		},
		{ // File cache enabled.
			enableFileCache: true,
		},
	}
}

func flagsFromScenario(scenario scenarioConfig, rapidAppendsCacheDir string) []string {
	metadataCacheEnableFlags := []string{metadataCacheEnableFlag}
	metadataCacheDisableFlags := []string{metadataCacheDisableFlag}
	fileCacheEnableFlags := []string{fileCacheMaxSizeFlag, cacheDirFlagPrefix + rapidAppendsCacheDir}
	fileCacheDisableFlags := []string{}
	commonFlags := []string{writeRapidAppendsEnableFlag}

	flags := commonFlags
	if scenario.enableMetadataCache {
		flags = append(flags, metadataCacheEnableFlags...)
	} else {
		flags = append(flags, metadataCacheDisableFlags...)
	}
	if scenario.enableFileCache {
		flags = append(flags, fileCacheEnableFlags...)
	} else {
		flags = append(flags, fileCacheDisableFlags...)
	}
	return flags
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	if !setup.IsZonalBucketRun() {
		log.Fatalf("This package is not supposed to be run with Regional Buckets.")
	}
	// TODO(b/431926259): Add support for mountedDir tests as this
	// package has multi-mount scenario tests and currently we only
	// pass single mountedDir to test package.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		log.Fatalf("This package doesn't support --mountedDirectory option currently.")
	}
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// Set up test directory for primary mount.
	setup.SetUpTestDirForTestBucketFlag()
	primaryMount.rootDir = setup.MntDir()
	primaryMount.logFilePath = setup.LogFile()
	// TODO(b/432179045): `--write-global-max-blocks=-1` is needed right now because of a bug in global semaphore release.
	// Remove this flag once bug is fixed.
	primaryMountFlags := []string{"--write-experimental-enable-rapid-appends=true", "--metadata-cache-ttl-secs=0", "--write-global-max-blocks=-1"}
	err := static_mounting.MountGcsfuseWithStaticMounting(primaryMountFlags)
	if err != nil {
		log.Fatalf("Unable to mount primary mount: %v", err)
	}
	// Setup Package Test Directory for primary mount.
	primaryMount.testDirPath = setup.SetupTestDirectory(testDirName)
	defer setup.UnmountGCSFuse(primaryMount.rootDir)

	// Set up test directory for secondary mount.
	setup.SetUpTestDirForTestBucketFlag()
	secondaryMount.rootDir = setup.MntDir()
	secondaryMount.logFilePath = setup.LogFile()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting

	var successCode int
	for _, scenario = range scenariosToBeRun() {
		successCode = func() int {
			// Create a cache-dir if needed.
			var rapidAppendsCacheDir string
			if scenario.enableFileCache {
				rapidAppendsCacheDir, err = os.MkdirTemp("", "rapid_appends_cache_dir_*")
				if err != nil {
					log.Fatalf("Failed to create cache dir for rapid append tests: %v", err)
				}
				defer func() {
					err := os.RemoveAll(rapidAppendsCacheDir)
					if err != nil {
						log.Fatalf("Error while cleaning up cache dir %q: %v", rapidAppendsCacheDir, err)
					}
				}()
			}

			flags = flagsFromScenario(scenario, rapidAppendsCacheDir)
			log.Printf("Running tests with flags: %v", flags)
			return m.Run()
		}()
		if successCode != 0 {
			break
		}
	}
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
