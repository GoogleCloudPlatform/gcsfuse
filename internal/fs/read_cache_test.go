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

// A collection of tests for a file system where the file cache is enabled.
package fs_test

import (
	"io"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	. "github.com/jacobsa/ogletest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestReadCacheTestSuite(t *testing.T) {
	suite.Run(t, new(FileCacheTest))
	suite.Run(t, new(FileCacheWithCacheForRangeRead))
	suite.Run(t, new(FileCacheIsDisabledWithCacheDirAndZeroMaxSize))
	suite.Run(t, new(FileCacheDestroyTest))
}

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

const (
	FileCacheSizeInMb     = 10
	DefaultObjectName     = "foo.txt"
	RenamedObjectName     = "bar.txt"
	DefaultObjectSizeInMb = 5

	NestedDefaultObjectName = "dir/foo.txt"
	DefaultDir              = "dir"

	RenamedDir       = "renamed_dir"
	UserTempLocation = "my/temp"
)

var CacheDir = path.Join(os.Getenv("HOME"), "cache-dir")
var FileCacheDir = path.Join(CacheDir, util.FileCache)

// A collection of tests for a file system where the file cache is enabled
// with cache-file-for-range-read set to False.
type FileCacheTest struct {
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *FileCacheTest) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			MaxSizeMB:             FileCacheSizeInMb,
			CacheFileForRangeRead: false,
		},
		CacheDir: config.CacheDir(CacheDir),
	}
	t.fsTest.SetupSuite()
}

func (t *FileCacheTest) TearDownTest() {
	t.fsTest.TearDownTest()
	err := os.RemoveAll(FileCacheDir)
	assert.Nil(t.T(), err)
}

func generateRandomString(length int) string {
	return string(testutil.GenerateRandomBytes(length))
}

func closeFile(t *fsTest, file *os.File) {
	err := file.Close()
	assert.Nil(t.T(), err)
}

func sequentialReadShouldPopulateCache(t *fsTest, cacheDir string) {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(t, file)
	assert.Nil(t.T(), err)

	// reading object with cache enabled should cache the object into file.
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	AssertEq(objectContent, string(buf))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(cacheDir, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(buf, cachedContent))
}

func cacheFilePermissionTest(t *fsTest, fileMode os.FileMode) {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(t, file)
	assert.Nil(t.T(), err)

	// reading object with cache enabled should cache the object into file.
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent, string(buf)))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	stat, err := os.Stat(downloadPath)
	assert.Nil(t.T(), err)
	// confirm file mode is as expected
	AssertEq(fileMode, stat.Mode())
}

func writeShouldNotPopulateCache(t *fsTest) {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, util.DefaultFilePerm)
	defer closeFile(t, file)
	assert.Nil(t.T(), err)

	// writing file with cache enabled should not populate cache.
	buf := []byte(objectContent)
	n, err := file.Write(buf)
	assert.Nil(t.T(), err)
	AssertEq(n, len(objectContent))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	_, err = os.Stat(downloadPath)
	assert.NotNil(t.T(), err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func sequentialToRandomReadShouldPopulateCache(t *fsTest) {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(t, file)
	assert.Nil(t.T(), err)
	// Sequential read
	buf := make([]byte, util.MiB)
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent[:util.MiB], string(buf)))

	// random read
	offsetForRandom := int64(3)
	_, err = file.Seek(offsetForRandom, 0)
	assert.Nil(t.T(), err)
	_, err = file.Read(buf)

	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent[offsetForRandom:offsetForRandom+util.MiB], string(buf)))
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(cachedContent[offsetForRandom:offsetForRandom+util.MiB], buf))
}

