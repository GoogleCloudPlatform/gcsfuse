package only_dir_mounting

import (
	"fmt"
	"log"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

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
		if err = MountGcsfuseWithOnlyDir(flags[i], "Test"); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}
		setup.ExecuteTestForFlags(flags[i], m)
	}

	// "Test" directory not exist in bucket.
	for i := 0; i < len(flags); i++ {
		if err = MountGcsfuseWithOnlyDir(flags[i], "Test"); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}
		setup.ExecuteTestForFlags(flags[i], m)
	}
	return
}

func RunTests(flags [][]string, m *testing.M) (successCode int) {
	setup.ParseSetUpFlags()

	setup.RunTests(m)

	successCode = mountGcsFuseForFlags(flags, m)

	log.Printf("Test log: %s\n", setup.LogFile())

	return successCode
}
