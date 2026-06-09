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

package fs_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fusetesting"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type stallingBucket struct {
	gcs.Bucket
	mu            sync.Mutex
	stallDuration time.Duration
	failWithErr   error
}

func (sb *stallingBucket) RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (*gcs.Folder, error) {
	sb.mu.Lock()
	duration := sb.stallDuration
	errToReturn := sb.failWithErr
	sb.mu.Unlock()

	time.Sleep(duration)
	if errToReturn != nil {
		return nil, errToReturn
	}
	return sb.Bucket.RenameFolder(ctx, folderName, destinationFolderId)
}

func (sb *stallingBucket) setStallDuration(d time.Duration) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.stallDuration = d
}

func (sb *stallingBucket) setFailWithErr(err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.failWithErr = err
}

type HNSDirectoryRenameLatencyTests struct {
	suite.Suite
	fsTest
}

func TestHNSDirectoryRenameLatencyTests(t *testing.T) {
	suite.Run(t, new(HNSDirectoryRenameLatencyTests))
}

func (t *HNSDirectoryRenameLatencyTests) SetupSuite() {
	mtimeClock = timeutil.RealClock()
	underlying := fake.NewFakeBucket(mtimeClock, "some_bucket", gcs.BucketType{Hierarchical: true})

	bucket = &stallingBucket{
		Bucket:        underlying,
		stallDuration: 2 * time.Second,
	}

	t.serverCfg.ImplicitDirectories = false
	t.serverCfg.NewConfig = &cfg.Config{
		EnableHns:                true,
		EnableAtomicRenameObject: true,
	}
	t.serverCfg.MetricHandle = metrics.NewNoopMetrics()
	t.serverCfg.TraceHandle = tracing.NewNoopTracer()

	t.fsTest.SetUpTestSuite()
}

func (t *HNSDirectoryRenameLatencyTests) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
	bucket = nil
}

func (t *HNSDirectoryRenameLatencyTests) SetupTest() {
	// Disable stall during setup to ensure setup runs fast
	if sb, ok := bucket.(*stallingBucket); ok {
		sb.setStallDuration(0)
		sb.setFailWithErr(nil)
	}

	// Clean up any leftovers
	entries, _ := fusetesting.ReadDirPicky(mntDir)
	for _, e := range entries {
		os.RemoveAll(path.Join(mntDir, e.Name()))
	}

	// Prepare standard folders and files
	err := t.createFolders([]string{"sub/", "sub/foo/", "sub/foo/child/", "sub/bar/", "unrelated_dir/"})
	require.NoError(t.T(), err)

	err = t.createObjects(map[string]string{
		"sub/foo/file.txt":       "foo file content",
		"sub/foo/child/file.txt": "child file content",
		"unrelated_dir/file.txt": "unrelated file content",
	})
	require.NoError(t.T(), err)

	// Configure default stall for the actual test execution
	if sb, ok := bucket.(*stallingBucket); ok {
		sb.setStallDuration(2 * time.Second)
		sb.setFailWithErr(nil)
	}
}

func (t *HNSDirectoryRenameLatencyTests) TearDownTest() {
	// Disable stall during teardown to avoid delays
	if sb, ok := bucket.(*stallingBucket); ok {
		sb.setStallDuration(0)
		sb.setFailWithErr(nil)
	}
	t.fsTest.TearDown()
}

// Test 1.1: Concurrent Lookup Blocks on Source Descendants
func (t *HNSDirectoryRenameLatencyTests) TestConcurrentLookupBlocksOnSourceDescendants() {
	src := path.Join(mntDir, "sub", "foo")
	dst := path.Join(mntDir, "sub", "bar_new")
	child := path.Join(src, "child")

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := os.Rename(src, dst)
		assert.NoError(t.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	_, err := os.Stat(child)
	elapsed := time.Since(start)

	assert.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err))
	assert.GreaterOrEqual(t.T(), elapsed.Seconds(), 1.5)

	wg.Wait()

	// Verify post-completion correctness
	_, err = os.Stat(src)
	assert.True(t.T(), os.IsNotExist(err))
	_, err = os.Stat(dst)
	assert.NoError(t.T(), err)
	_, err = os.Stat(path.Join(dst, "child"))
	assert.NoError(t.T(), err)
}

