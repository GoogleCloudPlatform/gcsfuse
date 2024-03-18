// Copyright 2023 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy
//
//of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Provides integration tests for move directory.
package rename_dir_limit_test

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const SrcMoveDirectory = "srcMoveDir"
const SubSrcMoveDirectory = "subSrcMoveDir"
const SrcMoveFile = "srcMoveFile"
const SrcMoveFileContent = "This is from move file in srcMove directory.\n"
const DestMoveDirectory = "destMoveDir"
const DestNonEmptyMoveDirectory = "destNonEmptyMoveDirectory"
const SubDirInNonEmptyDestMoveDirectory = "subDestMoveDir"
const DestMoveDirectoryNotExist = "notExist"
const NumberOfObjectsInSrcMoveDirectory = 2
const NumberOfObjectsInNonEmptyDestMoveDirectory = 2
const DestEmptyMoveDirectory = "destEmptyMoveDirectory"
const EmptySrcDirectoryMoveTest = "emptySrcDirectoryMoveTest"
const NumberOfObjectsInEmptyDestMoveDirectory = 1

func checkIfSrcDirectoryGetsRemovedAfterMoveOperation(srcDirPath string, t *testing.T) {
	_, err := os.Stat(srcDirPath)

	if err == nil {
		t.Errorf("Directory exist after move operation.")
	}
}

// Create below directory structure.
// srcMoveDir                  -- Dir
// srcMoveDir/srcMoveFile      -- File
// srcMoveDir/subSrcMoveDir    -- Dir
func createSrcDirectoryWithObjectsForMoveDirTest(dirPath string, t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// testBucket/srcMoveDir
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", dirPath, err)
		return
	}

	// testBucket/subSrcMoveDir
	subDirPath := path.Join(dirPath, SubSrcMoveDirectory)
	err = os.Mkdir(subDirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", subDirPath, err)
		return
	}

	// testBucket/srcMoveDir/srcMoveFile
	filePath := path.Join(dirPath, SrcMoveFile)

	file, err := os.Create(filePath)
	if err != nil {
		t.Errorf("Error in creating file %v:", err)
	}

	// Closing file at the end
	defer operations.CloseFile(file)

	err = operations.WriteFile(file.Name(), SrcMoveFileContent)
	if err != nil {
		t.Errorf("File at %v", err)
	}
}

func checkIfMovedDirectoryHasCorrectData(destDir string, t *testing.T) {
	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Comparing number of objects in the testBucket - 2
	if len(obj) != NumberOfObjectsInSrcMoveDirectory {
		t.Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// Comparing first object name and type
	// Name - testBucket/destMoveDir/srcMoveFile, Type - file
	if obj[0].Name() != SrcMoveFile || obj[0].IsDir() == true {
		t.Errorf("Object Listed for bucket directory is incorrect.")
	}

	// Comparing second object name and type
	// Name - testBucket/destMoveDir/srcMoveDir, Type - dir
	if obj[1].Name() != SubSrcMoveDirectory || obj[1].IsDir() != true {
		t.Errorf("Object Listed for bucket directory is incorrect.")
	}

	destFile := path.Join(destDir, SrcMoveFile)

	content, err := operations.ReadFile(destFile)
	if err != nil {
		t.Errorf("ReadAll: %v", err)
	}
	if got, want := string(content), SrcMoveFileContent; got != want {
		t.Errorf("File content %q not match %q", got, want)
	}
}

// Move SrcDirectory objects in DestDirectory
// srcMoveDir                  -- Dir
// srcMoveDir/srcMoveFile      -- File
// srcMoveDir/subSrcMoveDir    -- Dir

// destMoveDir                  -- Dir
// destMoveDir/srcMoveFile      -- File
// destMoveDir/subSrcMoveDir    -- Dir
func TestMoveDirectoryInNonExistingDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), SrcMoveDirectory)

	createSrcDirectoryWithObjectsForMoveDirTest(srcDir, t)

	destDir := path.Join(setup.MntDir(), DestMoveDirectoryNotExist)

	err := operations.MoveDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in moving directory: %v", err)
	}

	checkIfMovedDirectoryHasCorrectData(destDir, t)
	checkIfSrcDirectoryGetsRemovedAfterMoveOperation(srcDir, t)
}

