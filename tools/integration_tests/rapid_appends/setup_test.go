package rapid_appends

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName           = "RapidAppendsTest"
	fileNamePrefix        = "rapid-append-file-"
	contentSizeForBW      = 3
	blockSize             = operations.OneMiB
	numAppends            = 2
	appendSize            = 10
	unfinalizedObjectSize = 10
	metadataCacheTTLSecs  = 70
	fileOpenModeRPlus     = os.O_RDWR
	fileOpenModeAppend    = os.O_APPEND | os.O_WRONLY
)

var (
	testEnv env
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	cfg           *test_suite.TestConfig
	bucketType    string
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.RapidAppends) == 0 {
		log.Println("No configuration found for rapid_appends tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.RapidAppends = make([]test_suite.TestConfig, 1)
		cfg.RapidAppends[0].TestBucket = setup.TestBucket()
		cfg.RapidAppends[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.RapidAppends[0].LogFile = setup.LogFile()
		cfg.RapidAppends[0].Configs = make([]test_suite.ConfigItem, 4)

		// 1. TestSingleMountAppendsTestSuite
		cfg.RapidAppends[0].Configs[0].Flags = []string{"--write-block-size-mb=1"}
		cfg.RapidAppends[0].Configs[0].Compatible = map[string]bool{"flat": false, "hns": false, "zonal": true}
		cfg.RapidAppends[0].Configs[0].Run = "TestSingleMountAppendsTestSuite"

		// 2. TestDualMountAppendsTestSuite
		cfg.RapidAppends[0].Configs[1].Flags = []string{"--write-block-size-mb=1"}
		cfg.RapidAppends[0].Configs[1].SecondaryFlags = []string{"--write-block-size-mb=1"}
		cfg.RapidAppends[0].Configs[1].Compatible = map[string]bool{"flat": false, "hns": false, "zonal": true}
		cfg.RapidAppends[0].Configs[1].Run = "TestDualMountAppendsTestSuite"

		// 3. TestSingleMountReadsTestSuite
		cfg.RapidAppends[0].Configs[2].Flags = []string{
			"",                             // NoCache
			"--metadata-cache-ttl-secs=70", // MetadataCache
			"--file-cache-max-size-mb=-1 --cache-dir=/gcsfuse-tmp/cache",                              // FileCache
			"--metadata-cache-ttl-secs=70 --file-cache-max-size-mb=-1 --cache-dir=/gcsfuse-tmp/cache", // MetadataAndFileCache
		}
		cfg.RapidAppends[0].Configs[2].Compatible = map[string]bool{"flat": false, "hns": false, "zonal": true}
		cfg.RapidAppends[0].Configs[2].Run = "TestSingleMountReadsTestSuite"

		// 4. TestDualMountReadsTestSuite
		cfg.RapidAppends[0].Configs[3].Flags = []string{
			"",
			"--metadata-cache-ttl-secs=70",
			"--file-cache-max-size-mb=-1 --cache-dir=/gcsfuse-tmp/cache-primary",
			"--metadata-cache-ttl-secs=70 --file-cache-max-size-mb=-1 --cache-dir=/gcsfuse-tmp/cache-primary",
		}
		cfg.RapidAppends[0].Configs[3].SecondaryFlags = []string{
			"--write-block-size-mb=1",
			"--write-block-size-mb=1",
			"--write-block-size-mb=1",
			"--write-block-size-mb=1",
		}
		cfg.RapidAppends[0].Configs[3].Compatible = map[string]bool{"flat": false, "hns": false, "zonal": true}
		cfg.RapidAppends[0].Configs[3].Run = "TestDualMountReadsTestSuite"
	}

	testEnv.ctx = context.Background()
	testEnv.cfg = &cfg.RapidAppends[0]
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, testEnv.cfg)
	if !setup.IsZonalBucketRun() {
		log.Fatalf("This test package is only compatible for zonal bucket runs")
	}

	// 2. Create storage client before running tests.
	var err error
	testEnv.storageClient, err = client.CreateStorageClient(testEnv.ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer testEnv.storageClient.Close()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		// For GKE, we expect both directories to be mounted if it's a dual mount test.
		// If using config, GKEMountedDirectorySecondary should be set.
		testEnv.cfg.GCSFuseMountedDirectory = testEnv.cfg.GKEMountedDirectory
		testEnv.cfg.GCSFuseMountedDirectorySecondary = testEnv.cfg.GKEMountedDirectorySecondary
		os.Exit(m.Run())
	}

	// For GCE environment
	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	// Override GKE specific paths with GCSFuse paths if running in GCE environment.
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())

	// For dual mount, we create another directory.
	secondaryDir, err := os.MkdirTemp(setup.TestDir(), "gcsfuse-secondary-mount")
	if err != nil {
		log.Fatalf("Failed to create secondary mount directory: %v", err)
	}
	testEnv.cfg.GCSFuseMountedDirectorySecondary = secondaryDir

	log.Println("Running static mounting tests for rapid appends...")
	successCode := m.Run()

	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
