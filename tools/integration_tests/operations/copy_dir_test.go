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

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

// Create below directory structure.
// srcCopyDir               -- Dir
// srcCopyDir/copy.txt      -- File
// srcCopyDir/subSrcCopyDir -- Dir
func createSrcDirectoryWithObjects(dirPath string, t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

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

	// Closing file at the end
	defer operations.CloseFile(file)
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
// srcCopyDir               -- Dir
// srcCopyDir/copy.txt      -- File
// srcCopyDir/subSrcCopyDir -- Dir

// destCopyDir               -- Dir
// destCopyDir/copy.txt      -- File
// destCopyDir/subSrcCopyDir -- Dir
func TestCopyDirectoryInNonExistingDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), SrcCopyDirectory)

	createSrcDirectoryWithObjects(srcDir, t)

	destDir := path.Join(setup.MntDir(), DestCopyDirectoryNotExist)

	err := operations.CopyDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in copying directory: %v", err)
	}

	checkIfCopiedDirectoryHasCorrectData(destDir, t)
}

// Copy SrcDirectory in DestDirectory
// srcCopyDir               -- Dir
// srcCopyDir/copy.txt      -- File
// srcCopyDir/subSrcCopyDir -- Dir

// destCopyDir                          -- Dir
// destCopyDir/srcCopyDir               -- Dir
// destCopyDir/srcCopyDir/copy.txt      -- File
// destCopyDir/srcCopyDir/subSrcCopyDir -- Dir
func TestCopyDirectoryInEmptyDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), SrcCopyDirectory)

	createSrcDirectoryWithObjects(srcDir, t)

	// Create below directory
	// destCopyDir               -- Dir
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
}

func createDestNonEmptyDirectory(t *testing.T) {
	destDir := path.Join(setup.MntDir(), DestNonEmptyCopyDirectory)
	operations.CreateDirectoryWithNFiles(0, destDir, "", t)

	destSubDir := path.Join(destDir, SubDirInNonEmptyDestCopyDirectory)
	operations.CreateDirectoryWithNFiles(0, destSubDir, "", t)
}

func TestCopyDirectoryInNonEmptyDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), SrcCopyDirectory)

	createSrcDirectoryWithObjects(srcDir, t)

	// Create below directory
	// destCopyDir               -- Dir
	destDir := path.Join(setup.MntDir(), DestNonEmptyCopyDirectory)
	createDestNonEmptyDirectory(t)

	err := operations.CopyDir(srcDir, destDir)
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
	if len(obj) != NumberOfObjectsInNonEmptyDestCopyDirectory {
		t.Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// destCopyDirectory/srcCopyDirectory  - Dir
	if obj[0].Name() != SrcCopyDirectory || obj[0].IsDir() != true {
		t.Errorf("Error in copying directory.")
		return
	}

	// destCopyDirectory/subDirInNonEmptyDestCopyDirectory  - Dir
	if obj[1].Name() != SubDirInNonEmptyDestCopyDirectory || obj[1].IsDir() != true {
		t.Errorf("Existing object affected.")
		return
	}

	destSrc := path.Join(destDir, SrcCopyDirectory)
	checkIfCopiedDirectoryHasCorrectData(destSrc, t)
}

func checkIfCopiedEmptyDirectoryHasNoData(destSrc string, t *testing.T) {
	objs, err := os.ReadDir(destSrc)
	if err != nil {
		log.Fatal(err)
	}

	if len(objs) != 0 {
		t.Errorf("Directory has incorrect data.")
	}
}

// Copy SrcDirectory in DestDirectory
// emptySrcDirectoryCopyTest

// destNonEmptyCopyDirectory
// destNonEmptyCopyDirectory/subDirInNonEmptyDestCopyDirectory

