// Copyright 2026 Google LLC
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

package shared_chunk_cache

import (
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Suite Definitions
////////////////////////////////////////////////////////////////////////

// Struct to store the details of a mount point
type mountPoint struct {
	rootDir     string // Root directory of the test folder, which contains mnt and gcsfuse.log.
	mntDir      string // Directory where the GCS bucket is mounted. This is 'mnt' inside rootDir.
	testDirPath string // Path to the 'SharedChunkCacheTest' directory inside mntDir.
	logFilePath string // Path to the GCSFuse log file. This is gcsfuse.log inside rootDir.
}

// BaseSuite provides the common structure and configuration-driven setup logic.
type BaseSuite struct {
	suite.Suite
	primaryFlags   []string
	secondaryFlags []string
	primaryMount   mountPoint
	secondaryMount mountPoint
	sharedCacheDir string
}

// SharedChunkCacheTestSuite groups all shared chunk cache tests.
type SharedChunkCacheTestSuite struct{ BaseSuite }

////////////////////////////////////////////////////////////////////////
// Common Suite Logic
////////////////////////////////////////////////////////////////////////

func (t *BaseSuite) SetupTest() {
	// Set up the shared cache directory
	if testEnv.cfg.GKEMountedDirectory != "" {
		t.sharedCacheDir = path.Join(GKETempDir, "shared-cache", "gcsfuse-shared-chunk-cache")
	} else {
		t.sharedCacheDir = path.Join(setup.TestDir(), GKETempDir, "shared-cache", "gcsfuse-shared-chunk-cache")
	}

	// Clean up cache directory before each test to ensure clean state
	operations.RemoveDir(t.sharedCacheDir)

	if testEnv.cfg.GKEMountedDirectory != "" {
		// GKE Mode: Already mounted
		t.primaryMount.mntDir = testEnv.cfg.GKEMountedDirectory
		t.primaryMount.testDirPath = path.Join(t.primaryMount.mntDir, testDirName)
		t.primaryMount.logFilePath = testEnv.cfg.LogFile // Might be empty, but that's fine for GKE

		if len(t.secondaryFlags) > 0 {
			t.secondaryMount.mntDir = testEnv.cfg.GKEMountedDirectorySecondary
			t.secondaryMount.testDirPath = path.Join(t.secondaryMount.mntDir, testDirName)
		}
	} else {
		// GCE Mode: Mount it
		t.primaryMount.setupTestDir(testEnv.cfg.GCSFuseMountedDirectory, testEnv.cfg.LogFile)
		t.mountGcsfuse(t.primaryMount, "primary", t.primaryFlags)

		if len(t.secondaryFlags) > 0 {
			secondaryLog := path.Join(path.Dir(testEnv.cfg.LogFile), "gcsfuse_secondary.log")
			t.secondaryMount.setupTestDir(testEnv.cfg.GCSFuseMountedDirectorySecondary, secondaryLog)
			t.mountGcsfuse(t.secondaryMount, "secondary", t.secondaryFlags)
		}
	}
}

func (t *BaseSuite) TearDownTest() {
	if t.T().Failed() {
		// Save logs for both mounts on failure to aid debugging.
		testName := strings.ReplaceAll(t.T().Name(), "/", "_")
		if t.primaryMount.logFilePath != "" {
			setup.SaveLogFileAsArtifact(t.primaryMount.logFilePath, "gcsfuse-primary-log-"+testName)
		}
		if len(t.secondaryFlags) > 0 && t.secondaryMount.logFilePath != "" {
			setup.SaveLogFileAsArtifact(t.secondaryMount.logFilePath, "gcsfuse-secondary-log-"+testName)
		}
	}

	if testEnv.cfg.GKEMountedDirectory != "" {
		// GKE Mode: Just cleanup files
		setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	} else {
		// GCE Mode: Unmount and clean up
		t.unmountAndCleanupMount(t.primaryMount, "primary")
		if len(t.secondaryFlags) > 0 {
			t.unmountAndCleanupMount(t.secondaryMount, "secondary")
		}
	}

	// Clean up shared cache directory after each test
	operations.RemoveDir(t.sharedCacheDir)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (mnt *mountPoint) setupTestDir(mountDir, logFile string) {
	mnt.rootDir = setup.TestDir()
	mnt.mntDir = mountDir
	mnt.logFilePath = logFile
	mnt.testDirPath = path.Join(mountDir, testDirName)
}

func (t *BaseSuite) mountGcsfuse(mnt mountPoint, mountType string, flags []string) {
	setup.SetMntDir(mnt.mntDir)
	setup.SetLogFile(mnt.logFilePath)
	err := static_mounting.MountGcsfuseWithStaticMounting(flags)
	require.NoError(t.T(), err, "Unable to mount %s: %v", mountType, err)
	mnt.testDirPath = setup.SetupTestDirectory(testDirName)
	log.Printf("Running tests with %s mount flags %v", mountType, flags)
}

func (t *BaseSuite) unmountAndCleanupMount(m mountPoint, name string) {
	setup.UnmountGCSFuse(m.mntDir)
	// Cleaning up the intermediate generated test files.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
}

func (t *BaseSuite) createTestFile(fileName string, fileSize int) {
	t.T().Helper()
	testFilePath := path.Join(t.primaryMount.testDirPath, fileName)
	operations.CreateFileOfSize(int64(fileSize), testFilePath, t.T())
}

func (t *BaseSuite) getCachedChunkCount() int {
	t.T().Helper()
	count := 0
	err := filepath.WalkDir(t.sharedCacheDir, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(filePath, ".bin") {
			count++
		}
		return nil
	})
	if err != nil {
		t.T().Logf("Error walking cache directory: %v", err)
	}
	return count
}

func (t *BaseSuite) getCacheFileModTimes() map[string]os.FileInfo {
	t.T().Helper()
	modTimes := make(map[string]os.FileInfo)
	err := filepath.WalkDir(t.sharedCacheDir, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(filePath, ".bin") {
			info, err := os.Stat(filePath)
			if err == nil {
				modTimes[filePath] = info
			}
		}
		return nil
	})
	if err != nil {
		t.T().Logf("Error walking cache directory: %v", err)
	}
	return modTimes
}

