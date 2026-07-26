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

package stale_handle

import (
	"log"
	"os"
	"path"
	"syscall"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type staleFileHandleOpenFile struct {
	suite.Suite
	flags    []string
	f1       *os.File
	fileName string
	data     string
}

func (s *staleFileHandleOpenFile) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, mountFunc)
	setup.SetMntDir(mountDir)
	testEnv.testDirPath = SetupTestDirectory(testEnv.ctx, testEnv.storageClient, testDirName)
	s.data = setup.GenerateRandomString(5 * util.MiB)
}

func (s *staleFileHandleOpenFile) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *staleFileHandleOpenFile) SetupTest() {
	s.fileName = path.Base(s.T().Name()) + setup.GenerateRandomString(5)
	err := CreateObjectOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testDirName, s.fileName), "")
	assert.NoError(s.T(), err)
	s.f1 = operations.OpenFileWithODirect(s.T(), path.Join(testEnv.testDirPath, s.fileName))
}

func (s *staleFileHandleOpenFile) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *staleFileHandleOpenFile) TestOpenFileWhenClobbered() {
	defer func() { _ = s.f1.Close() }()

	// 1. Get old inode ID.
	fi1, err := os.Stat(s.f1.Name())
	assert.NoError(s.T(), err)
	stat1 := fi1.Sys().(*syscall.Stat_t)
	oldInodeId := stat1.Ino

	// 2. Clobber the file on GCS.
	err = WriteToObject(testEnv.ctx, testEnv.storageClient, path.Join(testDirName, s.fileName), FileContents, storage.Conditions{})
	assert.NoError(s.T(), err)

	// 3. Open the file again. It should succeed (VFS retry).
	fh2, err := os.OpenFile(path.Join(testEnv.testDirPath, s.fileName), os.O_RDWR|syscall.O_DIRECT, 0)
	assert.NoError(s.T(), err)
	defer operations.CloseFileShouldNotThrowError(s.T(), fh2)

	// 4. Get new inode ID.
	fi2, err := fh2.Stat()
	assert.NoError(s.T(), err)
	stat2 := fi2.Sys().(*syscall.Stat_t)
	newInodeId := stat2.Ino

	// 5. Verify they are different.
	assert.NotEqual(s.T(), oldInodeId, newInodeId)

	// 6. Verify we can write to the new fd.
	_, err = fh2.Write([]byte("chips"))
	assert.NoError(s.T(), err)
}

func TestStaleHandleOpenFileWhenClobbered(t *testing.T) {
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, new(staleFileHandleOpenFile))
		return
	}

	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		s := new(staleFileHandleOpenFile)
		s.flags = flags
		log.Printf("Running OpenFile tests with flags: %s", s.flags)
		suite.Run(t, s)
	}
}
