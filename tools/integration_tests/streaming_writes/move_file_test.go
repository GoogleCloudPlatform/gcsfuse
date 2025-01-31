package streaming_writes

import (
	"path"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/require"
)

func (t *defaultMountEmptyGCSFile) TestMoveBeforeFileIsFlushed() {
	operations.WriteWithoutClose(t.f1, FileContents, t.T())
	operations.WriteWithoutClose(t.f1, FileContents, t.T())
	operations.VerifyStatFile(t.filePath, int64(2*len(FileContents)), FilePerms, t.T())
	err := t.f1.Sync()
	require.NoError(t.T(), err)

	newFile := "newFile.txt"
	destDirPath := path.Join(testDirPath, newFile)
	err = operations.Move(t.filePath, destDirPath)

	// Validate that move didn't throw any error.
	require.NoError(t.T(), err)
	// Verify the new object contents.
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, newFile, FileContents+FileContents, t.T())
	require.NoError(t.T(), t.f1.Close())
	// Check if old object is deleted.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, t.fileName, t.T())
}