func (t *FileCacheTest) TestReadShouldChangeLRU() {
	objectName1 := DefaultObjectName + "1"
	objectContent1 := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objectName2 := DefaultObjectName + "2"
	objectContent2 := generateRandomString((FileCacheSizeInMb - DefaultObjectSizeInMb) * util.MiB)
	objectName3 := DefaultObjectName + "3"
	objectContent3 := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	// Check that file 3 size should be <= min(file size 1, file size 2)
	AssertLe(len(objectContent3), len(objectContent1))
	AssertLe(len(objectContent3), len(objectContent2))
	objects := map[string]string{objectName1: objectContent1, objectName2: objectContent2, objectName3: objectContent3}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	// Open and read files for object 1 & 2, filet 1 should be LRU after that.
	buf := make([]byte, 10)
	fileHandle1, err := os.OpenFile(path.Join(mntDir, objectName1), os.O_RDONLY|syscall.O_DIRECT, 0644)
	defer closeFile(&t.fsTest, fileHandle1)
	assert.Nil(t.T(), err)
	_, err = fileHandle1.ReadAt(buf, 0)
	assert.Nil(t.T(), err)
	AssertEq(string(buf), objectContent1[0:len(buf)])
	fileHandle2, err := os.OpenFile(path.Join(mntDir, objectName2), os.O_RDONLY|syscall.O_DIRECT, 0644)
	defer closeFile(&t.fsTest, fileHandle2)
	assert.Nil(t.T(), err)
	_, err = fileHandle2.ReadAt(buf, 0)
	assert.Nil(t.T(), err)
	AssertEq(string(buf), objectContent2[0:len(buf)])
	// Assert cache files are created.
	objectPath1 := util.GetObjectPath(bucket.Name(), objectName1)
	downloadPath1 := util.GetDownloadPath(FileCacheDir, objectPath1)
	objectPath2 := util.GetObjectPath(bucket.Name(), objectName2)
	downloadPath2 := util.GetDownloadPath(FileCacheDir, objectPath2)
	_, err = os.Stat(downloadPath1)
	assert.Nil(t.T(), err)
	_, err = os.Stat(downloadPath2)
	assert.Nil(t.T(), err)

	// Read file 1, so file 2 becomes LRU and then read file 3. Doing this should
	// evict file 2 and not file 1.
	_, err = fileHandle1.ReadAt(buf, 0)
	assert.Nil(t.T(), err)
	AssertEq(string(buf), objectContent1[0:len(buf)])
	fileHandle3, err := os.OpenFile(path.Join(mntDir, objectName3), os.O_RDONLY|syscall.O_DIRECT, 0644)
	defer closeFile(&t.fsTest, fileHandle3)
	assert.Nil(t.T(), err)
	_, err = fileHandle3.ReadAt(buf, 0)
	assert.Nil(t.T(), err)
	AssertEq(string(buf), objectContent3[0:len(buf)])

	// Cache for file 2 should be evicted.
	_, err = os.Stat(downloadPath2)
	AssertTrue(os.IsNotExist(err))
	// Cache for file 1 shouldn't be evicted.
	_, err = os.Stat(downloadPath1)
	assert.Nil(t.T(), err)
}

func (t *FileCacheTest) TestSequentialReadShouldPopulateCache() {
	sequentialReadShouldPopulateCache(&t.fsTest, FileCacheDir)
}

func (t *FileCacheTest) TestSequentialToRandomReadShouldPopulateCache() {
	sequentialToRandomReadShouldPopulateCache(&t.fsTest)
}

func (t *FileCacheTest) TestCacheFilePermission() {
	cacheFilePermissionTest(&t.fsTest, util.DefaultFilePerm)
}

func (t *FileCacheTest) TestWriteShouldNotPopulateCache() {
	writeShouldNotPopulateCache(&t.fsTest)
}

