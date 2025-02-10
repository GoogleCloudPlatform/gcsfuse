// Copyright 2025 Google LLC
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

package local_file

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type CommonLocalFileTestSuite struct {
	filePath           string
	testDirName        string
	testDirPath        string
	fh                 *os.File
	ctx                context.Context
	storageClient      *storage.Client
	CloseStorageClient func() error
	flags              []string
	test_suite.TestifySuite
}

type localFileTestSuite struct {
	CommonLocalFileTestSuite
	suite.Suite
}

func (t *localFileTestSuite) SetupSuite() {
	log.Println("Setting up suite")
	t.ctx = context.Background()
	t.CloseStorageClient = CreateStorageClientWithCancel(&t.ctx, &t.storageClient)
	if !testing.Short() {
		t.flags = append(t.flags, "--client-protocol=grpc")
	}
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()
	setup.MountGCSFuseWithGivenMountFunc(t.flags, MountFunc)

	if setup.DynamicBucketMounted() != "" {
		// Skip dynamic mounting tests if the mounted bucket is not hierarchical
		if !setup.IsHierarchicalBucket(t.ctx, t.storageClient) {
			t.T().Skip("Skipping Dynamic Mounting tests as the bucket is not hierarchical...")
		} else {
			rootMntDir = setup.MntDir()
			// Changing mntDir to path of bucket mounted in mntDir for testing.
			mntDirOfTestBucket := path.Join(setup.MntDir(), setup.DynamicBucketMounted())
			setup.SetMntDir(mntDirOfTestBucket)
			if setup.DynamicBucketMounted() == dynamic_mounting.GetTestBucketNameForDynamicMounting() {
				// Create dynamic test bucket for dynamic mounting tests.
				dynamic_mounting.CreateTestBucketForDynamicMounting(t.ctx, t.storageClient)
			}
		}
	}
}

func (t *localFileTestSuite) TearDownSuite() {
	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(t.ctx, t.storageClient, path.Join(setup.TestBucket(), LocalFileTestDirName))
	if setup.DynamicBucketMounted() != "" {
		if setup.DynamicBucketMounted() == dynamic_mounting.GetTestBucketNameForDynamicMounting() {
			// Delete dynamic bucket
			dynamic_mounting.DeleteTestBucketForDynamicMounting(t.ctx, t.storageClient, dynamic_mounting.GetTestBucketNameForDynamicMounting())
		}
		// Currently mntDir is mntDir/bucketName.
		// Unmounting can happen on rootMntDir. Changing mntDir to rootMntDir for unmounting.
		setup.SetMntDir(rootMntDir)
	}
	// Close storage client.
	err := t.CloseStorageClient()
	if err != nil {
		log.Fatalf("closeStorageClient failed: %v", err)
	}
	// Unmount GCSFuse.
	setup.UnmountGCSFuse(setup.MntDir())
}

func (t *localFileTestSuite) SetupTest() {
	log.Println("Setting up test")
	t.testDirPath = setup.SetupTestDirectory(t.testDirName)
	t.filePath, t.fh = CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, FileName1, t.T())
}
