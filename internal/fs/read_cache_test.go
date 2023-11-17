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

// A collection of tests for a file system where the file cache is enabled
// without download-file-for-random-read set to True.

package fs_test

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/config"
	. "github.com/jacobsa/ogletest"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////
const FileCacheSizeInMb = 10

var CacheLocation = path.Join(os.Getenv("HOME"), "cache-dir")

const DefaultObjectName = "foo.txt"
const DefaultObjectSizeInMb = 5

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
	_ = os.RemoveAll(CacheLocation)
}

func generateRandomString(length int) string {
	randBytes := make([]byte, length)
	for i := 0; i < length; i++ {
		randBytes[i] = byte(rand.Intn(26) + 65)
	}
	return string(randBytes)
}

func closeFile(file *os.File) {
	err := file.Close()
	AssertEq(nil, err)
}

func sequentialReadShouldPopulateCache(t *fsTest) {
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
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(buf, cachedContent))
}

func cacheFilePermissionWithoutAllowOther(t *fsTest, fileMode os.FileMode) {
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
	AssertEq(string(buf), objectContent)

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
	fmt.Println(err.Error())
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
	AssertEq(objectContent[:util.MiB], string(buf))

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
	sequentialReadShouldPopulateCache(&t.fsTest)
}

func (t *FileCacheTest) CacheFilePermissionWithoutAllowOther() {
	cacheFilePermissionWithoutAllowOther(&t.fsTest, util.DefaultFilePerm)
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
	AssertEq(string(buf), objectContent)

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
	AssertEq(objectContent1, string(gotObjectContent1))
	// verify file is cached
	objectPath1 := util.GetObjectPath(bucket.Name(), objectName1)
	downloadPath1 := util.GetDownloadPath(CacheLocation, objectPath1)
	cachedContent, err := os.ReadFile(downloadPath1)
	AssertEq(nil, err)
	AssertEq(objectContent1, string(cachedContent))

	// read the second file, so first should be evicted.
	filePath2 := path.Join(mntDir, objectName2)
	gotObjectContent2, err := os.ReadFile(filePath2)
	AssertEq(nil, err)
	AssertEq(objectContent2, string(gotObjectContent2))

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
	AssertEq(objectContent[util.MiB:2*util.MiB], string(buf))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	// ToDo(raj-prince): This is not correct behavior i.e. we are making space
	// for file even in case of random reads when downloadFileForRandomRead is
	// False.
	stat, err := os.Stat(downloadPath)
	AssertEq(nil, err)
	AssertEq(0, stat.Size())
}

func (t *FileCacheTest) SequentialToRandomReadShouldPopulateCache() {
	sequentialReadShouldPopulateCache(&t.fsTest)
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