// Move SrcDirectory in DestDirectory
// srcMoveDir                  -- Dir
// srcMoveDir/srcMoveFile      -- File
// srcMoveDir/subSrcMoveDir    -- Dir

// destMoveDir                             -- Dir
// destMoveDir/srcMoveDir                  -- Dir
// destMoveDir/srcMoveDir/srcMoveFile      -- File
// destMoveDir/srcMoveDir/subSrcMoveDir    -- Dir
func TestMoveDirectoryInEmptyDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), SrcMoveDirectory)

	createSrcDirectoryWithObjectsForMoveDirTest(srcDir, t)

	// Create below directory
	// destMoveDir               -- Dir
	destDir := path.Join(setup.MntDir(), DestMoveDirectory)
	err := os.Mkdir(destDir, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory: %v", err)
	}

	err = operations.MoveDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in moving directory: %v", err)
	}

	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destMoveDirectory has the correct directory copied.
	// destMoveDirectory
	// destMoveDirectory/srcMoveDirectory
	if len(obj) != 1 || obj[0].Name() != SrcMoveDirectory || obj[0].IsDir() != true {
		t.Errorf("Error in moving directory.")
		return
	}

	destSrc := path.Join(destDir, SrcMoveDirectory)
	checkIfMovedDirectoryHasCorrectData(destSrc, t)
	checkIfSrcDirectoryGetsRemovedAfterMoveOperation(srcDir, t)
}

func createDestNonEmptyDirectoryForMoveTest(t *testing.T) {
	destDir := path.Join(setup.MntDir(), DestNonEmptyMoveDirectory)
	operations.CreateDirectoryWithNFiles(0, destDir, "", t)

	destSubDir := path.Join(destDir, SubDirInNonEmptyDestMoveDirectory)
	operations.CreateDirectoryWithNFiles(0, destSubDir, "", t)
}

func TestMoveDirectoryInNonEmptyDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), SrcMoveDirectory)

	createSrcDirectoryWithObjectsForMoveDirTest(srcDir, t)

	// Create below directory
	// destMoveDir               -- Dir
	destDir := path.Join(setup.MntDir(), DestNonEmptyMoveDirectory)
	createDestNonEmptyDirectoryForMoveTest(t)

	err := operations.MoveDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in moving directory: %v", err)
	}

	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destMoveDirectory has the correct directory copied.
	// destMoveDirectory
	// destMoveDirectory/srcMoveDirectory
	// destMoveDirectory/subDestMoveDirectory
	if len(obj) != NumberOfObjectsInNonEmptyDestMoveDirectory {
		t.Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// destMoveDirectory/srcMoveDirectory  - Dir
	if obj[0].Name() != SrcMoveDirectory || obj[0].IsDir() != true {
		t.Errorf("Error in moving directory.")
		return
	}

	// destMoveDirectory/subDirInNonEmptyDestMoveDirectory  - Dir
	if obj[1].Name() != SubDirInNonEmptyDestMoveDirectory || obj[1].IsDir() != true {
		t.Errorf("Existing object affected.")
		return
	}

	destSrc := path.Join(destDir, SrcMoveDirectory)
	checkIfMovedDirectoryHasCorrectData(destSrc, t)
	checkIfSrcDirectoryGetsRemovedAfterMoveOperation(srcDir, t)
}

func checkIfMovedEmptyDirectoryHasNoData(destSrc string, t *testing.T) {
	objs, err := os.ReadDir(destSrc)
	if err != nil {
		log.Fatal(err)
	}

	if len(objs) != 0 {
		t.Errorf("Directory has incorrect data.")
	}
}

// Move SrcDirectory in DestDirectory
// emptySrcDirectoryMoveTest

// destNonEmptyMoveDirectory
// destNonEmptyMoveDirectory/subDirInNonEmptyDestMoveDirectory