////////////////////////////////////////////////////////////////////////
// Test Cases
////////////////////////////////////////////////////////////////////////

// TestCacheMiss tests that when a small read triggers chunk caching from the first mount.
// Reading 2MB starting at 10MB offset will download and cache the entire 10MB chunk (2nd chunk).
func (t *SharedChunkCacheTestSuite) TestCacheMiss() {
	const (
		testFileName = "test_cache_miss.txt"
		fileSize     = 30 * util.MiB
		readSize     = 2 * util.MiB  // Read only 2MB
		readOffset   = 10 * util.MiB // Start at 10MB (in the 2nd chunk)
	)

	// Arrange: Set up test file and verify initial cache state
	t.createTestFile(testFileName, fileSize)
	initialCacheCount := t.getCachedChunkCount()
	require.Equal(t.T(), 0, initialCacheCount, "Cache should be empty initially")

	// Act: Read 2MB from the 2nd chunk (triggers download and caching of entire 10MB chunk)
	primaryFilePath := path.Join(t.primaryMount.testDirPath, testFileName)
	startTime := time.Now()
	chunk, err := operations.ReadChunkFromFile(primaryFilePath, readSize, readOffset, os.O_RDONLY)
	cacheMissTime := time.Since(startTime)

	// Assert: Verify read succeeded and entire chunk was cached
	require.NoError(t.T(), err, "Failed to read chunk from primary mount")
	require.Equal(t.T(), int(readSize), len(chunk), "Read chunk size mismatch")
	cachedChunkCount := t.getCachedChunkCount()
	require.Equal(t.T(), 1, cachedChunkCount, "Cache should contain exactly 1 chunk after reading from 2nd chunk")
	t.T().Logf("Cache miss test: Read %d bytes from offset %d in %v (cache miss), entire chunk cached (%d chunk)",
		readSize, readOffset, cacheMissTime, cachedChunkCount)
}

