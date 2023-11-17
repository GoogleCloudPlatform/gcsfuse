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
// with download-file-for-random-read set to True.

package fs_test

import (
	"os"
	"path"
	"reflect"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/config"
	. "github.com/jacobsa/ogletest"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

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
	_ = os.RemoveAll(CacheLocation)
}

func (t *FileCacheWithDownloadForRandomRead) RandomReadShouldPopulateCache() {
	objectContent := generateRandomString(DefaultObjectSizeInMb * util.MiB)
	objects := map[string]string{DefaultObjectName: objectContent}
	err := t.createObjects(objects)
	AssertEq(nil, err)
	filePath := path.Join(mntDir, DefaultObjectName)
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, util.DefaultFilePerm)
	defer closeFile(file)
	AssertEq(nil, err)

	// random read should also download
	buf := make([]byte, len(objectContent)-util.MiB)
	_, err = file.Seek(util.MiB, 0)
	AssertEq(nil, err)
	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent[util.MiB:], string(buf)))

	objectPath := util.GetObjectPath(bucket.Name(), DefaultObjectName)
	downloadPath := util.GetDownloadPath(CacheLocation, objectPath)
	cachedContent, err := os.ReadFile(downloadPath)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent, string(cachedContent)))
}

func (t *FileCacheWithDownloadForRandomRead) SequentialReadShouldPopulateCache() {
	sequentialReadShouldPopulateCache(&t.fsTest)
}

func (t *FileCacheWithDownloadForRandomRead) CacheFilePermissionWithAllowOther() {
	cacheFilePermissionWithoutAllowOther(&t.fsTest, util.FilePermWithAllowOther)
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
	AssertEq(nil, err)
	buf := make([]byte, len(objectContent))
	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(changedContent, string(buf)))
}
