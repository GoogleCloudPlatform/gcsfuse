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
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/config"
	testutil "github.com/googlecloudplatform/gcsfuse/internal/util"
	. "github.com/jacobsa/ogletest"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

const (
	FileCacheSizeInMb     = 10
	DefaultObjectName     = "foo.txt"
	DefaultObjectSizeInMb = 5
)

var CacheLocation = path.Join(os.Getenv("HOME"), "cache-dir")

// A collection of tests for a file system where the file cache is enabled
// with download-file-for-random-read set to False.
type FileCacheTest struct {
	fsTest
}

func init() {
	RegisterTestSuite(&FileCacheTest{})
}

func (t *FileCacheTest) SetUpTestSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			MaxSizeInMB:               FileCacheSizeInMb,
			DownloadFileForRandomRead: false,
		},
		CacheLocation: config.CacheLocation(CacheLocation),
	}
	t.fsTest.SetUpTestSuite()
}

func (t *FileCacheTest) TearDown() {
	t.fsTest.TearDown()
	err := os.RemoveAll(CacheLocation)
	AssertEq(nil, err)
}

func generateRandomString(length int) string {
	return string(testutil.GenerateRandomBytes(length))
}

func closeFile(file *os.File) {
	err := file.Close()
	AssertEq(nil, err)
}

func sequentialReadShouldPopulateCache(t *fsTest, cacheLocation string) {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)

	// reading object with cache enabled should cache the object into file.
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertEq(objectContent, string(buf))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(cacheLocation, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(buf, cachedContent))
}

func cacheFilePermissionTest(t *fsTest, fileMode os.FileMode) {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)

	// reading object with cache enabled should cache the object into file.
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent, string(buf)))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	stat, err := os.Stat(downloadPath)
	AssertEq(nil, err)
	// confirm file mode is as expected
	AssertEq(fileMode, stat.Mode())
}

func writeShouldNotPopulateCache(t *fsTest) {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)

	// writing file with cache enabled should not populate cache.
	buf := []byte(objectContent)
	n, err := file.Write(buf)
	AssertEq(nil, err)
	AssertEq(n, len(objectContent))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	_, err = os.Stat(downloadPath)
	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func sequentialToRandomReadShouldPopulateCache(t *fsTest) {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)
	// Sequential read
	buf := make([]byte, util.MiB)
	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent[:util.MiB], string(buf)))

	// random read
	offsetForRandom := int64(3)
	_, err = file.Seek(offsetForRandom, 0)
	AssertEq(nil, err)
	_, err = file.Read(buf)

	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent[offsetForRandom:offsetForRandom+util.MiB], string(buf)))
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(cachedContent[offsetForRandom:offsetForRandom+util.MiB], buf))
}

func (t *FileCacheTest) SequentialReadShouldPopulateCache() {
	sequentialReadShouldPopulateCache(&t.fsTest, CacheLocation)
}

func (t *FileCacheTest) SequentialToRandomReadShouldPopulateCache() {
	sequentialToRandomReadShouldPopulateCache(&t.fsTest)
}

func (t *FileCacheTest) CacheFilePermissionWithoutAllowOther() {
	cacheFilePermissionTest(&t.fsTest, util.DefaultFilePerm)
}

func (t *FileCacheTest) WriteShouldNotPopulateCache() {
	writeShouldNotPopulateCache(&t.fsTest)
}

