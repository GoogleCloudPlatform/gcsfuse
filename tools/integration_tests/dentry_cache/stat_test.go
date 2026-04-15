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
	"fmt"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	RetryInterval = 5 * time.Second
	RetryTimeout  = 2 * time.Minute
)

type statWithDentryCacheEnabledTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	suite.Suite
}

func (s *statWithDentryCacheEnabledTest) SetupTest() {
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *statWithDentryCacheEnabledTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *statWithDentryCacheEnabledTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *statWithDentryCacheEnabledTest) SetupSuite() {
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *statWithDentryCacheEnabledTest) TestStatWithDentryCacheEnabled() {
	successfulFilePath := operations.RetryUntil(s.ctx, s.T(), RetryInterval, RetryTimeout, func() (string, error) {
		testFileName := fmt.Sprintf("%s_%d", s.T().Name(), time.Now().UnixNano())
		filePath := path.Join(testEnv.testDirPath, testFileName)
		client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, testFileName, initialContentSize, s.T())

		smallContent, err := operations.GenerateRandomData(updatedContentSize)
		if err != nil {
			return "", err
		}

		// Stat file to cache the entry
		_, err = os.Stat(filePath)
		if err != nil {
			return "", err
		}

		start := time.Now()

		// Modify the object on GCS.
		objectName := path.Join(testDirName, testFileName)
		err = client.WriteToObject(s.ctx, s.storageClient, objectName, string(smallContent), storage.Conditions{})
		if err != nil {
			return "", err
		}

		// Stat again, it should give old cached attributes.
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return "", err
		}

		elapsed := time.Since(start)

		if elapsed >= 2*time.Second {
			return "", fmt.Errorf("operation took %v, exceeding the 2-second TTL", elapsed)
		}

		if fileInfo.Size() != int64(initialContentSize) {
			return "", fmt.Errorf("expected cached size %d, got %d", initialContentSize, fileInfo.Size())
		}

		return filePath, nil
	})

	// Wait for a period more than the timeout (2 second), so that entry expires in cache.
	time.Sleep(2100 * time.Millisecond)
	// Stat again, it should give updated attributes.
	fileInfo, err := os.Stat(successfulFilePath)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), int64(updatedContentSize), fileInfo.Size())
}

func (s *statWithDentryCacheEnabledTest) TestStatWhenFileIsDeletedDirectlyFromGCS() {
	successfulFilePath := operations.RetryUntil(s.ctx, s.T(), RetryInterval, RetryTimeout, func() (string, error) {
		testFileName := fmt.Sprintf("%s_%d", s.T().Name(), time.Now().UnixNano())
		filePath := path.Join(testEnv.testDirPath, testFileName)
		client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, testFileName, initialContentSize, s.T())

		// Stat file to cache the entry
		_, err := os.Stat(filePath)
		if err != nil {
			return "", err
		}

		start := time.Now()

		// Delete the object directly from GCS.
		objectName := path.Join(testDirName, testFileName)
		err = client.DeleteObjectOnGCS(s.ctx, s.storageClient, objectName)
		if err != nil {
			return "", err
		}

		// Stat again, it should give old cached attributes rather than giving not found error.
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return "", err
		}

		elapsed := time.Since(start)

		if elapsed >= 2*time.Second {
			return "", fmt.Errorf("operation took %v, exceeding the 2-second TTL", elapsed)
		}

		if fileInfo.Size() != int64(initialContentSize) {
			return "", fmt.Errorf("expected cached size %d, got %d", initialContentSize, fileInfo.Size())
		}

		return filePath, nil
	})

	// Wait for a period more than the timeout (2 second), so that entry expires in cache.
	time.Sleep(2100 * time.Millisecond)
	// Stat again, it should give error as file does not exist.
	_, err := os.Stat(successfulFilePath)
	assert.NotNil(s.T(), err)
}

func TestStatWithDentryCacheEnabledTest(t *testing.T) {
	ts := &statWithDentryCacheEnabledTest{ctx: context.Background(), storageClient: testEnv.storageClient}

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
