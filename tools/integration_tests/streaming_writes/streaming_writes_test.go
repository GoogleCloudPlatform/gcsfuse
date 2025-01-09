package streaming_writes

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	testDirName = "StreamingWritesTest"
)

var (
	testDirPath string
	mountFunc   func([]string) error
	// root directory is the directory to be unmounted.
	rootDir       string
	storageClient *storage.Client
	ctx           context.Context
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	if setup.MountedDirectory() != "" {
		// Once streaming writes are enabled by default, we can run all defaultMount tests here.
		log.Printf("These tests will not run with mounted directory..")
		return
	}

	// Create storage client before running tests.
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()
	rootDir = setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()
	os.Exit(successCode)
}
