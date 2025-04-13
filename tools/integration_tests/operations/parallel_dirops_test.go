// Copyright 2024 Google LLC
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

// A collection of E2E tests for a file system where parallel dirops are allowed.
// Dirops refers to readdir and lookup operations. These tests are complimentary
// to the composite tests.
package operations_test

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/all_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

// createDirectoryStructureForParallelDiropsTest creates the following files and
// directory structure.
// bucket
//
//	file1.txt
//	file2.txt
//	explicitDir1/file1.txt
//	explicitDir1/file2.txt
//	explicitDir2/file1.txt
//
// Also returns the path to test directory.
func (s *OperationSuite) createDirectoryStructureForParallelDiropsTest() string {
	testDir := setup.SetupTestDirectoryOnMntDir(s.mountConfiguration.MntDir(), TestDirName(s.T()))
	setup.CleanUpDir(testDir)

	// Create explicitDir1 structure
	explicitDir1 := path.Join(testDir, "explicitDir1")
	operations.CreateDirectory(explicitDir1, s.T())
	filePath1 := path.Join(explicitDir1, "file1.txt")
	operations.CreateFileOfSize(5, filePath1, s.T())
	filePath2 := path.Join(explicitDir1, "file2.txt")
	operations.CreateFileOfSize(10, filePath2, s.T())

	// Create explicitDir2 structure
	explicitDir2 := path.Join(testDir, "explicitDir2")
	operations.CreateDirectory(explicitDir2, s.T())
	filePath1 = path.Join(explicitDir2, "file1.txt")
	operations.CreateFileOfSize(11, filePath1, s.T())

	filePath1 = path.Join(testDir, "file1.txt")
	operations.CreateFileOfSize(5, filePath1, s.T())
	filePath2 = path.Join(testDir, "file2.txt")
	operations.CreateFileOfSize(3, filePath2, s.T())

	return testDir
}

// lookUpFileStat performs a lookup for the given file path and returns the FileInfo and error.
func lookUpFileStat(wg *sync.WaitGroup, filePath string, result *os.FileInfo, err *error) {
	defer wg.Done()
	fileInfo, lookupErr := os.Stat(filePath)
	*result = fileInfo
	*err = lookupErr
}

func (s *OperationSuite) TestParallelLookUpsForSameFile() {
	// Create directory structure for testing.
	testDir := s.createDirectoryStructureForParallelDiropsTest()
	var stat1, stat2 os.FileInfo
	var err1, err2 error

	// Parallel lookups of file just under mount.
	filePath := path.Join(testDir, "file1.txt")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go lookUpFileStat(&wg, filePath, &stat1, &err1)
	go lookUpFileStat(&wg, filePath, &stat2, &err2)
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(s.T(), err1)
	assert.NoError(s.T(), err2)
	assert.Equal(s.T(), int64(5), stat1.Size())
	assert.Equal(s.T(), int64(5), stat2.Size())
	assert.Contains(s.T(), filePath, stat1.Name())
	assert.Contains(s.T(), filePath, stat2.Name())

	// Parallel lookups of file under a directory in mount.
	filePath = path.Join(testDir, "explicitDir1/file2.txt")
	wg.Add(2)
	go lookUpFileStat(&wg, filePath, &stat1, &err1)
	go lookUpFileStat(&wg, filePath, &stat2, &err2)
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(s.T(), err1)
	assert.NoError(s.T(), err2)
	assert.Equal(s.T(), int64(10), stat1.Size())
	assert.Equal(s.T(), int64(10), stat2.Size())
	assert.Contains(s.T(), filePath, stat1.Name())
	assert.Contains(s.T(), filePath, stat2.Name())
}

func (s *OperationSuite) TestParallelReadDirs() {
	// Create directory structure for testing.
	testDir := s.createDirectoryStructureForParallelDiropsTest()
	readDirFunc := func(wg *sync.WaitGroup, dirPath string, dirEntries *[]os.DirEntry, err *error) {
		defer wg.Done()
		*dirEntries, *err = os.ReadDir(dirPath)
	}
	var dirEntries1, dirEntries2 []os.DirEntry
	var err1, err2 error

	// Parallel readDirs of explicit dir under mount.
	dirPath := path.Join(testDir, "explicitDir1")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go readDirFunc(&wg, dirPath, &dirEntries1, &err1)
	go readDirFunc(&wg, dirPath, &dirEntries2, &err2)

	wg.Wait()

	// Assert both readDirs passed and give correct information
	assert.NoError(s.T(), err1)
	assert.NoError(s.T(), err2)
	assert.Equal(s.T(), 2, len(dirEntries1))
	assert.Equal(s.T(), 2, len(dirEntries2))
	assert.Contains(s.T(), "file1.txt", dirEntries1[0].Name())
	assert.Contains(s.T(), "file2.txt", dirEntries1[1].Name())
	assert.Contains(s.T(), "file1.txt", dirEntries2[0].Name())
	assert.Contains(s.T(), "file2.txt", dirEntries2[1].Name())

	// Parallel readDirs of a directory and its parent directory.
	dirPath = path.Join(testDir, "explicitDir1")
	parentDirPath := testDir
	wg = sync.WaitGroup{}
	wg.Add(2)
	go readDirFunc(&wg, dirPath, &dirEntries1, &err1)
	go readDirFunc(&wg, parentDirPath, &dirEntries2, &err2)
	wg.Wait()

	// Assert both readDirs passed and give correct information
	assert.NoError(s.T(), err1)
	assert.NoError(s.T(), err2)
	assert.Equal(s.T(), 2, len(dirEntries1))
	assert.Equal(s.T(), 4, len(dirEntries2))
	assert.Contains(s.T(), "file1.txt", dirEntries1[0].Name())
	assert.Contains(s.T(), "file2.txt", dirEntries1[1].Name())
	assert.Contains(s.T(), "explicitDir1", dirEntries2[0].Name())
	assert.Contains(s.T(), "explicitDir2", dirEntries2[1].Name())
	assert.Contains(s.T(), "file1.txt", dirEntries2[2].Name())
	assert.Contains(s.T(), "file2.txt", dirEntries2[3].Name())
}

