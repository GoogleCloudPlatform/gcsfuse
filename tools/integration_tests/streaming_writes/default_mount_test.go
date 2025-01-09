package streaming_writes

import (
	"os"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

type defaultMountCommonTest struct {
	f1 *os.File
	suite.Suite
}

func (t *defaultMountCommonTest) SetupSuite() {
	flags := []string{"--enable-streaming-writes=true", "--write-block-size-mb=1", "--write-max-blocks-per-file=2"}
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (t *defaultMountCommonTest) TearDownSuite() {
	setup.UnmountGCSFuse(rootDir)
}