func (t *FileCacheTest) TestFileSizeGreaterThanCacheSize() {
	objectContent := generateRandomString((FileCacheSizeInMb + 1) * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(&t.fsTest, file)
	assert.Nil(t.T(), err)

	// reading object with size greater than cache size
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent, string(buf)))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	_, err = os.Stat(downloadPath)
	assert.NotNil(t.T(), err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func (t *FileCacheTest) TestEvictionWhenFileCacheIsFull() {
	objectName1 := DefaultObjectName + "1"
	objectContent1 := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objectName2 := DefaultObjectName + "2"
	objectContent2 := generateRandomString((FileCacheSizeInMb - DefaultObjectSizeInMb + 1) * util.MiB)
	objects := map[string]string{objectName1: objectContent1, objectName2: objectContent2}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	// read object 1 which should populate the cache
	filePath1 := path.Join(mntDir, objectName1)
	gotObjectContent1, err := os.ReadFile(filePath1)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent1, string(gotObjectContent1)))
	// verify file is cached
	objectPath1 := util.GetObjectPath(bucket.Name(), objectName1)
	downloadPath1 := util.GetDownloadPath(FileCacheDir, objectPath1)
	cachedContent, err := os.ReadFile(downloadPath1)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent1, string(cachedContent)))

	// read the second file, so first should be evicted.
	filePath2 := path.Join(mntDir, objectName2)
	gotObjectContent2, err := os.ReadFile(filePath2)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent2, string(gotObjectContent2)))

	_, err = os.Stat(downloadPath1)
	assert.NotNil(t.T(), err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func (t *FileCacheTest) TestRandomReadShouldNotPopulateCache() {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(&t.fsTest, file)
	assert.Nil(t.T(), err)
	// randomly read object with cache enabled should not populate cache
	buf := make([]byte, util.MiB)
	_, err = file.Seek(util.MiB, 0)
	assert.Nil(t.T(), err)

	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent[util.MiB:2*util.MiB], string(buf)))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheDir, objectPath)
	// Cache should not be populated
	_, err = os.Stat(downloadPath)
	assert.NotNil(t.T(), err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func (t *FileCacheTest) TestReadWithNewHandleAfterDeletingFileFromCacheShouldFail() {
	objectContent := generateRandomString(util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	assert.Nil(t.T(), err)
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	closeFile(&t.fsTest, file)
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	file, err = os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	assert.Nil(t.T(), err)
	// delete the file in cache
	err = os.Remove(downloadPath)
	assert.Nil(t.T(), err)
	defer closeFile(&t.fsTest, file)
	assert.Nil(t.T(), err)

	// reading again should throw error
	_, err = file.Read(buf)

	assert.NotNil(t.T(), err)
	AssertTrue(strings.Contains(err.Error(), "input/output error"))
}

func (t *FileCacheTest) TestReadWithOldHandleAfterDeletingFileFromCacheShouldNotFail() {
	objectContent := generateRandomString(util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	assert.Nil(t.T(), err)
	defer closeFile(&t.fsTest, file)
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	// delete the file in cache
	err = os.Remove(downloadPath)
	assert.Nil(t.T(), err)
	// Read with old handle.
	_, err = file.Seek(0, 0)
	assert.Nil(t.T(), err)

	_, err = file.Read(buf)

	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(string(buf), objectContent))
}

func (t *FileCacheTest) TestDeletingObjectShouldInvalidateTheCorrespondingCache() {
	objectContent := generateRandomString(util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	assert.Nil(t.T(), err)
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	closeFile(&t.fsTest, file)
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	_, err = os.Stat(downloadPath)
	assert.Nil(t.T(), err)

	// Delete the object.
	err = os.Remove(filePath)
	assert.Nil(t.T(), err)

	_, err = os.Stat(downloadPath)
	assert.NotNil(t.T(), err)
	AssertTrue(os.IsNotExist(err))
}

func (t *FileCacheTest) TestRenamingObjectShouldInvalidateTheCorrespondingCache() {
	objectContent := generateRandomString(util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	assert.Nil(t.T(), err)
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	closeFile(&t.fsTest, file)
	renamedPath := path.Join(mntDir, RenamedObjectName)
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	_, err = os.Stat(downloadPath)
	assert.Nil(t.T(), err)

	// Rename the object.
	err = os.Rename(filePath, renamedPath)
	assert.Nil(t.T(), err)

	_, err = os.Stat(downloadPath)
	assert.NotNil(t.T(), err)
	AssertTrue(os.IsNotExist(err))
}

func (t *FileCacheTest) TestRenamingDirShouldInvalidateTheCacheOfNestedObject() {
	objectContent := generateRandomString(util.MiB)
	objects := map[string]string{NestedDefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, NestedDefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	assert.Nil(t.T(), err)
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	closeFile(&t.fsTest, file)
	dir := path.Join(mntDir, DefaultDir)
	renamedDir := path.Join(mntDir, RenamedDir)
	objectPath := util.GetObjectPath(bucket.Name(), NestedDefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	_, err = os.Stat(downloadPath)
	assert.Nil(t.T(), err)

	// Rename dir.
	err = os.Rename(dir, renamedDir)
	assert.Nil(t.T(), err)

	_, err = os.Stat(downloadPath)
	assert.NotNil(t.T(), err)
	AssertTrue(os.IsNotExist(err))
}

func (t *FileCacheTest) TestConcurrentReadsFromSameFileHandle() {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(&t.fsTest, file)
	assert.Nil(t.T(), err)
	wg := sync.WaitGroup{}
	readFunc := func(offset int64, length int64) {
		defer wg.Done()
		buf := make([]byte, length)
		_, err := file.Seek(offset, 0)
		assert.Nil(t.T(), err)
		_, err = file.Read(buf)
		ExpectTrue(err == nil || err == io.EOF)
		// we can't compare the data as seek is of same file and can be changed by
		// concurrent go routines.
	}
	wg.Add(1)
	// initiate sequential read first
	readFunc(0, util.MiB)

	// read concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go readFunc(int64(i)*util.MiB, util.MiB)
	}
	wg.Wait()

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent, string(cachedContent)))
}

func (t *FileCacheTest) TestFileSizeEqualToFileCacheSize() {
	objectContent := generateRandomString(FileCacheSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)

	// reading object with cache enabled should cache the object into file.
	gotContent, err := os.ReadFile(filePath)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent, string(cachedContent)))
}

func (t *FileCacheTest) TestWriteToFileCachedAndThenReadingItShouldBeCorrect() {
	sequentialToRandomReadShouldPopulateCache(&t.fsTest)
	// write content to file that is cached.
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, util.DefaultFilePerm)
	defer closeFile(&t.fsTest, file)
	assert.Nil(t.T(), err)
	// Write to file after reading
	buf := []byte(objectContent)
	n, err := file.Write(buf)
	assert.Nil(t.T(), err)
	AssertEq(n, len(objectContent))

	// read the file again
	gotContent, err := os.ReadFile(filePath)
	assert.Nil(t.T(), err)

	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))
	// the file in cache should not be updated because the file that is being
	// cached is still dirty.
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	assert.Nil(t.T(), err)
	AssertFalse(reflect.DeepEqual(objectContent, string(cachedContent)))
}

