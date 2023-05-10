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

// Provides integration tests for write flows.
package operations_test

import (
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/fileoperationhelper"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func TestWriteAtEndOfFile(t *testing.T) {
	fileName := setup.CreateTempFile()

	err := fileoperationhelper.WriteAtEndOfFile(fileName, "line 3\n")
	if err != nil {
		t.Errorf("AppendString: %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 1\nline 2\nline 3\n")
}

func TestWriteAtStartOfFile(t *testing.T) {
	fileName := setup.CreateTempFile()

	err := fileoperationhelper.WriteAtStartOfFile(fileName, "line 4\n")
	if err != nil {
		t.Errorf("WriteString-Start: %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 4\nline 2\n")
}

func TestWriteAtRandom(t *testing.T) {
	fileName := setup.CreateTempFile()

	// Write at 7th byte which corresponds to the start of 2nd line
	// thus changing line2\n with line5\n.
	err := fileoperationhelper.WriteAtRandom(fileName, "line 5\n")
	if err != nil {
		t.Errorf("WriteString-Random: %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 1\nline 5\n")
}

func TestCreateFile(t *testing.T) {
	fileName := setup.CreateTempFile()

	// Stat the file to check if it exists.
	if _, err := os.Stat(fileName); err != nil {
		t.Errorf("File not found, %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 1\nline 2\n")
}