// TestCacheHit tests that when a small read is served from the shared chunk cache.
// Reading 2MB starting at 10MB offset should be served from the cached 10MB chunk (2nd chunk).
// This test verifies cache hits by checking that cache files are not modified.
func (t *SharedChunkCacheTestSuite) TestCacheHit() {
	const (
		testFileName = "test_cache_hit.txt"
		fileSize     = 30 * util.MiB
		readSize     = 2 * util.MiB  // Read only 2MB
		readOffset   = 10 * util.MiB // Start at 10MB (in the 2nd chunk)
	)

	// Arrange: Set up test file and populate cache via primary mount
	t.createTestFile(testFileName, fileSize)
	primaryFilePath := path.Join(t.primaryMount.testDirPath, testFileName)
	primaryStartTime := time.Now()
	primaryChunk, err := operations.ReadChunkFromFile(primaryFilePath, readSize, readOffset, os.O_RDONLY)
	primaryCacheMissTime := time.Since(primaryStartTime)
	require.NoError(t.T(), err, "Failed to read chunk from primary mount")
	require.Equal(t.T(), int(readSize), len(primaryChunk), "Read chunk size mismatch on primary mount")
	cachedChunkCount := t.getCachedChunkCount()
	require.Equal(t.T(), 1, cachedChunkCount, "Cache should contain exactly 1 chunk after reading from 2nd chunk")
	t.T().Logf("Primary read (cache miss): %d bytes from offset %d in %v, entire chunk cached (%d chunk)",
		readSize, readOffset, primaryCacheMissTime, cachedChunkCount)
	// Capture cache file modification times before secondary read
	cacheFileTimes := t.getCacheFileModTimes()
	require.NotEmpty(t.T(), cacheFileTimes, "Should have cache files to track")
	t.T().Logf("Captured modification times for %d cache files", len(cacheFileTimes))

	// Act: Read the same 2MB from the secondary mount (should be served from cache)
	secondaryFilePath := path.Join(t.secondaryMount.testDirPath, testFileName)
	// Warm up metadata cache on secondary mount by doing a stat first
	// This ensures the timed read doesn't include initial metadata lookup overhead
	_, err = os.Stat(secondaryFilePath)
	require.NoError(t.T(), err, "Failed to stat file on secondary mount")
	secondaryStartTime := time.Now()
	secondaryChunk, err := operations.ReadChunkFromFile(secondaryFilePath, readSize, readOffset, os.O_RDONLY)
	cacheHitTime := time.Since(secondaryStartTime)

	// Assert: Verify read succeeded from cache without re-downloading
	require.NoError(t.T(), err, "Failed to read chunk from secondary mount")
	require.Equal(t.T(), int(readSize), len(secondaryChunk), "Read chunk size mismatch on secondary mount")
	require.Equal(t.T(), primaryChunk, secondaryChunk, "Chunk content from both mounts should match")
	finalCacheCount := t.getCachedChunkCount()
	require.Equal(t.T(), cachedChunkCount, finalCacheCount,
		"Cache size should remain the same after reading from secondary mount (cache hit)")
	// Verify cache files were NOT modified (proving they weren't re-downloaded)
	newCacheFileTimes := t.getCacheFileModTimes()
	for filePath, oldInfo := range cacheFileTimes {
		newInfo, exists := newCacheFileTimes[filePath]
		require.True(t.T(), exists, "Cache file should still exist: %s", filePath)
		require.Equal(t.T(), oldInfo.ModTime(), newInfo.ModTime(),
			"Cache file should not be modified (proving cache hit): %s", filePath)
	}
	speedup := float64(primaryCacheMissTime) / float64(cacheHitTime)
	t.T().Logf("Secondary read (cache hit): %d bytes from offset %d in %v, cache files unchanged (%d chunk)",
		readSize, readOffset, cacheHitTime, finalCacheCount)
	t.T().Logf("Performance: Cache miss=%v, Cache hit=%v, Speedup=%.2fx",
		primaryCacheMissTime, cacheHitTime, speedup)
}

