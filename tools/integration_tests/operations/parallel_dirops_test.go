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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

type testDirStrucure struct {
	testDir                 string
	explicitDir1Name        string
	file1InExplicitDir1Name string
	file2InExplicitDir1Name string
	explicitDir2Name        string
	file1InExplicitDir2Name string
	file1Name               string
	file2Name               string
}

// createDirStructure creates the following files and
// directory structure.
// bucket
//
//	file1Name
//	file2Name
//	explicitDir1Name/file1InExplicitDir1Name
//	explicitDir1Name/file2InExplicitDir1Name
//	explicitDir2Name/file1InExplicitDir2Name
//
// Also returns the path to test directory.
func createDirStructure(t *testing.T) testDirStrucure {
	var tds testDirStrucure
	tds.testDir = setup.SetupTestDirectory(DirForOperationTests + "-" + setup.GenerateRandomString(5))

	// Create explicitDir1 structure
	tds.explicitDir1Name = "explicitDir1-" + setup.GenerateRandomString(5)
	explicitDir1 := path.Join(tds.testDir, tds.explicitDir1Name)
	operations.CreateDirectory(explicitDir1, t)
	tds.file1InExplicitDir1Name = "file1-" + setup.GenerateRandomString(5) + ".txt"
	filePath1 := path.Join(explicitDir1, tds.file1InExplicitDir1Name)
	operations.CreateFileOfSize(5, filePath1, t)
	tds.file2InExplicitDir1Name = "file2-" + setup.GenerateRandomString(5) + ".txt"
	filePath2 := path.Join(explicitDir1, tds.file2InExplicitDir1Name)
	operations.CreateFileOfSize(10, filePath2, t)

	// Create explicitDir2 structure
	tds.explicitDir2Name = "explicitDir2-" + setup.GenerateRandomString(5)
	explicitDir2 := path.Join(tds.testDir, tds.explicitDir2Name)
	operations.CreateDirectory(explicitDir2, t)
	tds.file1InExplicitDir2Name = "file1-" + setup.GenerateRandomString(5) + ".txt"
	filePath1 = path.Join(explicitDir2, tds.file1InExplicitDir2Name)
	operations.CreateFileOfSize(11, filePath1, t)

	tds.file1Name = "file1-" + setup.GenerateRandomString(5) + ".txt"
	filePath1 = path.Join(tds.testDir, tds.file1Name)
	operations.CreateFileOfSize(5, filePath1, t)
	tds.file2Name = "file2-" + setup.GenerateRandomString(5) + ".txt"
	filePath2 = path.Join(tds.testDir, tds.file2Name)
	operations.CreateFileOfSize(3, filePath2, t)

	return tds
}

// deleteDirStructure deletes the following files and
// directory structure.
// bucket
//
//	file1Name
//	file2Name
//	explicitDir1Name/file1InExplicitDir1Name
//	explicitDir1Name/file2InExplicitDir1Name
//	explicitDir2Name/file1InExplicitDir2Name
//
// Also returns the path to test directory.
func deleteDirStructure(tds testDirStrucure) {
	setup.CleanUpDir(tds.testDir)
}

// lookUpFileStat performs a lookup for the given file path and returns the FileInfo and error.
func lookUpFileStat(wg *sync.WaitGroup, filePath string, result *os.FileInfo, err *error) {
	defer wg.Done()
	fileInfo, lookupErr := os.Stat(filePath)
	*result = fileInfo
	*err = lookupErr
}