func (s *OperationSuite) TestParallelLookUpAndDeleteSameDir() {
	// Create directory structure for testing.
	testDir := s.createDirectoryStructureForParallelDiropsTest()
	deleteFunc := func(wg *sync.WaitGroup, dirPath string, err *error) {
		defer wg.Done()
		*err = os.RemoveAll(dirPath)
	}
	var statInfo os.FileInfo
	var lookUpErr, deleteErr error

	// Parallel lookup and deletion of explicit dir under mount.
	dirPath := path.Join(testDir, "explicitDir1")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go lookUpFileStat(&wg, dirPath, &statInfo, &lookUpErr)
	go deleteFunc(&wg, dirPath, &deleteErr)
	wg.Wait()

	assert.NoError(s.T(), deleteErr)
	_, err := os.Stat(dirPath)
	assert.True(s.T(), os.IsNotExist(err))
	// Assert either dir is looked up first or deleted first
	if lookUpErr == nil {
		assert.NotNil(s.T(), statInfo, "statInfo should not be nil when lookUpErr is nil")
		assert.Contains(s.T(), statInfo.Name(), "explicitDir1")
		assert.True(s.T(), statInfo.IsDir(), "The created path should be a directory")
	} else {
		assert.True(s.T(), os.IsNotExist(lookUpErr))
	}
}

func (s *OperationSuite) TestParallelLookUpsForDifferentFiles() {
	// Create directory structure for testing.
	testDir := s.createDirectoryStructureForParallelDiropsTest()
	var stat1, stat2 os.FileInfo
	var err1, err2 error

	// Parallel lookups of two files just under mount.
	filePath1 := path.Join(testDir, "file1.txt")
	filePath2 := path.Join(testDir, "file2.txt")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go lookUpFileStat(&wg, filePath1, &stat1, &err1)
	go lookUpFileStat(&wg, filePath2, &stat2, &err2)

	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(s.T(), err1)
	assert.NoError(s.T(), err2)
	assert.Equal(s.T(), int64(5), stat1.Size())
	assert.Equal(s.T(), int64(3), stat2.Size())
	assert.Contains(s.T(), filePath1, stat1.Name())
	assert.Contains(s.T(), filePath2, stat2.Name())

	// Parallel lookups of two files under a directory in mount.
	filePath1 = path.Join(testDir, "explicitDir1", "file1.txt")
	filePath2 = path.Join(testDir, "explicitDir1", "file2.txt")
	wg = sync.WaitGroup{}
	wg.Add(2)
	go lookUpFileStat(&wg, filePath1, &stat1, &err1)
	go lookUpFileStat(&wg, filePath2, &stat2, &err2)
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(s.T(), err1)
	assert.NoError(s.T(), err2)
	assert.Equal(s.T(), int64(5), stat1.Size())
	assert.Equal(s.T(), int64(10), stat2.Size())
	assert.Contains(s.T(), filePath1, stat1.Name())
	assert.Contains(s.T(), filePath2, stat2.Name())
}

func (s *OperationSuite) TestParallelReadDirAndMkdirInsideSameDir() {
	// Create directory structure for testing.
	testDir := s.createDirectoryStructureForParallelDiropsTest()
	readDirFunc := func(wg *sync.WaitGroup, dirPath string, dirEntries *[]os.DirEntry, err *error) {
		defer wg.Done()
		*err = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			*dirEntries = append(*dirEntries, d)
			return nil
		})
	}
	mkdirFunc := func(wg *sync.WaitGroup, dirPath string, err *error) {
		defer wg.Done()
		*err = os.Mkdir(dirPath, setup.DirPermission_0755)
	}
	var dirEntries []os.DirEntry
	var readDirErr, mkdirErr error

	// Parallel readDirs and mkdir inside the same directory.
	newDirPath := path.Join(testDir, "newDir")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go readDirFunc(&wg, testDir, &dirEntries, &readDirErr)
	go mkdirFunc(&wg, newDirPath, &mkdirErr)
	wg.Wait()

	// Assert both listing and mkdir succeeded
	assert.NoError(s.T(), readDirErr)
	assert.NoError(s.T(), mkdirErr)
	dirStatInfo, err := os.Stat(newDirPath)
	assert.NoError(s.T(), err)
	assert.True(s.T(), dirStatInfo.IsDir(), "The created path should be a directory")
	// List should happen either before or after creation of newDir.
	assert.GreaterOrEqual(s.T(), len(dirEntries), 8)
	assert.LessOrEqual(s.T(), len(dirEntries), 9)
	if len(dirEntries) == 9 {
		assert.Contains(s.T(), dirEntries[8].Name(), "newDir")
	}
}

