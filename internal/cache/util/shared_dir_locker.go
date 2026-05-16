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
	"hash/fnv"
	"sync"
)

// numStripes defines the number of individual read-write mutexes available.
// A prime or near-prime value like 1024 provides a good distribution of hashes
// across the stripes, drastically reducing lock contention for concurrent
// accesses to different directory paths compared to a single global lock.
const numStripes = 1024

// SharedDirLocker manages a set of striped reader-writer locks for cache directories.
// It implements the DirLocker interface defined in internal/util.
//
// Why an array and not a map?
// Using a fixed-size array of mutexes (striped locking) avoids the need to lock
// the data structure itself (as would be required for a map of string -> Mutex).
// This guarantees that locking paths that hash to different stripes is completely
// wait-free with respect to each other, improving concurrency for high-throughput
// scenarios. It also limits the memory footprint.
type SharedDirLocker struct {
	mu [numStripes]sync.RWMutex
}

// NewSharedDirLocker creates and returns a new instance of SharedDirLocker.
func NewSharedDirLocker() *SharedDirLocker {
	return &SharedDirLocker{}
}

func (s *SharedDirLocker) getMutex(path string) *sync.RWMutex {
	// We use the FNV-1a non-cryptographic hash algorithm because it is
	// extremely fast, produces a good avalanche effect (good distribution),
	// and operates directly on the byte representation of the path string.
	h := fnv.New32a()
	h.Write([]byte(path))
	// Modulo the hash sum by the number of stripes to deterministically map
	// the path string to a specific mutex index in the array.
	return &s.mu[h.Sum32()%numStripes]
}

// ReadLock acquires a shared lock for the specified directory path.
// This should be invoked before non-destructive, non-exclusive operations
// within the directory, such as listing its contents, creating files inside it,
// or deleting files inside it. It prevents the directory itself from being removed.
func (s *SharedDirLocker) ReadLock(path string) {
	s.getMutex(path).RLock()
}

// ReadUnlock releases a shared lock for the specified directory path.
func (s *SharedDirLocker) ReadUnlock(path string) {
	s.getMutex(path).RUnlock()
}

// WriteLock acquires an exclusive lock for the specified directory path.
// This should be invoked before operations that alter the state of the directory
// itself, such as deleting the directory or renaming it. It ensures no other
// goroutine is actively listing or writing to it concurrently.
func (s *SharedDirLocker) WriteLock(path string) {
	s.getMutex(path).Lock()
}

// WriteUnlock releases an exclusive lock for the specified directory path.
func (s *SharedDirLocker) WriteUnlock(path string) {
	s.getMutex(path).Unlock()
}
