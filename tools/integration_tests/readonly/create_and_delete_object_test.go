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

// Provides integration tests for file operations with --o=ro flag set.
package readonly_test

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func CreateFile(filePath string, fileName string, t *testing.T) {
	f := path.Join(filePath, fileName)

	file, err := os.OpenFile(f, os.O_CREATE, setup.FilePermission_0600)

	defer file.Close()

	// It will throw an error read-only file system or permission denied.
	if err == nil {
		t.Errorf("File is created in read-only file system.")
	}
}

func TestCreateFile(t *testing.T) {
	CreateFile(setup.MntDir(), "testFile.txt", t)
}

func TestCreateFileInSubDirectory(t *testing.T) {
	CreateFile(setup.MntDir()+"/Test", "testFile.txt", t)
}