func TestParallelLookUpsForSameFile(t *testing.T) {
	// Create directory structure for testing.
	tds := createDirStructure(t)
	defer deleteDirStructure(tds)
	var stat1, stat2 os.FileInfo
	var err1, err2 error

	// Parallel lookups of file just under mount.
	filePath := path.Join(tds.testDir, tds.file1Name)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go lookUpFileStat(&wg, filePath, &stat1, &err1)
	go lookUpFileStat(&wg, filePath, &stat2, &err2)
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, int64(5), stat1.Size())
	assert.Equal(t, int64(5), stat2.Size())
	assert.Contains(t, filePath, stat1.Name())
	assert.Contains(t, filePath, stat2.Name())

	// Parallel lookups of file under a directory in mount.
	filePath = path.Join(tds.testDir, tds.explicitDir1Name, tds.file2InExplicitDir1Name)
	wg.Add(2)
	go lookUpFileStat(&wg, filePath, &stat1, &err1)
	go lookUpFileStat(&wg, filePath, &stat2, &err2)
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, int64(10), stat1.Size())
	assert.Equal(t, int64(10), stat2.Size())
	assert.Contains(t, filePath, stat1.Name())
	assert.Contains(t, filePath, stat2.Name())
}

func TestParallelReadDirs(t *testing.T) {
	// Create directory structure for testing.
	tds := createDirStructure(t)
	defer deleteDirStructure(tds)
	readDirFunc := func(wg *sync.WaitGroup, dirPath string, dirEntries *[]os.DirEntry, err *error) {
		defer wg.Done()
		*dirEntries, *err = os.ReadDir(dirPath)
	}
	var dirEntries1, dirEntries2 []os.DirEntry
	var err1, err2 error

	// Parallel readDirs of explicit dir under mount.
	dirPath := path.Join(tds.testDir, tds.explicitDir1Name)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go readDirFunc(&wg, dirPath, &dirEntries1, &err1)
	go readDirFunc(&wg, dirPath, &dirEntries2, &err2)

	wg.Wait()

	// Assert both readDirs passed and give correct information
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 2, len(dirEntries1))
	assert.Equal(t, 2, len(dirEntries2))
	assert.Contains(t, tds.file1InExplicitDir1Name, dirEntries1[0].Name())
	assert.Contains(t, tds.file2InExplicitDir1Name, dirEntries1[1].Name())
	assert.Contains(t, tds.file1InExplicitDir1Name, dirEntries2[0].Name())
	assert.Contains(t, tds.file2InExplicitDir1Name, dirEntries2[1].Name())

	// Parallel readDirs of a directory and its parent directory.
	dirPath = path.Join(tds.testDir, tds.explicitDir1Name)
	parentDirPath := tds.testDir
	wg = sync.WaitGroup{}
	wg.Add(2)
	go readDirFunc(&wg, dirPath, &dirEntries1, &err1)
	go readDirFunc(&wg, parentDirPath, &dirEntries2, &err2)
	wg.Wait()

	// Assert both readDirs passed and give correct information
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 2, len(dirEntries1))
	assert.Equal(t, 4, len(dirEntries2))
	assert.Contains(t, tds.file1InExplicitDir1Name, dirEntries1[0].Name())
	assert.Contains(t, tds.file2InExplicitDir1Name, dirEntries1[1].Name())
	assert.Contains(t, tds.explicitDir1Name, dirEntries2[0].Name())
	assert.Contains(t, tds.explicitDir2Name, dirEntries2[1].Name())
	assert.Contains(t, tds.file1Name, dirEntries2[2].Name())
	assert.Contains(t, tds.file2Name, dirEntries2[3].Name())
}

func TestParallelLookUpAndDeleteSameDir(t *testing.T) {
	// Create directory structure for testing.
	tds := createDirStructure(t)
	defer deleteDirStructure(tds)
	deleteFunc := func(wg *sync.WaitGroup, dirPath string, err *error) {
		defer wg.Done()
		*err = os.RemoveAll(dirPath)
	}
	var statInfo os.FileInfo
	var lookUpErr, deleteErr error

	// Parallel lookup and deletion of explicit dir under mount.
	dirPath := path.Join(tds.testDir, tds.explicitDir1Name)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go lookUpFileStat(&wg, dirPath, &statInfo, &lookUpErr)
	go deleteFunc(&wg, dirPath, &deleteErr)
	wg.Wait()

	assert.NoError(t, deleteErr)
	_, err := os.Stat(dirPath)
	assert.True(t, os.IsNotExist(err))
	// Assert either dir is looked up first or deleted first
	if lookUpErr == nil {
		assert.NotNil(t, statInfo, "statInfo should not be nil when lookUpErr is nil")
		assert.Contains(t, statInfo.Name(), tds.explicitDir1Name)
		assert.True(t, statInfo.IsDir(), "The created path should be a directory")
	} else {
		assert.True(t, os.IsNotExist(lookUpErr))
	}
}

