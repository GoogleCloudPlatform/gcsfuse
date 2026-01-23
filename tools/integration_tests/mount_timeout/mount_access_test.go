// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mount_timeout

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	// This role is a custom role and granting this role to any service account grants only storage.objects.list permission.
	// Custom roles follow the naming pattern projects/<project-id>/roles/<custom-role-name>
	listPermCustomRoleName = "storage.objects.list"
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

func (testSuite *MountAccessTest) TestMountingWithMinimalAccessSucceeds() {
	serviceAccount, localKeyFilePath := creds_tests.CreateCredentials(gCtx)
	creds_tests.ApplyCustomRoleToServiceAccountOnBucket(gCtx, gStorageClient, serviceAccount, listPermCustomRoleName, setup.TestBucket())
	defer creds_tests.RevokeCustomRoleFromServiceAccountOnBucket(gCtx, gStorageClient, serviceAccount, listPermCustomRoleName, setup.TestBucket())

	err := testSuite.mountWithKeyFile(setup.TestBucket(), localKeyFilePath)

	assert.NoError(testSuite.T(), err)
}

func TestMountAccess(t *testing.T) {
	// Set log file.
	setup.SetLogFile(fmt.Sprintf("%s%s.txt", logfilePathPrefix, t.Name()))

	suite.Run(t, &MountAccessTest{})
}
