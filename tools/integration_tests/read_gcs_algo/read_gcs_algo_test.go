package read_gcs_algo

import (
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const OneMB = 1024 * 1024
const DirForReadAlgoTests = "dirForReadAlgoTests"

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()
	// Do not enable fileCache as we want to test gcs read flow.
	mountConfigFlags := [][]string{{"--implicit-dirs=true"}}
	successCode := static_mounting.RunTests(mountConfigFlags, m)
	os.Exit(successCode)
}
