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
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
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
func createDirectoryStructureForParallelDiropsTest(t *testing.T) {
	_ = setup.SetupTestDirectory(DirForOperationTests)

	ctx := context.Background()
	storageClient, err := client.CreateStorageClient(ctx)
	if err != nil {
		t.Fatalf("failed while creting client for parallel dirops test client.CreateStorageClient: %v", err)
	}

	// Create explicitDir1 structure
	explicitDir1 := path.Join(DirForOperationTests, "explicitDir1")
	err = client.CreateObjectOnGCS(ctx, storageClient, explicitDir1+"/", "")
	if err != nil {
		t.Fatalf("error while creating explicitdir1 : %v", err)
	}
	filePath1 := path.Join(explicitDir1, "file1.txt")
	err = client.CreateObjectOnGCS(ctx, storageClient, filePath1, "12345")
	if err != nil {
		t.Fatalf("error while creating explicitdir1/file1.txt : %v", err)
	}
	filePath2 := path.Join(explicitDir1, "file2.txt")
	err = client.CreateObjectOnGCS(ctx, storageClient, filePath2, "6789101112")
	if err != nil {
		t.Fatalf("error while creating explicitdir1/file2.txt : %v", err)
	}

	// Create explicitDir2 structure
	explicitDir2 := path.Join(DirForOperationTests, "explicitDir2")
	err = client.CreateObjectOnGCS(ctx, storageClient, explicitDir2+"/", "")
	if err != nil {
		t.Fatalf("error while creating explicitdir2: %v", err)
	}
	filePath1 = path.Join(explicitDir2, "file1.txt")
	err = client.CreateObjectOnGCS(ctx, storageClient, filePath1, "-1234556789")
	if err != nil {
		t.Fatalf("error while creating explicitdir2/file1.txt : %v", err)
	}

	filePath1 = path.Join(DirForOperationTests, "file1.txt")
	err = client.CreateObjectOnGCS(ctx, storageClient, filePath1, "abcdef")
	if err != nil {
		t.Fatalf("error while creating file1.txt: %v", err)
	}
	filePath2 = path.Join(DirForOperationTests, "file2.txt")
	err = client.CreateObjectOnGCS(ctx, storageClient, filePath2, "xyz")
	if err != nil {
		t.Fatalf("error while creating file2.txt: %v", err)
	}
}

func TestParallelReadDirAndMkdirInsideSameDir(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	// Create directory structure for testing.
	createDirectoryStructureForParallelDiropsTest(t)
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
		err := os.Mkdir(dirPath, 0600)
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
	assert.Nil(t, readDirErr)
	assert.Nil(t, mkdirErr)
	dirStatInfo, err := os.Stat(newDirPath)
	assert.Nil(t, err)
	assert.True(t, dirStatInfo.IsDir())
	// List should happen either before or after creation of newDir.
	assert.GreaterOrEqual(t, len(dirEntries), 8)
	assert.LessOrEqual(t, len(dirEntries), 9)
	if len(dirEntries) == 9 {
		assert.Contains(t, dirEntries[8].Name(), "newDir")
	}
}

func TestParallelLookUpAndDeleteSameFile(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	// Create directory structure for testing.
	createDirectoryStructureForParallelDiropsTest(t)
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

	// Assert either file is looked up first or deleted first
	assert.Nil(t, deleteErr)
	_, err := os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))
	if lookUpErr == nil {
		assert.Equal(t, int64(5), fileInfo.Size())
		assert.Contains(t, fileInfo.Name(), "file1.txt")
		assert.False(t, fileInfo.IsDir())
	} else {
		assert.True(t, os.IsNotExist(lookUpErr))
	}
}

func TestParallelLookUpAndRenameSameFile(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	// Create directory structure for testing.
	createDirectoryStructureForParallelDiropsTest(t)
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

	// Assert either file is renamed first or looked up first
	assert.Nil(t, renameErr)
	newFileInfo, err := os.Stat(newFilePath)
	assert.Nil(t, err)
	assert.Contains(t, newFileInfo.Name(), "newFile.txt")
	assert.False(t, newFileInfo.IsDir())
	assert.Equal(t, int64(5), newFileInfo.Size())
	if lookUpErr == nil {
		assert.Equal(t, int64(5), fileInfo.Size())
		assert.Contains(t, fileInfo.Name(), "file1.txt")
		assert.False(t, fileInfo.IsDir())
	} else {
		assert.True(t, os.IsNotExist(lookUpErr))
	}
}

func TestParallelLookUpAndMkdirSameDir(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	// Create directory structure for testing.
	createDirectoryStructureForParallelDiropsTest(t)
	lookUpFunc := func(wg *sync.WaitGroup, dirPath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(dirPath)
		return fileInfo, err
	}
	mkdirFunc := func(wg *sync.WaitGroup, dirPath string) error {
		defer wg.Done()
		err := os.Mkdir(dirPath, 0600)
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

	// Assert either directory is created first or looked up first
	assert.Nil(t, mkdirErr)
	dirStatInfo, err := os.Stat(dirPath)
	assert.Nil(t, err)
	assert.True(t, dirStatInfo.IsDir())
	if lookUpErr == nil {
		assert.Contains(t, statInfo.Name(), "newDir")
		assert.True(t, statInfo.IsDir())
	} else {
		assert.True(t, os.IsNotExist(lookUpErr))
	}
}
