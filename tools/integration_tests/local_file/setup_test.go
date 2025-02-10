// Copyright 2023 Google LLC
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

// Provides integration tests for file and directory operations.

package local_file

import (
	"log"
	"os"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as local_file tests validates content from the bucket.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		setup.RunTestsForMountedDirectoryFlag(m)
	}

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	log.Println("Running static mounting tests...")
	MountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()
	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted)
		MountFunc = only_dir_mounting.MountGcsfuseWithOnlyDir
		successCode = m.Run()
		setup.SetOnlyDirMounted("")
	}

	// Dynamic mounting tests create a bucket and perform tests on that bucket,
	// which is not a hierarchical bucket. So we skip these tests in setupSuite
	//if we find the bucket is not hierarchical.
	if successCode == 0 {
		MountFunc = dynamic_mounting.MountGcsfuseWithDynamicMounting
		log.Printf("Running dynamic mounting tests for test bucket [%v]...", setup.TestBucket())
		// SetDynamicBucketMounted to the passed test bucket.
		setup.SetDynamicBucketMounted(setup.TestBucket())
		successCode = m.Run()
		if successCode == 0 {
			log.Printf("Running dynamic mounting tests for created test bucket [%v]...", dynamic_mounting.GetTestBucketNameForDynamicMounting())
			// SetDynamicBucketMounted to the passed test bucket.
			setup.SetDynamicBucketMounted(dynamic_mounting.GetTestBucketNameForDynamicMounting())
			successCode = m.Run()
		}
		// Reset SetDynamicBucketMounted to empty after tests are done.
		setup.SetDynamicBucketMounted("")
	}
	os.Exit(successCode)
}

func TestLocalFileTestSuiteWithImplicitDir(t *testing.T) {
	s := new(localFileTestSuite)
	s.CommonLocalFileTestSuite.TestifySuite = &s.Suite
	s.CommonLocalFileTestSuite.flags = []string{"--implicit-dirs=true", "--rename-dir-limit=3"}
	s.CommonLocalFileTestSuite.testDirName = LocalFileTestDirName
	suite.Run(t, s)
}

func TestLocalFileTestSuiteWithoutImplicitDir(t *testing.T) {
	s := new(localFileTestSuite)
	s.CommonLocalFileTestSuite.TestifySuite = &s.Suite
	s.CommonLocalFileTestSuite.flags = []string{"--implicit-dirs=false", "--rename-dir-limit=3"}
	s.CommonLocalFileTestSuite.testDirName = LocalFileTestDirName
	suite.Run(t, s)
}
