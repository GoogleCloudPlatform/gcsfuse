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

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

// createDirectoryStructureForParallelDiropsTest creates the following files and
// directory structure.
// bucket
//
//			file1.txt
//			file2.txt
//			explicitDir1
//					file1.txt
//					file2.txt
//	    explicitDir2
//					file1.txt
//
// Also returns the path to test directory.
func createDirectoryStructureForParallelDiropsTest(t *testing.T) string {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	setup.CleanUpDir(testDir)

	// Create explicitDir1 structure
	explicitDir1 := path.Join(testDir, "explicitDir1")
	operations.CreateDirectory(explicitDir1, t)
	filePath1 := path.Join(explicitDir1, "file1.txt")
	operations.CreateFileOfSize(5, filePath1, t)
	filePath2 := path.Join(explicitDir1, "file2.txt")
	operations.CreateFileOfSize(10, filePath2, t)

	// Create explicitDir2 structure
	explicitDir2 := path.Join(testDir, "explicitDir2")
	operations.CreateDirectory(explicitDir2, t)
	filePath1 = path.Join(explicitDir2, "file1.txt")
	operations.CreateFileOfSize(11, filePath1, t)

	filePath1 = path.Join(testDir, "file1.txt")
	operations.CreateFileOfSize(5, filePath1, t)
	filePath2 = path.Join(testDir, "file2.txt")
	operations.CreateFileOfSize(3, filePath2, t)

	return testDir
}

func TestParallelLookUpsForSameFile(t *testing.T) {
	// Create directory structure for testing.
	testDir := createDirectoryStructureForParallelDiropsTest(t)
	lookUpFunc := func(wg *sync.WaitGroup, filePath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(filePath)
		return fileInfo, err
	}
	var stat1, stat2 os.FileInfo
	var err1, err2 error

	// Parallel lookups of file just under mount.
	filePath := path.Join(testDir, "file1.txt")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		stat1, err1 = lookUpFunc(&wg, filePath)
	}()
	go func() {
		stat2, err2 = lookUpFunc(&wg, filePath)
	}()
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, int64(5), stat1.Size())
	assert.Equal(t, int64(5), stat2.Size())
	assert.Contains(t, filePath, stat1.Name())
	assert.Contains(t, filePath, stat2.Name())

	// Parallel lookups of file under a directory in mount.
	filePath = path.Join(testDir, "explicitDir1/file2.txt")
	wg.Add(2)
	go func() {
		stat1, err1 = lookUpFunc(&wg, filePath)
	}()
	go func() {
		stat2, err2 = lookUpFunc(&wg, filePath)
	}()
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
	testDir := createDirectoryStructureForParallelDiropsTest(t)
	readDirFunc := func(wg *sync.WaitGroup, dirPath string) ([]os.DirEntry, error) {
		defer wg.Done()
		dirEntries, err := os.ReadDir(dirPath)
		return dirEntries, err
	}
	var dirEntries1, dirEntries2 []os.DirEntry
	var err1, err2 error

	// Parallel readDirs of explicit dir under mount.
	dirPath := path.Join(testDir, "explicitDir1")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		dirEntries1, err1 = readDirFunc(&wg, dirPath)
	}()
	go func() {
		dirEntries2, err2 = readDirFunc(&wg, dirPath)
	}()
	wg.Wait()

	// Assert both readDirs passed and give correct information
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 2, len(dirEntries1))
	assert.Equal(t, 2, len(dirEntries2))
	assert.Contains(t, "file1.txt", dirEntries1[0].Name())
	assert.Contains(t, "file2.txt", dirEntries1[1].Name())
	assert.Contains(t, "file1.txt", dirEntries2[0].Name())
	assert.Contains(t, "file2.txt", dirEntries2[1].Name())

	// Parallel readDirs of a directory and its parent directory.
	dirPath = path.Join(testDir, "explicitDir1")
	parentDirPath := testDir
	wg = sync.WaitGroup{}
	wg.Add(2)
	go func() {
		dirEntries1, err1 = readDirFunc(&wg, dirPath)
	}()
	go func() {
		dirEntries2, err2 = readDirFunc(&wg, parentDirPath)
	}()
	wg.Wait()

	// Assert both readDirs passed and give correct information
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 2, len(dirEntries1))
	assert.Equal(t, 4, len(dirEntries2))
	assert.Contains(t, "file1.txt", dirEntries1[0].Name())
	assert.Contains(t, "file2.txt", dirEntries1[1].Name())
	assert.Contains(t, "explicitDir1", dirEntries2[0].Name())
	assert.Contains(t, "explicitDir2", dirEntries2[1].Name())
	assert.Contains(t, "file1.txt", dirEntries2[2].Name())
	assert.Contains(t, "file2.txt", dirEntries2[3].Name())
}