func (t *FileCacheTest) FileSizeGreaterThanCacheSize() {
	objectContent := generateRandomString((FileCacheSizeInMb + 1) * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)

	// reading object with size greater than cache size
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent, string(buf)))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	_, err = os.Stat(downloadPath)
	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func (t *FileCacheTest) EvictionWhenFileCacheIsFull() {
	objectName1 := DefaultObjectName + "1"
	objectContent1 := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objectName2 := DefaultObjectName + "2"
	objectContent2 := generateRandomString((FileCacheSizeInMb - DefaultObjectSizeInMb + 1) * util.MiB)
	objects := map[string]string{objectName1: objectContent1, objectName2: objectContent2}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	// read object 1 which should populate the cache
	filePath1 := path.Join(mntDir, objectName1)
	gotObjectContent1, err := os.ReadFile(filePath1)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent1, string(gotObjectContent1)))
	// verify file is cached
	objectPath1 := util.GetObjectPath(bucket.Name(), objectName1)
	downloadPath1 := util.GetDownloadPath(CacheLocation, objectPath1)
	cachedContent, err := os.ReadFile(downloadPath1)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent1, string(cachedContent)))

	// read the second file, so first should be evicted.
	filePath2 := path.Join(mntDir, objectName2)
	gotObjectContent2, err := os.ReadFile(filePath2)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent2, string(gotObjectContent2)))

	_, err = os.Stat(downloadPath1)
	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func (t *FileCacheTest) RandomReadShouldNotPopulateCache() {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)
	// randomly read object with cache enabled should not populate cache
	buf := make([]byte, util.MiB)
	_, err = file.Seek(util.MiB, 0)
	AssertEq(nil, err)

	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent[util.MiB:2*util.MiB], string(buf)))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	// ToDo(raj-prince): This is not correct behavior i.e. we are making space
	// for file even in case of random reads when downloadFileForRandomRead is
	// False.
	stat, err := os.Stat(downloadPath)
	AssertEq(nil, err)
	AssertEq(0, stat.Size())
}

func (t *FileCacheTest) DeletingFileFromCacheShouldReadFromGCS() {
	objectContent := generateRandomString(util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	AssertEq(nil, err)
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	AssertEq(nil, err)
	closeFile(file)
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	// delete the file in cache
	err = os.Remove(downloadPath)
	AssertEq(nil, err)
	file, err = os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)

	// reading again should throw error
	_, err = file.Read(buf)

	// ToDo(raj-prince): This is a bug due to which new file is created and data
	// is served from that. Also, the data is mostly empty.
	AssertEq(nil, err)
	AssertFalse(reflect.DeepEqual(string(buf), objectContent))
}

func (t *FileCacheTest) ConcurrentReadsFromSameFileHandle() {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)
	wg := sync.WaitGroup{}
	readFunc := func(offset int64, length int64) {
		defer wg.Done()
		buf := make([]byte, length)
		_, err := file.Seek(offset, 0)
		ExpectEq(nil, err)
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
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent, string(cachedContent)))
}

func (t *FileCacheTest) FileSizeEqualToFileCacheSize() {
	objectContent := generateRandomString(FileCacheSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)

	// reading object with cache enabled should cache the object into file.
	gotContent, err := os.ReadFile(filePath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent, string(cachedContent)))
}

func (t *FileCacheTest) WriteToFileCachedAndThenReadingItShouldBeCorrect() {
	sequentialToRandomReadShouldPopulateCache(&t.fsTest)
	// write content to file that is cached.
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)
	// Write to file after reading
	buf := []byte(objectContent)
	n, err := file.Write(buf)
	AssertEq(nil, err)
	AssertEq(n, len(objectContent))

	// read the file again
	gotContent, err := os.ReadFile(filePath)
	AssertEq(nil, err)

	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))
	// the file in cache should not be updated because the file that is being
	// cached is still dirty.
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	AssertEq(nil, err)
	AssertFalse(reflect.DeepEqual(objectContent, string(cachedContent)))
}

func (t *FileCacheTest) SyncToFileCachedAndThenReadingItShouldBeCorrect() {
	sequentialToRandomReadShouldPopulateCache(&t.fsTest)
	// write and sync content to file that is cached.
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)
	// Write and sync to file after reading
	buf := []byte(objectContent)
	n, err := file.Write(buf)
	AssertEq(nil, err)
	AssertEq(n, len(objectContent))
	err = file.Sync()
	AssertEq(nil, err)

	// read the file again
	gotContent, err := os.ReadFile(filePath)
	AssertEq(nil, err)

	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent, string(cachedContent)))
}

// A collection of tests for a file system where the file cache is enabled
// with download-file-for-random-read set to True.
type FileCacheWithDownloadForRandomRead struct {
	fsTest
}

func init() {
	RegisterTestSuite(&FileCacheWithDownloadForRandomRead{})
}

