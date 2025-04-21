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

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"

	client_util "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const PrefixBucketForDynamicMountingTest = "gcsfuse-dynamic-mounting-test-"

func MountGcsfuseWithDynamicMounting(flags []string) (err error) {
	return MountGcsfuseWithDynamicMountingMntDirLogFile(flags, setup.MntDir(), setup.LogFile())
}

func MountGcsfuseWithDynamicMountingMntDirLogFile(flags []string, mntDir, logFile string) (err error) {
	defaultArg := []string{"--log-severity=trace",
		"--log-file=" + logFile,
		mntDir}

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

func executeTestsForDynamicMounting(flags [][]string, createdBucket string, m *testing.M) (successCode int) {
	rootMntDir := setup.MntDir()

	// In dynamic mounting all the buckets mounted in mntDir which user has permission.
	// mntDir - bucket1, bucket2, bucket3, ...
	// We will test on passed testBucket and one created bucket.

	// SetDynamicBucketMounted to the passed test bucket.
	setup.SetDynamicBucketMounted(setup.TestBucket())
	// Test on testBucket
	successCode = runTestsOnGivenMountedTestBucket(setup.TestBucket(), flags, rootMntDir, m)

	testBucketForDynamicMounting := CreateTestBucketForDynamicMounting(ctx, client)
	// Test on created bucket.
	// SetDynamicBucketMounted to the mounted bucket.
	setup.SetDynamicBucketMounted(createdBucket)
	if successCode == 0 {
		successCode = runTestsOnGivenMountedTestBucket(createdBucket, flags, rootMntDir, m)
	}
	// Reset SetDynamicBucketMounted to empty after tests are done.
	setup.SetDynamicBucketMounted("")

	// Setting back the original mntDir after testing.
	setup.SetMntDir(rootMntDir)
	if err := client_util.DeleteBucket(ctx, client, testBucketForDynamicMounting); err != nil {
		log.Fatalf("Failed to delete the bucket : %s. Error: %v", testBucketForDynamicMounting, err)
	}
	return
}

func CreateTestBucketForDynamicMounting(ctx context.Context, client *storage.Client) (bucketName string, err error) {
	projectID, err := metadata.ProjectIDWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get project ID of instance: %v", err)
	}

	// Create bucket handle and attributes
	storageClassAndLocation := &storage.BucketAttrs{
		Location: "us-west1",
	}

	bucketName = PrefixBucketForDynamicMountingTest + setup.GenerateRandomString(5)
	bucket := client.Bucket(bucketName)
	if err := bucket.Create(ctx, projectID, storageClassAndLocation); err != nil {
		return "", fmt.Errorf("failed to create bucket: %v", err)
	}
	return
}

func RunTests(ctx context.Context, client *storage.Client, flags [][]string, m *testing.M) (successCode int) {
	log.Println("Running dynamic mounting tests...")

	createdBucket, err := CreateTestBucketForDynamicMounting(ctx, client)
	if err != nil {
		log.Fatalf("Failed to create bucket for dynamic mounting test: %v", err)
	}

	successCode = executeTestsForDynamicMounting(flags, createdBucket, m)

	log.Printf("Test log: %s\n", setup.LogFile())

	if err := client_util.DeleteBucket(ctx, client, createdBucket); err != nil {
		log.Fatalf("Failed to delete the created bucket for dynamic mounting test: %s. Error: %v", createdBucket, err)
	}

	return successCode
}