// Output
// destNonEmptyMoveDirectory
// destNonEmptyMoveDirectory/subDirInNonEmptyDestMoveDirectory
// destNonEmptyMoveDirectory/emptySrcDirectoryMoveTest
func TestMoveEmptyDirectoryInNonEmptyDirectory(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	srcDir := path.Join(setup.MntDir(), EmptySrcDirectoryMoveTest)
	operations.CreateDirectoryWithNFiles(0, srcDir, "", t)

	// Create below directory
	// destNonEmptyMoveDirectory                                                -- Dir
	// destNonEmptyMoveDirectory/subDirInNonEmptyDestMoveDirectory              -- Dir
	destDir := path.Join(setup.MntDir(), DestNonEmptyMoveDirectory)
	createDestNonEmptyDirectoryForMoveTest(t)

	err := operations.MoveDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in moving directory: %v", err)
	}

	objs, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destMoveDirectory has the correct directory copied.
	// destNonEmptyMoveDirectory
	// destNonEmptyMoveDirectory/emptyDirectoryMoveTest           - Dir
	// destNonEmptyMoveDirectory/subDestMoveDirectory             - Dir
	if len(objs) != NumberOfObjectsInNonEmptyDestMoveDirectory {
		t.Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// destNonEmptyMoveDirectory/srcMoveDirectory  - Dir
	if objs[0].Name() != EmptySrcDirectoryMoveTest || objs[0].IsDir() != true {
		t.Errorf("Error in moving directory.")
		return
	}

	// destNonEmptyMoveDirectory/subDirInNonEmptyDestMoveDirectory  - Dir
	if objs[1].Name() != SubDirInNonEmptyDestMoveDirectory || objs[1].IsDir() != true {
		t.Errorf("Existing object affected.")
		return
	}

	movDirPath := path.Join(destDir, EmptySrcDirectoryMoveTest)
	checkIfMovedEmptyDirectoryHasNoData(movDirPath, t)
	checkIfSrcDirectoryGetsRemovedAfterMoveOperation(srcDir, t)
}

// Move SrcDirectory in DestDirectory
// emptySrcDirectoryMoveTest

// destEmptyMoveDirectory

// Output
// destEmptyMoveDirectory/emptySrcDirectoryMoveTest
func TestMoveEmptyDirectoryInEmptyDirectory(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	srcDir := path.Join(setup.MntDir(), EmptySrcDirectoryMoveTest)
	operations.CreateDirectoryWithNFiles(0, srcDir, "", t)

	// Create below directory
	// destMoveDir               -- Dir
	destDir := path.Join(setup.MntDir(), DestEmptyMoveDirectory)
	operations.CreateDirectoryWithNFiles(0, destDir, "", t)

	err := operations.MoveDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in moving directory: %v", err)
	}

	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destMoveDirectory has the correct directory copied.
	// destEmptyMoveDirectory
	// destEmptyMoveDirectory/emptyDirectoryMoveTest
	if len(obj) != NumberOfObjectsInEmptyDestMoveDirectory {
		t.Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// destEmptyMoveDirectory/srcMoveDirectory  - Dir
	if obj[0].Name() != EmptySrcDirectoryMoveTest || obj[0].IsDir() != true {
		t.Errorf("Error in moving directory.")
		return
	}

	movDirPath := path.Join(destDir, EmptySrcDirectoryMoveTest)
	checkIfMovedEmptyDirectoryHasNoData(movDirPath, t)
	checkIfSrcDirectoryGetsRemovedAfterMoveOperation(srcDir, t)
}

// Move SrcDirectory in DestDirectory
// emptySrcDirectoryMoveTest

// Output
// destMoveDirectoryNotExist
func TestMoveEmptyDirectoryInNonExistingDirectory(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	srcDir := path.Join(setup.MntDir(), EmptySrcDirectoryMoveTest)
	operations.CreateDirectoryWithNFiles(0, srcDir, "", t)

	// destMoveDirectoryNotExist             -- Dir
	destDir := path.Join(setup.MntDir(), DestMoveDirectoryNotExist)

	_, err := os.Stat(destDir)
	if err == nil {
		t.Errorf("destMoveDirectoryNotExist directory exist.")
	}

	err = operations.MoveDir(srcDir, destDir)
	if err != nil {
		t.Errorf("Error in moving directory: %v", err)
	}

	checkIfMovedEmptyDirectoryHasNoData(destDir, t)
	checkIfSrcDirectoryGetsRemovedAfterMoveOperation(srcDir, t)
}
