// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dentry_cache

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/suite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

type deleteOperationTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	suite.Suite
}

func (s *deleteOperationTest) SetupTest() {
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *deleteOperationTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *deleteOperationTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *deleteOperationTest) SetupSuite() {
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *deleteOperationTest) TestDeleteFileWhenFileIsClobbered() {
	testFileName := s.T().Name()
	// Create a file with initial content directly in GCS.
	filePath := path.Join(testEnv.testDirPath, testFileName)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, testFileName, initialContentSize, s.T())
	// Stat file to cache the entry
	_, err := os.Stat(filePath)
	require.Nil(s.T(), err)
	// Modify the object on GCS.
	objectName := path.Join(testDirName, testFileName) // This is correct, objectName is relative to bucket.
	smallContent, err := operations.GenerateRandomData(updatedContentSize)
	require.Nil(s.T(), err)
	require.Nil(s.T(), client.WriteToObject(s.ctx, s.storageClient, objectName, string(smallContent), storage.Conditions{}))

	// Deleting the file should not give error
	err = os.Remove(filePath)

	assert.Nil(s.T(), err)
}

func TestDeleteOperationTest(t *testing.T) {
	ts := &deleteOperationTest{ctx: context.Background(), storageClient: testEnv.storageClient}

	// Run tests for mounted directory if the flag is set.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
