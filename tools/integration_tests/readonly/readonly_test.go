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

// Run shell script
func runScriptForTestData(script string, testBucket string) {
	cmd := exec.Command("/bin/bash", script, testBucket)
	_, err := cmd.Output()
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	flags := [][]string{{"--o=ro", "--implicit-dirs=true"}, {"--file-mode=544", "--dir-mode=544", "--implicit-dirs=true"}}

	// Create objects in bucket for testing.
	runScriptForTestData("testdata/create_objects.sh", setup.TestBucket())

	successCode := setup.RunTests(flags, m)

	// Delete objects from bucket after testing.
	runScriptForTestData("testdata/delete_objects.sh", setup.TestBucket())

	os.Exit(successCode)
}
