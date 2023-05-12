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

// Provides integration tests when --key-file flag is set or GOOGLE_APPLICATION_CREDENTIALS environment variable is set.

package key_file_and_google_application_creds

import (
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.RunScriptForTestData("testdata/get_key_file.sh", "key-file-integration-test-gcs-fuse")
	flags := [][]string{{"--key-file=viewer_creds.json", "--implicit-dirs"}}

	if setup.TestBucket() != "" && setup.MountedDirectory() != "" {
		log.Printf("Both --testbucket and --mountedDirectory can't be specified at the same time.")
		os.Exit(1)
	}

	successCode := setup.RunTests(flags, m)

	os.Exit(successCode)
}
