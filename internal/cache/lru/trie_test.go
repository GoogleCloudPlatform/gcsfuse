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

package lru

import (
	"container/list"
	"reflect"
	"sort"
	"testing"
)

func TestSplitPath(t *testing.T) {
	// Arrange
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{"empty", "", nil},
		{"root", "/", nil},
		{"dot", ".", nil},
		{"single", "a", []string{"a"}},
		{"simple", "a/b/c", []string{"a", "b", "c"}},
		{"leading slash", "/a/b/c", []string{"a", "b", "c"}},
		{"trailing slash", "a/b/c/", []string{"a", "b", "c"}},
		{"redundant slashes", "a//b///c", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			actual := splitPath(tt.path)
			// Assert
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("splitPath(%q) = %v, expected %v", tt.path, actual, tt.expected)
			}
		})
	}
}

func TestFileCacheTrieIndexer_SetAndGet(t *testing.T) {
	// Arrange
	ti := NewFileCacheTrieIndexer(nil)
	l := list.New()
	elem1 := l.PushBack(1)

	// Act
	ti.Set("a/b/c.txt", elem1)
	val, ok := ti.Get("a/b/c.txt")

	// Assert
	if !ok || val != elem1 {
		t.Errorf("Get(a/b/c.txt) = %v, %v; expected %v, true", val, ok, elem1)
	}
	if ti.Len() != 1 {
		t.Errorf("Len() = %d, expected 1", ti.Len())
	}
}

func TestFileCacheTrieIndexer_SetOverwrite(t *testing.T) {
	// Arrange
	ti := NewFileCacheTrieIndexer(nil)
	l := list.New()
	elem1 := l.PushBack(1)
	elem2 := l.PushBack(2)
	ti.Set("a/b/c.txt", elem1)

	// Act
	ti.Set("a/b/c.txt", elem2)
	val, ok := ti.Get("a/b/c.txt")

	// Assert
	if !ok || val != elem2 {
		t.Errorf("Get(a/b/c.txt) after overwrite = %v, %v; expected %v, true", val, ok, elem2)
	}
	if ti.Len() != 1 {
		t.Errorf("Len() = %d, expected 1", ti.Len())
	}
}

func TestFileCacheTrieIndexer_GetDirectoryFails(t *testing.T) {
	// Arrange
	ti := NewFileCacheTrieIndexer(nil)
	l := list.New()
	elem1 := l.PushBack(1)
	ti.Set("a/b/c.txt", elem1)

	// Act
	val, ok := ti.Get("a/b")

	// Assert
	if ok || val != nil {
		t.Errorf("Get(a/b) = %v, %v; expected nil, false", val, ok)
	}
}

func TestFileCacheTrieIndexer_Delete(t *testing.T) {
	// Arrange
	ti := NewFileCacheTrieIndexer(nil)
	l := list.New()
	elem1 := l.PushBack(1)
	ti.Set("a/b/c.txt", elem1)

	// Act
	ti.Delete("a/b/c.txt")
	val, ok := ti.Get("a/b/c.txt")

	// Assert
	if ok || val != nil {
		t.Errorf("Get(a/b/c.txt) after delete = %v, %v; expected nil, false", val, ok)
	}
	if ti.Len() != 0 {
		t.Errorf("Len() after delete = %d, expected 0", ti.Len())
	}
}

func TestFileCacheTrieIndexer_OnEmptyDir_PartialDelete(t *testing.T) {
	// Arrange
	emptyDirs := []string{}
	ti := NewFileCacheTrieIndexer(func(dirPath string) {
		emptyDirs = append(emptyDirs, dirPath)
	})
	l := list.New()
	elem1 := l.PushBack(1)
	elem2 := l.PushBack(2)
	ti.Set("a/b/c/f1.txt", elem1)
	ti.Set("a/b/f2.txt", elem2)

	// Act
	ti.Delete("a/b/c/f1.txt")

	// Assert
	expected := []string{"a/b/c"}
	if !reflect.DeepEqual(emptyDirs, expected) {
		t.Errorf("emptyDirs = %v, expected %v", emptyDirs, expected)
	}
}

