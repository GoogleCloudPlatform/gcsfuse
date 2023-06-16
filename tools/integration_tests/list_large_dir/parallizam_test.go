package list_large_dir_test

import (
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/list_large_dir"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestWithP(t *testing.T) {
	// Create twelve thousand files in the directoryWithTwelveThousandFiles directory.
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)
	//err := os.Mkdir(dirPath, setup.FilePermission_0600)
	//if err != nil {
	//	t.Errorf("Error in creating directory: %v", err)
	//}
	list_large_dir.Test(dirPath, PrefixFileInDirectoryWithTwelveThousandFiles, t)
}

func TestWithOutP(t *testing.T) {
	// Create twelve thousand files in the directoryWithTwelveThousandFiles directory.
	dirPath := path.Join(setup.TestBucket(), DirectoryWithTwelveThousandFiles)
	for i := 0; i < 12000; i++ {
		filePath := path.Join(os.Getenv("HOME"), PrefixFileInDirectoryWithTwelveThousandFiles+strconv.Itoa(i))
		_, err := os.Create(filePath)
		if err != nil {
			t.Errorf("Error in creating file.")
		}
		setup.RunScriptForTestData("testdata/create_objects.sh", dirPath)
	}
}