func (t *FileCacheTest) TestSyncToFileCachedAndThenReadingItShouldBeCorrect() {
	sequentialToRandomReadShouldPopulateCache(&t.fsTest)
	// write and sync content to file that is cached.
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, util.DefaultFilePerm)
	defer closeFile(&t.fsTest, file)
	assert.Nil(t.T(), err)
	// Write and sync to file after reading
	buf := []byte(objectContent)
	n, err := file.Write(buf)
	assert.Nil(t.T(), err)
	AssertEq(n, len(objectContent))
	err = file.Sync()
	assert.Nil(t.T(), err)

	// read the file again
	gotContent, err := os.ReadFile(filePath)
	assert.Nil(t.T(), err)

	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent, string(cachedContent)))
}

// A collection of tests for a file system where the file cache is enabled
// with cache-file-for-range-read set to True.
type FileCacheWithCacheForRangeRead struct {
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *FileCacheWithCacheForRangeRead) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			MaxSizeMB:             -1,
			CacheFileForRangeRead: true,
		},
		CacheDir: config.CacheDir(CacheDir),
	}
	t.fsTest.SetupSuite()
}

func (t *FileCacheWithCacheForRangeRead) TearDownTest() {
	t.fsTest.TearDownTest()
	err := os.RemoveAll(FileCacheDir)
	assert.Nil(t.T(), err)
}

func (t *FileCacheWithCacheForRangeRead) TestRandomReadShouldPopulateCache() {
	hundredKiB := 100 * util.KiB
	tenKiB := 10 * util.KiB
	objectContent := generateRandomString(hundredKiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	assert.Nil(t.T(), err)
	defer closeFile(&t.fsTest, file)

	// Random read should also download
	buf := make([]byte, tenKiB)
	_, err = file.Seek(int64(tenKiB), 0)
	assert.Nil(t.T(), err)
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent[tenKiB:2*tenKiB], string(buf)))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	// Sleep for async job to complete download
	time.Sleep(50 * time.Millisecond)
	cacheFile, err := os.OpenFile(downloadPath, os.O_RDWR|syscall.O_DIRECT, 0644)
	assert.Nil(t.T(), err)
	defer closeFile(&t.fsTest, cacheFile)
	cachedContent := make([]byte, hundredKiB)
	_, err = cacheFile.Read(cachedContent)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent, string(cachedContent)))
}

func (t *FileCacheWithCacheForRangeRead) TestSequentialReadShouldPopulateCache() {
	sequentialReadShouldPopulateCache(&t.fsTest, FileCacheDir)
}

func (t *FileCacheWithCacheForRangeRead) TestCacheFilePermission() {
	cacheFilePermissionTest(&t.fsTest, util.DefaultFilePerm)
}

func (t *FileCacheWithCacheForRangeRead) TestWriteShouldNotPopulateCache() {
	writeShouldNotPopulateCache(&t.fsTest)
}

func (t *FileCacheWithCacheForRangeRead) TestSequentialToRandomReadShouldPopulateCache() {
	sequentialToRandomReadShouldPopulateCache(&t.fsTest)
}

