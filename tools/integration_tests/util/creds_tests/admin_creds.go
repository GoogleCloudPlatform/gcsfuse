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

package creds_tests

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func RunTestsForKeyFileAndGoogleApplicationCredentials(testFlagSet [][]string, m *testing.M) (successCode int) {
	// Saving testBucket value for setting back after testing.
	testBucket := setup.TestBucket()

	// Set the testBucket value to the bucket belonging to a different project for testing credentials.
	setup.SetTestBucket("integration-test-gcsfuse")

	// Testing without --key-file and GOOGLE_APPLICATION_CREDENTIALS env variable set
	for i := 0; i < len(testFlagSet); i++ {
		err := mounting.MountGcsfuse(testFlagSet[i])
		if err == nil {
			log.Print("Error: Mounting successful without key file.")
		}
	}

	setup.RunScriptForTestData("../util/creds_tests/testdata/get_creds.sh", "key-file-integration-tests")

	// Get the credential path to pass as a key file.
	creds_path := path.Join(os.Getenv("HOME"), "admin_creds.json")

	// Testing with GOOGLE_APPLICATION_CREDENTIALS env variable
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", creds_path)

	successCode = static_mounting.RunTests(testFlagSet, m)

	if successCode != 0 {
		setup.SetTestBucket(testBucket)
		return
	}

	// Testing with --key-file and GOOGLE_APPLICATION_CREDENTIALS env variable set
	keyFileFlag := "--key-file=" + creds_path

	for i := 0; i < len(testFlagSet); i++ {
		testFlagSet[i] = append(testFlagSet[i], keyFileFlag)
	}

	successCode = static_mounting.RunTests(testFlagSet, m)

	if successCode != 0 {
		setup.SetTestBucket(testBucket)
		return
	}

	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

	// Testing with --key-file flag only
	successCode = static_mounting.RunTests(testFlagSet, m)

	if successCode != 0 {
		setup.SetTestBucket(testBucket)
		return
	}

	setup.SetTestBucket(testBucket)

	return successCode
}
