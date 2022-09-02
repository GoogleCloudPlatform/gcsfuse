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

// Provides integration tests for write flows with implicit_dir flag set.
package implicitdir_test

import (
	"os"
	"testing"
)

func TestWriteAtEndOfFile(t *testing.T) {
	fileName := createTempFile()
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Errorf("Open file for append: %v", err)
	}

	if _, err = f.WriteString("line 3\n"); err != nil {
		t.Errorf("AppendString: %v", err)
	}

	f.Close()

	compareFileContents(t, fileName, "line 1\nline 2\nline 3\n")
}

func TestWriteAtStartOfFile(t *testing.T) {
	fileName := createTempFile()
	f, err := os.OpenFile(fileName, os.O_WRONLY, 0600)
	if err != nil {
		t.Errorf("Open file for write at start: %v", err)
	}

	if _, err = f.WriteAt([]byte("line 4\n"), 0); err != nil {
		t.Errorf("WriteString-Start: %v", err)
	}

	f.Close()

	compareFileContents(t, fileName, "line 4\nline 2\n")
}

func TestWriteAtRandom(t *testing.T) {
	fileName := createTempFile()

	f, err := os.OpenFile(fileName, os.O_WRONLY, 0600)
	if err != nil {
		t.Errorf("Open file for write at random: %v", err)
	}

	if _, err = f.WriteAt([]byte("line 5\n"), 7); err != nil {
		t.Errorf("WriteString-Random: %v", err)
	}

	f.Close()

	compareFileContents(t, fileName, "line 1\nline 5\n")
}
