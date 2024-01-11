package read_cache

import (
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
	cacheSubDirectoryName = "gcsfuse-file-cache"
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

func createConfigFile() string {
	cacheLocationPath = path.Join(setup.TestDir(), "cache-dir")

	// Set up config file for file cache.
	mountConfig := config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			// Keeping the size as low because the operations are performed on small
			// files
			MaxSizeInMB: 10,
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

	// Run static mounting tests.
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()

	// Run dynamic mounting tests.
	// Save mount directory variable to have path of bucket to run tests.
	mountDir = path.Join(setup.MntDir(), setup.TestBucket())
	mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMounting
	successCode = m.Run()

	// OnlyDir mounting tests.
	setup.SetOnlyDirMounted(testDirName)
	mountDir = rootDir
	mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDir

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(path.Join(setup.TestBucket(), testDirName))
	setup.RemoveBinFileCopiedForTesting()
	os.Exit(successCode)
}
