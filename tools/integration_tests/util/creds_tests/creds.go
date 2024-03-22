// Copyright 2023 Google Inc. All Rights Reserved.
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
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const NameOfServiceAccount = "creds-integration-tests"
const CredentialsSecretName = "gcsfuse-integration-tests"

var WhitelistedGcpProjects = []string{"gcs-fuse-test", "gcs-fuse-test-ml"}

func CreateCredentials() (serviceAccount, localKeyFilePath string) {
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
	gcloudSecretAccessCmd := fmt.Sprintf("secrets versions access latest --secret %s", CredentialsSecretName)
	creds, err := operations.ExecuteGcloudCommandf(gcloudSecretAccessCmd)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error while fetching key file %v", err))
	}

	// Create and write creds to local file.
	file, err := os.Create(localKeyFilePath)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error while creating credentials file %v", err))
	}
	_, err = io.WriteString(file, string(creds))
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error while writing credentials to local file %v", err))
	}
	operations.CloseFile(file)

	return
}

func ApplyPermissionToServiceAccount(serviceAccount, permission string) {
	// Provide permission to service account for testing.
	_, err := operations.ExecuteGsutilCommandf(fmt.Sprintf("iam ch serviceAccount:%s:%s gs://%s", serviceAccount, permission, setup.TestBucket()))
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error while setting permissions to SA: %v", err))
	}
	// Waiting for 2 minutes as it usually takes within 2 minutes for policy
	// changes to propagate: https://cloud.google.com/iam/docs/access-change-propagation
	time.Sleep(120 * time.Second)
}

func RevokePermission(serviceAccount, permission, bucket string) {
	cmd := fmt.Sprintf("iam ch -d serviceAccount:%s:%s gs://%s", serviceAccount, permission, bucket)
	// Revoke the permission after testing.
	_, err := operations.ExecuteGsutilCommandf(cmd)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in unsetting permissions to SA: %v", err))
	}
}

func RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSet(testFlagSet [][]string, permission string, m *testing.M) (successCode int) {
	serviceAccount, localKeyFilePath := CreateCredentials()
	ApplyPermissionToServiceAccount(serviceAccount, permission)
	defer RevokePermission(serviceAccount, permission, setup.TestBucket())

	// Without –key-file flag and GOOGLE_APPLICATION_CREDENTIALS
	// This case will not get covered as gcsfuse internally authenticates from a metadata server on GCE VM.
	// https://github.com/golang/oauth2/blob/master/google/default.go#L160

	// Testing with GOOGLE_APPLICATION_CREDENTIALS env variable
	err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", localKeyFilePath)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in setting environment variable: %v", err))
	}

	successCode = static_mounting.RunTests(testFlagSet, m)

	if successCode != 0 {
		return
	}

	// Testing with --key-file and GOOGLE_APPLICATION_CREDENTIALS env variable set
	keyFileFlag := "--key-file=" + localKeyFilePath

	for i := 0; i < len(testFlagSet); i++ {
		testFlagSet[i] = append(testFlagSet[i], keyFileFlag)
	}

	successCode = static_mounting.RunTests(testFlagSet, m)

	if successCode != 0 {
		return
	}

	err = os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in unsetting environment variable: %v", err))
	}

	// Testing with --key-file flag only
	successCode = static_mounting.RunTests(testFlagSet, m)

	if successCode != 0 {
		return
	}

	return successCode
}
