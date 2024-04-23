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
//
// Provides integration tests for symlink operation on local files.

package interrupt

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
)

var (
	testDirPath string
)

type ignoreInterruptsTest struct{}

func (s *ignoreInterruptsTest) Teardown(t *testing.T) {}

func (s *ignoreInterruptsTest) Setup(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *ignoreInterruptsTest) TestGitClone(t *testing.T) {
	output, err := operations.ExecuteToolCommandf("git", "clone %s %s", "https://github.com/gcsfuse-github-machine-user-bot/test-repository.git", testDirPath)

	if err != nil {
		t.Errorf("Git clone failed: %s: %v", string(output), err)
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestIgnoreInterrupts(t *testing.T) {
	ts := &ignoreInterruptsTest{}
	test_setup.RunTests(t, ts)
}
