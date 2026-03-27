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

package util

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSharedDirLocker_Basic(t *testing.T) {
	// Arrange
	locker := NewSharedDirLocker()
	path := "/test/path" // dummy path, need not actually exist

	// Act & Assert
	// Test ReadLock
	locker.ReadLock(path)
	locker.ReadUnlock(path)
	// Test WriteLock
	locker.WriteLock(path)
	locker.WriteUnlock(path)
}

func TestSharedDirLocker_ConcurrentReaders(t *testing.T) {
	// Arrange
	locker := NewSharedDirLocker()
	path := "/test/path" // dummy path, need not actually exist
	var wg sync.WaitGroup
	numReaders := 10

	// Act
	wg.Add(numReaders)
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			locker.ReadLock(path)
			time.Sleep(1 * time.Millisecond)
			locker.ReadUnlock(path)
		}()
	}

	// Assert
	// This should not hang/deadlock
	wg.Wait()
}

func TestSharedDirLocker_WriterBlocksReaders(t *testing.T) {
	// Arrange
	locker := NewSharedDirLocker()
	path := "/test/path" // dummy path, need not actually exist
	readAcquired := make(chan bool, 1)

	// Act
	locker.WriteLock(path)
	go func() {
		locker.ReadLock(path)
		readAcquired <- true
		locker.ReadUnlock(path)
	}()

	// Assert
	select {
	case <-readAcquired:
		t.Fatal("ReadLock should have been blocked by WriteLock")
	case <-time.After(10 * time.Millisecond):
		// Expected: blocked
	}
	locker.WriteUnlock(path)
	select {
	case <-readAcquired:
		// Success: ReadLock was acquired after WriteUnlock
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ReadLock was not acquired after WriteUnlock")
	}
}

func TestSharedDirLocker_ExclusiveWriter(t *testing.T) {
	// Arrange
	locker := NewSharedDirLocker()
	path := "/test/path" // dummy path, need not actually exist
	writeAcquired := make(chan bool, 1)

	// Act
	locker.WriteLock(path)
	go func() {
		locker.WriteLock(path)
		writeAcquired <- true
		locker.WriteUnlock(path)
	}()

	// Assert
	select {
	case <-writeAcquired:
		t.Fatal("Second WriteLock should have been blocked")
	case <-time.After(10 * time.Millisecond):
		// Expected: blocked
	}
	locker.WriteUnlock(path)
	select {
	case <-writeAcquired:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Second WriteLock was not acquired after first WriteUnlock")
	}
}

func TestSharedDirLocker_StripingDifferentPaths(t *testing.T) {
	// Arrange
	locker := NewSharedDirLocker()
	var path1, path2 string
	// Find two paths that map to different stripes
	for i := 0; ; i++ {
		p := fmt.Sprintf("/path%d", i)
		if path1 == "" {
			path1 = p
			continue
		}
		if locker.getMutex(p) != locker.getMutex(path1) {
			path2 = p
			break
		}
	}
	acquired := make(chan bool, 1)

	// Act
	locker.WriteLock(path1)
	go func() {
		locker.WriteLock(path2)
		acquired <- true
		locker.WriteUnlock(path2)
	}()

	// Assert
	select {
	case <-acquired:
		// Success: different stripes don't block each other
	case <-time.After(100 * time.Millisecond):
		t.Fatal("WriteLock on different stripe was blocked")
	}
	locker.WriteUnlock(path1)
}

func TestSharedDirLocker_StripingSamePath(t *testing.T) {
	// Arrange
	locker := NewSharedDirLocker()
	path := "/some/path" // dummy path, need not actually exist

	// Act & Assert
	assert.Equal(t, locker.getMutex(path), locker.getMutex(path))
}
