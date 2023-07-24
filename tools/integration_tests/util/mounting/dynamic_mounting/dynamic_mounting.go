//Copyright 2023 Google Inc. All Rights Reserved.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package dynamic_mounting

import (
	"fmt"
	"log"
	"path"
	"testing"

	"cloud.google.com/go/compute/metadata"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const BucketForDynamicMountingTest = "gcsfuse-dynamic-mounting-test"

func mountGcsfuseWithDynamicMounting(flags []string) (err error) {
	defaultArg := []string{"--debug_gcs",
		"--debug_fs",
		"--debug_fuse",
		"--log-file=" + setup.LogFile(),
		"--log-format=text",
		setup.MntDir()}

	for i := 0; i < len(defaultArg); i++ {
		flags = append(flags, defaultArg[i])
	}

	err = mounting.MountGcsfuse(setup.BinFile(), flags)

	return err
}

func runTestsOnGivenMountedTestBucket(bucketName string, flags []string, m *testing.M) (successCode int) {
	mntDirOfTestBucket := path.Join(setup.MntDir(), bucketName)
	setup.SetMntDir(mntDirOfTestBucket)
	successCode = setup.ExecuteTestForFlagsSet(flags, m)
	return
}

func executeTestsForDynamicMounting(flags [][]string, m *testing.M) (successCode int) {
	var err error

	mntDir := setup.MntDir()

	for i := 0; i < len(flags); i++ {
		if err = mountGcsfuseWithDynamicMounting(flags[i]); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}

		// In dynamic mounting all the buckets mounted in mntDir which user has permission.
		// mntDir - bucket1, bucket2, bucket3, ...
		// We will test on passed testBucket and created bucket.
		successCode = runTestsOnGivenMountedTestBucket(setup.TestBucket(), flags[i], m)
		successCode = runTestsOnGivenMountedTestBucket(BucketForDynamicMountingTest, flags[i], m)
	}

	// Setting back the original mntDir after testing.
	setup.SetMntDir(mntDir)
	return
}

func RunTests(flags [][]string, m *testing.M) (successCode int) {
	project_id, err := metadata.ProjectID()
	if err != nil {
		log.Printf("Error in fetching project id: %v", err)
	}

	setup.RunScriptForTestData("../util/mounting/dynamic_mounting/testdata/create_bucket.sh", BucketForDynamicMountingTest, project_id)

	successCode = executeTestsForDynamicMounting(flags, m)

	log.Printf("Test log: %s\n", setup.LogFile())

	// Deleting bucket after testing.
	setup.RunScriptForTestData("../util/mounting/dynamic_mounting/testdata/delete_bucket.sh", "gcsfuse-dynamic-mounting-test")

	return successCode
}
