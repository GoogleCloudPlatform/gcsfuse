package read_cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
)

const (
	MiB          = 1024 * 1024
	fileSize     = 3 * MiB
	chunksRead   = fileSize / MiB
	testFileName = "foo"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type testStruct struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *testStruct) Setup(t *testing.T) {
	if setup.MountedDirectory() != "" {
		t.Log("Skipping setup as tests are running test in GKE environment")
		return
	}
	// Mount GCSFuse.
	if err := mountFunc(s.flags); err != nil {
		t.Errorf("Failed to mount GCSFuse: %v", err)
	}
	setup.SetMntDir(mountDir)
	testDirPath = client.SetupTestDirectoryForROMount(s.ctx, s.storageClient, testDirName)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, testFileName, fileSize, t)
}

func (s *testStruct) Teardown(t *testing.T) {
	if setup.MountedDirectory() != "" {
		t.Log("Skipping teardown as tests are running in GKE environment")
		return
	}
	// unmount gcsfuse
	setup.SetMntDir(rootDir)
	err := setup.UnMount()
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
	}
	// delete log file created
	err = os.Remove(setup.LogFile())
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in deleting log file: %v", err))
	}
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *testStruct) TestSecondSequentialReadIsCacheHit(t *testing.T) {
	// Read file 1st time.
	expectedOutcome1 := readFileAndGetExpectedOutcome(testDirPath, testFileName, t)
	s.validateFileInCacheDirectory(t)
	// Read file 2nd time.
	expectedOutcome2 := readFileAndGetExpectedOutcome(testDirPath, testFileName, t)

	// Validate that the content read by read operation matches content on GCS.
	client.ValidateObjectContentsFromGCS(s.ctx, s.storageClient, testDirName, testFileName,
		expectedOutcome1.content, t)
	client.ValidateObjectContentsFromGCS(s.ctx, s.storageClient, testDirName, testFileName,
		expectedOutcome2.content, t)
	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, t)
	validate(expectedOutcome2, structuredReadLogs[1], true, true, chunksRead, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func Test(t *testing.T) {
	// Define flag set to run the tests.
	mountConfigFilePath := createConfigFile()
	flagSet := [][]string{
		{"--implicit-dirs=true", "--config-file=" + mountConfigFilePath},
		{"--implicit-dirs=false", "--config-file=" + mountConfigFilePath},
	}

	// Create storage client before running tests.
	var err error
	ts := &testStruct{}
	ts.ctx = context.Background()
	ctx, cancel := context.WithTimeout(ts.ctx, time.Minute*15)
	ts.storageClient, err = client.CreateStorageClient(ctx)
	if err != nil {
		log.Fatalf("client.CreateStorageClient: %v", err)
	}
	// Defer close storage client and release resources.
	defer func() {
		err := ts.storageClient.Close()
		if err != nil {
			t.Log("Failed to close storage client")
		}
		defer cancel()
	}()

	// Run tests.
	for _, flags := range flagSet {
		// Run tests without ro flag.
		ts.flags = flags
		test_setup.RunTests(t, ts)
		// Run tests with ro flag.
		ts.flags = append(flags, "--o=ro")
		test_setup.RunTests(t, ts)
	}
}
