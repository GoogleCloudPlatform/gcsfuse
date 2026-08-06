// Copyright 2023 Google LLC
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

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
)

// Create below directory structure.
// srcCopyDir               -- Dir
// srcCopyDir/copy.txt      -- File
// srcCopyDir/subSrcCopyDir -- Dir
func (s *operationsTestSuite) createSrcDirectoryWithObjects(dirPath string) string {
	// testBucket/srcCopyDir
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		s.T().Errorf("Mkdir at %q: %v", dirPath, err)
		return ""
	}

	// testBucket/subSrcCopyDir
	subDirPath := path.Join(dirPath, SubSrcCopyDirectory)
	err = os.Mkdir(subDirPath, setup.FilePermission_0600)
	if err != nil {
		s.T().Errorf("Mkdir at %q: %v", subDirPath, err)
		return ""
	}

	// testBucket/srcCopyDir/copy.txt
	filePath := path.Join(dirPath, SrcCopyFile)

	file, err := os.Create(filePath)
	if err != nil {
		s.T().Errorf("Error in creating file %v:", err)
	}

	err = operations.WriteFile(file.Name(), SrcCopyFileContent)
	if err != nil {
		s.T().Errorf("File at %v", err)
	}

	// Closing file at the end
	defer operations.CloseFileShouldNotThrowError(s.T(), file)

	return dirPath
}

func (s *operationsTestSuite) checkIfCopiedDirectoryHasCorrectData(destDir string) {
	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Comparing number of objects in the testBucket - 2
	if len(obj) != NumberOfObjectsInSrcCopyDirectory {
		s.T().Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// Comparing first object name and type
	// Name - testBucket/destCopyDir/copy.txt, Type - file
	if obj[0].Name() != SrcCopyFile || obj[0].IsDir() == true {
		s.T().Errorf("Object Listed for bucket directory is incorrect.")
	}

	// Comparing second object name and type
	// Name - testBucket/destCopyDir/srcCopyDir, Type - dir
	if obj[1].Name() != SubSrcCopyDirectory || obj[1].IsDir() != true {
		s.T().Errorf("Object Listed for bucket directory is incorrect.")
	}

	destFile := path.Join(destDir, SrcCopyFile)

	content, err := operations.ReadFile(destFile)
	if err != nil {
		s.T().Errorf("ReadAll: %v", err)
	}
	if got, want := string(content), SrcCopyFileContent; got != want {
		s.T().Errorf("File content %q not match %q", got, want)
	}
}

// Copy SrcDirectory objects in DestDirectory
// srcCopyDir               -- Dir
// srcCopyDir/copy.txt      -- File
// srcCopyDir/subSrcCopyDir -- Dir

// destCopyDir               -- Dir
// destCopyDir/copy.txt      -- File
// destCopyDir/subSrcCopyDir -- Dir
func (s *operationsTestSuite) TestCopyDirectoryInNonExistingDirectory() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	srcDir := s.createSrcDirectoryWithObjects(path.Join(testDir, SrcCopyDirectory))
	destDir := path.Join(testDir, DestCopyDirectoryNotExist)

	err := operations.CopyDir(srcDir, destDir)
	require.NoError(s.T(), err, "Error in copying directory")

	s.checkIfCopiedDirectoryHasCorrectData(destDir)
}

// Copy SrcDirectory in DestDirectory
// srcCopyDir               -- Dir
// srcCopyDir/copy.txt      -- File
// srcCopyDir/subSrcCopyDir -- Dir

// destCopyDir                          -- Dir
// destCopyDir/srcCopyDir               -- Dir
// destCopyDir/srcCopyDir/copy.txt      -- File
// destCopyDir/srcCopyDir/subSrcCopyDir -- Dir
func (s *operationsTestSuite) TestCopyDirectoryInEmptyDirectory() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	srcDir := s.createSrcDirectoryWithObjects(path.Join(testDir, SrcCopyDirectory))

	// Create below directory
	// destCopyDir               -- Dir
	destDir := path.Join(testDir, DestCopyDirectory)
	err := os.Mkdir(destDir, setup.FilePermission_0600)
	if err != nil {
		s.T().Errorf("Error in creating directory: %v", err)
	}

	err = operations.CopyDir(srcDir, destDir)
	require.NoError(s.T(), err, "Error in copying directory")

	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destCopyDirectory has the correct directory copied.
	// destCopyDirectory
	// destCopyDirectory/srcCopyDirectory
	if len(obj) != 1 || obj[0].Name() != SrcCopyDirectory || obj[0].IsDir() != true {
		s.T().Errorf("Error in copying directory.")
		return
	}

	destSrc := path.Join(destDir, SrcCopyDirectory)
	s.checkIfCopiedDirectoryHasCorrectData(destSrc)
}

func (s *operationsTestSuite) createDestNonEmptyDirectory(dirPath string) string {
	operations.CreateDirectoryWithNFiles(0, dirPath, "", s.T())

	destSubDir := path.Join(dirPath, SubDirInNonEmptyDestCopyDirectory)
	operations.CreateDirectoryWithNFiles(0, destSubDir, "", s.T())

	return dirPath
}

