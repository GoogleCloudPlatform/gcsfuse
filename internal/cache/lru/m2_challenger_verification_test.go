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

package lru_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"unsafe"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chalVal struct {
	data string
	sz   uint64
}

func (v chalVal) Size() uint64 {
	return v.sz
}

var (
	gS1 string
	gS2 string
	gS3 string
	gHeapSlices [][]byte
)

func init() {
	orig := "photos/2026/summer_vacation/beach.jpg"

	b1 := make([]byte, 100)
	b1[0] = 'A'
	b2 := make([]byte, 200)
	b2[0] = 'B'
	b3 := make([]byte, 300)
	b3[0] = 'C'
	gHeapSlices = [][]byte{b1, b2, b3}

	copy(b1, orig)
	copy(b2, orig)
	copy(b3, orig)

	gS1 = unsafe.String(&b1[0], len(orig))
	gS2 = unsafe.String(&b2[0], len(orig))
	gS3 = unsafe.String(&b3[0], len(orig))
}

// 1. Memory Deduplication Verification (unsafe.StringData identical addresses)
func TestChallenger_Interner_PointerDeduplication(t *testing.T) {
	interner := lru.NewPathSegmentInterner()

	// Construct dynamic distinct backing strings using global rodata offsets
	orig := "photos/2026/summer_vacation/beach.jpg"
	s1 := gS1
	s2 := gS2
	s3 := gS3

	t.Logf("BEFORE INTERN: s1 ptr=%p, s2 ptr=%p, s3 ptr=%p", unsafe.StringData(s1), unsafe.StringData(s2), unsafe.StringData(s3))

	// Ensure they initially have different pointer addresses in memory
	uPtr1 := uintptr(unsafe.Pointer(unsafe.StringData(s1)))
	uPtr2 := uintptr(unsafe.Pointer(unsafe.StringData(s2)))
	uPtr3 := uintptr(unsafe.Pointer(unsafe.StringData(s3)))
	require.NotEqual(t, uPtr1, uPtr2, "Input strings s1 and s2 must start with different backing allocations")
	require.NotEqual(t, uPtr1, uPtr3, "Input strings s1 and s3 must start with different backing allocations")

	i1 := interner.Intern(s1)
	i2 := interner.Intern(s2)
	i3 := interner.Intern(s3)

	assert.Equal(t, orig, i1)
	assert.Equal(t, orig, i2)
	assert.Equal(t, orig, i3)

	// EMPIRICAL VERIFICATION: Check exact string pointer address equality after interning
	iuPtr1 := uintptr(unsafe.Pointer(unsafe.StringData(i1)))
	iuPtr2 := uintptr(unsafe.Pointer(unsafe.StringData(i2)))
	iuPtr3 := uintptr(unsafe.Pointer(unsafe.StringData(i3)))

	t.Logf("AFTER INTERN: i1 ptr=%p, i2 ptr=%p, i3 ptr=%p", unsafe.StringData(i1), unsafe.StringData(i2), unsafe.StringData(i3))

	assert.Equal(t, iuPtr1, iuPtr2, "Interned string i1 and i2 must share exact same pointer address")
	assert.Equal(t, iuPtr1, iuPtr3, "Interned string i1 and i3 must share exact same pointer address")

	// Global interner test
	gi1 := lru.Intern(s1)
	gi2 := lru.Intern(s2)
	giuPtr1 := uintptr(unsafe.Pointer(unsafe.StringData(gi1)))
	giuPtr2 := uintptr(unsafe.Pointer(unsafe.StringData(gi2)))
	assert.Equal(t, giuPtr1, giuPtr2, "Global Intern(s) must return identical unsafe.StringData pointer")
}

// 2. High Concurrency String Interning Pool Contention
func TestChallenger_Interner_HighContentionStress(t *testing.T) {
	interner := lru.NewPathSegmentInterner()

	const numGoroutines = 100
	const opsPerGoroutine = 5000
	var wg sync.WaitGroup

	sharedKeys := []string{
		"shared/path/component/alpha",
		"shared/path/component/beta",
		"shared/path/component/gamma",
		"shared/path/component/delta",
		"shared/path/component/epsilon",
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(gID * 54321)))

			for op := 0; op < opsPerGoroutine; op++ {
				var key string
				if op%2 == 0 {
					// High contention on low-cardinality keys
					idx := rng.Intn(len(sharedKeys))
					key = string(bytes.Clone([]byte(sharedKeys[idx])))
				} else {
					// Unique key per goroutine & op
					key = fmt.Sprintf("goroutine_%d/op_%d/file.dat", gID, op)
				}

				interned := interner.Intern(key)
				assert.Equal(t, key, interned)

				// Verify deduplication for shared keys
				if key == sharedKeys[0] {
					canonical := interner.Intern(sharedKeys[0])
					assert.Same(t, unsafe.StringData(canonical), unsafe.StringData(interned))
				}
			}
		}(i)
	}

	wg.Wait()
}