// TestCacheHitSingleMount tests cache behavior within a single mount.
// First read causes a cache miss (downloads and caches the chunk).
// Subsequent read of the same chunk should be a cache hit (served from cache).
func (t *SharedChunkCacheTestSuite) TestCacheHitSingleMount() {
	const (
		testFileName = "test_cache_hit_single_mount.txt"
		fileSize     = 30 * util.MiB
		readSize     = 2 * util.MiB  // Read only 2MB
		readOffset   = 10 * util.MiB // Start at 10MB (in the 2nd chunk)
	)

	// Arrange: Set up test file, verify empty cache, and perform first read to populate cache
	t.createTestFile(testFileName, fileSize)
	initialCacheCount := t.getCachedChunkCount()
	require.Equal(t.T(), 0, initialCacheCount, "Cache should be empty initially")
	primaryFilePath := path.Join(t.primaryMount.testDirPath, testFileName)
	firstReadStart := time.Now()
	firstChunk, err := operations.ReadChunkFromFile(primaryFilePath, readSize, readOffset, os.O_RDONLY)
	cacheMissTime := time.Since(firstReadStart)
	require.NoError(t.T(), err, "Failed to read chunk on first read (cache miss)")
	require.Equal(t.T(), int(readSize), len(firstChunk), "Read chunk size mismatch on first read")
	cachedChunkCount := t.getCachedChunkCount()
	require.Equal(t.T(), 1, cachedChunkCount, "Cache should contain exactly 1 chunk after first read")
	t.T().Logf("First read (cache miss): %d bytes from offset %d in %v, chunk cached (%d chunk)",
		readSize, readOffset, cacheMissTime, cachedChunkCount)
	// Capture cache file modification times before second read
	cacheFileTimes := t.getCacheFileModTimes()
	require.NotEmpty(t.T(), cacheFileTimes, "Should have cache files to track")

	// Act: Read the same chunk again from the same mount (should hit cache)
	secondReadStart := time.Now()
	secondChunk, err := operations.ReadChunkFromFile(primaryFilePath, readSize, readOffset, os.O_RDONLY)
	cacheHitTime := time.Since(secondReadStart)

	// Assert: Verify second read succeeded from cache without re-downloading
	require.NoError(t.T(), err, "Failed to read chunk on second read (cache hit)")
	require.Equal(t.T(), int(readSize), len(secondChunk), "Read chunk size mismatch on second read")
	require.Equal(t.T(), firstChunk, secondChunk, "Content should match between first and second read")
	finalCacheCount := t.getCachedChunkCount()
	require.Equal(t.T(), cachedChunkCount, finalCacheCount,
		"Cache size should remain the same after second read (cache hit)")
	// Verify cache files were NOT modified (proving cache hit)
	newCacheFileTimes := t.getCacheFileModTimes()
	for filePath, oldInfo := range cacheFileTimes {
		newInfo, exists := newCacheFileTimes[filePath]
		require.True(t.T(), exists, "Cache file should still exist: %s", filePath)
		require.Equal(t.T(), oldInfo.ModTime(), newInfo.ModTime(),
			"Cache file should not be modified (proving cache hit): %s", filePath)
	}
	speedup := float64(cacheMissTime) / float64(cacheHitTime)
	t.T().Logf("Second read (cache hit): %d bytes from offset %d in %v, cache files unchanged (%d chunk)",
		readSize, readOffset, cacheHitTime, finalCacheCount)
	t.T().Logf("Single mount performance: Cache miss=%v, Cache hit=%v, Speedup=%.2fx",
		cacheMissTime, cacheHitTime, speedup)
}

////////////////////////////////////////////////////////////////////////
// Test Suite Runner
////////////////////////////////////////////////////////////////////////

func RunTests(t *testing.T, runName string, factory func(primaryFlags, secondaryFlags []string) suite.TestingSuite) {
	for _, cfg := range testEnv.cfg.Configs {
		if cfg.Run == runName {
			for i, flagStr := range cfg.Flags {
				primaryFlags := strings.Fields(flagStr)
				var secondaryFlags []string
				if len(cfg.SecondaryFlags) > i {
					secondaryFlags = strings.Fields(cfg.SecondaryFlags[i])
				}
				suite.Run(t, factory(primaryFlags, secondaryFlags))
			}
		}
	}
}

func TestSharedChunkCacheTestSuite(t *testing.T) {
	RunTests(t, "TestSharedChunkCacheTestSuite", func(primaryFlags, secondaryFlags []string) suite.TestingSuite {
		s := &SharedChunkCacheTestSuite{}
		s.primaryFlags = primaryFlags
		s.secondaryFlags = secondaryFlags
		return s
	})
}
