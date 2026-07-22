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
	"strconv"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
)

const poolSize = 2000000

var (
	stringPool     []string
	stringPoolOnce sync.Once
)

// getStringPool initializes and returns a global pool of unique strings.
// This is run once for the entire test suite execution.
func getStringPool() []string {
	stringPoolOnce.Do(func() {
		stringPool = make([]string, poolSize)
		for i := 0; i < poolSize; i++ {
			stringPool[i] = "dir/subdir/file" + strconv.Itoa(i)
		}
	})
	return stringPool
}

// ==========================================
// INSERTION BENCHMARKS
// ==========================================

// Benchmark Insertion with inode.Name keys (Static Mount - Baseline)
func Benchmark_MapKey_InodeName_Insert_Static(b *testing.B) {
	pool := getStringPool()
	root := inode.NewRootName("")
	m := make(map[inode.Name]interface{})
	i := 0
	for b.Loop() {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		m[name] = nil
		i++
	}
}

// Benchmark Insertion with inode.Name keys (Dynamic Mount - Baseline)
func Benchmark_MapKey_InodeName_Insert_Dynamic(b *testing.B) {
	pool := getStringPool()
	root := inode.NewRootName("bucket-name")
	m := make(map[inode.Name]interface{})
	i := 0
	for b.Loop() {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		m[name] = nil
		i++
	}
}

// Benchmark Insertion with string keys (Static Mount - Optimized)
func Benchmark_MapKey_String_Insert_Static(b *testing.B) {
	pool := getStringPool()
	root := inode.NewRootName("")
	m := make(map[string]interface{})
	i := 0
	for b.Loop() {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		m[name.LocalName()] = nil
		i++
	}
}

// Benchmark Insertion with string keys (Dynamic Mount - Optimized)
func Benchmark_MapKey_String_Insert_Dynamic(b *testing.B) {
	pool := getStringPool()
	root := inode.NewRootName("bucket-name")
	m := make(map[string]interface{})
	i := 0
	for b.Loop() {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		m[name.LocalName()] = nil
		i++
	}
}

// ==========================================
// LOOKUP BENCHMARKS
// ==========================================

// Benchmark Lookup with inode.Name keys (Static Mount - Baseline)
func Benchmark_MapKey_InodeName_Lookup_Static(b *testing.B) {
	pool := getStringPool()
	root := inode.NewRootName("")
	m := make(map[inode.Name]interface{})
	for i := 0; i < b.N; i++ {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		m[name] = nil
	}

	i := 0
	for b.Loop() {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		_ = m[name]
		i++
	}
}

// Benchmark Lookup with inode.Name keys (Dynamic Mount - Baseline)
func Benchmark_MapKey_InodeName_Lookup_Dynamic(b *testing.B) {
	pool := getStringPool()
	root := inode.NewRootName("bucket-name")
	m := make(map[inode.Name]interface{})
	for i := 0; i < b.N; i++ {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		m[name] = nil
	}

	i := 0
	for b.Loop() {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		_ = m[name]
		i++
	}
}

// Benchmark Lookup with string keys (Static Mount - Optimized)
func Benchmark_MapKey_String_Lookup_Static(b *testing.B) {
	pool := getStringPool()
	root := inode.NewRootName("")
	m := make(map[string]interface{})
	for i := 0; i < b.N; i++ {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		m[name.LocalName()] = nil
	}

	i := 0
	for b.Loop() {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		_ = m[name.LocalName()]
		i++
	}
}

// Benchmark Lookup with string keys (Dynamic Mount - Optimized)
func Benchmark_MapKey_String_Lookup_Dynamic(b *testing.B) {
	pool := getStringPool()
	root := inode.NewRootName("bucket-name")
	m := make(map[string]interface{})
	for i := 0; i < b.N; i++ {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		m[name.LocalName()] = nil
	}

	i := 0
	for b.Loop() {
		name := inode.NewDescendantName(root, pool[i%poolSize])
		_ = m[name.LocalName()]
		i++
	}
}
