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

package grpc_validation

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/util"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel/sdk/resource"
)

// gRPC directPath can be established in a scenario like VM in us-central1-a and
// the bucket in us-central1. Since we are creating the buckets dynamically, we need
// to select regions and not zones.
// Note for single region bucket testing,
// We can work with 2 regions only since for testing success case, we are actually finding out
// the test region using go library. Just for failure case, we need to find out a region which
// is bound to be one of these two since continent differs.
// Example case: Test Region : us-central1 , Failure Test Region : europe-west4
//
//	Test Region : us-west1, Failure test Region : us-central1
var single_regions = []string{
	"us-central1",
	"europe-west4",
}

// gRPC is now supported for multi region buckets as well.
// Same logic, if the test VM is in us, then failure region = eu
// If the test VM is in non-US, then failure region = us
var multi_regions = []string{
	"us",
	"eu",
}
var gcp_project = "gcs-fuse-test"
var logFilePrefix = "grpc_validation_mnt_log"

// A directory containing outputs created by build_gcsfuse, set up and deleted
// in TestMain.
var gBuildDir string

// Since gRPC directpath does not work over cloudtop, so these validation tests will be skipped
// when run on cloudtop.
var cloudtopProd = "cloudtop-prod"

////////////////////////////////////////////////////////////////////////
// Helper Functions
////////////////////////////////////////////////////////////////////////

// For both multi region buckets and single region buckets test, we need to decide failure case
// region based on the region of the VM on which the test is running.
func findTestExecutionEnvironment(ctx context.Context) (string, error) {
	detectedAttrs, err := resource.New(ctx, resource.WithDetectors(gcp.NewDetector()))
	if err != nil {
		log.Printf("Error fetching the test environment.All tests will be skipped.")
		return "", err
	}
	attrs := detectedAttrs.Set()
	if v, exists := attrs.Value("gcp.gce.instance.hostname"); exists && strings.Contains(strings.ToLower(v.AsString()), cloudtopProd) {
		return cloudtopProd, nil
	}
	if v, exists := attrs.Value("cloud.region"); exists {
		return v.AsString(), nil
	}
	return cloudtopProd, nil
}

// For testing with single region buckets, we need to find the region for success case.
// If the input is 'us-west1-a' which is the test VM Zone, then this function returns 'us-west1' used
// creating the test buckets.
func findSingleRegionForGRPCDirectPathSuccessCase(testRegion string) string {
	parts := strings.Split(testRegion, "-")
	if len(parts) >= 2 {
		return strings.Join(parts[:len(parts)-1], "-") //rejoin the first parts of the zone, excluding the last part.
	}
	return ""
}

// For testing with multi region buckets, we need to find the region for success case.
// If the input is 'us-west1-a' which is the test VM Zone, then this function returns 'us' used
// creating the test buckets.
func findMultiRegionForGRPCDirectPathSuccessCase(testRegion string) string {
	parts := strings.Split(testRegion, "-")
	if len(parts) > 0 {
		return parts[0] // Return the first part of the string i.e. if us-central1 then us
	}
	return ""
}

// Generic function to pick a region other than the passed value to validate for failure scenario.
func pickFailureRegionFromListOfRegions(successRegion string, regions []string) string {
	for _, otherRegion := range regions {
		if otherRegion != successRegion {
			return otherRegion
		}
	}
	return ""
}

func createTestBucketName(region string) string {
	epoch := time.Now().UnixNano() // Get the current Unix epoch time
	return fmt.Sprintf("grpc_validation_%s_%d", region, epoch)
}

// Based on the test case, we need to create the bucket in/out of region with the test VM.
// This has to be done dynamically at the time of test setup.
// Based on what region we pass, the test bucket will be multi region or single region.
func createTestBucket(ctx context.Context, client *storage.Client, testBucketRegion, testBucketName string) (err error) {
	bucket := client.Bucket(testBucketName)
	if err = bucket.Create(ctx, gcp_project, &storage.BucketAttrs{Location: testBucketRegion}); err != nil {
		log.Printf("Error while creating bucket")
		return err
	}

	return nil
}

// Delete the test bucket after the test is complete.
func DeleteBucket(ctx context.Context, client *storage.Client, testBucketName string) (err error) {
	if testBucketName == "" {
		return errors.New("Bucket Name must not be empty!")
	}

	bucket := client.Bucket(testBucketName)
	if err = bucket.Delete(ctx); err != nil {
		log.Printf("Error while deleting bucket : %s", testBucketName)
		return err
	}

	return nil
}

func TestMain(m *testing.M) {
	// Parse flags from the setup.
	setup.ParseSetUpFlags()

	var err error

	if setup.TestInstalledPackage() {
		// when testInstalledPackage flag is set, gcsfuse is preinstalled on the
		// machine. Hence, here we are overwriting gBuildDir to /.
		gBuildDir = "/"
		code := m.Run()
		os.Exit(code)
	}

	// To test locally built package
	// Set up a directory into which we will build.
	gBuildDir, err = os.MkdirTemp("", "gcsfuse_integration_tests")
	if err != nil {
		log.Fatalf("TempDir: %p", err)
		return
	}

	// Build into that directory.
	err = util.BuildGcsfuse(gBuildDir)
	if err != nil {
		log.Fatalf("buildGcsfuse: %p", err)
		return
	}

	// Run tests.
	code := m.Run()

	// Clean up and exit.
	os.RemoveAll(gBuildDir)
	os.Exit(code)
}
