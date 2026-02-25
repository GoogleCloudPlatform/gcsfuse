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

const numStripes = 1024

// SharedDirLocker manages a set of striped reader-writer locks for cache directories.
// It implements the DirLocker interface defined in internal/util.
type SharedDirLocker struct {
	mu [numStripes]sync.RWMutex
}

// NewSharedDirLocker creates and returns a new instance of SharedDirLocker.
func NewSharedDirLocker() *SharedDirLocker {
	return &SharedDirLocker{}
}

func (s *SharedDirLocker) getLocker(path string) *sync.RWMutex {
	h := fnv.New32a()
	h.Write([]byte(path))
	return &s.mu[h.Sum32()%numStripes]
}

// ReadLock acquires a shared lock for the specified directory path.
func (s *SharedDirLocker) ReadLock(path string) {
	s.getLocker(path).RLock()
}

// ReadUnlock releases a shared lock for the specified directory path.
func (s *SharedDirLocker) ReadUnlock(path string) {
	s.getLocker(path).RUnlock()
}

// WriteLock acquires an exclusive lock for the specified directory path.
func (s *SharedDirLocker) WriteLock(path string) {
	s.getLocker(path).Lock()
}

// WriteUnlock releases an exclusive lock for the specified directory path.
func (s *SharedDirLocker) WriteUnlock(path string) {
	s.getLocker(path).Unlock()
}
