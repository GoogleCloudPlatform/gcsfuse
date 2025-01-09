package streaming_writes

import (
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/stretchr/testify/suite"
)

type defaultMountLocalFile struct {
	defaultMountCommonTest
}

func (t *defaultMountLocalFile) SetupTest() {
	// Create a local file.
	_, t.f1 = CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
}

// Executes all tests that run with single streamingWrites configuration for localFiles.
func TestDefaultMountLocalFileTest(t *testing.T) {
	suite.Run(t, new(defaultMountLocalFile))
}