func (t *FileCacheWithDownloadForRandomRead) SetUpTestSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			MaxSizeInMB:               -1,
			DownloadFileForRandomRead: true,
		},
		CacheLocation: config.CacheLocation(CacheLocation),
	}
	t.serverCfg.AllowOther = true
	t.fsTest.SetUpTestSuite()
}

func (t *FileCacheWithDownloadForRandomRead) TearDown() {
	t.fsTest.TearDown()
	err := os.RemoveAll(CacheLocation)
	AssertEq(nil, err)
}

func (t *FileCacheWithDownloadForRandomRead) RandomReadShouldPopulateCache() {
	objectContent := generateRandomString(1 * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)

	// random read should also download
	buf := make([]byte, len(objectContent)-util.MiB/2)
	_, err = file.Seek(util.MiB/2, 0)
	AssertEq(nil, err)
	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent[util.MiB/2:], string(buf)))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	AssertEq(nil, err)
	// Sleep to get the file downloaded to cache.
	time.Sleep(4 * time.Millisecond)
	AssertTrue(reflect.DeepEqual(objectContent, string(cachedContent)))
}

func (t *FileCacheWithDownloadForRandomRead) SequentialReadShouldPopulateCache() {
	sequentialReadShouldPopulateCache(&t.fsTest, CacheLocation)
}

func (t *FileCacheWithDownloadForRandomRead) CacheFilePermissionWithAllowOther() {
	cacheFilePermissionTest(&t.fsTest, util.FilePermWithAllowOther)
}

func (t *FileCacheWithDownloadForRandomRead) WriteShouldNotPopulateCache() {
	writeShouldNotPopulateCache(&t.fsTest)
}

func (t *FileCacheWithDownloadForRandomRead) SequentialToRandomReadShouldPopulateCache() {
	sequentialToRandomReadShouldPopulateCache(&t.fsTest)
}

func (t *FileCacheWithDownloadForRandomRead) NewGenerationShouldRebuildCache() {
	objectContent := generateRandomString(2 * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)
	// read to populate cache
	gotContent, err := os.ReadFile(filePath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))

	// Change generation and size of object
	objectContent = generateRandomString(util.MiB)
	objects = map[string]string{DefaultObjectName: objectContent}
	err = t.createObjects(objects)
	AssertEq(nil, err)

	// advance clock for stat cache to hit ttl
	cacheClock.AdvanceTime(time.Second * 60)
	// read again to hit updated cache
	gotContent, err = os.ReadFile(filePath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))
	// check cache also contains updated content
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	cacheContent, err := os.ReadFile(downloadPath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent, string(cacheContent)))
}

func (t *FileCacheTest) ModifyFileInCacheAndThenReadShouldGiveModifiedData() {
	objectContent := generateRandomString(util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)
	// read to populate cache
	gotContent, err := os.ReadFile(filePath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent, string(gotContent)))

	// change the file in cache
	changedContent := generateRandomString(util.MiB)
	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	// modify the file in cache
	err = os.WriteFile(downloadPath, []byte(changedContent), util.FilePermWithAllowOther)
	AssertEq(nil, err)

	// read the file again, should give modified content
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.FilePermWithAllowOther)
	defer closeFile(file)
	AssertEq(nil, err)
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(changedContent, string(buf)))
}

// A collection of tests for a file system where the file cache is enabled
// with default cache location.
type FileCacheWithDefaultCacheLocation struct {
	fsTest
}

func init() {
	RegisterTestSuite(&FileCacheWithDefaultCacheLocation{})
}

func (t *FileCacheWithDefaultCacheLocation) SetUpTestSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			MaxSizeInMB:               -1,
			DownloadFileForRandomRead: true,
		},
	}
	t.serverCfg.AllowOther = true
	t.fsTest.SetUpTestSuite()
}

func (t *FileCacheWithDefaultCacheLocation) TearDown() {
	t.fsTest.TearDown()
	err := os.RemoveAll(CacheLocation)
	AssertEq(nil, err)
}

func (t *FileCacheWithDefaultCacheLocation) DefaultLocationIsTempDir() {
	sequentialReadShouldPopulateCache(&t.fsTest, os.TempDir())
}
