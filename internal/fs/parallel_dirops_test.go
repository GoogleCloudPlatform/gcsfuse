// Copyright 2015 Google Inc. All Rights Reserved.
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

// A collection of tests for a file system where parallel dirops are allowed.
// Dirops refers to readdir and lookup operations.

package fs_test

import (
	"os"
	"path"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ParallelDiropsTest struct {
	suite.Suite
	fsTest
}

type ParallelDiropsWithoutCachesTest struct {
	ParallelDiropsTest
}

// createFilesAndDirStructureInBucket creates the following files and directory
// structure.
// bucket
//
//			file1.txt
//			file2.txt
//			explicitDir1
//					file1.txt
//					file2.txt
//	    implicitDir1
//					file1.txt
func (t *ParallelDiropsTest) createFilesAndDirStructureInBucket() {
	assert.Nil(
		t.T(),
		t.createObjects(
			map[string]string{
				"file1.txt":              "abcdef",
				"file2.txt":              "xyz",
				"explicitDir1/":          "",
				"explicitDir1/file1.txt": "12345",
				"explicitDir1/file2.txt": "6789101112",
				"implicitDir1/file1.txt": "-1234556789",
			}))
}

func (t *ParallelDiropsTest) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileSystemConfig: config.FileSystemConfig{
			DisableParallelDirops: false,
		}}
	t.serverCfg.RenameDirLimit = 10
	t.fsTest.SetUpTestSuite()
}

func (t *ParallelDiropsWithoutCachesTest) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileSystemConfig: config.FileSystemConfig{
			DisableParallelDirops: false,
		}}
	t.serverCfg.RenameDirLimit = 10
	t.serverCfg.DirTypeCacheTTL = 0
	t.serverCfg.InodeAttributeCacheTTL = 0
	t.fsTest.SetUpTestSuite()
}

func (t *ParallelDiropsTest) SetupTest() {
	t.createFilesAndDirStructureInBucket()
}

func (t *ParallelDiropsTest) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *ParallelDiropsTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *ParallelDiropsTest) TestParallelLookUpsForSameFile() {
	lookUpFunc := func(wg *sync.WaitGroup, filePath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(filePath)
		return fileInfo, err
	}
	var stat1, stat2 os.FileInfo
	var err1, err2 error

	// Parallel lookups of file just under mount.
	filePath := path.Join(mntDir, "file1.txt")
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
	assert.NoError(t.T(), err1)
	assert.NoError(t.T(), err2)
	assert.Equal(t.T(), int64(6), stat1.Size())
	assert.Equal(t.T(), int64(6), stat2.Size())
	assert.Contains(t.T(), filePath, stat1.Name())
	assert.Contains(t.T(), filePath, stat2.Name())

	// Parallel lookups of file under a directory in mount.
	filePath = path.Join(mntDir, "explicitDir1/file2.txt")
	wg.Add(2)
	go func() {
		stat1, err1 = lookUpFunc(&wg, filePath)
	}()
	go func() {
		stat2, err2 = lookUpFunc(&wg, filePath)
	}()
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(t.T(), err1)
	assert.NoError(t.T(), err2)
	assert.Equal(t.T(), int64(10), stat1.Size())
	assert.Equal(t.T(), int64(10), stat2.Size())
	assert.Contains(t.T(), filePath, stat1.Name())
	assert.Contains(t.T(), filePath, stat2.Name())
}

func (t *ParallelDiropsTest) TestParallelLookUpsForSameDir() {
	lookUpFunc := func(wg *sync.WaitGroup, dirPath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(dirPath)
		return fileInfo, err
	}
	var stat1, stat2 os.FileInfo
	var err1, err2 error

	// Parallel lookups of explicit dir under mount.
	dirPath := path.Join(mntDir, "explicitDir1")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		stat1, err1 = lookUpFunc(&wg, dirPath)
	}()
	go func() {
		stat2, err2 = lookUpFunc(&wg, dirPath)
	}()
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(t.T(), err1)
	assert.NoError(t.T(), err2)
	assert.True(t.T(), stat1.IsDir())
	assert.True(t.T(), stat2.IsDir())
	assert.Contains(t.T(), dirPath, stat1.Name())
	assert.Contains(t.T(), dirPath, stat2.Name())

	// Parallel lookups of implicit dir in mount.
	dirPath = path.Join(mntDir, "implicitDir1/")
	wg.Add(2)
	go func() {
		stat1, err1 = lookUpFunc(&wg, dirPath)
	}()
	go func() {
		stat2, err2 = lookUpFunc(&wg, dirPath)
	}()
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(t.T(), err1)
	assert.NoError(t.T(), err2)
	assert.True(t.T(), stat1.IsDir())
	assert.True(t.T(), stat2.IsDir())
	assert.Contains(t.T(), dirPath, stat1.Name())
	assert.Contains(t.T(), dirPath, stat2.Name())
}

