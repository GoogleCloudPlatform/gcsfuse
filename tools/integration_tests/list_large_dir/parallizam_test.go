package list_large_dir_test

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/list_large_dir"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestWithP(t *testing.T) {
	// Create twelve thousand files in the directoryWithTwelveThousandFiles directory.
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory: %v", err)
	}
	list_large_dir.Test(dirPath, PrefixFileInDirectoryWithTwelveThousandFiles, t)
}

func TestWithOutP(t *testing.T) {
	// Create twelve thousand files in the directoryWithTwelveThousandFiles directory.
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)
	operations.CreateDirectoryWithNFiles(1000, dirPath, PrefixFileInDirectoryWithTwelveThousandFiles, t)
}