// Test 1.2: Concurrent Lookup Blocks on Destination Descendants
func (t *HNSDirectoryRenameLatencyTests) TestConcurrentLookupBlocksOnDestinationDescendants() {
	src := path.Join(mntDir, "sub", "foo")
	dst := path.Join(mntDir, "sub", "bar_new")
	childDst := path.Join(dst, "child")

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := os.Rename(src, dst)
		assert.NoError(t.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	_, err := os.Stat(childDst)
	elapsed := time.Since(start)

	assert.NoError(t.T(), err)
	assert.GreaterOrEqual(t.T(), elapsed.Seconds(), 1.5)

	wg.Wait()
}

// Test 1.3: Concurrent File Creation Blocks
func (t *HNSDirectoryRenameLatencyTests) TestConcurrentFileCreationBlocks() {
	src := path.Join(mntDir, "sub", "foo")
	dst := path.Join(mntDir, "sub", "bar_new")
	newFile := path.Join(src, "new_file.txt")

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := os.Rename(src, dst)
		assert.NoError(t.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	f, err := os.Create(newFile)
	elapsed := time.Since(start)

	if err == nil {
		f.Close()
	}
	assert.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err))
	assert.GreaterOrEqual(t.T(), elapsed.Seconds(), 1.5)

	wg.Wait()
}

// Test 1.4: Concurrent Directory Creation Blocks
func (t *HNSDirectoryRenameLatencyTests) TestConcurrentDirectoryCreationBlocks() {
	src := path.Join(mntDir, "sub", "foo")
	dst := path.Join(mntDir, "sub", "bar_new")
	newSubdir := path.Join(src, "new_subdir")

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := os.Rename(src, dst)
		assert.NoError(t.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	err := os.Mkdir(newSubdir, dirPerms)
	elapsed := time.Since(start)

	assert.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err))
	assert.GreaterOrEqual(t.T(), elapsed.Seconds(), 1.5)

	wg.Wait()
}