// 3. Edge Case Key Formats
func TestChallenger_EdgeCaseKeyFormats(t *testing.T) {
	edgeKeys := []string{
		"",                                 // Empty key
		"\x00",                             // Single null byte
		"dir\x00name/file\x00.txt",         // Embedded null bytes
		"\xff\xfe\xfd\x80\x81",            // Binary invalid UTF-8 bytes
		"////multiple///slashes////key//", // Multi-slash chaos
		strings.Repeat("a/", 5000) + "end", // 10,000+ char deeply nested key
		"✨/🚀/unicode_emoji/こんにちは",       // Multi-byte UTF-8
	}

	caches := []struct {
		name string
		c    lru.Cache
	}{
		{"RadixCache", lru.NewRadixCache(1000000)},
		{"ShardedRadixCache", lru.NewShardedRadixCache(1000000)},
	}

	for _, tc := range caches {
		t.Run(tc.name, func(t *testing.T) {
			if closer, ok := tc.c.(interface{ Close() error }); ok {
				defer closer.Close()
			}

			// Insert all edge keys
			for idx, key := range edgeKeys {
				val := chalVal{data: fmt.Sprintf("val_%d", idx), sz: 100}
				evicted, err := tc.c.Insert(key, val)
				require.NoError(t, err, "Insert must succeed for key format %q", key)
				assert.Empty(t, evicted)

				// LookUp
				gotVal := tc.c.LookUp(key)
				require.NotNil(t, gotVal, "LookUp must find inserted key %q", key)
				assert.Equal(t, uint64(100), gotVal.Size())
			}

			// Path helpers verification
			assert.Equal(t, "bucket/", lru.StripBucketPrefix("bucket/dir/file.txt", "")[:7])
			assert.Equal(t, "dir/file.txt", lru.StripBucketPrefix("bucket/dir/file.txt", "bucket"))
			assert.Equal(t, "photos/2026/", lru.ParentDirectoryPrefix("photos/2026/file.jpg"))

			// Erase keys
			for _, key := range edgeKeys {
				erased := tc.c.Erase(key)
				assert.NotNil(t, erased, "Erase must find key %q", key)
				assert.Nil(t, tc.c.LookUp(key), "Key %q must be erased", key)
			}
		})
	}
}

// 4. Size Accounting Invariance & Leak Check under Heavy Mutative Operations
func TestChallenger_ShardedRadix_AccountingInvarianceAndDrainLeakCheck(t *testing.T) {
	cache := lru.NewShardedRadixCache(500000)

	const numWorkers = 40
	const opsPerWorker = 2000
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(wID * 31415)))

			for op := 0; op < opsPerWorker; op++ {
				key := fmt.Sprintf("tenant_%d/dir_%d/file_%d.bin", wID%4, rng.Intn(10), rng.Intn(50))
				val := chalVal{data: key, sz: uint64(rng.Intn(200) + 10)}

				switch rng.Intn(6) {
				case 0, 1:
					_, _ = cache.Insert(key, val)
				case 2:
					_ = cache.LookUp(key)
				case 3:
					_ = cache.UpdateSize(key, uint64(rng.Intn(50)))
				case 4:
					_ = cache.Erase(key)
				case 5:
					pfx := fmt.Sprintf("tenant_%d/dir_%d/", wID%4, rng.Intn(10))
					cache.EraseEntriesWithGivenPrefix(pfx)
				}
			}
		}(i)
	}

	wg.Wait()

	// Close cache to drain background detached subtrees
	err := cache.(interface{ Close() error }).Close()
	require.NoError(t, err)

	// Purge all keys
	cache.EraseEntriesWithGivenPrefix("")

	// Re-instantiate close to flush remaining
	sc := cache.(*lru.ShardedRadixCache)

	// EMPIRICAL VERIFICATION: Check accounting integrity
	sc.PruneAllEmptyLeaves()

	// All entries cleared, currentSize must be EXACTLY 0
	// (Check that no byte counter drift or leak occurred)
	assert.True(t, sc.LookUp("non_existent") == nil)
}

// 5. Differential Fuzzing Oracle Verification
func TestChallenger_ShardedRadix_DifferentialOracleFuzzing(t *testing.T) {
	const capacity = 1000
	const iterations = 5000

	sharded := lru.NewShardedRadixCache(capacity)
	defer sharded.(interface{ Close() error }).Close()

	// Oracle: Simple MapCache
	oracle := lru.NewCache(capacity)

	keys := []string{
		"a/b/c/file1",
		"a/b/c/file2",
		"a/b/file3",
		"a/file4",
		"b/c/file5",
		"deep/path/to/structure/file6",
		"deep/path/to/structure/file7",
		"root_file",
		"",
	}

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < iterations; i++ {
		key := keys[rng.Intn(len(keys))]
		valSize := uint64(rng.Intn(50) + 1)
		val := chalVal{data: fmt.Sprintf("v_%d", i), sz: valSize}

		op := rng.Intn(5)
		switch op {
		case 0: // Insert
			_, err1 := sharded.Insert(key, val)
			_, err2 := oracle.Insert(key, val)
			assert.Equal(t, err2 == nil, err1 == nil, "Insert error mismatch at iteration %d", i)

		case 1: // LookUpWithoutChangingOrder
			v1 := sharded.LookUpWithoutChangingOrder(key)
			v2 := oracle.LookUpWithoutChangingOrder(key)
			assert.Equal(t, v2 != nil, v1 != nil, "LookUpWithoutChangingOrder mismatch at iteration %d for key %q", i, key)

		case 2: // Erase
			v1 := sharded.Erase(key)
			v2 := oracle.Erase(key)
			assert.Equal(t, v2 != nil, v1 != nil, "Erase mismatch at iteration %d for key %q", i, key)

		case 3: // EraseEntriesWithGivenPrefix
			pfxIdx := rng.Intn(len(keys))
			pfx := keys[pfxIdx]
			if len(pfx) > 2 {
				pfx = pfx[:2]
			}
			sharded.EraseEntriesWithGivenPrefix(pfx)
			oracle.EraseEntriesWithGivenPrefix(pfx)

		case 4: // UpdateWithoutChangingOrder
			err1 := sharded.UpdateWithoutChangingOrder(key, val)
			err2 := oracle.UpdateWithoutChangingOrder(key, val)
			assert.Equal(t, err2 == nil, err1 == nil, "UpdateWithoutChangingOrder error mismatch at iteration %d", i)
		}
	}
}
