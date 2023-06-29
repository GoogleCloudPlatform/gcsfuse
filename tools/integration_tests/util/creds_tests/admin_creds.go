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
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSet(testFlagSet [][]string, m *testing.M) (successCode int) {
	// Saving testBucket value for setting back after testing.
	testBucket := setup.TestBucket()

	// Set the testBucket value to the bucket belonging to a different project for testing credentials.
	setup.SetTestBucket("integration-test-gcsfuse")

	// Set back the original testBucket, which we passed through --testBucket flag.
	defer setup.SetTestBucket(testBucket)

	// Testing without --key-file and GOOGLE_APPLICATION_CREDENTIALS env variable set
	for i := 0; i < len(testFlagSet); i++ {
		flags := testFlagSet[i]
		flags = append(flags, setup.TestBucket())
		flags = append(flags, setup.MntDir())
		err := mounting.MountGcsfuse(flags)
		if err == nil {
			log.Print("Error: Mounting successful without key file.")
		}
	}

	// Get credential from bucket.
	setup.RunScriptForTestData("../util/creds_tests/testdata/get_creds.sh", "integration-test-data-gcsfuse")
	buf := &bytes.Buffer{}
	key, err := createKey(buf, "admin-perm-integration-test@gcs-fuse-test.iam.gserviceaccount.com")
	if err != nil {
		_, _ = base64.StdEncoding.DecodeString(key.PrivateKeyData)
	}

	// Delete credentials after testing.
	defer setup.RunScriptForTestData("../util/creds_tests/testdata/delete_creds.sh", "")

	// Get the credential path to pass as a key file.
	creds_path := path.Join(os.Getenv("HOME"), "admin_creds.json")

	// Testing with GOOGLE_APPLICATION_CREDENTIALS env variable
	err = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", creds_path)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in setting environment variable: %v", err))
	}

	successCode = static_mounting.RunTests(testFlagSet, m)

	if successCode != 0 {
		return
	}

	// Testing with --key-file and GOOGLE_APPLICATION_CREDENTIALS env variable set
	keyFileFlag := "--key-file=" + creds_path

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
