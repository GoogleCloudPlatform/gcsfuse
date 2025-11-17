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
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	client_util "github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
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
var singleRegions = []string{
	"us-central1",
	"europe-west4",
}

// gRPC is now supported for multi region buckets as well.
// Same logic, if the test VM is in us, then failure region = eu
// If the test VM is in non-US, then failure region = us
var multiRegions = []string{
	"us",
	"eu",
}
var gcpProject = "gcs-fuse-test"
var (
	ctx        context.Context
	client     *storage.Client
	testRegion string
)

// Since gRPC directpath does not work over cloudtop, so these validation tests will be skipped
// when run on cloudtop.
var cloudtopProd = "cloudtop-prod"

////////////////////////////////////////////////////////////////////////
// Helper Functions
////////////////////////////////////////////////////////////////////////

// For both multi region buckets and single region buckets test, we need to decide failure case
// region based on the region of the VM on which the test is running.
// Reference for the GCP resource attribute: https://opentelemetry.io/docs/specs/semconv/attributes-registry/cloud/#cloud-availability-zone
func findTestExecutionEnvironment(ctx context.Context) (string, error) {
	detectedAttrs, err := resource.New(ctx, resource.WithDetectors(gcp.NewDetector()))
	if err != nil {
		log.Printf("Error fetching the test environment.All tests will be skipped. Error : %v", err)
		return "", err
	}
	attrs := detectedAttrs.Set()
	if v, exists := attrs.Value("gcp.gce.instance.hostname"); exists && strings.Contains(strings.ToLower(v.AsString()), cloudtopProd) {
		return cloudtopProd, nil
	}
	if v, exists := attrs.Value("cloud.availability_zone"); exists {
		return v.AsString(), nil
	}
	return "", nil
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

// Creating test bucket name with unique suffix.
func createTestBucketName(region string) string {
	epoch := time.Now().UnixNano() // Get the current Unix epoch time
	return fmt.Sprintf("grpc_validation_%s_%d", region, epoch)
}

// Based on the test case, we need to create the bucket in/out of region with the test VM.
// This has to be done dynamically at the time of test setup.
// Based on what region we pass, the test bucket will be multi region or single region.
func createTestBucket(testBucketRegion, testBucketName string) (err error) {
	bucket := client.Bucket(testBucketName)
	if err = bucket.Create(ctx, gcpProject, &storage.BucketAttrs{Location: testBucketRegion}); err != nil {
		log.Printf("Error while creating bucket, error: %v", err)
		return err
	}
	return nil
}

func TestMain(m *testing.M) {
	// Parse flags from the setup.
	var err error
	setup.ParseSetUpFlags()
	if setup.IsPresubmitRun() {
		log.Println("Skipping test package : grpc_validation since this is a presubmit test run")
		os.Exit(0)
	}
	setup.SetUpTestDirForTestBucketFlag()

	// Creating a common storage client for the test
	ctx = context.Background()
	if client, err = client_util.CreateStorageClient(ctx); err != nil {
		log.Fatalf("Creation of storage client failed with error : %v", err)
	}
	defer client.Close()

	testRegion, err = findTestExecutionEnvironment(ctx)
	if err != nil {
		log.Fatalf("Failed to retrieve test VM region: %v", err)
	}

	if testRegion == cloudtopProd {
		log.Println("Skipping tests due to cloudtop environment.")
		os.Exit(0)
	}

	// Run tests.
	code := m.Run()
	// Exit.
	os.Exit(code)
}
