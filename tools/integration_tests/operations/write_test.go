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
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestWriteAtEndOfFile(t *testing.T) {
	fileName := setup.CreateTempFile()

	err := operations.WriteFileInAppendMode(fileName, "line 3\n")
	if err != nil {
		t.Errorf("AppendString: %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 1\nline 2\nline 3\n")
}

func TestWriteAtStartOfFile(t *testing.T) {
	fileName := setup.CreateTempFile()

	err := operations.WriteFile(fileName, "line 4\n")
	if err != nil {
		t.Errorf("WriteString-Start: %v", err)
	}

	setup.CompareFileContents(t, fileName, "line 4\nline 2\n")
}

func TestWriteAtRandom(t *testing.T) {
	fileName := setup.CreateTempFile()

	f, err := os.OpenFile(fileName, os.O_WRONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Open file for write at random: %v", err)
	}

	// Write at 7th byte which corresponds to the start of 2nd line
	// thus changing line2\n with line5\n.
	if _, err = f.WriteAt([]byte("line 5\n"), 7); err != nil {
		t.Errorf("WriteString-Random: %v", err)
	}
	f.Close()

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
