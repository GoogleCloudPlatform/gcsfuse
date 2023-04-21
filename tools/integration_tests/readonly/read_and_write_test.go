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

// Provides integration tests for file operations with --o=ro flag set.
package readonly_test

import (
	"fmt"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func TestReadFile(t *testing.T) {
	f := path.Join(setup.MntDir(), "Test1.txt")

	file, err := os.OpenFile(f, os.O_RDONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in the opening the file %v", err))
	}
	defer file.Close()

	content, err := os.ReadFile(file.Name())
	if err != nil {
		t.Errorf("ReadAll: %v", err)
	}
	if got, want := string(content), "This is from file Test1\n"; got != want {
		t.Errorf("File content %q not match %q", got, want)
	}
}

func TestReadFileFromSubDirectory(t *testing.T) {
	f := path.Join(setup.MntDir(), "Test", "a.txt")

	file, err := os.OpenFile(f, os.O_RDONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in the opening the file %v", err))
	}
	defer file.Close()

	content, err := os.ReadFile(file.Name())
	if err != nil {
		t.Errorf("ReadAll: %v", err)
	}
	if got, want := string(content), "This is from directory Test file a\n"; got != want {
		t.Errorf("File content %q not match %q", got, want)
	}
}
