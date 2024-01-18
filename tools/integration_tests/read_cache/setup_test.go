// Copyright 2024 Google Inc. All Rights Reserved.
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

package read_cache

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	testDirName           = "ReadCacheTest"
	onlyDirMounted        = "Test"
	cacheSubDirectoryName = "gcsfuse-file-cache"
	smallContent          = "small content"
	smallContentSize      = 13
	MiB                   = 1024 * 1024
	chunkSizeToRead       = MiB
	fileSize              = 3 * MiB
	chunksRead            = fileSize / MiB
	testFileName          = "foo"
	cacheCapacityInMB     = 9
	largeFileSize         = 15 * MiB
	largeFileName         = "15MBFile"
	largeFileChunksRead   = 15
	chunksReadAfterUpdate = 1
	metadataCacheTTlInSec = 10
)

var (
	testDirPath       string
	cacheLocationPath string
	mountFunc         func([]string) error
	// mount directory is where our tests run.
	mountDir string
	// root directory is the directory to be unmounted.
	rootDir string
)

func createConfigFile(cacheSize int64) string {
	cacheLocationPath = path.Join(setup.TestDir(), "cache-dir")

	// Set up config file for file cache.
	mountConfig := config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			// Keeping the size as low because the operations are performed on small
			// files
			MaxSizeInMB: cacheSize,
		},
		CacheLocation: config.CacheLocation(cacheLocationPath),
		LogConfig: config.LogConfig{
			Severity:        config.TRACE,
			Format:          "json",
			FilePath:        setup.LogFile(),
			LogRotateConfig: config.DefaultLogRotateConfig(),
		},
	}
	filePath := setup.YAMLConfigFile(mountConfig, "config.yaml")
	return filePath
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	setup.RunTestsForMountedDirectoryFlag(m)

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Save mount and root directory variables.
	mountDir, rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		// Save mount directory variable to have path of bucket to run tests.
		mountDir = path.Join(setup.MntDir(), setup.TestBucket())
		mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMounting
		successCode = m.Run()
	}

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		mountDir = rootDir
		mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDir
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(path.Join(setup.TestBucket(), setup.OnlyDirMounted(), testDirName))
	}

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(path.Join(setup.TestBucket(), testDirName))
	setup.RemoveBinFileCopiedForTesting()
	os.Exit(successCode)
}
