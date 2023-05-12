package only_dir_mounting

import (
	"fmt"
	"log"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const DirectoryInTestBucket = "Test"

func MountGcsfuseWithOnlyDir(flags []string, dir string) (err error) {
	defaultArg := []string{"--only-dir",
		dir,
		"--debug_gcs",
		"--debug_fs",
		"--debug_fuse",
		"--log-file=" + setup.LogFile(),
		"--log-format=text",
		setup.TestBucket(),
		setup.MntDir()}

	err = setup.MountGcsfuse(defaultArg, flags)

	return err
}

func mountGcsFuseForFlags(flags [][]string, m *testing.M) (successCode int) {
	var err error

	// "Test" directory not exist in bucket.
	for i := 0; i < len(flags); i++ {
		if err = MountGcsfuseWithOnlyDir(flags[i], DirectoryInTestBucket); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}
		setup.ExecuteTestForFlags(flags[i], m)
	}

	// "Test" directory exist in bucket.
	mountDir := path.Join(setup.MntDir(), DirectoryInTestBucket)
	// Clean the bucket for readonly testing.
	setup.RunScriptForTestData("testdata/delete_objects.sh", setup.TestBucket())

	// Create Test directory in bucket.
	setup.RunScriptForTestData("testdata/create_objects.sh", mountDir)

	for i := 0; i < len(flags); i++ {
		if err = MountGcsfuseWithOnlyDir(flags[i], DirectoryInTestBucket); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}
		setup.ExecuteTestForFlags(flags[i], m)
	}

	setup.RunScriptForTestData("testdata/delete_objects.sh", setup.TestBucket())
	return
}

func RunTests(flags [][]string, m *testing.M) (successCode int) {
	setup.ParseSetUpFlags()

	setup.RunTests(m)

	successCode = mountGcsFuseForFlags(flags, m)

	log.Printf("Test log: %s\n", setup.LogFile())

	return successCode
}