func (t *ParallelDiropsTest) TestParallelReadDirsForSameDir() {
	readDirFunc := func(wg *sync.WaitGroup, dirPath string) ([]os.DirEntry, error) {
		defer wg.Done()
		dirEntries, err := os.ReadDir(dirPath)
		return dirEntries, err
	}
	var dirEntries1, dirEntries2 []os.DirEntry
	var err1, err2 error

	// Parallel readDirs of explicit dir under mount.
	dirPath := path.Join(mntDir, "explicitDir1")
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
	assert.NoError(t.T(), err1)
	assert.NoError(t.T(), err2)
	assert.Contains(t.T(), dirEntries1[0].Name(), "file1.txt")
	assert.Contains(t.T(), dirEntries1[1].Name(), "file2.txt")
	assert.Contains(t.T(), dirEntries2[0].Name(), "file1.txt")
	assert.Contains(t.T(), dirEntries2[1].Name(), "file2.txt")

	// Parallel readDirs of implicit dir under mount.
	dirPath = path.Join(mntDir, "implicitDir1")
	wg = sync.WaitGroup{}
	wg.Add(2)
	go func() {
		dirEntries1, err1 = readDirFunc(&wg, dirPath)
	}()
	go func() {
		dirEntries2, err2 = readDirFunc(&wg, dirPath)
	}()
	wg.Wait()

	// Assert both readDirs passed and give correct information
	assert.NoError(t.T(), err1)
	assert.NoError(t.T(), err2)
	assert.Contains(t.T(), dirEntries1[0].Name(), "file1.txt")
	assert.Contains(t.T(), dirEntries2[0].Name(), "file1.txt")

	// Parallel readDirs of a directory and its parent directory.
	dirPath = path.Join(mntDir, "explicitDir1")
	parentDirPath := mntDir
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
	assert.NoError(t.T(), err1)
	assert.NoError(t.T(), err2)
	assert.Contains(t.T(), dirEntries1[0].Name(), "file1.txt")
	assert.Contains(t.T(), dirEntries1[1].Name(), "file2.txt")
	assert.Contains(t.T(), dirEntries2[0].Name(), "explicitDir1")
	assert.Contains(t.T(), dirEntries2[1].Name(), "file1.txt")
	assert.Contains(t.T(), dirEntries2[2].Name(), "file2.txt")
	assert.Contains(t.T(), dirEntries2[3].Name(), "implicitDir1")

}

func (t *ParallelDiropsTest) TestParallelReadDirAndMkdirSameDir() {
	readDirFunc := func(wg *sync.WaitGroup, dirPath string) ([]os.DirEntry, error) {
		defer wg.Done()
		dirEntries, err := os.ReadDir(dirPath)
		return dirEntries, err
	}
	mkdirFunc := func(wg *sync.WaitGroup, dirPath string) error {
		defer wg.Done()
		err := os.Mkdir(dirPath, 0600)
		return err
	}
	var dirEntries []os.DirEntry
	var readDirErr, mkdirErr error

	// Parallel readDirs and mkdir of a new directory.
	dirPath := path.Join(mntDir, "newDir")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		dirEntries, readDirErr = readDirFunc(&wg, dirPath)
	}()
	go func() {
		mkdirErr = mkdirFunc(&wg, dirPath)
	}()
	wg.Wait()

	// Assert either directory is created first or listed first
	assert.NoError(t.T(), mkdirErr)
	dirStatInfo, err := os.Stat(dirPath)
	assert.NoError(t.T(), err)
	assert.True(t.T(), dirStatInfo.IsDir())
	if readDirErr == nil {
		assert.Equal(t.T(), 0, len(dirEntries))
	} else {
		assert.True(t.T(), os.IsNotExist(readDirErr))
	}

	// Parallel readDirs and mkdir of a new dir inside already present dir.
	dirPath = path.Join(mntDir, "explicitDir1", "newDir")
	wg = sync.WaitGroup{}
	wg.Add(2)
	go func() {
		dirEntries, readDirErr = readDirFunc(&wg, dirPath)
	}()
	go func() {
		mkdirErr = mkdirFunc(&wg, dirPath)
	}()
	wg.Wait()

	// Assert either directory is created first or listed first
	assert.NoError(t.T(), mkdirErr)
	dirStatInfo, err = os.Stat(dirPath)
	assert.NoError(t.T(), err)
	assert.True(t.T(), dirStatInfo.IsDir())
	if readDirErr == nil {
		assert.Equal(t.T(), 0, len(dirEntries))
	} else {
		assert.True(t.T(), os.IsNotExist(readDirErr))
	}
}

