// Copyright 2023 Google LLC
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

// Provides integration tests for copy file.
package operations_test

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

func TestCopyFile(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	fileName := path.Join(testDir, tempFileName+setup.GenerateRandomString(5))

	operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, t)
	defer os.Remove(fileName)

	content, err := operations.ReadFile(fileName)
	if err != nil {
		t.Errorf("Read: %v", err)
	}

	newFileName := fileName + "Copy"
	if _, err := os.Stat(newFileName); err == nil {
		t.Errorf("Copied file %s already present", newFileName)
	}

	err = operations.CopyFile(fileName, newFileName)
	if err != nil {
		t.Errorf("Error : %v", err)
	}
	defer os.Remove(newFileName)

	// Check if the data in the copied file matches the original file,
	// and the data in original file is unchanged.
	setup.CompareFileContents(t, newFileName, string(content))
	setup.CompareFileContents(t, fileName, string(content))
}