func TestParallelLookUpsForDifferentFiles(t *testing.T) {
	// Create directory structure for testing.
	tds := createDirStructure(t)
	defer deleteDirStructure(tds)
	var stat1, stat2 os.FileInfo
	var err1, err2 error

	// Parallel lookups of two files just under mount.
	filePath1 := path.Join(tds.testDir, tds.file1Name)
	filePath2 := path.Join(tds.testDir, tds.file2Name)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go lookUpFileStat(&wg, filePath1, &stat1, &err1)
	go lookUpFileStat(&wg, filePath2, &stat2, &err2)

	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, int64(5), stat1.Size())
	assert.Equal(t, int64(3), stat2.Size())
	assert.Contains(t, filePath1, stat1.Name())
	assert.Contains(t, filePath2, stat2.Name())

	// Parallel lookups of two files under a directory in mount.
	filePath1 = path.Join(tds.testDir, tds.explicitDir1Name, tds.file1InExplicitDir1Name)
	filePath2 = path.Join(tds.testDir, tds.explicitDir1Name, tds.file2InExplicitDir1Name)
	wg = sync.WaitGroup{}
	wg.Add(2)
	go lookUpFileStat(&wg, filePath1, &stat1, &err1)
	go lookUpFileStat(&wg, filePath2, &stat2, &err2)
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, int64(5), stat1.Size())
	assert.Equal(t, int64(10), stat2.Size())
	assert.Contains(t, filePath1, stat1.Name())
	assert.Contains(t, filePath2, stat2.Name())
}

func TestParallelReadDirAndMkdirInsideSameDir(t *testing.T) {
	// Create directory structure for testing.
	tds := createDirStructure(t)
	defer deleteDirStructure(tds)
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
	newDirPath := path.Join(tds.testDir, "newDir")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go readDirFunc(&wg, tds.testDir, &dirEntries, &readDirErr)
	go mkdirFunc(&wg, newDirPath, &mkdirErr)
	wg.Wait()

	// Assert both listing and mkdir succeeded
	assert.NoError(t, readDirErr)
	assert.NoError(t, mkdirErr)
	dirStatInfo, err := os.Stat(newDirPath)
	assert.NoError(t, err)
	assert.True(t, dirStatInfo.IsDir(), "The created path should be a directory")
	// List should happen either before or after creation of newDir.
	assert.GreaterOrEqual(t, len(dirEntries), 8)
	assert.LessOrEqual(t, len(dirEntries), 9)
	if len(dirEntries) == 9 {
		assert.Contains(t, dirEntries[8].Name(), "newDir")
	}
}

