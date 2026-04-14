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

// Provide tests for cases where bucket with requester-pays feature is
// mounted and used through gcsfuse.
package requester_pays_bucket

import (
	"context"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName        = "RequesterPaysBucketTests"
	onlyDirTestDirName = "OnlyDirRequesterPaysBucketTests"

	// requesterPaysServiceAccountName is the name of the service account used for requester-pays testing.
	// This service account must exist in the active GCP project where tests are run
	// (e.g., "gcs-fuse-test" or "gcs-fuse-test-ml").
	// The test expects a JSON key for this SA to be stored in Secret Manager
	// within the same project, under the secret name specified by requesterPaysCredsSecretName.
	// Note: The user or service account running the test package, as well as the
	// requester-pays service account (requester-pays-tester), must be granted the
	// Storage Admin and Service Usage Consumer roles on the project.
	// For example, adding the Service Usage Consumer role to
	// requester-pays-tester@gcs-fuse-test.iam.gserviceaccount.com in the gcs-fuse-test project.
	requesterPaysServiceAccountName = "requester-pays-tester"
	requesterPaysCredsSecretName    = "requester-pays-tester"
	targetBillingProject            = "gcs-fuse-test"
)

// To prevent global variable pollution, enhance code clarity,
// and avoid inadvertent errors. We strongly suggest that, all new package-level
// variables (which would otherwise be declared with `var` at the package root) should
// be added as fields to this 'env' struct instead.
type env struct {
	testDirPath   string
	storageClient *storage.Client
	ctx           context.Context
	bucketName    string
}

var (
	testEnv env
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	if setup.IsZonalBucketRun() {
		log.Fatal("Test not supported for zonal bucket as they don't support requester-pays feature")
	}

	// Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.RequesterPaysBucket) == 0 {
		log.Println("No configuration found for requester pays bucket tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.RequesterPaysBucket = make([]test_suite.TestConfig, 1)
		cfg.RequesterPaysBucket[0].TestBucket = setup.TestBucket()
		cfg.RequesterPaysBucket[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.RequesterPaysBucket[0].Configs = make([]test_suite.ConfigItem, 1)
		cfg.RequesterPaysBucket[0].Configs[0].Flags = []string{
			"--billing-project=${BILLING_PROJECT} --key-file=${KEY_FILE}",
			"--billing-project=${BILLING_PROJECT} --client-protocol=grpc --key-file=${KEY_FILE}",
			"--billing-project=${BILLING_PROJECT} --client-protocol=grpc --grpc-path-strategy=direct-path-only --key-file=${KEY_FILE}",
		}
		cfg.RequesterPaysBucket[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": false}
	}

	testEnv.ctx = context.Background()

	// When not running in GKE environment.
	if cfg.RequesterPaysBucket[0].GKEMountedDirectory == "" {
		// Replace ${BILLING_PROJECT} placeholder in flags with the default billing project.
		for i := range cfg.RequesterPaysBucket[0].Configs {
			for j := range cfg.RequesterPaysBucket[0].Configs[i].Flags {
				cfg.RequesterPaysBucket[0].Configs[i].Flags[j] = strings.ReplaceAll(cfg.RequesterPaysBucket[0].Configs[i].Flags[j], "${BILLING_PROJECT}", targetBillingProject)
			}
		}
		// Setup service account credentials for requester-pays testing.
		_, localKeyFilePath := creds_tests.CreateCredentialsForSA(testEnv.ctx, requesterPaysServiceAccountName, requesterPaysCredsSecretName)
		defer func() {
			if err := os.Remove(localKeyFilePath); err != nil {
				log.Printf("Failed to delete temp credentials file %s: %v", localKeyFilePath, err)
			}
		}()
		setup.SetKeyFile(localKeyFilePath)
		for i := range cfg.RequesterPaysBucket[0].Configs {
			for j := range cfg.RequesterPaysBucket[0].Configs[i].Flags {
				cfg.RequesterPaysBucket[0].Configs[i].Flags[j] = strings.ReplaceAll(cfg.RequesterPaysBucket[0].Configs[i].Flags[j], "${KEY_FILE}", localKeyFilePath)
			}
		}
	}

	// Extract billing project from flags.
	var billingProject string
	for _, flag := range cfg.RequesterPaysBucket[0].Configs[0].Flags {
		if strings.Contains(flag, "--billing-project=") {
			parts := strings.Split(flag, "--billing-project=")
			if len(parts) > 1 {
				billingProject = strings.Fields(parts[1])[0]
				break
			}
		}
	}
	if billingProject == "" {
		log.Fatal("Billing project is not set. It must be set using environment variables in GKE envrionment and by replacing the '${BILLING_PROJECT}' string in non-GKE environements.")
	}
	setup.SetBillingProject(billingProject)

	// Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	testEnv.bucketName = strings.Split(cfg.RequesterPaysBucket[0].TestBucket, "/")[0]
	wasEnabled := client.MustEnableRequesterPays(testEnv.storageClient, testEnv.ctx, testEnv.bucketName)
	if wasEnabled {
		defer client.MustDisableRequesterPays(testEnv.storageClient, testEnv.ctx, testEnv.bucketName)
	}

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as RequesterPaysBucket tests validates content from the bucket.
	if cfg.RequesterPaysBucket[0].GKEMountedDirectory != "" && cfg.RequesterPaysBucket[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.RequesterPaysBucket[0].GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// Build the flag sets dynamically from the config.
	bucketType := setup.TestEnvironment(testEnv.ctx, &cfg.RequesterPaysBucket[0])
	flags := setup.BuildFlagSets(cfg.RequesterPaysBucket[0], bucketType, "")
	setup.SetUpTestDirForTestBucket(&cfg.RequesterPaysBucket[0])

	log.Println("Running static mounting tests...")
	successCode := static_mounting.RunTestsWithConfigFile(&cfg.RequesterPaysBucket[0], flags, m)

	if successCode == 0 {
		log.Printf("Running only-dir mounting tests ...")
		successCode = only_dir_mounting.RunTestsWithConfigFile(&cfg.RequesterPaysBucket[0], flags, onlyDirTestDirName, m)
	}

	// If failed, then save the gcsfuse log file(s).
	setup.SaveLogFileInCaseOfFailure(successCode)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
