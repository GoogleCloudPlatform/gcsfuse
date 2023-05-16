// Copyright 2023 Google Inc. All Rights Reserved.
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

// Provide functions to test with key file and GOOGLE_APPLICATION_CREDENTIALS env variable is set.
package key_file

import (
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func RunTestsForKeyFileAndGoogleApplicationCredentials(credFilePath string, flags []string, m *testing.M) (successCode int) {
	// Revoking gcloud credentials to test with service account credentials.
	setup.RunScriptForTestData("../util/key_file/testdata/revoke_gcloud_creds.sh", "")

	setup.RunScriptForTestData("../util/key_file/testdata/get_key_files.sh", "key-file-integration-test-gcs-fuse")

	// Testing with --key-file
	defaultArg := []string{"--key-file=" + credFilePath}

	for i := 0; i < len(defaultArg); i++ {
		flags = append(flags, defaultArg[i])
	}

	f1 := [][]string{flags}
	successCode = setup.RunTests(f1, m)

	if successCode != 0 {
		return
	}

	// Testing with GOOGLE_APPLICATION_CREDENTIALS env variable
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credFilePath)

	f := [][]string{flags}
	successCode = setup.RunTests(f, m)

	if successCode != 0 {
		return
	}

	// Testing with key-file and GOOGLE_APPLICATION_CREDENTIALS env variable
	successCode = setup.RunTests(f1, m)
	if successCode != 0 {
		return
	}

	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

	// Testing without key-file and GOOGLE_APPLICATION_CREDENTIALS env variable
	flagSet := []string{"--implicit-dirs"}
	err := setup.MountGcsfuse(flagSet)
	if err == nil {
		log.Printf("Mounted successfully without credentials.")
		return 1
	} else {
		successCode = 0
	}

	// Delete key file after using it.
	setup.RunScriptForTestData("../util/key_file/testdata/delete_key_files.sh", "")

	return successCode
}