func (t *FileCacheWithCacheForRangeRead) TestNewGenerationShouldRebuildCache() {
	objectContent := generateRandomString(2 * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	// read to populate cache
	gotContent, err := os.ReadFile(filePath)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))

	// Change generation and size of object
	objectContent = generateRandomString(util.MiB)
	objects = map[string]string{DefaultObjectName: objectContent}
	err = t.createObjects(objects)
	assert.Nil(t.T(), err)

	// advance clock for stat cache to hit ttl
	cacheClock.AdvanceTime(time.Second * 60)
	// read again to hit updated cache
	gotContent, err = os.ReadFile(filePath)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))
	// check cache also contains updated content
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	cacheContent, err := os.ReadFile(downloadPath)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent, string(cacheContent)))
}

func (t *FileCacheTest) TestModifyFileInCacheAndThenReadShouldGiveModifiedData() {
	objectContent := generateRandomString(util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	// read to populate cache
	gotContent, err := os.ReadFile(filePath)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))

	// change the file in cache
	changedContent := generateRandomString(util.MiB)
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	// modify the file in cache
	err = os.WriteFile(downloadPath, []byte(changedContent), os.FileMode(0655))
	assert.Nil(t.T(), err)

	// read the file again, should give modified content
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, os.FileMode(0655))
	defer closeFile(&t.fsTest, file)
	assert.Nil(t.T(), err)
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	AssertTrue(reflect.DeepEqual(changedContent, string(buf)))
}

// Tests for file system where the file cache is disabled if cache-dir is passed
// but file-cache: max-size-mb is 0.
type FileCacheIsDisabledWithCacheDirAndZeroMaxSize struct {
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *FileCacheIsDisabledWithCacheDirAndZeroMaxSize) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			MaxSizeMB:             0,
			CacheFileForRangeRead: true,
		},
		CacheDir: config.CacheDir(CacheDir),
	}
	t.fsTest.SetupSuite()
}

func (t *FileCacheIsDisabledWithCacheDirAndZeroMaxSize) TearDownTest() {
	t.fsTest.TearDownTest()
}

func (t *FileCacheIsDisabledWithCacheDirAndZeroMaxSize) TearDownSuite() {
	t.fsTest.TearDownSuite()
}

func (t *FileCacheIsDisabledWithCacheDirAndZeroMaxSize) TestReadingFileDoesNotPopulateCache() {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(&t.fsTest, file)
	assert.Nil(t.T(), err)

	// Reading object with cache disabled should not cache the object into file.
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	assert.Nil(t.T(), err)
	AssertEq(objectContent, string(buf))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(FileCacheDir, objectPath)
	_, err = os.Stat(downloadPath)
	assert.NotNil(t.T(), err)
	AssertTrue(os.IsNotExist(err))
}

// Test to check cache is not deleted at the time of unmounting.
type FileCacheDestroyTest struct {
	suite.Suite
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.TearDownTestSuite
	fsTest
}

func (t *FileCacheDestroyTest) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			MaxSizeMB:             -1,
			CacheFileForRangeRead: true,
		},
		CacheDir: config.CacheDir(CacheDir),
	}
	t.fsTest.SetupSuite()
}

func (t *FileCacheDestroyTest) TearDownSuite() {
	// Do nothing as fs is unmounted in the test itself
	// t.fsTest.TearDownSuite()
}

func (t *FileCacheDestroyTest) TearDownTest() {
	// Do nothing and just delete cache as fs is unmounted in the test itself
	err := os.RemoveAll(FileCacheDir)
	assert.Nil(t.T(), err)
	t.fsTest.TearDownTest()
}

func (t *FileCacheDestroyTest) TestCacheIsNotDeletedOnUnmount() {
	// Read to populate cache
	objectContent := generateRandomString(50)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	assert.Nil(t.T(), err)
	filePath := path.Join(mntDir, DefaultObjectName)
	gotContent, err := os.ReadFile(filePath)
	assert.Nil(t.T(), err)
	ExpectTrue(reflect.DeepEqual(objectContent, string(gotContent)))
	_, err = os.Stat(FileCacheDir)
	assert.Nil(t.T(), err)

	t.fsTest.TearDownTest()
	t.fsTest.TearDownSuite()

	if err != nil {
		AddFailure("MountedFileSystem.Unmount: %v", err)
		AbortTest()
	}
	// Check the cache location is not deleted
	_, err = os.Stat(FileCacheDir)
	assert.Nil(t.T(), err)
}