func (t *ParallelDiropsTest) TestParallelLookUpAndCreateSameFile() {
	lookUpFunc := func(wg *sync.WaitGroup, filePath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(filePath)
		return fileInfo, err
	}
	createFileFunc := func(wg *sync.WaitGroup, filePath string) (*os.File, error) {
		defer wg.Done()
		file, err := os.Create(filePath)
		return file, err
	}
	var fileInfo os.FileInfo
	var lookUpErr, createErr error
	var file *os.File

	// Parallel lookup and creation of a file.
	filePath := path.Join(mntDir, "explicitDir1", "file3.txt")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		fileInfo, lookUpErr = lookUpFunc(&wg, filePath)
	}()
	go func() {
		file, createErr = createFileFunc(&wg, filePath)
	}()
	wg.Wait()

	// Assert either file is created first or looked up first
	assert.NoError(t.T(), createErr)
	assert.Contains(t.T(), file.Name(), "file3.txt")
	if lookUpErr == nil {
		assert.Equal(t.T(), int64(0), fileInfo.Size())
		assert.Contains(t.T(), fileInfo.Name(), "file3.txt")
		assert.False(t.T(), fileInfo.IsDir())
	} else {
		assert.True(t.T(), os.IsNotExist(lookUpErr))
	}
}

func (t *ParallelDiropsTest) TestParallelLookUpAndDeleteSameFile() {
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
	filePath := path.Join(mntDir, "explicitDir1", "file1.txt")
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
	assert.NoError(t.T(), deleteErr)
	_, err := os.Stat(filePath)
	assert.True(t.T(), os.IsNotExist(err))
	if lookUpErr == nil {
		assert.Equal(t.T(), int64(5), fileInfo.Size())
		assert.Contains(t.T(), fileInfo.Name(), "file1.txt")
		assert.False(t.T(), fileInfo.IsDir())
	} else {
		assert.True(t.T(), os.IsNotExist(lookUpErr))
	}
}

func (t *ParallelDiropsTest) TestParallelLookUpAndRenameSameFile() {
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
	filePath := path.Join(mntDir, "explicitDir1", "file1.txt")
	newFilePath := path.Join(mntDir, "newFile.txt")
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
	assert.NoError(t.T(), renameErr)
	newFileInfo, err := os.Stat(newFilePath)
	assert.NoError(t.T(), err)
	assert.Contains(t.T(), newFileInfo.Name(), "newFile.txt")
	assert.False(t.T(), newFileInfo.IsDir())
	assert.Equal(t.T(), int64(5), newFileInfo.Size())
	if lookUpErr == nil {
		assert.Equal(t.T(), int64(5), fileInfo.Size())
		assert.Contains(t.T(), fileInfo.Name(), "file1.txt")
		assert.False(t.T(), fileInfo.IsDir())
	} else {
		assert.True(t.T(), os.IsNotExist(lookUpErr))
	}
}

func (t *ParallelDiropsTest) TestParallelLookUpAndDeleteSameDir() {
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

	// Parallel lookups of explicit dir under mount.
	dirPath := path.Join(mntDir, "explicitDir1")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		statInfo, lookUpErr = lookUpFunc(&wg, dirPath)
	}()
	go func() {
		deleteErr = deleteFunc(&wg, dirPath)
	}()
	wg.Wait()

	// Assert either dir is created first or deleted first
	assert.NoError(t.T(), deleteErr)
	_, err := os.Stat(dirPath)
	assert.True(t.T(), os.IsNotExist(err))
	if lookUpErr == nil {
		assert.Contains(t.T(), statInfo.Name(), "explicitDir1")
		assert.True(t.T(), statInfo.IsDir())
	} else {
		assert.True(t.T(), os.IsNotExist(lookUpErr))
	}
}

func (t *ParallelDiropsTest) TestParallelLookUpAndMkdirSameDir() {
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
	dirPath := path.Join(mntDir, "newDir")
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
	assert.NoError(t.T(), mkdirErr)
	dirStatInfo, err := os.Stat(dirPath)
	assert.NoError(t.T(), err)
	assert.True(t.T(), dirStatInfo.IsDir())
	if lookUpErr == nil {
		assert.Contains(t.T(), statInfo.Name(), "newDir")
		assert.True(t.T(), statInfo.IsDir())
	} else {
		assert.True(t.T(), os.IsNotExist(lookUpErr))
	}
}

