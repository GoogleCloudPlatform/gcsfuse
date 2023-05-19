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

// Provides integration tests for copy directory.
package operations_test

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

// Create below directory structure.
// srcCopyDir               -- Directory
// srcCopyDir/copy.txt      -- File
// srcCopyDir/subSrcCopyDir -- Directory
func createSrcDirectoryWithObjects(dirPath string, t *testing.T) {
	// testBucket/srcCopyDir
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", dirPath, err)
		return
	}

	// testBucket/subSrcCopyDir
	subDirPath := path.Join(dirPath, SubSrcCopyDirectory)
	err = os.Mkdir(subDirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", subDirPath, err)
		return
	}

	// testBucket/srcCopyDir/copy.txt
	filePath := path.Join(dirPath, SrcCopyFile)

	file, err := os.Create(filePath)
	if err != nil {
		t.Errorf("Error in creating file %v:", err)
	}

	err = operations.WriteFile(file.Name(), SrcCopyFileContent)
	if err != nil {
		t.Errorf("File at %v", err)
	}
}

func checkIfCopiedDirectoryHasCorrectData(destDir string, t *testing.T) {
	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Comparing number of objects in the testBucket - 2
	if len(obj) != NumberOfObjectsInSrcCopyDirectory {
		t.Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// Comparing first object name and type
	// Name - testBucket/destCopyDir/copy.txt, Type - file
	if obj[0].Name() != SrcCopyFile || obj[0].IsDir() == true {
		t.Errorf("Object Listed for bucket directory is incorrect.")
	}

	// Comparing second object name and type
	// Name - testBucket/destCopyDir/srcCopyDir, Type - dir
	if obj[1].Name() != SubSrcCopyDirectory || obj[1].IsDir() != true {
		t.Errorf("Object Listed for bucket directory is incorrect.")
	}

	destFile := path.Join(destDir, SrcCopyFile)

	content, err := operations.ReadFile(destFile)
	if err != nil {
		t.Errorf("ReadAll: %v", err)
	}
	if got, want := string(content), SrcCopyFileContent; got != want {
		t.Errorf("File content %q not match %q", got, want)
	}
}

// Copy SrcDirectory objects in DestDirectory
// srcCopyDir               -- Directory
// srcCopyDir/copy.txt      -- File
// srcCopyDir/subSrcCopyDir -- Directory

// destCopyDir               -- Directory
// destCopyDir/copy.txt      -- File
// destCopyDir/subSrcCopyDir -- Directory
func TestCopyDirectoryInNonExistingDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), SrcCopyDirectory)

	createSrcDirectoryWithObjects(srcDir, t)

	destDir := path.Join(setup.MntDir(), DestCopyDirectoryNotExist)

	err := operations.CopyDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in copying directory: %v", err)
	}

	checkIfCopiedDirectoryHasCorrectData(destDir, t)

	os.RemoveAll(srcDir)
	os.RemoveAll(destDir)
}

// Copy SrcDirectory in DestDirectory
// srcCopyDir               -- Directory
// srcCopyDir/copy.txt      -- File
// srcCopyDir/subSrcCopyDir -- Directory

// destCopyDir                          -- Directory
// destCopyDir/srcCopyDir               -- Directory
// destCopyDir/srcCopyDir/copy.txt      -- File
// destCopyDir/srcCopyDir/subSrcCopyDir -- Directory
func TestCopyDirectoryInEmptyDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), SrcCopyDirectory)

	createSrcDirectoryWithObjects(srcDir, t)

	// Create below directory
	// destCopyDir               -- Directory
	destDir := path.Join(setup.MntDir(), DestCopyDirectory)
	err := os.Mkdir(destDir, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory: %v", err)
	}

	err = operations.CopyDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in copying directory: %v", err)
	}

	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destCopyDirectory has the correct directory copied.
	// destCopyDirectory
	// destCopyDirectory/srcCopyDirectory
	if len(obj) != 1 || obj[0].Name() != SrcCopyDirectory || obj[0].IsDir() != true {
		t.Errorf("Error in copying directory.")
		return
	}

	destSrc := path.Join(destDir, SrcCopyDirectory)
	checkIfCopiedDirectoryHasCorrectData(destSrc, t)

	os.RemoveAll(srcDir)
	os.RemoveAll(destDir)
}

func TestCopyDirectoryInNonEmptyDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), SrcCopyDirectory)

	createSrcDirectoryWithObjects(srcDir, t)

	// Create below directory
	// destCopyDir               -- Directory
	destDir := path.Join(setup.MntDir(), DestNonEmptyCopyDirectory)
	err := os.Mkdir(destDir, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory: %v", err)
	}

	destSubDir := path.Join(destDir, SubDirInNonEmptyDestCopyDirectory)
	err = os.Mkdir(destSubDir, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory: %v", err)
	}

	err = operations.CopyDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in copying directory: %v", err)
	}

	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destCopyDirectory has the correct directory copied.
	// destCopyDirectory
	// destCopyDirectory/srcCopyDirectory
	// destCopyDirectory/subDestCopyDirectory
	if len(obj) != NumberOfObjectsInDestCopyDirectory {
		t.Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// destCopyDirectory/srcCopyDirectory  - Directory
	if obj[0].Name() != SrcCopyDirectory || obj[0].IsDir() != true {
		t.Errorf("Error in copying directory.")
		return
	}

	// destCopyDirectory/subDirInNonEmptyDestCopyDirectory  - Directory
	if obj[1].Name() != SubDirInNonEmptyDestCopyDirectory || obj[1].IsDir() != true {
		t.Errorf("Existing object affected.")
		return
	}

	destSrc := path.Join(destDir, SrcCopyDirectory)
	checkIfCopiedDirectoryHasCorrectData(destSrc, t)

	os.RemoveAll(srcDir)
	os.RemoveAll(destDir)
}
