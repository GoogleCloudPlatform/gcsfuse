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

package helpers

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	FileName1                     = "foo1"
	FileName2                     = "foo2"
	FileName3                     = "foo3"
	ImplicitDirName               = "implicit"
	ImplicitFileName1             = "implicitFile1"
	ExplicitDirName               = "explicit"
	ExplicitFileName1             = "explicitFile1"
	DirPerms          os.FileMode = 0755
	FilePerms         os.FileMode = 0644
	FileContents                  = "teststring"
	GCSFileContent                = "gcsContent"
)

func CreateLocalFile(fileName string, t *testing.T) (filePath string, f *os.File) {
	// Creating a file shouldn't create file on GCS.
	filePath = path.Join(setup.MntDir(), fileName)
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, FilePerms)
	if err != nil {
		t.Fatalf("CreateLocalFile(%s): %v", fileName, err)
	}

	ValidateObjectNotFoundErr(fileName, t)
	return
}

func ValidateObjectNotFoundErr(fileName string, t *testing.T) {
	_, err := ReadObjectFromGCS(fileName)
	if err == nil || !strings.Contains(err.Error(), "storage: object doesn't exist") {
		t.Fatalf("Incorrect error returned from GCS for file %s: %v", fileName, err)
	}
}

func CloseLocalFile(f *os.File, fileName string, t *testing.T) {
	err := f.Close()
	if err != nil {
		t.Fatalf("%s.Close(): %v", fileName, err)
	}
}

func ValidateObjectContents(fileName string, expectedContent string, t *testing.T) {
	gotContent, err := ReadObjectFromGCS(fileName)
	if err != nil {
		t.Fatalf("Error while reading synced local file from GCS, Err: %v", err)
	}

	if expectedContent != gotContent {
		t.Fatalf("GCS file %s content mismatch. Got: %s, Expected: %s ", fileName, gotContent, expectedContent)
	}
}

func CloseFileAndValidateObjectContents(f *os.File, fileName string, contents string, t *testing.T) {
	CloseLocalFile(f, fileName, t)
	ValidateObjectContents(fileName, contents, t)
}

func WritingToLocalFileSHouldNotThrowError(fh *os.File, content string, t *testing.T) {
	_, err := fh.Write([]byte(content))
	if err != nil {
		t.Fatalf("Error while writing to local file. err: %v", err)
	}
}

func WritingToLocalFileShouldNotWriteToGCS(fh *os.File, fileName string, t *testing.T) {
	WritingToLocalFileSHouldNotThrowError(fh, FileContents, t)
	ValidateObjectNotFoundErr(fileName, t)
}

func NewFileShouldGetSyncedToGCSAtClose(fileName string, t *testing.T) {
	// Create a local file.
	_, fh := CreateLocalFile(fileName, t)

	// Writing contents to local file shouldn't create file on GCS.
	WritingToLocalFileShouldNotWriteToGCS(fh, fileName, t)

	// Close the file and validate if the file is created on GCS.
	CloseFileAndValidateObjectContents(fh, fileName, FileContents, t)
}

func ValidateNoFileOrDirError(filename string, t *testing.T) {
	_, err := os.Stat(path.Join(setup.MntDir(), filename))
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("os.Stat on unlinked local file. Expected: %s, Got: %v",
			"no such file or directory", err)
	}
}

func ReadDirectory(dirPath string, t *testing.T) (entries []os.DirEntry) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		t.Fatalf("os.ReadDir(%s) err: %v", dirPath, err)
	}
	return
}

func VerifyLocalFileEntry(entry os.DirEntry, fileName string, size int64, t *testing.T) {
	if entry.IsDir() {
		t.Fatalf("Expected: file entry, Got: directory entry.")
	}
	if entry.Name() != fileName {
		t.Fatalf("File name, Expected: %s, Got: %s", fileName, entry.Name())
	}
	fileInfo, err := entry.Info()
	if err != nil {
		t.Fatalf("%s.Info() err: %v", fileName, err)
	}
	if fileInfo.Size() != size {
		t.Fatalf("Local file %s size, Expected: %d, Got: %d", fileName, size, fileInfo.Size())
	}
}

