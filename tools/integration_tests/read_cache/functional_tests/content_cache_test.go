package functional_tests_test

import (
	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_suite"
	"log"
	"os/exec"
	"path"
	"path/filepath"
	"testing"
)

const (
	MiB          = 1024 * 1024
	filePerms    = 0644
	testFileName = "foo"
)

type testStruct struct {
}

func createReadCacheConfigFile() string {
	//cacheLocationPath = path.Join(setup.TestDir(), "cache-dir")
	cacheLocationPath = "/tmp/cache_temp"

	// Set up config file for file cache.
	mountConfig2 := config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			// Keeping the size as low because the operations are performed on small
			// files
			MaxSizeInMB: 10,
		},
		CacheLocation: config.CacheLocation(cacheLocationPath),
		LogConfig: config.LogConfig{
			Severity: config.TRACE,
			Format:   "json",
			FilePath: setup.LogFile(),
		},
	}
	filePath := setup.YAMLConfigFile(mountConfig2, "config.yaml")
	return filePath
}

func setupFileInTestDirectory(t *testing.T, size int64) {
	randomData, err := operations.GenerateRandomData(size)
	if err != nil {
		t.Errorf("operations.GenerateRandomData: %v", err)
	}
	// Setup file with 5 MiB content in test directory.
	testDirPath := path.Join(setup.MntDir(), testDirName)
	filePath := path.Join(testDirPath, testFileName)
	operations.CreateFileWithContent(filePath, filePerms, string(randomData), t)
}

func (s *testStruct) Setup(t *testing.T) {
	t.Log("Running per-test setup code")
	// Mount GCSFuse.
	if setup.MountedDirectory() != "" {
		t.Log("Skipping setup as tests are running test in GKE environment")
		return
	}

	// mount GCSFuse with read cache config file
	flags := []string{"--config-file=" + createReadCacheConfigFile(), setup.TestBucket(), setup.MntDir()}
	mountCmd := exec.Command(
		setup.BinFile(),
		flags...,
	)
	_, err := mountCmd.CombinedOutput()
	if err != nil {
		log.Println(mountCmd.String())
		t.Errorf("Failed to mount GCSFuse with flags = %v: %v\n", flags, err)
	}
	testDirPath = setup.SetupTestDirectory(testDirName)
	setupFileInTestDirectory(t, 5*MiB)
}

func (s *testStruct) Teardown(t *testing.T) {
	t.Log("Running per-test teardown code")
	if setup.MountedDirectory() != "" {
		t.Log("Skipping teardown as tests are running test in GKE environment")
		return
	}
	// unmount gcsfuse
	// delete log file created
}

func (s *testStruct) TestSecondReadIsCacheHit(t *testing.T) {
	// Read file 1st iteration.
	_, err := operations.ReadFile(path.Join(testDirPath, testFileName))
	if err != nil {
		t.Errorf("Failed to read file in first iteration: %v", err)
	}
	// Validate that the file is now present in cache directory
	fileInfo, err := operations.StatFile(path.Join(cacheLocationPath, "gcsfuse-file-cache", setup.TestBucket(), testDirName, testFileName))
	if err != nil {
		t.Errorf("Failed to find cache file at location %v err: %v", cacheLocationPath, err)
	}
	if (*fileInfo).Size() != 5*MiB {
		t.Errorf("Incorrect cached file size. Expected %d, Got: %d", 5*MiB, (*fileInfo).Size())
	}

	// Read file 2nd iteration.
	_, err = operations.ReadFile(path.Join(testDirPath, testFileName))
	if err != nil {
		t.Errorf("Failed to read file in first iteration: %v", err)
	}
	// Parse the log file and validate cache hit and miss from the parsed JSON
	parserScriptPath, err := filepath.Abs("../../../log_parser/json_parser.py")
	if err != nil {
		t.Errorf("failed to fetch path to log parser script: %v", err)
	}
	_, err = operations.ExecuteToolCommandf("python3", "%s %s %s", parserScriptPath, setup.LogFile(), path.Join(setup.TestDir(), "parsed_logs.json"))
	if err != nil {
		t.Errorf("Failed to parse logs %s: %v", setup.LogFile(), err)
	}

}

//func (s *testStruct) TestSomethingElse(t *testing.T) {
//	t.Log("TestSomethingElse")
//}

func Test(t *testing.T) {
	test_suite.RunSubTests(t, &testStruct{})
}