// Output
// destNonEmptyCopyDirectory
// destNonEmptyCopyDirectory/subDirInNonEmptyDestCopyDirectory
// destNonEmptyCopyDirectory/emptySrcDirectoryCopyTest
func TestCopyEmptyDirectoryInNonEmptyDirectory(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	srcDir := path.Join(setup.MntDir(), EmptySrcDirectoryCopyTest)
	operations.CreateDirectoryWithNFiles(0, srcDir, "", t)

	// Create below directory
	// destNonEmptyCopyDirectory                                                -- Dir
	// destNonEmptyCopyDirectory/subDirInNonEmptyDestCopyDirectory              -- Dir
	destDir := path.Join(setup.MntDir(), DestNonEmptyCopyDirectory)
	createDestNonEmptyDirectory(t)

	err := operations.CopyDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in copying directory: %v", err)
	}

	objs, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destCopyDirectory has the correct directory copied.
	// destNonEmptyCopyDirectory
	// destNonEmptyCopyDirectory/emptyDirectoryCopyTest           - Dir
	// destNonEmptyCopyDirectory/subDestCopyDirectory             - Dir
	if len(objs) != NumberOfObjectsInNonEmptyDestCopyDirectory {
		t.Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// destNonEmptyCopyDirectory/srcCopyDirectory  - Dir
	if objs[0].Name() != EmptySrcDirectoryCopyTest || objs[0].IsDir() != true {
		t.Errorf("Error in copying directory.")
		return
	}

	// destNonEmptyCopyDirectory/subDirInNonEmptyDestCopyDirectory  - Dir
	if objs[1].Name() != SubDirInNonEmptyDestCopyDirectory || objs[1].IsDir() != true {
		t.Errorf("Existing object affected.")
		return
	}

	copyDirPath := path.Join(destDir, EmptySrcDirectoryCopyTest)
	checkIfCopiedEmptyDirectoryHasNoData(copyDirPath, t)
}

// Copy SrcDirectory in DestDirectory
// emptySrcDirectoryCopyTest

// destEmptyCopyDirectory

// Output
// destEmptyCopyDirectory/emptySrcDirectoryCopyTest
func TestCopyEmptyDirectoryInEmptyDirectory(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	srcDir := path.Join(setup.MntDir(), EmptySrcDirectoryCopyTest)
	operations.CreateDirectoryWithNFiles(0, srcDir, "", t)

	// Create below directory
	// destCopyDir               -- Dir
	destDir := path.Join(setup.MntDir(), DestEmptyCopyDirectory)
	operations.CreateDirectoryWithNFiles(0, destDir, "", t)

	err := operations.CopyDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in copying directory: %v", err)
	}

	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destCopyDirectory has the correct directory copied.
	// destEmptyCopyDirectory
	// destEmptyCopyDirectory/emptyDirectoryCopyTest
	if len(obj) != NumberOfObjectsInEmptyDestCopyDirectory {
		t.Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// destEmptyCopyDirectory/srcCopyDirectory  - Dir
	if obj[0].Name() != EmptySrcDirectoryCopyTest || obj[0].IsDir() != true {
		t.Errorf("Error in copying directory.")
		return
	}

	copyDirPath := path.Join(destDir, EmptySrcDirectoryCopyTest)
	checkIfCopiedEmptyDirectoryHasNoData(copyDirPath, t)
}

// Copy SrcDirectory in DestDirectory
// emptySrcDirectoryCopyTest

// Output
// destCopyDirectoryNotExist
func TestCopyEmptyDirectoryInNonExistingDirectory(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	srcDir := path.Join(setup.MntDir(), EmptySrcDirectoryCopyTest)
	operations.CreateDirectoryWithNFiles(0, srcDir, "", t)

	// destCopyDirectoryNotExist             -- Dir
	destDir := path.Join(setup.MntDir(), DestCopyDirectoryNotExist)

	_, err := os.Stat(destDir)
	if err == nil {
		t.Errorf("destCopyDirectoryNotExist directory exist.")
	}

	err = operations.CopyDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in copying directory: %v", err)
	}

	checkIfCopiedEmptyDirectoryHasNoData(destDir, t)
}
