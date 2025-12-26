// Copyright 2024 Google LLC
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

package mount_timeout

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/util"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel/sdk/resource"
)

var (
	// A directory containing outputs created by build_gcsfuse, set up and deleted
	// in TestMain.
	gBuildDir string

	// On Linux, the path to fusermount, whose directory must be in gcsfuse's PATH
	// variable in order to successfully mount. Set by TestMain.
	gFusermountPath string

	// storage client for permission setup on the bucket for mount access.
	gStorageClient *storage.Client

	gCtx context.Context
)

const (
	// Constants specific to non-ZB E2E runs.
	testEnvGCEUSCentral                    string        = "gce-us-central"
	testEnvGCENonUSCentral                 string        = "gce-non-us-central"
	testEnvNonGCE                          string        = "non-gce"
	multiRegionUSBucket                    string        = "mount_timeout_test_bucket_us"
	multiRegionAsiaBucket                  string        = "mount_timeout_test_bucket_asia"
	dualRegionUSBucket                     string        = "mount_timeout_test_bucket_nam4"
	dualRegionAsiaBucket                   string        = "mount_timeout_test_bucket_asia1"
	singleRegionUSCentralBucket            string        = "mount_timeout_test_bucket_us-central1"
	singleRegionAsiaEastBucket             string        = "mount_timeout_test_bucket_asia-east1"
	singleRegionAsiaEastExpectedMountTime  time.Duration = 5500 * time.Millisecond
	multiRegionUSExpectedMountTime         time.Duration = 4500 * time.Millisecond
	multiRegionAsiaExpectedMountTime       time.Duration = 7500 * time.Millisecond
	dualRegionUSExpectedMountTime          time.Duration = 4500 * time.Millisecond
	dualRegionAsiaExpectedMountTime        time.Duration = 6250 * time.Millisecond
	singleRegionUSCentralExpectedMountTime time.Duration = 2500 * time.Millisecond
	// Constants specific to ZB E2E runs.
	testEnvZoneGCEUSCentral1A       string        = "gce-zone-us-central1-a"
	testEnvZoneGCEUSWEST4A          string        = "gce-zone-us-west4a-a"
	testEnvZoneGCEOther             string        = "gce-zone-other"
	zonalUSCentral1ABucket          string        = "mount_timeout_test_bucket_zb_usc1a"
	zonalUSWest4ABucket             string        = "mount_timeout_test_bucket_zb_usw4a"
	zonalSameZoneExpectedMountTime  time.Duration = 2500 * time.Millisecond
	zonalCrossZoneExpectedMountTime time.Duration = 5000 * time.Millisecond
	// Commont constants.
	relaxedExpectedMountTime time.Duration = 8000 * time.Millisecond
	logfilePathPrefix        string        = "/tmp/gcsfuse_mount_timeout_"
)

// findTestExecutionEnvironment determines the environment in which the tests are running.
// It uses the GCP resource detector to identify the environment.
//
// If the tests are running on a GCE instance with a hostname containing non-gce.
// it returns testEnvNonGCE since it implies that the tests are being run on cloudtop.
//
// If the tests are running on a VM in the "us-central" region, it returns gce-us-central .
// Otherwise, if running in any other region, it returns gce-non-us-central.
//
// For all other cases, it returns non-gce.
func findTestExecutionEnvironment(ctx context.Context) string {
	detectedAttrs, err := resource.New(ctx, resource.WithDetectors(gcp.NewDetector()))
	if err != nil {
		log.Printf("Error fetching the test environment.All tests will be skipped.")
	}
	attrs := detectedAttrs.Set()
	if v, exists := attrs.Value("gcp.gce.instance.hostname"); exists && strings.Contains(strings.ToLower(v.AsString()), "cloudtop-prod") {
		if !setup.IsZonalBucketRun() {
			return testEnvNonGCE
		} else {
			return testEnvZoneGCEOther
		}
	}
	if !setup.IsZonalBucketRun() {
		if v, exists := attrs.Value("cloud.region"); exists {
			if strings.Contains(strings.ToLower(v.AsString()), "us-central") {
				return testEnvGCEUSCentral
			} else {
				return testEnvGCENonUSCentral
			}
		}
		return testEnvNonGCE
	} else {
		if v, exists := attrs.Value("cloud.availability_zone"); exists {
			switch strings.ToLower(v.AsString()) {
			case "us-central1-a":
				return testEnvZoneGCEUSCentral1A
			case "us-west4-a":
				return testEnvZoneGCEUSWEST4A
			default:
				return testEnvZoneGCEOther
			}
		}
		return testEnvZoneGCEOther
	}
}

func TestMain(m *testing.M) {
	// Parse flags from the setup.
	setup.ParseSetUpFlags()

	var err error

	// Find fusermount if we're running on Linux.
	if runtime.GOOS == "linux" {
		gFusermountPath, err = exec.LookPath("fusermount")
		if err != nil {
			log.Fatalf("LookPath(fusermount): %p", err)
		}
	}
	gCtx = context.Background()

	// Create storage client for direct bucket access.
	gStorageClient, err = client.CreateStorageClient(gCtx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer gStorageClient.Close()

	testEnv := findTestExecutionEnvironment(gCtx)
	err = os.Setenv("TEST_ENV", testEnv)
	if err != nil {
		fmt.Println("Error setting environment variable:", err)
		return
	}

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
