//Copyright 2023 Google LLC
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
	"context"
	"fmt"
	"log"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const PrefixBucketForDynamicMountingTest = "gcsfuse-dynamic-mounting-test-"

func MountGcsfuseWithDynamicMounting(flags []string) (err error) {
	defaultArg := []string{"--log-severity=trace",
		"--log-file=" + setup.LogFile(),
		setup.MntDir()}

	for i := 0; i < len(defaultArg); i++ {
		flags = append(flags, defaultArg[i])
	}

	err = mounting.MountGcsfuse(setup.BinFile(), flags)

	return err
}

func runTestsOnGivenMountedTestBucket(bucketName string, flags [][]string, rootMntDir string, m *testing.M) (successCode int) {
	for i := 0; i < len(flags); i++ {
		if err := MountGcsfuseWithDynamicMounting(flags[i]); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}

		// Changing mntDir to path of bucket mounted in mntDir for testing.
		mntDirOfTestBucket := path.Join(setup.MntDir(), bucketName)

		setup.SetMntDir(mntDirOfTestBucket)

		log.Printf("Running dynamic mounting tests with flags: %s", flags[i])
		// Running tests on flags.
		successCode = setup.ExecuteTest(m)

		// Currently mntDir is mntDir/bucketName.
		// Unmounting can happen on rootMntDir. Changing mntDir to rootMntDir for unmounting.
		setup.SetMntDir(rootMntDir)
		setup.UnMountAndThrowErrorInFailure(flags[i], successCode)
		if successCode != 0 {
			return
		}
	}
	return
}

func executeTestsForDynamicMounting(config *test_suite.TestConfig, flagsSet [][]string, m *testing.M) (successCode int) {
	rootMntDir := setup.MntDir()

	// In dynamic mounting all the buckets mounted in mntDir which user has permission.
	// mntDir - bucket1, bucket2, bucket3, ...

	// SetDynamicBucketMounted to the passed test bucket.
	setup.SetDynamicBucketMounted(config.TestBucket)
	successCode = runTestsOnGivenMountedTestBucket(config.TestBucket, flagsSet, rootMntDir, m)

	// Reset SetDynamicBucketMounted to empty after tests are done.
	setup.SetDynamicBucketMounted("")

	// Setting back the original mntDir after testing.
	setup.SetMntDir(rootMntDir)
	return
}

// Deprecated: Use RunTestsWithConfigFile instead.
// TODO(b/438068132): cleanup deprecated methods after migration is complete.
func RunTests(ctx context.Context, client *storage.Client, flags [][]string, m *testing.M) (successCode int) {
	config := &test_suite.TestConfig{
		TestBucket:       setup.TestBucket(),
		MountedDirectory: setup.MountedDirectory(),
		LogFile:          setup.LogFile(),
	}
	return RunTestsWithConfigFile(config, flags, m)
}

func RunTestsWithConfigFile(config *test_suite.TestConfig, flagsSet [][]string, m *testing.M) (successCode int) {
	log.Println("Running dynamic mounting tests...")
	successCode = executeTestsForDynamicMounting(config, flagsSet, m)
	log.Printf("Test log: %s\n", setup.LogFile())
	return successCode
}