func (s *operationsTestSuite) TestCopyDirectoryInNonEmptyDirectory() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	srcDir := s.createSrcDirectoryWithObjects(path.Join(testDir, SrcCopyDirectory))

	// Create below directory
	// destCopyDir               -- Dir
	destDir := s.createDestNonEmptyDirectory(path.Join(testDir, DestNonEmptyCopyDirectory))

	err := operations.CopyDir(srcDir, destDir)
	require.NoError(s.T(), err, "Error in copying directory")

	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destCopyDirectory has the correct directory copied.
	// destCopyDirectory
	// destCopyDirectory/srcCopyDirectory
	// destCopyDirectory/subDestCopyDirectory
	if len(obj) != NumberOfObjectsInNonEmptyDestCopyDirectory {
		s.T().Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// destCopyDirectory/srcCopyDirectory  - Dir
	if obj[0].Name() != SrcCopyDirectory || obj[0].IsDir() != true {
		s.T().Errorf("Error in copying directory.")
		return
	}

	// destCopyDirectory/subDirInNonEmptyDestCopyDirectory  - Dir
	if obj[1].Name() != SubDirInNonEmptyDestCopyDirectory || obj[1].IsDir() != true {
		s.T().Errorf("Existing object affected.")
		return
	}

	destSrc := path.Join(destDir, SrcCopyDirectory)
	s.checkIfCopiedDirectoryHasCorrectData(destSrc)
}

func (s *operationsTestSuite) checkIfCopiedEmptyDirectoryHasNoData(destSrc string) {
	objs, err := os.ReadDir(destSrc)
	if err != nil {
		log.Fatal(err)
	}

	if len(objs) != 0 {
		s.T().Errorf("Directory has incorrect data.")
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
func (s *operationsTestSuite) TestCopyEmptyDirectoryInNonEmptyDirectory() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	srcDir := path.Join(testDir, EmptySrcDirectoryCopyTest)
	operations.CreateDirectoryWithNFiles(0, srcDir, "", s.T())

	// Create below directory
	// destNonEmptyCopyDirectory                                                -- Dir
	// destNonEmptyCopyDirectory/subDirInNonEmptyDestCopyDirectory              -- Dir
	destDir := s.createDestNonEmptyDirectory(path.Join(testDir, DestNonEmptyCopyDirectory))

	err := operations.CopyDir(srcDir, destDir)
	require.NoError(s.T(), err, "Error in copying directory")

	objs, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destCopyDirectory has the correct directory copied.
	// destNonEmptyCopyDirectory
	// destNonEmptyCopyDirectory/emptyDirectoryCopyTest           - Dir
	// destNonEmptyCopyDirectory/subDestCopyDirectory             - Dir
	if len(objs) != NumberOfObjectsInNonEmptyDestCopyDirectory {
		s.T().Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// destNonEmptyCopyDirectory/srcCopyDirectory  - Dir
	if objs[0].Name() != EmptySrcDirectoryCopyTest || objs[0].IsDir() != true {
		s.T().Errorf("Error in copying directory.")
		return
	}

	// destNonEmptyCopyDirectory/subDirInNonEmptyDestCopyDirectory  - Dir
	if objs[1].Name() != SubDirInNonEmptyDestCopyDirectory || objs[1].IsDir() != true {
		s.T().Errorf("Existing object affected.")
		return
	}

	copyDirPath := path.Join(destDir, EmptySrcDirectoryCopyTest)
	s.checkIfCopiedEmptyDirectoryHasNoData(copyDirPath)
}

// Copy SrcDirectory in DestDirectory
// emptySrcDirectoryCopyTest

// destEmptyCopyDirectory

// Output
// destEmptyCopyDirectory/emptySrcDirectoryCopyTest
func (s *operationsTestSuite) TestCopyEmptyDirectoryInEmptyDirectory() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	srcDir := path.Join(testDir, EmptySrcDirectoryCopyTest)
	operations.CreateDirectoryWithNFiles(0, srcDir, "", s.T())

	// Create below directory
	// destCopyDir               -- Dir
	destDir := path.Join(testDir, DestEmptyCopyDirectory)
	operations.CreateDirectoryWithNFiles(0, destDir, "", s.T())

	err := operations.CopyDir(srcDir, destDir)
	require.NoError(s.T(), err, "Error in copying directory")

	obj, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatal(err)
	}

	// Check if destCopyDirectory has the correct directory copied.
	// destEmptyCopyDirectory
	// destEmptyCopyDirectory/emptyDirectoryCopyTest
	if len(obj) != NumberOfObjectsInEmptyDestCopyDirectory {
		s.T().Errorf("The number of objects in the current directory doesn't match.")
		return
	}

	// destEmptyCopyDirectory/srcCopyDirectory  - Dir
	if obj[0].Name() != EmptySrcDirectoryCopyTest || obj[0].IsDir() != true {
		s.T().Errorf("Error in copying directory.")
		return
	}

	copyDirPath := path.Join(destDir, EmptySrcDirectoryCopyTest)
	s.checkIfCopiedEmptyDirectoryHasNoData(copyDirPath)
}

// Copy SrcDirectory in DestDirectory
// emptySrcDirectoryCopyTest

// Output
// destCopyDirectoryNotExist
func (s *operationsTestSuite) TestCopyEmptyDirectoryInNonExistingDirectory() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	srcDir := path.Join(testDir, EmptySrcDirectoryCopyTest)
	operations.CreateDirectoryWithNFiles(0, srcDir, "", s.T())

	// destCopyDirectoryNotExist             -- Dir
	destDir := path.Join(testDir, DestCopyDirectoryNotExist)

	_, err := os.Stat(destDir)
	if err == nil {
		s.T().Errorf("destCopyDirectoryNotExist directory exist.")
	}

	err = operations.CopyDir(srcDir, destDir)
	require.NoError(s.T(), err, "Error in copying directory")

	s.checkIfCopiedEmptyDirectoryHasNoData(destDir)
}