// Test 1.5: Concurrent Unrelated Lookup Executes Immediately
func (t *HNSDirectoryRenameLatencyTests) TestConcurrentUnrelatedLookupExecutesImmediately() {
	src := path.Join(mntDir, "sub", "foo")
	dst := path.Join(mntDir, "sub", "bar_new")
	unrelatedFile := path.Join(mntDir, "unrelated_dir", "file.txt")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := os.Rename(src, dst)
		assert.NoError(t.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	statStart := time.Now()
	_, err := os.Stat(unrelatedFile)
	statElapsed := time.Since(statStart)

	assert.NoError(t.T(), err)
	assert.Less(t.T(), statElapsed.Seconds(), 0.5)

	wg.Wait()
}

// Test 2.1: Lookup on Parent Directory is Unblocked
func (t *HNSDirectoryRenameLatencyTests) TestLookupOnParentDirectoryIsUnblocked() {
	src := path.Join(mntDir, "sub", "foo")
	dst := path.Join(mntDir, "sub", "bar_new")
	unrelated := path.Join(mntDir, "unrelated_dir")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := os.Rename(src, dst)
		assert.NoError(t.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	statStart := time.Now()
	_, err := os.Stat(unrelated)
	statElapsed := time.Since(statStart)

	assert.NoError(t.T(), err)
	assert.Less(t.T(), statElapsed.Seconds(), 0.5)

	wg.Wait()
}

// Test 2.2: Deeply Nested Descendants Block
func (t *HNSDirectoryRenameLatencyTests) TestDeeplyNestedDescendantsBlock() {
	err := t.createFolders([]string{"sub/foo/l1/", "sub/foo/l1/l2/", "sub/foo/l1/l2/l3/"})
	require.NoError(t.T(), err)
	err = t.createObjects(map[string]string{
		"sub/foo/l1/l2/l3/file.txt": "deep content",
	})
	require.NoError(t.T(), err)

	src := path.Join(mntDir, "sub", "foo")
	dst := path.Join(mntDir, "sub", "bar_new")
	deepFile := path.Join(src, "l1", "l2", "l3", "file.txt")

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := os.Rename(src, dst)
		assert.NoError(t.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	_, err = os.Stat(deepFile)
	elapsed := time.Since(start)

	assert.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err))
	assert.GreaterOrEqual(t.T(), elapsed.Seconds(), 1.5)

	wg.Wait()
}

// Test 2.3: Failed Rename Releases Waiters Correctly
func (t *HNSDirectoryRenameLatencyTests) TestFailedRenameReleasesWaitersCorrectly() {
	src := path.Join(mntDir, "sub", "foo")
	dst := path.Join(mntDir, "sub", "bar_new")
	child := path.Join(src, "child")

	if sb, ok := bucket.(*stallingBucket); ok {
		sb.setFailWithErr(os.ErrPermission)
	}

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := os.Rename(src, dst)
		assert.Error(t.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	_, err := os.Stat(child)
	elapsed := time.Since(start)

	assert.NoError(t.T(), err)
	assert.GreaterOrEqual(t.T(), elapsed.Seconds(), 1.5)

	wg.Wait()
}

// Test 3.1: Lexicographical Ordering Deadlock Prevention
func (t *HNSDirectoryRenameLatencyTests) TestLexicographicalOrderingDeadlockPrevention() {
	err := t.createFolders([]string{"sub/A/", "sub/B/"})
	require.NoError(t.T(), err)
	err = t.createObjects(map[string]string{
		"sub/A/file.txt": "A content",
		"sub/B/file.txt": "B content",
	})
	require.NoError(t.T(), err)

	pathA := path.Join(mntDir, "sub", "A")
	pathB := path.Join(mntDir, "sub", "B")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_ = os.Rename(pathA, pathB)
	}()

	go func() {
		defer wg.Done()
		_ = os.Rename(pathB, pathA)
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(10 * time.Second):
		t.T().Fatal("Deadlock detected: Renames did not complete within 10s")
	}
}

// Test 3.2: Overlapping Renames
func (t *HNSDirectoryRenameLatencyTests) TestOverlappingRenames() {
	src := path.Join(mntDir, "sub", "foo")
	dst := path.Join(mntDir, "sub", "bar_new")
	childSrc := path.Join(src, "child")
	childDst := path.Join(src, "child_new")

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		err := os.Rename(src, dst)
		assert.NoError(t.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	var secondRenameElapsed time.Duration
	go func() {
		defer wg.Done()
		secondStart := time.Now()
		err := os.Rename(childSrc, childDst)
		secondRenameElapsed = time.Since(secondStart)
		assert.Error(t.T(), err)
		assert.True(t.T(), os.IsNotExist(err))
	}()

	wg.Wait()
	totalElapsed := time.Since(start)

	assert.GreaterOrEqual(t.T(), secondRenameElapsed.Seconds(), 1.5)
	assert.GreaterOrEqual(t.T(), totalElapsed.Seconds(), 2.0)
}

// Test 4.1: Parallel Heavy Workload Simulation
func (t *HNSDirectoryRenameLatencyTests) TestParallelHeavyWorkloadSimulation() {
	src := path.Join(mntDir, "sub", "foo")
	dst := path.Join(mntDir, "sub", "bar_new")
	unrelatedDir := path.Join(mntDir, "unrelated_dir")

	var wg sync.WaitGroup
	stopUnrelated := make(chan struct{})
	unrelatedCount := 0
	var mu sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-stopUnrelated:
				return
			case <-ticker.C:
				filePath := path.Join(unrelatedDir, fmt.Sprintf("file_%d.txt", i))
				f, err := os.Create(filePath)
				if err == nil {
					_, writeErr := f.Write([]byte("unrelated content"))
					assert.NoError(t.T(), writeErr)
					assert.NoError(t.T(), f.Close())
					_, err = os.Stat(filePath)
					assert.NoError(t.T(), err)
					removeErr := os.Remove(filePath)
					assert.NoError(t.T(), removeErr)
				}
				mu.Lock()
				unrelatedCount++
				mu.Unlock()
				i++
			}
		}
	}()

	time.Sleep(100 * time.Millisecond)
	renameStart := time.Now()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := os.Rename(src, dst)
		assert.NoError(t.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	targetStart := time.Now()
	_, err := os.Stat(path.Join(src, "file.txt"))
	targetElapsed := time.Since(targetStart)

	assert.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err))
	assert.GreaterOrEqual(t.T(), targetElapsed.Seconds(), 1.5)

	close(stopUnrelated)
	wg.Wait()

	renameElapsed := time.Since(renameStart)
	assert.GreaterOrEqual(t.T(), renameElapsed.Seconds(), 2.0)

	mu.Lock()
	count := unrelatedCount
	mu.Unlock()
	assert.GreaterOrEqual(t.T(), count, 50)
}