func TestFileCacheTrieIndexer_OnEmptyDir_RecursiveDelete(t *testing.T) {
	// Arrange
	emptyDirs := []string{}
	ti := NewFileCacheTrieIndexer(func(dirPath string) {
		emptyDirs = append(emptyDirs, dirPath)
	})
	l := list.New()
	elem2 := l.PushBack(2)
	ti.Set("a/b/f2.txt", elem2)

	// Act
	// 1. Evict the file. This should signal the immediate parent "a/b".
	ti.Delete("a/b/f2.txt")

	// 2. Simulate the background worker successfully deleting "a/b" from disk
	// and calling back into the LRU to delete the directory node.
	// This should in turn signal the grandparent "a".
	ti.DeleteDirIfEmpty("a/b")

	// Assert
	expected := []string{"a/b", "a"}
	if !reflect.DeepEqual(emptyDirs, expected) {
		t.Errorf("emptyDirs = %v, expected %v", emptyDirs, expected)
	}
}

func TestFileCacheTrieIndexer_KeysWithPrefix(t *testing.T) {
	// Arrange
	ti := NewFileCacheTrieIndexer(nil)
	l := list.New()
	elem := l.PushBack(1)
	ti.Set("a/b/c/f1.txt", elem)
	ti.Set("a/b/c/f2.txt", elem)
	ti.Set("a/b/d/f3.txt", elem)
	ti.Set("a/x.txt", elem)
	tests := []struct {
		prefix   string
		expected []string
	}{
		{"", []string{"a/b/c/f1.txt", "a/b/c/f2.txt", "a/b/d/f3.txt", "a/x.txt"}},
		{"a/b/c", []string{"a/b/c/f1.txt", "a/b/c/f2.txt"}},
		{"a/b/c/", []string{"a/b/c/f1.txt", "a/b/c/f2.txt"}},
		{"a/b/c/f", []string{"a/b/c/f1.txt", "a/b/c/f2.txt"}},
		{"a/b/c/f1", []string{"a/b/c/f1.txt"}},
		{"a/b/d", []string{"a/b/d/f3.txt"}},
		{"a", []string{"a/b/c/f1.txt", "a/b/c/f2.txt", "a/b/d/f3.txt", "a/x.txt"}},
		{"z", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			// Act
			actual := ti.KeysWithPrefix(tt.prefix)
			// Assert
			sort.Strings(actual)
			sort.Strings(tt.expected)
			if !reflect.DeepEqual(actual, tt.expected) {
				if len(actual) == 0 && len(tt.expected) == 0 {
					return
				}
				t.Errorf("KeysWithPrefix(%q) = %v, expected %v", tt.prefix, actual, tt.expected)
			}
		})
	}
}

func assertPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Test %q did not panic as expected", name)
		}
	}()
	f()
}

func TestFileCacheTrieIndexer_PanicOnIntermediateFile_Set(t *testing.T) {
	// Arrange
	ti := NewFileCacheTrieIndexer(nil)
	l := list.New()
	elem := l.PushBack(1)
	ti.Set("a/b/c.txt", elem)

	// Act & Assert
	assertPanic(t, "Set", func() {
		ti.Set("a/b/c.txt/d.txt", elem)
	})
}

func TestFileCacheTrieIndexer_PanicOnIntermediateFile_Get(t *testing.T) {
	// Arrange
	ti := NewFileCacheTrieIndexer(nil)
	l := list.New()
	elem := l.PushBack(1)
	ti.Set("a/b/c.txt", elem)

	// Act & Assert
	assertPanic(t, "Get", func() {
		ti.Get("a/b/c.txt/d.txt")
	})
}

func TestFileCacheTrieIndexer_PanicOnIntermediateFile_Delete(t *testing.T) {
	// Arrange
	ti := NewFileCacheTrieIndexer(nil)
	l := list.New()
	elem := l.PushBack(1)
	ti.Set("a/b/c.txt", elem)

	// Act & Assert
	assertPanic(t, "Delete", func() {
		ti.Delete("a/b/c.txt/d.txt")
	})
}

func TestFileCacheTrieIndexer_PanicOnIntermediateFile_KeysWithPrefix(t *testing.T) {
	// Arrange
	ti := NewFileCacheTrieIndexer(nil)
	l := list.New()
	elem := l.PushBack(1)
	ti.Set("a/b/c.txt", elem)

	// Act & Assert
	assertPanic(t, "KeysWithPrefix", func() {
		ti.KeysWithPrefix("a/b/c.txt/d")
	})
}
