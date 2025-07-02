package unfinalized_appends

import (
	"os"
	"path"
	"syscall"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UnfinalizedAppendsSuite struct {
	appendFileHandle *os.File
	appendableWriter *storage.Writer
	fileName         string
	fileSize         int
	initialContent   string
	suite.Suite
}

func (t *UnfinalizedAppendsSuite) SetupSuite() {
	setup.SetMntDir(gOtherRootDir)
	setup.SetLogFile(gOtherLogFilePath)
	setup.MountGCSFuseWithGivenMountFunc(gFlags, gMountFunc)

	setup.SetMntDir(gRootDir)
	setup.SetLogFile(gLogFilePath)
	setup.MountGCSFuseWithGivenMountFunc(gFlags, gMountFunc)
}

func (t *UnfinalizedAppendsSuite) TearDownSuite() {
	//setup.UnmountGCSFuse(gRootDir)
	//setup.UnmountGCSFuse(gOtherRootDir)
}

func (t *UnfinalizedAppendsSuite) SetupTest() {
	t.fileName = FileName1 + setup.GenerateRandomString(5)
	var err error
	// Create unfinalized object of
	t.appendableWriter = client.CreateUnfinalizedObject(gCtx, t.T(), gStorageClient, path.Join(testDirName, t.fileName), 10)
	t.fileSize = 10

	t.appendFileHandle, err = os.OpenFile(path.Join(gTestDirPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, operations.FilePermission_0600)
	if err != nil {
		t.T().Fatalf("Unable to open unfinalized file %s in append mode: %v", t.fileName, err)
		return
	}
	initialContent, err := operations.ReadFile(path.Join(gTestDirPath, t.fileName))
	if err != nil {
		t.T().Fatalf("Unable to read the unfinalized object %s from GCS: %v", t.fileName, err)
	}
	assert.Equal(t.T(), 10, len(initialContent))
	t.initialContent = string(initialContent)
}

func (t *UnfinalizedAppendsSuite) AppendToFile(appendContent string) {
	n, err := t.appendFileHandle.WriteString(appendContent)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(appendContent), n)
	t.fileSize += n
}

func (t *UnfinalizedAppendsSuite) TestAppendsFromSameMountAreReadableInRealTime() {
	expectedContent := t.initialContent
	for i := 0; i < 3; i++ {
		fileContent := setup.GenerateRandomString(util.MiB + util.MiB)
		t.AppendToFile(fileContent)
		expectedContent += fileContent
		// Sync append file handle.
		operations.SyncFile(t.appendFileHandle, t.T())
		// Read content of file from different mount.
		gotContent, err := operations.ReadFile(path.Join(gOtherTestDirPath, t.fileName))
		//log.Printf("Expected Content: [%v]", expectedContent)
		//log.Printf("Got content [%v]", string(gotContent))
		assert.NoError(t.T(), err)
		assert.Equal(t.T(), t.fileSize, len(gotContent))
		//assert.Equal(t.T(), expectedContent, string(gotContent))
	}
}

func TestUnfinalizedAppendsSuite(t *testing.T) {
	appendSuite := new(UnfinalizedAppendsSuite)
	suite.Run(t, appendSuite)
}
