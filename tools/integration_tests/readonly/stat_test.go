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
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func statExistingObj(objPath string, t *testing.T) {
	_, err := os.Stat(objPath)
	if err != nil {
		t.Errorf("Fail to stat the object.")
	}
}

func TestStatFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "Test1.txt")
	statExistingObj(filePath, t)
}

func TestStatSubDirectoryFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "Test", "a.txt")
	statExistingObj(filePath, t)
}

func TestStatDirectory(t *testing.T) {
	DirPath := path.Join(setup.MntDir(), "Test")
	statExistingObj(DirPath, t)
}

func TestStatSubDirectory(t *testing.T) {
	DirPath := path.Join(setup.MntDir(), "Test", "b")
	statExistingObj(DirPath, t)
}

func statNotExistingObj(objPath string, t *testing.T) {
	_, err := os.Stat(objPath)
	if err == nil {
		t.Errorf("Object exist!!")
	}
}

func TestStatNotExistingFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "test.txt")
	statNotExistingObj(filePath, t)
}

func TestStatNotExistingSubDirectoryFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "Test", "test.txt")
	statNotExistingObj(filePath, t)
}

func TestStatNotExistingDirectory(t *testing.T) {
	DirPath := path.Join(setup.MntDir(), "test")
	statNotExistingObj(DirPath, t)
}

func TestStatNotExistingSubDirectory(t *testing.T) {
	DirPath := path.Join(setup.MntDir(), "Test", "test")
	statNotExistingObj(DirPath, t)
}
