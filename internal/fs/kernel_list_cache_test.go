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

// A collection of tests which tests the kernel-list-cache feature, in which
// directory listing 2nd time is served from kernel page-cache unless not invalidated.
// Base of all the tests: how to detect if directory listing is served from page-cache
// or from GCSFuse?
// (a) GCSFuse file-system ensures different content, when listing happens on the same directory.
// (b) If two consecutive directory listing for the same directory are same, that means
//     2nd listing is served from kernel-page-cache.
// (c) If not then, both 1st and 2nd listing are served from GCSFuse filesystem.

package fs_test

import (
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/cfg"
	"github.com/vipnydav/gcsfuse/v3/internal/util"
	"github.com/vipnydav/gcsfuse/v3/metrics"
)

const (
	kernelListCacheTtlSeconds = 1000
)

type KernelListCacheTestCommon struct {
	suite.Suite
	fsTest
}

func (t *KernelListCacheTestCommon) SetupTest() {
	t.createFilesAndDirStructureInBucket()
	cacheClock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
}

func (t *KernelListCacheTestCommon) TearDownTest() {
	cacheClock.AdvanceTime(util.MaxTimeDuration)
	t.deleteFilesAndDirStructureInBucket()
	t.fsTest.TearDown()
}

func (t *KernelListCacheTestCommon) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func TestKernelListCacheTestSuite(t *testing.T) {
	suite.Run(t, new(KernelListCacheTestCommon))
}

// getFilesAndDirStructureObjects returns the following files and directory
// objects.
//
//	explicitDir/
//	explicitDir/file1.txt
//	explicitDir/file2.txt
//	implicitDir/file1.txt
//	implicitDir/file2.txt
func getFilesAndDirStructureObjects() map[string]string {
	return map[string]string{
		"explicitDir/":          "",
		"explicitDir/file1.txt": "12345",
		"explicitDir/file2.txt": "6789101112",
		"implicitDir/file1.txt": "-1234556789",
		"implicitDir/file2.txt": "kdfkdj9",
	}
}
func (t *KernelListCacheTestCommon) createFilesAndDirStructureInBucket() {
	assert.Nil(t.T(), t.createObjects(getFilesAndDirStructureObjects()))
}

func (t *KernelListCacheTestCommon) deleteFilesAndDirStructureInBucket() {
	filesAndDirStructure := getFilesAndDirStructureObjects()
	for k := range filesAndDirStructure {
		assert.Nil(t.T(), t.deleteObject(k))
	}
}

func (t *KernelListCacheTestCommon) deleteObjectOrFail(objectName string) {
	assert.Nil(t.T(), t.deleteObject(objectName))
}

type KernelListCacheTestWithPositiveTtl struct {
	suite.Suite
	fsTest
	KernelListCacheTestCommon
}

func (t *KernelListCacheTestWithPositiveTtl) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.NewConfig = &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			KernelListCacheTtlSecs: kernelListCacheTtlSeconds,
		},
		MetadataCache: cfg.MetadataCacheConfig{
			TtlSecs: 0,
		},
	}
	t.serverCfg.RenameDirLimit = 10
	t.serverCfg.MetricHandle = metrics.NewNoopMetrics()
	t.fsTest.SetUpTestSuite()
}

func TestKernelListCacheTestWithPositiveTtlSuite(t *testing.T) {
	SkipTestForUnsupportedKernelVersion(t)
	suite.Run(t, new(KernelListCacheTestWithPositiveTtl))
}

// Test_Parallel_OpenDirAndLookUpInode helps in detecting the deadlock when
// OpenDir() and LookUpInode() request for same directory comes in parallel.
func (t *KernelListCacheTestWithPositiveTtl) Test_Parallel_OpenDirAndLookUpInode() {
	var wg sync.WaitGroup
	wg.Add(2)
	// Fail if the operation takes more than timeout.
	timeout := 5 * time.Second
	iterationsPerGoroutine := 100

	go func() {
		defer wg.Done()
		for range iterationsPerGoroutine {
			f, err := os.Open(path.Join(mntDir, "explicitDir"))
			assert.Nil(t.T(), err)

			err = f.Close()
			assert.Nil(t.T(), err)
		}
	}()
	go func() {
		defer wg.Done()
		for range iterationsPerGoroutine {
			_, err := os.Stat(path.Join(mntDir, "explicitDir"))
			assert.Nil(t.T(), err)
		}
	}()

	// Wait for goroutines or timeout.
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()
	select {
	case <-done:
		// Operation completed successfully before timeout.
	case <-time.After(timeout):
		assert.FailNow(t.T(), "Possible deadlock")
	}
}

