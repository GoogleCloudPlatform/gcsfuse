package mount_timeout

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	onlyListPermCustomRole = "roles/storage.objects.list"
)

type MountAccessTest struct {
	suite.Suite
	// Path to the gcsfuse binary.
	gcsfusePath string

	// A temporary directory into which a file system may be mounted. Removed in
	// TearDown.
	dir string
}

func (testSuite *MountAccessTest) SetupTest() {
	var err error
	testSuite.gcsfusePath = path.Join(gBuildDir, "bin/gcsfuse")
	// Set up the temporary directory.
	testSuite.dir, err = os.MkdirTemp("", "mount_timeout_test")
	assert.NoError(testSuite.T(), err)
}

func (testSuite *MountAccessTest) TearDownTest() {
	err := os.Remove(testSuite.dir)
	assert.NoError(testSuite.T(), err)
}

// mountWithKeyFile mounts the bucket with the given key file.
// It returns any error during mounting or unmounting.
func (testSuite *MountAccessTest) mountWithKeyFile(bucketName, keyFile string) (err error) {
	logFile := setup.LogFile()
	defer func() {
		if err != nil {
			setup.SaveLogFileAsArtifact(logFile, setup.GCSFuseLogFilePrefix+filepath.Base(logFile))
		}
	}()

	args := []string{"--key-file=" + keyFile, "--log-severity=trace", "--log-file=" + logFile, bucketName, testSuite.dir}

	if err = mounting.MountGcsfuse(testSuite.gcsfusePath, args); err != nil {
		err = fmt.Errorf("mount failed for bucket %q with key-file having minimal access: %w", bucketName, err)
		return err
	}
	if err = unmountAndWait(testSuite.dir); err != nil {
		err = fmt.Errorf("unmountAndWait failed for bucket %q on mount point %q with err: %w", bucketName, testSuite.dir, err)
		return err
	}
	return nil
}

func (testSuite *MountAccessTest) TestMountWithSAWithMinimalAccessSucceeds() {
	serviceAccount, localKeyFilePath := creds_tests.CreateCredentials(gCtx)
	creds_tests.ApplyCustomRoleToServiceAccountOnBucket(gCtx, gStorageClient, serviceAccount, onlyListPermCustomRole, setup.TestBucket())

	err := testSuite.mountWithKeyFile(setup.TestBucket(), localKeyFilePath)

	assert.NoError(testSuite.T(), err)
}