func VerifyDirectoryEntry(entry os.DirEntry, dirName string, t *testing.T) {
	if !entry.IsDir() {
		t.Fatalf("Expected: directory entry, Got: file entry.")
	}
	if entry.Name() != dirName {
		t.Fatalf("File name, Expected: %s, Got: %s", dirName, entry.Name())
	}
}

func UnlinkShouldNotThrowError(filePath string, t *testing.T) {
	err := os.Remove(filePath)

	// Verify os.Remove() operation succeeds.
	if err != nil {
		t.Fatalf("os.Remove(%s): %v", filePath, err)
	}
}

func SyncOnLocalFileShouldNotThrowError(fh *os.File, fileName string, t *testing.T) {
	err := fh.Sync()

	// Verify fh.Sync operation succeeds.
	if err != nil {
		t.Fatalf("%s.Sync(): %v", fileName, err)
	}
}

func CreateExplicitDirShouldNotThrowError(t *testing.T) {
	err := os.Mkdir(path.Join(setup.MntDir(), ExplicitDirName), DirPerms)

	// Verify MkDir operation succeeds.
	if err != nil {
		t.Fatalf("Error while creating directory, err: %v", err)
	}
}

func RemoveDirShouldNotThrowError(dirName string, t *testing.T) {
	dirPath := path.Join(setup.MntDir(), dirName)
	err := os.RemoveAll(dirPath)

	// Verify rmDir operation succeeds.
	if err != nil {
		t.Fatalf("os.RemoveAll(%s): %v", dirPath, err)
	}
}

func SymLinkShouldNotThrowError(filePath, symlinkName string, t *testing.T) {
	err := os.Symlink(filePath, symlinkName)

	// Verify os.SymLink operation succeeds.
	if err != nil {
		t.Fatalf("os.Symlink(%s, %s): %v", filePath, symlinkName, err)
	}
}

func VerifyReadLink(filePath, symlinkName string, t *testing.T) {
	target, err := os.Readlink(symlinkName)

	// Verify os.Readlink operation succeeds.
	if err != nil {
		t.Fatalf("os.Readlink(%s): %v", symlinkName, err)
	}
	if filePath != target {
		t.Fatalf("Symlink target mismatch. Expected: %s, Got: %s", filePath, target)
	}
}

func VerifyReadFile(symlinkName string, t *testing.T) {
	contents, err := os.ReadFile(symlinkName)

	// Verify os.ReadFile operation succeeds.
	if err != nil {
		t.Fatalf("os.ReadFile(%s): %v", symlinkName, err)
	}
	if FileContents != string(contents) {
		t.Fatalf("Symlink content mismatch. Expected: %s, Got: %s", FileContents, contents)
	}
}

func VerifyCountOfEntries(expected, got int, t *testing.T) {
	if expected != got {
		t.Fatalf("entry count mismatch, expected: %d, got: %d", expected, got)
	}
}

func VerifyRenameOperationNotSupported(err error, t *testing.T) {
	if err == nil || !strings.Contains(err.Error(), "operation not supported") {
		t.Fatalf("os.Rename(), expected err: %s, got err: %v",
			"operation not supported", err)
	}
}

func VerifyStatOnLocalFile(filePath string, fileSize int64, t *testing.T) {
	// Stat the file to validate if file is truncated correctly.
	fi, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("os.Stat err: %v", err)
	}
	if fi.Name() != path.Base(filePath) {
		t.Fatalf("File name mismatch in stat call. Expected: %s, Got: %s", path.Base(filePath), fi.Name())
	}
	if fi.Size() != fileSize {
		t.Fatalf("File size mismatch in stat call. Expected: %d, Got: %d", fileSize, fi.Size())
	}
	if fi.Mode() != FilePerms {
		t.Fatalf("File permissions mismatch in stat call. Expected: %v, Got: %v", FilePerms, fi.Mode())
	}
}
