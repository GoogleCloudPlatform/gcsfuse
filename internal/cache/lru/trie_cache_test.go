// Copyright 2025 Google LLC
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

package lru

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TrieCacheTest struct {
	suite.Suite
	cache *TrieCache
}

func TestTrieCache(t *testing.T) {
	suite.Run(t, new(TrieCacheTest))
}

type testValue struct {
	size uint64
}

func (v testValue) Size() uint64 {
	return v.size
}

func (t *TrieCacheTest) SetupTest() {
	t.cache = NewTrieCache(100)
}

func (t *TrieCacheTest) TestInsertAndLookUp() {
	val := testValue{size: 10}
	evicted, err := t.cache.Insert("key1", val)
	assert.NoError(t.T(), err)
	assert.Empty(t.T(), evicted)

	got := t.cache.LookUp("key1")
	assert.Equal(t.T(), val, got)

	got = t.cache.LookUp("key2")
	assert.Nil(t.T(), got)
}

func (t *TrieCacheTest) TestErase() {
	val := testValue{size: 10}
	t.cache.Insert("key1", val)

	erased := t.cache.Erase("key1")
	assert.Equal(t.T(), val, erased)

	got := t.cache.LookUp("key1")
	assert.Nil(t.T(), got)
}

func (t *TrieCacheTest) TestEraseEntriesWithGivenPrefix() {
	// Insert keys: "a", "a/b", "a/c", "b", "b/d"
	t.cache.Insert("a", testValue{size: 10})
	t.cache.Insert("a/b", testValue{size: 10})
	t.cache.Insert("a/c", testValue{size: 10})
	t.cache.Insert("b", testValue{size: 10})
	t.cache.Insert("b/d", testValue{size: 10})

	// Erase prefix "a"
	t.cache.EraseEntriesWithGivenPrefix("a")

	// "a", "a/b", "a/c" should be gone
	assert.Nil(t.T(), t.cache.LookUp("a"))
	assert.Nil(t.T(), t.cache.LookUp("a/b"))
	assert.Nil(t.T(), t.cache.LookUp("a/c"))

	// "b", "b/d" should remain
	assert.NotNil(t.T(), t.cache.LookUp("b"))
	assert.NotNil(t.T(), t.cache.LookUp("b/d"))
}

func (t *TrieCacheTest) TestEraseEntriesWithGivenPrefix_ExactMatch() {
	t.cache.Insert("apple", testValue{size: 10})
	t.cache.Insert("apple_pie", testValue{size: 10})

	t.cache.EraseEntriesWithGivenPrefix("apple")

	assert.Nil(t.T(), t.cache.LookUp("apple"))
	assert.Nil(t.T(), t.cache.LookUp("apple_pie"))
}

func (t *TrieCacheTest) TestEraseEntriesWithGivenPrefix_NoMatch() {
	t.cache.Insert("apple", testValue{size: 10})

	t.cache.EraseEntriesWithGivenPrefix("banana")

	assert.NotNil(t.T(), t.cache.LookUp("apple"))
}

func (t *TrieCacheTest) TestEraseEntriesWithGivenPrefix_EmptyPrefix() {
	t.cache.Insert("apple", testValue{size: 10})
	t.cache.Insert("banana", testValue{size: 10})

	t.cache.EraseEntriesWithGivenPrefix("")

	assert.Nil(t.T(), t.cache.LookUp("apple"))
	assert.Nil(t.T(), t.cache.LookUp("banana"))
	assert.Equal(t.T(), 0, t.cache.Len())
}

func (t *TrieCacheTest) TestLRUEviction() {
	// Max size 100. Each item 40.
	t.cache = NewTrieCache(100)
	v := testValue{size: 40}

	t.cache.Insert("1", v) // 40
	t.cache.Insert("2", v) // 80
	t.cache.Insert("3", v) // 120 -> evict "1" (LRU)

	assert.Nil(t.T(), t.cache.LookUp("1"))
	assert.NotNil(t.T(), t.cache.LookUp("2"))
	assert.NotNil(t.T(), t.cache.LookUp("3"))
}

func (t *TrieCacheTest) TestLRUEviction_WithPrefixErase() {
	// Ensure prefix erase updates LRU list correctly (removes from list)
	t.cache = NewTrieCache(100)
	v := testValue{size: 10}

	t.cache.Insert("a", v)
	t.cache.Insert("b", v)
	t.cache.Insert("c", v)

	t.cache.EraseEntriesWithGivenPrefix("b")

	assert.NotNil(t.T(), t.cache.LookUp("a"))
	assert.Nil(t.T(), t.cache.LookUp("b"))
	assert.NotNil(t.T(), t.cache.LookUp("c"))

	// Check internal consistency if possible, or just rely on no panic/correct behavior
	assert.Equal(t.T(), 2, t.cache.Len())
}