func (t *ParallelDiropsTest) TestParallelLookUpAndRenameSameDir() {
	lookUpFunc := func(wg *sync.WaitGroup, dirPath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(dirPath)
		return fileInfo, err
	}
	renameFunc := func(wg *sync.WaitGroup, oldPath string, newPath string) error {
		defer wg.Done()
		err := os.Rename(oldPath, newPath)
		return err
	}
	var statInfo os.FileInfo
	var lookUpErr, renameErr error

	// Parallel lookup and rename of a directory.
	dirPath := path.Join(mntDir, "explicitDir1")
	newDirPath := path.Join(mntDir, "newDir")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		statInfo, lookUpErr = lookUpFunc(&wg, dirPath)
	}()
	go func() {
		renameErr = renameFunc(&wg, dirPath, newDirPath)
	}()
	wg.Wait()

	// Assert either directory is renamed first or looked up first
	assert.NoError(t.T(), renameErr)
	dirStatInfo, err := os.Stat(newDirPath)
	assert.NoError(t.T(), err)
	assert.True(t.T(), dirStatInfo.IsDir())
	if lookUpErr == nil {
		assert.Contains(t.T(), statInfo.Name(), "explicitDir1")
		assert.True(t.T(), statInfo.IsDir())
	} else {
		assert.True(t.T(), os.IsNotExist(lookUpErr))
	}
}

func (t *ParallelDiropsTest) TestParallelLookUpsForDifferentFiles() {
	lookUpFunc := func(wg *sync.WaitGroup, filePath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(filePath)
		return fileInfo, err
	}
	var stat1, stat2 os.FileInfo
	var err1, err2 error

	// Parallel lookups of two files just under mount.
	filePath1 := path.Join(mntDir, "file1.txt")
	filePath2 := path.Join(mntDir, "file2.txt")
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
	assert.NoError(t.T(), err1)
	assert.NoError(t.T(), err2)
	assert.Equal(t.T(), int64(6), stat1.Size())
	assert.Equal(t.T(), int64(3), stat2.Size())
	assert.Contains(t.T(), filePath1, stat1.Name())
	assert.Contains(t.T(), filePath2, stat2.Name())

	// Parallel lookups of two files under a directory in mount.
	filePath1 = path.Join(mntDir, "explicitDir1", "file1.txt")
	filePath2 = path.Join(mntDir, "explicitDir1", "file2.txt")
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
	assert.NoError(t.T(), err1)
	assert.NoError(t.T(), err2)
	assert.Equal(t.T(), int64(5), stat1.Size())
	assert.Equal(t.T(), int64(10), stat2.Size())
	assert.Contains(t.T(), filePath1, stat1.Name())
	assert.Contains(t.T(), filePath2, stat2.Name())
}

func (t *ParallelDiropsTest) TestParallelLookUpsForDifferentDirs() {
	lookUpFunc := func(wg *sync.WaitGroup, dirPath string) (os.FileInfo, error) {
		defer wg.Done()
		fileInfo, err := os.Stat(dirPath)
		return fileInfo, err
	}
	var stat1, stat2 os.FileInfo
	var err1, err2 error

	// Parallel lookups of two dirs under mount.
	dirPath1 := path.Join(mntDir, "explicitDir1")
	dirPath2 := path.Join(mntDir, "implicitDir1")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		stat1, err1 = lookUpFunc(&wg, dirPath1)
	}()
	go func() {
		stat2, err2 = lookUpFunc(&wg, dirPath2)
	}()
	wg.Wait()

	// Assert both stats passed and give correct information
	assert.NoError(t.T(), err1)
	assert.NoError(t.T(), err2)
	assert.True(t.T(), stat1.IsDir())
	assert.True(t.T(), stat2.IsDir())
	assert.Contains(t.T(), dirPath1, stat1.Name())
	assert.Contains(t.T(), dirPath2, stat2.Name())
}

func (t *ParallelDiropsTest) TestParallelReadDirsForDifferentDirs() {
	readDirFunc := func(wg *sync.WaitGroup, dirPath string) ([]os.DirEntry, error) {
		defer wg.Done()
		dirEntries, err := os.ReadDir(dirPath)
		return dirEntries, err
	}
	var dirEntries1, dirEntries2 []os.DirEntry
	var err1, err2 error

	// Parallel readDirs of explicit dir under mount.
	dirPath1 := path.Join(mntDir, "explicitDir1")
	dirPath2 := path.Join(mntDir, "implicitDir1")
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		dirEntries1, err1 = readDirFunc(&wg, dirPath1)
	}()
	go func() {
		dirEntries2, err2 = readDirFunc(&wg, dirPath2)
	}()
	wg.Wait()

	// Assert both readDirs passed and give correct information
	assert.NoError(t.T(), err1)
	assert.NoError(t.T(), err2)
	assert.Contains(t.T(), dirEntries1[0].Name(), "file1.txt")
	assert.Contains(t.T(), dirEntries1[1].Name(), "file2.txt")
	assert.Contains(t.T(), dirEntries2[0].Name(), "file1.txt")
}

func TestParallelDiropsTestSuite(t *testing.T) {
	suite.Run(t, new(ParallelDiropsTest))
}

func TestParallelDiropsWithoutCachesTestSuite(t *testing.T) {
	suite.Run(t, new(ParallelDiropsWithoutCachesTest))
}
