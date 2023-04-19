// Copyright 2021 Google Inc. All Rights Reserved.
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

// Provides integration tests when --o=ro flag is set.
package readonly_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	flags := [][]string{{"--o=ro", "--implicit-dirs=true"}}

	// Set environment variable to use testBucket in creating objects.
	os.Setenv("TEST_BUCKET", setup.TestBucket())

	// Create objects in bucket for testing.
	cmd := exec.Command("/bin/bash", "create_objects.sh")
	_, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	successCode := setup.RunTests(flags, m)

	// Delete objects from bucket after testing.
	cmd = exec.Command("/bin/bash", "delete_objects.sh")
	_, err = cmd.Output()
	if err != nil {
		panic(err)
	}

	// Unset environment variable after testing
	os.Unsetenv("TEST_BUCKET")

	os.Exit(successCode)
}
