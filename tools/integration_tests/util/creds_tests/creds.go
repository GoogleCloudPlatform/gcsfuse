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
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/compute/metadata"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const NameOfServiceAccount = "creds-test-gcsfuse"

func setPermission(permission string) {
	// Provide admin permission to the bucket.
	setup.RunScriptForTestData("../util/creds_tests/testdata/provide_permission.sh", setup.TestBucket(), ServiceAccount, permission)
}

func RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSet(testFlagSet [][]string, permission string, m *testing.M) (successCode int) {
	id, err := metadata.ProjectID()
	if err != nil {
		log.Printf("Error in fetching project id: %v", err)
	}

	// Finding service account id from project-id and service account name.
	serviceAccount := NameOfServiceAccount + "@" + id + ".iam.gserviceaccount.com"

	// Create service account
	setup.RunScriptForTestData("../util/creds_tests/testdata/create_service_account.sh", NameOfServiceAccount)

	cred_file_path := path.Join(os.Getenv("HOME"), "creds.json")

	// Create credentials
	setup.RunScriptForTestData("../util/creds_tests/testdata/create_key_file.sh", cred_file_path, serviceAccount)

	// Provide permission to service account for testing.
	setPermission(permission)

	// Revoke the permission and delete creds and service account after testing.
	defer setup.RunScriptForTestData("../util/creds_tests/testdata/revoke_permission_and_delete_service_account_and_creds.sh", serviceAccount, cred_file_path)

	// Testing with GOOGLE_APPLICATION_CREDENTIALS env variable
	err = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", cred_file_path)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in setting environment variable: %v", err))
	}

	successCode = static_mounting.RunTests(testFlagSet, m)

	if successCode != 0 {
		return
	}

	// Testing with --key-file and GOOGLE_APPLICATION_CREDENTIALS env variable set
	keyFileFlag := "--key-file=" + cred_file_path

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
