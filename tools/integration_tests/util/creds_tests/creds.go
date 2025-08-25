// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Run tests for --key-file flag and GOOGLE_APPLICATION_CREDENTIALS env variable

package creds_tests

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/iam"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const NameOfServiceAccount = "creds-integration-tests"
const CredentialsSecretName = "gcsfuse-integration-tests"

var WhitelistedGcpProjects = []string{"gcs-fuse-test", "gcs-fuse-test-ml"}

func CreateCredentials(ctx context.Context) (serviceAccount, localKeyFilePath string) {
	log.Println("Running credentials tests...")

	// Fetching project-id to get service account id.
	id, err := metadata.ProjectID()
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in fetching project id: %v", err))
	}
	// return if active GCP project is not in whitelisted gcp projects
	if !slices.Contains(WhitelistedGcpProjects, id) {
		log.Printf("The active GCP project is not one of: %s. So the credentials test will not run.", strings.Join(WhitelistedGcpProjects, ", "))
	}

	// Service account id format is name@project-id.iam.gserviceaccount.com
	serviceAccount = NameOfServiceAccount + "@" + id + ".iam.gserviceaccount.com"

	localKeyFilePath = path.Join(os.Getenv("HOME"), "creds.json")

	// Download credentials
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Failed to create secret manager client: %v", err))
	}
	defer client.Close()
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", id, CredentialsSecretName),
	}
	creds, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error while fetching key file %v", err))
	}

	// Create and write creds to local file.
	file, err := os.Create(localKeyFilePath)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error while creating credentials file %v", err))
	}
	_, err = io.Writer.Write(file, creds.Payload.Data)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error while writing credentials to local file %v", err))
	}
	operations.CloseFile(file)

	return
}

func ApplyPermissionToServiceAccount(ctx context.Context, storageClient *storage.Client, serviceAccount, permission, bucket string) {
	// Provide permission to service account for testing.
	bucketHandle := storageClient.Bucket(bucket)
	policy, err := bucketHandle.IAM().Policy(ctx)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error fetching: Bucket(%q).IAM().Policy: %v", bucket, err))
	}
	identity := fmt.Sprintf("serviceAccount:%s", serviceAccount)
	role := iam.RoleName(fmt.Sprintf("roles/storage.%s", permission))

	policy.Add(identity, role)
	if err := bucketHandle.IAM().SetPolicy(ctx, policy); err != nil {
		setup.LogAndExit(fmt.Sprintf("Error applying permission to service account: Bucket(%q).IAM().SetPolicy: %v", bucket, err))
	}
	// Waiting for 2 minutes as it usually takes within 2 minutes for policy
	// changes to propagate: https://cloud.google.com/iam/docs/access-change-propagation
	time.Sleep(120 * time.Second)
}

func RevokePermission(ctx context.Context, storageClient *storage.Client, serviceAccount, permission, bucket string) {
	// Revoke the permission to service account after testing.
	bucketHandle := storageClient.Bucket(bucket)
	policy, err := bucketHandle.IAM().Policy(ctx)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error fetching: Bucket(%q).IAM().Policy: %v", bucket, err))
	}
	identity := fmt.Sprintf("serviceAccount:%s", serviceAccount)
	role := iam.RoleName(fmt.Sprintf("roles/storage.%s", permission))

	policy.Remove(identity, role)
	if err := bucketHandle.IAM().SetPolicy(ctx, policy); err != nil {
		setup.LogAndExit(fmt.Sprintf("Error applying permission to service account: Bucket(%q).IAM().SetPolicy: %v", bucket, err))
	}
}

// Deprecated: Use RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSetWithConfigFile instead.
// TODO(b/438068132): cleanup deprecated methods after migration is complete.
func RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSet(ctx context.Context, storageClient *storage.Client, testFlagSet [][]string, permission string, m *testing.M) (successCode int) {
	config := &test_suite.TestConfig{
		TestBucket:       setup.TestBucket(),
		MountedDirectory: setup.MountedDirectory(),
		LogFile:          setup.LogFile(),
	}
	return RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSetWithConfigFile(config, ctx, storageClient, testFlagSet, permission, m)
}

func RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSetWithConfigFile(config *test_suite.TestConfig, ctx context.Context, storageClient *storage.Client, testFlagSet [][]string, permission string, m *testing.M) (successCode int) {
	serviceAccount, localKeyFilePath := CreateCredentials(ctx)
	ApplyPermissionToServiceAccount(ctx, storageClient, serviceAccount, permission, setup.TestBucket())
	defer RevokePermission(ctx, storageClient, serviceAccount, permission, setup.TestBucket())

	// Without –key-file flag and GOOGLE_APPLICATION_CREDENTIALS
	// This case will not get covered as gcsfuse internally authenticates from a metadata server on GCE VM.
	// https://github.com/golang/oauth2/blob/master/google/default.go#L160

	// Testing with GOOGLE_APPLICATION_CREDENTIALS env variable
	err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", localKeyFilePath)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in setting environment variable: %v", err))
	}

	successCode = static_mounting.RunTestsWithConfigFile(config, testFlagSet, m)

	if successCode != 0 {
		return
	}

	// Testing with --key-file and GOOGLE_APPLICATION_CREDENTIALS env variable set
	keyFileFlag := "--key-file=" + localKeyFilePath

	for i := 0; i < len(testFlagSet); i++ {
		testFlagSet[i] = append(testFlagSet[i], keyFileFlag)
	}

	successCode = static_mounting.RunTestsWithConfigFile(config, testFlagSet, m)

	if successCode != 0 {
		return
	}

	err = os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in unsetting environment variable: %v", err))
	}

	// Testing with --key-file flag only
	successCode = static_mounting.RunTestsWithConfigFile(config, testFlagSet, m)

	if successCode != 0 {
		return
	}

	return successCode
}