func TestParallelLookUpAndDeleteSameDir(t *testing.T) {
	// Create directory structure for testing.
	testDir := createDirectoryStructureForParallelDiropsTest(t)
	lookUpFunc := func(wg *sync.WaitGroup, dirPath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(dirPath)
		return fileInfo, err
	}
	deleteFunc := func(wg *sync.WaitGroup, dirPath string) error {
		defer wg.Done()
		err := os.RemoveAll(dirPath)
		return err
	}
	var statInfo os.FileInfo
	var lookUpErr, deleteErr error

	// Parallel lookup and deletion of explicit dir under mount.
	dirPath := path.Join(testDir, "explicitDir1")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		statInfo, lookUpErr = lookUpFunc(&wg, dirPath)
	}()
	go func() {
		deleteErr = deleteFunc(&wg, dirPath)
	}()
	wg.Wait()

	assert.NoError(t, deleteErr)
	_, err := os.Stat(dirPath)
	assert.True(t, os.IsNotExist(err))
	// Assert either dir is looked up first or deleted first
	if lookUpErr == nil {
		assert.Contains(t, statInfo.Name(), "explicitDir1")
		assert.True(t, statInfo.IsDir())
	} else {
		assert.True(t, os.IsNotExist(lookUpErr))
	}
}

func TestParallelLookUpsForDifferentFiles(t *testing.T) {
	// Create directory structure for testing.
	testDir := createDirectoryStructureForParallelDiropsTest(t)
	lookUpFunc := func(wg *sync.WaitGroup, filePath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(filePath)
		return fileInfo, err
	}
	var stat1, stat2 os.FileInfo
	var err1, err2 error

	// Parallel lookups of two files just under mount.
	filePath1 := path.Join(testDir, "file1.txt")
	filePath2 := path.Join(testDir, "file2.txt")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		stat1, err1 = lookUpFunc(&wg, filePath1)
	}()
	go func() {
		stat2, err2 = lookUpFunc(&wg, filePath2)
	}()
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, int64(5), stat1.Size())
	assert.Equal(t, int64(3), stat2.Size())
	assert.Contains(t, filePath1, stat1.Name())
	assert.Contains(t, filePath2, stat2.Name())

	// Parallel lookups of two files under a directory in mount.
	filePath1 = path.Join(testDir, "explicitDir1", "file1.txt")
	filePath2 = path.Join(testDir, "explicitDir1", "file2.txt")
	wg = sync.WaitGroup{}
	wg.Add(2)
	go func() {
		stat1, err1 = lookUpFunc(&wg, filePath1)
	}()
	go func() {
		stat2, err2 = lookUpFunc(&wg, filePath2)
	}()
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
	testDir := createDirectoryStructureForParallelDiropsTest(t)
	readDirFunc := func(wg *sync.WaitGroup, dirPath string) ([]os.DirEntry, error) {
		defer wg.Done()
		var dirEntries []os.DirEntry
		err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			dirEntries = append(dirEntries, d)
			return nil
		})
		return dirEntries, err
	}
	mkdirFunc := func(wg *sync.WaitGroup, dirPath string) error {
		defer wg.Done()
		err := os.Mkdir(dirPath, setup.DirPermission_0755)
		return err
	}
	var dirEntries []os.DirEntry
	var readDirErr, mkdirErr error

	// Parallel readDirs and mkdir inside the same directory.
	newDirPath := path.Join(testDir, "newDir")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		dirEntries, readDirErr = readDirFunc(&wg, testDir)
	}()
	go func() {
		mkdirErr = mkdirFunc(&wg, newDirPath)
	}()
	wg.Wait()

	// Assert both listing and mkdir succeeded
	assert.NoError(t, readDirErr)
	assert.NoError(t, mkdirErr)
	dirStatInfo, err := os.Stat(newDirPath)
	assert.NoError(t, err)
	assert.True(t, dirStatInfo.IsDir())
	// List should happen either before or after creation of newDir.
	assert.GreaterOrEqual(t, len(dirEntries), 8)
	assert.LessOrEqual(t, len(dirEntries), 9)
	if len(dirEntries) == 9 {
		assert.Contains(t, dirEntries[8].Name(), "newDir")
	}
}