// Test_Concurrent_ReadDir tests for potential deadlocks or race conditions
// when multiple goroutines call Readdir() concurrently on the same directory.
func (t *KernelListCacheTestWithPositiveTtl) Test_Concurrent_ReadDir() {
	var wg sync.WaitGroup
	goroutineCount := 10         // Number of concurrent goroutines
	iterationsPerGoroutine := 10 // Number of iterations per goroutine

	wg.Add(goroutineCount)
	timeout := 5 * time.Second

	dirPath := path.Join(mntDir, "explicitDir")

	for range goroutineCount {
		go func() {
			defer wg.Done()

			for range iterationsPerGoroutine {
				f, err := os.Open(dirPath)
				assert.Nil(t.T(), err)

				_, err = f.Readdirnames(-1) // Read all directory entries
				assert.Nil(t.T(), err)

				err = f.Close()
				assert.Nil(t.T(), err)
			}
		}()
	}

	// Wait for goroutines or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success: All Readdir operations finished before timeout
	case <-time.After(timeout):
		assert.FailNow(t.T(), "Possible deadlock or race condition detected during concurrent Readdir calls")
	}
}

// Test_Parallel_ReadDirAndFileOperations detects race conditions and deadlocks when one goroutine
// performs Readdir() while another concurrently creates and deletes files in the same directory.
func (t *KernelListCacheTestWithPositiveTtl) Test_Parallel_ReadDirAndFileOperations() {
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 5 * time.Second // Adjust timeout as needed
	iterationsPerGoroutine := 100

	dirPath := path.Join(mntDir, "explicitDir")

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for range iterationsPerGoroutine { // Adjust iteration count if needed
			f, err := os.Open(dirPath)
			assert.Nil(t.T(), err)

			_, err = f.Readdirnames(-1)
			assert.Nil(t.T(), err)

			err = f.Close()
			assert.Nil(t.T(), err)
		}
	}()

	// Goroutine 2: Creates and deletes files
	go func() {
		defer wg.Done()
		for range iterationsPerGoroutine { // Adjust iteration count if needed
			filePath := path.Join(dirPath, "tmp_file.txt")
			renamedFilePath := path.Join(dirPath, "renamed_tmp_file.txt")

			// Create
			f, err := os.Create(filePath)
			assert.Nil(t.T(), err)

			err = f.Close()
			assert.Nil(t.T(), err)

			// Rename
			err = os.Rename(filePath, renamedFilePath)
			assert.Nil(t.T(), err)

			// Delete
			err = os.Remove(renamedFilePath)
			assert.Nil(t.T(), err)
		}
	}()

	// Wait for goroutines or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success: Both operations finished before timeout
	case <-time.After(timeout):
		assert.FailNow(t.T(), "Possible deadlock or race condition detected")
	}
}

