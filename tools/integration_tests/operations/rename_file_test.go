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

// Provides integration tests for rename file.
package operations

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

func TestRenameFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	fileName := setup.CreateTempFile()

	content, err := operations.ReadFile(fileName)
	if err != nil {
		t.Errorf("Read: %v", err)
	}

	newFileName := fileName + "Rename"

	err = operations.RenameFile(fileName, newFileName)
	if err != nil {
		t.Errorf("Error in file copying: %v", err)
	}
	// Check if the data in the file is the same after renaming.
	setup.CompareFileContents(t, newFileName, string(content))
}
