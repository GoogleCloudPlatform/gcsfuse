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
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func CreateFile(filePath string, fileName string, t *testing.T) {
	f := path.Join(filePath, fileName)

	file, err := os.OpenFile(f, os.O_CREATE, setup.FilePermission_0600)

	// It will throw an error read-only file system or permission denied.
	if err == nil {
		t.Errorf("File is created in read-only file system.")
	}

	defer file.Close()
}

func TestCreateFile(t *testing.T) {
	CreateFile(setup.MntDir(), "testFile.txt", t)
}

func TestCreateFileInSubDirectory(t *testing.T) {
	CreateFile(setup.MntDir()+"/Test", "testFile.txt", t)
}

func CreateDir(dirPath string, t *testing.T) {
	err := os.Mkdir(dirPath, fs.ModeDir)

	// It will throw an error read-only file system or permission denied.
	if err == nil {
		t.Errorf("Directory is created in read-only file system.")
	}
}

func TestCreateDir(t *testing.T) {
	CreateDir(setup.MntDir()+"/test", t)
}

func TestCreateDirInSubDirectory(t *testing.T) {
	CreateDir(setup.MntDir()+"/Test"+"/test", t)
}

func DeleteObjects(objPath string, t *testing.T) {
	err := os.RemoveAll(objPath)

	// It will throw an error read-only file system or permission denied.
	if err == nil {
		t.Errorf("Objects are deleted in read-only file system.")
	}
}

func TestDeleteDir(t *testing.T) {
	DeleteObjects(setup.MntDir()+"/Test", t)
}

func TestDeleteFile(t *testing.T) {
	DeleteObjects(setup.MntDir()+"/"+"Test1.txt", t)
}

func TestDeleteSubDirectory(t *testing.T) {
	DeleteObjects(setup.MntDir()+"/Test"+"/b", t)
}

func TestDeleteFileInSubDirectory(t *testing.T) {
	DeleteObjects(setup.MntDir()+"/Test"+"/"+"a.txt", t)
}

func TestDeleteAllObjectsInBucket(t *testing.T) {
	DeleteObjects(setup.MntDir(), t)
}
