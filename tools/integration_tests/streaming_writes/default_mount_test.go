package streaming_writes

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
)

type defaultMountTest struct {
}

func (tt *defaultMountTest) Setup(t *testing.T) {
}

func (tt *defaultMountTest) Teardown(t *testing.T) {
}

// Executes all tests that run with single streamingWrites configuration.
func TestWithDefaultMount(t *testing.T) {
	flags := []string{"--enable-streaming-writes=true", "--write-block-size-mb=1", "--write-max-blocks-per-file=2"}
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	defer setup.UnmountGCSFuse(rootDir)
	testDirPath = setup.SetupTestDirectory(testDirName)

	ts := &defaultMountTest{}
	test_setup.RunTests(t, ts)
}
