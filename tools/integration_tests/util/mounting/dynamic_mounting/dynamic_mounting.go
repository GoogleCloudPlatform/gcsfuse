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
	"context"
	"fmt"
	"log"
	"path"
	"testing"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

// Adding prefix `golang-grpc-test` to white list the bucket for grpc so that
// we can run the grpc related e2e test.
const PrefixBucketForDynamicMountingTest = "golang-grpc-test-gcsfuse-dynamic-mounting-test-"

var testBucketForDynamicMounting = PrefixBucketForDynamicMountingTest + setup.GenerateRandomString(5)

func MountGcsfuseWithDynamicMounting(flags []string) (err error) {
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
	}
	return
}

func executeTestsForDynamicMounting(flags [][]string, m *testing.M) (successCode int) {
	rootMntDir := setup.MntDir()

	// In dynamic mounting all the buckets mounted in mntDir which user has permission.
	// mntDir - bucket1, bucket2, bucket3, ...
	// We will test on passed testBucket and one created bucket.

	// SetDynamicBucketMounted to the passed test bucket.
	setup.SetDynamicBucketMounted(setup.TestBucket())
	// Test on testBucket
	successCode = runTestsOnGivenMountedTestBucket(setup.TestBucket(), flags, rootMntDir, m)

	// Test on created bucket.
	// SetDynamicBucketMounted to the mounted bucket.
	setup.SetDynamicBucketMounted(testBucketForDynamicMounting)
	if successCode == 0 {
		successCode = runTestsOnGivenMountedTestBucket(testBucketForDynamicMounting, flags, rootMntDir, m)
	}
	// Reset SetDynamicBucketMounted to empty after tests are done.
	setup.SetDynamicBucketMounted("")

	// Setting back the original mntDir after testing.
	setup.SetMntDir(rootMntDir)
	return
}

func CreateTestBucketForDynamicMounting(ctx context.Context, client *storage.Client) (bucketName string) {
	projectID, err := metadata.ProjectID()
	if err != nil {
		log.Printf("Error in fetching project id: %v", err)
	}

	// Create bucket handle and attributes
	storageClassAndLocation := &storage.BucketAttrs{
		Location: "us-west1",
	}

	bucket := client.Bucket(testBucketForDynamicMounting)
	if err := bucket.Create(ctx, projectID, storageClassAndLocation); err != nil {
		log.Fatalf("DynamicBucket(%q).Create: %v", testBucketForDynamicMounting, err)
	}
	return testBucketForDynamicMounting
}

func DeleteTestBucketForDynamicMounting(ctx context.Context, client *storage.Client, bucketName string) {
	bucket := client.Bucket(bucketName)

	// Iterate through objects and delete them
	query := &storage.Query{}
	it := bucket.Objects(ctx, query)
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break // No more objects
		}
		if err != nil {
			log.Fatalf("Error iterating through objects: %v", err)
		}

		obj := bucket.Object(objAttrs.Name)
		err = obj.Delete(ctx)
		if err != nil {
			log.Fatalf("Failed to delete object %s: %v", objAttrs.Name, err)
		}
	}

	if err := bucket.Delete(ctx); err != nil {
		log.Printf("Bucket(%q).Delete: %v", bucketName, err)
	}
}

func RunTests(ctx context.Context, client *storage.Client, flags [][]string, m *testing.M) (successCode int) {
	log.Println("Running dynamic mounting tests...")

	CreateTestBucketForDynamicMounting(ctx, client)

	successCode = executeTestsForDynamicMounting(flags, m)

	log.Printf("Test log: %s\n", setup.LogFile())

	DeleteTestBucketForDynamicMounting(ctx, client, testBucketForDynamicMounting)

	return successCode
}
