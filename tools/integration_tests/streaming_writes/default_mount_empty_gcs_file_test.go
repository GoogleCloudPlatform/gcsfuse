package streaming_writes

import (
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

type defaultMountEmptyGCSFile struct {
	defaultMountCommonTest
}

func (t *defaultMountEmptyGCSFile) SetupTest() {
	t.createEmptyGCSFile()
}

func (t *defaultMountEmptyGCSFile) SetupSubTest() {
	t.createEmptyGCSFile()
}

func (t *defaultMountEmptyGCSFile) createEmptyGCSFile() {
	t.fileName = FileName1 + setup.GenerateRandomString(5)
	// Create an empty file on GCS.
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName, t.fileName, "", t.T())
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, t.fileName, "", t.T())
	filePath := path.Join(testDirPath, t.fileName)
	t.f1 = operations.OpenFile(filePath, t.T())
}

// Executes all tests that run with single streamingWrites configuration for empty GCS Files.
func TestDefaultMountEmptyGCSFileTest(t *testing.T) {
	suite.Run(t, new(defaultMountEmptyGCSFile))
}
