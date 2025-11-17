// Copyright 2024 Google LLC
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type readOnlyCredsTest struct {
	testDirPath string
	suite.Suite
}

func (r *readOnlyCredsTest) SetupTest() {
	r.testDirPath = path.Join(setup.MntDir(), testDirName)
}

func (r *readOnlyCredsTest) TearDownTest() {
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

func (r *readOnlyCredsTest) TestEmptyCreateFileFails_FailedFileNotInListing() {
	filePath := path.Join(r.testDirPath, testFileName)

	fh, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, operations.FilePermission_0777)
	if setup.IsZonalBucketRun() {
		require.Error(r.T(), err)
		assert.True(r.T(), strings.Contains(err.Error(), permissionDeniedError))
	} else {
		r.assertFileSyncFailsWithPermissionError(fh, r.T())
	}

	r.assertFailedFileNotInListing(r.T())
}

func (r *readOnlyCredsTest) TestNonEmptyCreateFileFails_FailedFileNotInListing() {
	filePath := path.Join(r.testDirPath, testFileName)

	fh, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, operations.FilePermission_0777)
	if setup.IsZonalBucketRun() {
		require.Error(r.T(), err)
		assert.True(r.T(), strings.Contains(err.Error(), permissionDeniedError))
	} else {
		operations.WriteWithoutClose(fh, content, r.T())
		operations.WriteWithoutClose(fh, content, r.T())
		r.assertFileSyncFailsWithPermissionError(fh, r.T())
	}

	r.assertFailedFileNotInListing(r.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestReadOnlyTest(t *testing.T) {
	ts := &readOnlyCredsTest{}

	// Run tests.
	suite.Run(t, ts)
}