func TestParallelLookUpAndDeleteSameFile(t *testing.T) {
	// Create directory structure for testing.
	testDir := createDirectoryStructureForParallelDiropsTest(t)
	lookUpFunc := func(wg *sync.WaitGroup, filePath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(filePath)
		return fileInfo, err
	}
	deleteFileFunc := func(wg *sync.WaitGroup, filePath string) error {
		defer wg.Done()
		err := os.Remove(filePath)
		return err
	}
	var fileInfo os.FileInfo
	var lookUpErr, deleteErr error

	// Parallel lookup and deletion of a file.
	filePath := path.Join(testDir, "explicitDir1", "file1.txt")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		fileInfo, lookUpErr = lookUpFunc(&wg, filePath)
	}()
	go func() {
		deleteErr = deleteFileFunc(&wg, filePath)
	}()
	wg.Wait()

	assert.NoError(t, deleteErr)
	_, err := os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))
	// Assert either file is looked up first or deleted first
	if lookUpErr == nil {
		assert.Equal(t, int64(5), fileInfo.Size())
		assert.Contains(t, fileInfo.Name(), "file1.txt")
		assert.False(t, fileInfo.IsDir())
	} else {
		assert.True(t, os.IsNotExist(lookUpErr))
	}
}

func TestParallelLookUpAndRenameSameFile(t *testing.T) {
	// Create directory structure for testing.
	testDir := createDirectoryStructureForParallelDiropsTest(t)
	lookUpFunc := func(wg *sync.WaitGroup, filePath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(filePath)
		return fileInfo, err
	}
	renameFunc := func(wg *sync.WaitGroup, oldFilePath string, newFilePath string) error {
		defer wg.Done()
		err := os.Rename(oldFilePath, newFilePath)
		return err
	}
	var fileInfo os.FileInfo
	var lookUpErr, renameErr error

	// Parallel lookup and rename of a file.
	filePath := path.Join(testDir, "explicitDir1", "file1.txt")
	newFilePath := path.Join(testDir, "newFile.txt")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		fileInfo, lookUpErr = lookUpFunc(&wg, filePath)
	}()
	go func() {
		renameErr = renameFunc(&wg, filePath, newFilePath)
	}()
	wg.Wait()

	assert.NoError(t, renameErr)
	newFileInfo, err := os.Stat(newFilePath)
	assert.NoError(t, err)
	assert.Contains(t, newFileInfo.Name(), "newFile.txt")
	assert.False(t, newFileInfo.IsDir())
	assert.Equal(t, int64(5), newFileInfo.Size())
	// Assert either file is renamed first or looked up first
	if lookUpErr == nil {
		assert.Equal(t, int64(5), fileInfo.Size())
		assert.Contains(t, fileInfo.Name(), "file1.txt")
		assert.False(t, fileInfo.IsDir())
	} else {
		assert.True(t, os.IsNotExist(lookUpErr))
	}
}

func TestParallelLookUpAndMkdirSameDir(t *testing.T) {
	// Create directory structure for testing.
	testDir := createDirectoryStructureForParallelDiropsTest(t)
	lookUpFunc := func(wg *sync.WaitGroup, dirPath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(dirPath)
		return fileInfo, err
	}
	mkdirFunc := func(wg *sync.WaitGroup, dirPath string) error {
		defer wg.Done()
		err := os.Mkdir(dirPath, setup.DirPermission_0755)
		return err
	}
	var statInfo os.FileInfo
	var lookUpErr, mkdirErr error

	// Parallel lookup and mkdir of a new directory.
	dirPath := path.Join(testDir, "newDir")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		statInfo, lookUpErr = lookUpFunc(&wg, dirPath)
	}()
	go func() {
		mkdirErr = mkdirFunc(&wg, dirPath)
	}()
	wg.Wait()

	assert.NoError(t, mkdirErr)
	// Assert either directory is created first or looked up first
	if lookUpErr == nil {
		assert.Contains(t, statInfo.Name(), "newDir")
		assert.True(t, statInfo.IsDir())
	} else {
		assert.True(t, os.IsNotExist(lookUpErr))
		dirStatInfo, err := os.Stat(dirPath)
		assert.NoError(t, err)
		assert.True(t, dirStatInfo.IsDir())
	}
}