func (s *OperationSuite) TestParallelLookUpAndDeleteSameFile() {
	// Create directory structure for testing.
	testDir := s.createDirectoryStructureForParallelDiropsTest()
	deleteFileFunc := func(wg *sync.WaitGroup, filePath string, err *error) {
		defer wg.Done()
		*err = os.Remove(filePath)
	}
	var fileInfo os.FileInfo
	var lookUpErr, deleteErr error

	// Parallel lookup and deletion of a file.
	filePath := path.Join(testDir, "explicitDir1", "file1.txt")
	wg := sync.WaitGroup{}
	wg.Add(2)

	go lookUpFileStat(&wg, filePath, &fileInfo, &lookUpErr)
	go deleteFileFunc(&wg, filePath, &deleteErr)

	wg.Wait()

	assert.NoError(s.T(), deleteErr)
	_, err := os.Stat(filePath)
	assert.True(s.T(), os.IsNotExist(err))
	// Assert either file is looked up first or deleted first
	if lookUpErr == nil {
		assert.NotNil(s.T(), fileInfo, "fileInfo should not be nil when lookUpErr is nil")
		assert.Equal(s.T(), int64(5), fileInfo.Size())
		assert.Contains(s.T(), fileInfo.Name(), "file1.txt")
		assert.False(s.T(), fileInfo.IsDir(), "The created path should not be a directory")
	} else {
		assert.True(s.T(), os.IsNotExist(lookUpErr))
	}
}

func (s *OperationSuite) TestParallelLookUpAndRenameSameFile() {
	// Create directory structure for testing.
	testDir := s.createDirectoryStructureForParallelDiropsTest()
	renameFunc := func(wg *sync.WaitGroup, oldFilePath string, newFilePath string, err *error) {
		defer wg.Done()
		*err = os.Rename(oldFilePath, newFilePath)
	}
	var fileInfo os.FileInfo
	var lookUpErr, renameErr error

	// Parallel lookup and rename of a file.
	filePath := path.Join(testDir, "explicitDir1", "file1.txt")
	newFilePath := path.Join(testDir, "newFile.txt")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go lookUpFileStat(&wg, filePath, &fileInfo, &lookUpErr)
	go renameFunc(&wg, filePath, newFilePath, &renameErr)

	wg.Wait()

	assert.NoError(s.T(), renameErr)
	newFileInfo, err := os.Stat(newFilePath)
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), newFileInfo.Name(), "newFile.txt")
	assert.False(s.T(), newFileInfo.IsDir())
	assert.Equal(s.T(), int64(5), newFileInfo.Size())
	// Assert either file is renamed first or looked up first
	if lookUpErr == nil {
		assert.NotNil(s.T(), fileInfo, "fileInfo should not be nil when lookUpErr is nil")
		assert.Equal(s.T(), int64(5), fileInfo.Size())
		assert.Contains(s.T(), fileInfo.Name(), "file1.txt")
		assert.False(s.T(), fileInfo.IsDir(), "The created path should not be a directory")
	} else {
		assert.True(s.T(), os.IsNotExist(lookUpErr))
	}
}

func (s *OperationSuite) TestParallelLookUpAndMkdirSameDir() {
	// Create directory structure for testing.
	testDir := s.createDirectoryStructureForParallelDiropsTest()
	mkdirFunc := func(wg *sync.WaitGroup, dirPath string, err *error) {
		defer wg.Done()
		*err = os.Mkdir(dirPath, setup.DirPermission_0755)
	}

	var statInfo os.FileInfo
	var lookUpErr, mkdirErr error

	dirPath := path.Join(testDir, "newDir")
	var wg sync.WaitGroup
	wg.Add(2)

	go lookUpFileStat(&wg, dirPath, &statInfo, &lookUpErr)
	go mkdirFunc(&wg, dirPath, &mkdirErr)
	wg.Wait()

	// Assert either directory is created first or looked up first
	assert.NoError(s.T(), mkdirErr, "mkdirFunc should not fail")

	if lookUpErr == nil {
		assert.NotNil(s.T(), statInfo, "statInfo should not be nil when lookUpErr is nil")
		assert.Contains(s.T(), statInfo.Name(), "newDir")
		assert.True(s.T(), statInfo.IsDir())
	} else {
		assert.True(s.T(), os.IsNotExist(lookUpErr), "lookUpErr should indicate directory does not exist")
		dirStatInfo, err := os.Stat(dirPath)
		assert.NoError(s.T(), err, "os.Stat should succeed after directory creation")
		assert.True(s.T(), dirStatInfo.IsDir(), "The created path should be a directory")
	}
}
