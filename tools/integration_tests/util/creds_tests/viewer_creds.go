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

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func RunTestsForKeyFileAndGoogleApplicationCredentials(testFlagSet [][]string, m *testing.M) (successCode int) {
	// Revoking gcloud credentials to test with service account credentials.
	//setup.RunScriptForTestData("../util/creds_tests/testdata/revoke_gcloud_creds.sh", "")

	//Testing without key-file and GOOGLE_APPLICATION_CREDENTIALS env variable
	//flagSet := []string{"--implicit-dirs"}
	//err := setup.MountGcsfuse(flagSet)
	//if err == nil {
	//	log.Print("Mounted successfully without credentials.")
	//	return 1
	//} else {
	//	successCode = 0
	//}

	setup.RunScriptForTestData("../util/creds_tests/testdata/get_creds.sh", "integration-test-tulsishah")

	creds_path := path.Join(os.Getenv("HOME"), "viewer_creds.json")

	// Testing with GOOGLE_APPLICATION_CREDENTIALS env variable
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", creds_path)

	log.Print(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	successCode = static_mounting.RunTests(testFlagSet, m)

	if successCode != 0 {
		return
	}

	keyFileFlag := "--key-file=" + creds_path

	for i := 0; i < len(testFlagSet); i++ {
		testFlagSet[i] = append(testFlagSet[i], keyFileFlag)
	}

	// Testing with --key-file and GOOGLE_APPLICATION_CREDENTIALS env variable set
	successCode = static_mounting.RunTests(testFlagSet, m)

	if successCode != 0 {
		return
	}

	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

	// Testing with --key-file flag only
	successCode = static_mounting.RunTests(testFlagSet, m)

	if successCode != 0 {
		return
	}

	//setup.RunScriptForTestData("../util/creds_tests/testdata/set_gcloud_creds.sh", "")

	return successCode
}