func TestParallelLookUpAndDeleteSameFile(t *testing.T) {
	// Create directory structure for testing.
	tds := createDirStructure(t)
	defer deleteDirStructure(tds)
	deleteFileFunc := func(wg *sync.WaitGroup, filePath string, err *error) {
		defer wg.Done()
		*err = os.Remove(filePath)
	}
	var fileInfo os.FileInfo
	var lookUpErr, deleteErr error

	// Parallel lookup and deletion of a file.
	filePath := path.Join(tds.testDir, tds.explicitDir1Name, tds.file1InExplicitDir1Name)
	wg := sync.WaitGroup{}
	wg.Add(2)

	go lookUpFileStat(&wg, filePath, &fileInfo, &lookUpErr)
	go deleteFileFunc(&wg, filePath, &deleteErr)

	wg.Wait()

	assert.NoError(t, deleteErr)
	_, err := os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))
	// Assert either file is looked up first or deleted first
	if lookUpErr == nil {
		assert.NotNil(t, fileInfo, "fileInfo should not be nil when lookUpErr is nil")
		assert.Equal(t, int64(5), fileInfo.Size())
		assert.Contains(t, fileInfo.Name(), tds.file1InExplicitDir1Name)
		assert.False(t, fileInfo.IsDir(), "The created path should not be a directory")
	} else {
		assert.True(t, os.IsNotExist(lookUpErr))
	}
}

func TestParallelLookUpAndRenameSameFile(t *testing.T) {
	// Create directory structure for testing.
	tds := createDirStructure(t)
	defer deleteDirStructure(tds)
	renameFunc := func(wg *sync.WaitGroup, oldFilePath string, newFilePath string, err *error) {
		defer wg.Done()
		*err = os.Rename(oldFilePath, newFilePath)
	}
	var fileInfo os.FileInfo
	var lookUpErr, renameErr error

	// Parallel lookup and rename of a file.
	filePath := path.Join(tds.testDir, tds.explicitDir1Name, tds.file1InExplicitDir1Name)
	newFilePath := path.Join(tds.testDir, "newFile.txt")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go lookUpFileStat(&wg, filePath, &fileInfo, &lookUpErr)
	go renameFunc(&wg, filePath, newFilePath, &renameErr)

	wg.Wait()

	assert.NoError(t, renameErr)
	newFileInfo, err := os.Stat(newFilePath)
	assert.NoError(t, err)
	assert.Contains(t, newFileInfo.Name(), "newFile.txt")
	assert.False(t, newFileInfo.IsDir())
	assert.Equal(t, int64(5), newFileInfo.Size())
	// Assert either file is renamed first or looked up first
	if lookUpErr == nil {
		assert.NotNil(t, fileInfo, "fileInfo should not be nil when lookUpErr is nil")
		assert.Equal(t, int64(5), fileInfo.Size())
		assert.Contains(t, fileInfo.Name(), tds.file1InExplicitDir1Name)
		assert.False(t, fileInfo.IsDir(), "The created path should not be a directory")
	} else {
		assert.True(t, os.IsNotExist(lookUpErr))
	}
}

func TestParallelLookUpAndMkdirSameDir(t *testing.T) {
	// Create directory structure for testing.
	tds := createDirStructure(t)
	defer deleteDirStructure(tds)
	mkdirFunc := func(wg *sync.WaitGroup, dirPath string, err *error) {
		defer wg.Done()
		*err = os.Mkdir(dirPath, setup.DirPermission_0755)
	}

	var statInfo os.FileInfo
	var lookUpErr, mkdirErr error

	dirPath := path.Join(tds.testDir, "newDir")
	var wg sync.WaitGroup
	wg.Add(2)

	go lookUpFileStat(&wg, dirPath, &statInfo, &lookUpErr)
	go mkdirFunc(&wg, dirPath, &mkdirErr)
	wg.Wait()

	// Assert either directory is created first or looked up first
	assert.NoError(t, mkdirErr, "mkdirFunc should not fail")

	if lookUpErr == nil {
		assert.NotNil(t, statInfo, "statInfo should not be nil when lookUpErr is nil")
		assert.Contains(t, statInfo.Name(), "newDir")
		assert.True(t, statInfo.IsDir())
	} else {
		assert.True(t, os.IsNotExist(lookUpErr), "lookUpErr should indicate directory does not exist")
		dirStatInfo, err := os.Stat(dirPath)
		assert.NoError(t, err, "os.Stat should succeed after directory creation")
		assert.True(t, dirStatInfo.IsDir(), "The created path should be a directory")
	}
}