// Test_Parallel_ReadDirAndDirOperations tests for potential deadlocks or race conditions when
// ReadDir() is called concurrently with directory creation and deletion operations.
func (t *KernelListCacheTestWithPositiveTtl) Test_Parallel_ReadDirAndDirOperations() {
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 5 * time.Second
	iterationsPerGoroutine := 100

	parentDir := path.Join(mntDir, "explicitDir")

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for range iterationsPerGoroutine {
			f, err := os.Open(parentDir)
			assert.Nil(t.T(), err)

			_, err = f.Readdirnames(0)
			assert.Nil(t.T(), err)

			err = f.Close()
			assert.Nil(t.T(), err)
		}
	}()

	// Goroutine 2: Creates and deletes directories
	go func() {
		defer wg.Done()
		for range iterationsPerGoroutine {
			dirPath := path.Join(parentDir, "test_dir")
			renamedDirPath := path.Join(parentDir, "renamed_test_dir")

			// Create
			err := os.Mkdir(dirPath, 0755)
			assert.Nil(t.T(), err)

			// Rename
			err = os.Rename(dirPath, renamedDirPath)
			assert.Nil(t.T(), err)

			// Delete
			err = os.Remove(renamedDirPath)
			assert.Nil(t.T(), err)
		}
	}()

	// Wait for goroutines or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success: Both operations finished before timeout
	case <-time.After(timeout):
		assert.FailNow(t.T(), "Possible deadlock or race condition detected during Readdir and directory operations")
	}
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() within ttl will be served from kernel page cache.
func (t *KernelListCacheTestWithPositiveTtl) TestKernelListCache_CacheHit() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file3.txt")
	// Advancing the clock within time.
	cacheClock.AdvanceTime(kernelListCacheTtlSeconds * time.Second / 2)

	// 2nd read, ReadDir() will be served from page-cache, that means no change in
	// response.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file2.txt", names2[1])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() within ttl will be served from kernel page cache.
func (t *KernelListCacheTestWithPositiveTtl) TestKernelListCache_CacheHitWithImplicitDir() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "implicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"implicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("implicitDir/file3.txt")
	// Advancing the clock within time.
	cacheClock.AdvanceTime(kernelListCacheTtlSeconds * time.Second / 2)

	// 2nd read, ReadDir() will be served from page-cache, that means no change in
	// response.
	f, err = os.Open(path.Join(mntDir, "implicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file2.txt", names2[1])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() out of ttl will also be served from GCSFuse filesystem.
func (t *KernelListCacheTestWithPositiveTtl) TestKernelListCache_CacheMiss() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file3.txt")
	// Advancing the time more than ttl.
	cacheClock.AdvanceTime(kernelListCacheTtlSeconds*time.Second + time.Second)

	// Since out of ttl, so invalidation happens and ReadDir() will be served from
	// gcsfuse filesystem.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 3, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file2.txt", names2[1])
	assert.Equal(t.T(), "file3.txt", names2[2])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() out of ttl will also be served from GCSFuse filesystem.
func (t *KernelListCacheTestWithPositiveTtl) TestKernelListCache_CacheMissWithImplicitDir() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "implicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"implicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("implicitDir/file3.txt")
	// Advancing the time more than ttl.
	cacheClock.AdvanceTime(kernelListCacheTtlSeconds*time.Second + time.Second)

	// Since out of ttl, so invalidation happens and ReadDir() will be served from
	// gcsfuse filesystem.
	f, err = os.Open(path.Join(mntDir, "implicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 3, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file2.txt", names2[1])
	assert.Equal(t.T(), "file3.txt", names2[2])
}

// TestKernelListCache_CacheHitAfterInvalidation:
// (a) First read will be served from GcsFuse filesystem.
// (b) Second read after ttl will also be served from GCSFuse file-system.
// (c) Third read within ttl will be served from kernel page cache.
func (t *KernelListCacheTestWithPositiveTtl) TestKernelListCache_CacheHitAfterInvalidation() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file3.txt")
	// Advancing the time more than ttl.
	cacheClock.AdvanceTime(kernelListCacheTtlSeconds*time.Second + time.Second)
	// Since out of ttl, so invalidation happens and ReadDir() will be served from
	// gcsfuse filesystem.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 3, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file2.txt", names2[1])
	assert.Equal(t.T(), "file3.txt", names2[2])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file4.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file4.txt")
	// Advancing the time within ttl.
	cacheClock.AdvanceTime(kernelListCacheTtlSeconds * time.Second / 2)

	// Within ttl, so will be served from kernel.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names3, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 3, len(names3))
	assert.Equal(t.T(), "file1.txt", names3[0])
	assert.Equal(t.T(), "file2.txt", names3[1])
	assert.Equal(t.T(), "file3.txt", names3[2])
}
