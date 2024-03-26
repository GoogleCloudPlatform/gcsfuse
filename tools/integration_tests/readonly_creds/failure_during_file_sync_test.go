// Copyright 2024 Google Inc. All Rights Reserved.
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

package readonly_creds

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type readOnlyCredsTest struct {
	testDirPath string
}

func (r *readOnlyCredsTest) Setup(t *testing.T) {
	r.testDirPath = path.Join(setup.MntDir(), testDirName)
}

func (r *readOnlyCredsTest) Teardown(t *testing.T) {
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (r *readOnlyCredsTest) assertFailedFileNotInListing(t *testing.T) {
	entries, err := os.ReadDir(r.testDirPath)
	if err != nil {
		t.Errorf("Failed to list directory %s: %v", r.testDirPath, err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected %s directory to be empty: %v", r.testDirPath, entries)
	}
}

func (r *readOnlyCredsTest) assertFileSyncFailsWithPermissionError(fh *os.File, t *testing.T) {
	err := fh.Close()
	if err == nil || !strings.Contains(err.Error(), permissionDeniedError) {
		t.Errorf("Expected error: %s, Got Error: %v", permissionDeniedError, err)
	}
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (r *readOnlyCredsTest) TestEmptyCreateFileFails_FailedFileNotInListing(t *testing.T) {
	filePath := path.Join(r.testDirPath, testFileName)

	fh := operations.CreateFile(filePath, operations.FilePermission_0777, t)
	r.assertFileSyncFailsWithPermissionError(fh, t)

	r.assertFailedFileNotInListing(t)
}

func (r *readOnlyCredsTest) TestNonEmptyCreateFileFails_FailedFileNotInListing(t *testing.T) {
	filePath := path.Join(r.testDirPath, testFileName)

	fh := operations.CreateFile(filePath, operations.FilePermission_0777, t)
	operations.WriteWithoutClose(fh, content, t)
	operations.WriteWithoutClose(fh, content, t)
	r.assertFileSyncFailsWithPermissionError(fh, t)

	r.assertFailedFileNotInListing(t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestReadOnlyTest(t *testing.T) {
	ts := &readOnlyCredsTest{}

	// Run tests.
	test_setup.RunTests(t, ts)
}
