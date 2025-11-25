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

package folder

import (
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestTrie(t *testing.T) { RunTests(t) }

type TrieTest struct {
}

func init() { RegisterTestSuite(&TrieTest{}) }

func (t *TrieTest) TestNewTrie() {
	trie := NewTrie()
	ExpectNe(nil, trie)
	ExpectNe(nil, trie.root)
	ExpectEq(0, len(trie.root.children))
	ExpectEq(nil, trie.root.file)
	ExpectEq(0, trie.CountLeafs())
}

func (t *TrieTest) TestCountFiles() {
	trie := NewTrie()
	ExpectEq(0, trie.CountFiles())

	// Insert some files
	trie.Insert("/a/b", &FileInfo{size: 1})
	trie.Insert("/a/c", &FileInfo{size: 2})
	ExpectEq(2, trie.CountFiles())

	// Insert a directory (should not be counted as a file)
	trie.InsertDir("/a/d")
	ExpectEq(2, trie.CountFiles())

	// Overwrite a file (count should not change)
	trie.Insert("/a/b", &FileInfo{size: 3})
	ExpectEq(2, trie.CountFiles())

	// Delete a file
	trie.Delete("/a/c")
	ExpectEq(1, trie.CountFiles())
}

func (t *TrieTest) TestPruneEmptyPath() {
	trie := NewTrie()

	// Setup: Create a deep path and a diverging path to test partial pruning.
	deepPath := "/a/b/c/d/e"
	divergingPath := "/a/b/x"
	trie.Insert(deepPath, &FileInfo{size: 1})
	trie.Insert(divergingPath, &FileInfo{size: 2})

	ExpectTrue(trie.PathExists(deepPath))
	ExpectTrue(trie.PathExists(divergingPath))
	ExpectEq(2, trie.CountLeafs())

	// Action 1: Delete the deep path. This should trigger pruneEmptyPath.
	trie.Delete(deepPath)

	// Assertions 1:
	// The deep path and its unique parents (/c, /d, /e) should be pruned.
	ExpectFalse(trie.PathExists(deepPath))
	ExpectFalse(trie.PathExists("/a/b/c/d"))
	ExpectFalse(trie.PathExists("/a/b/c"))

	// The shared path (/a/b) should NOT be pruned because of the diverging path.
	ExpectTrue(trie.PathExists("/a/b"))
	ExpectTrue(trie.PathExists(divergingPath))
	ExpectEq(1, trie.CountLeafs())

	// Action 2: Delete the diverging path. This should prune the rest of the branch.
	trie.Delete(divergingPath)

	// Assertions 2:
	// The entire /a branch should now be gone.
	ExpectFalse(trie.PathExists(divergingPath))
	ExpectFalse(trie.PathExists("/a/b"))
	ExpectFalse(trie.PathExists("/a"))
	ExpectEq(0, trie.CountLeafs())
}

func (t *TrieTest) TestInsertAndGet() {
	trie := NewTrie()
	path1 := "/a/b/c"
	file1 := &FileInfo{size: 10}
	path2 := "/a/b/d"
	file2 := &FileInfo{size: 20}

	// Insert and Get path1
	trie.Insert(path1, file1)
	val, ok := trie.Get(path1)
	ExpectTrue(ok)
	ExpectEq(file1, val)
	ExpectEq(1, trie.CountLeafs())

	// Insert and Get path2
	trie.Insert(path2, file2)
	val, ok = trie.Get(path2)
	ExpectTrue(ok)
	ExpectEq(file2, val)
	ExpectEq(2, trie.CountLeafs())

	// Get non-existent path
	val, ok = trie.Get("/a/b/x")
	ExpectFalse(ok)
	ExpectEq(nil, val)

	// Get prefix path which is not a leaf
	val, ok = trie.Get("/a/b")
	ExpectFalse(ok)
	ExpectEq(nil, val)

	// Overwrite existing file
	newFile1 := &FileInfo{size: 11}
	trie.Insert(path1, newFile1)
	val, ok = trie.Get(path1)
	ExpectTrue(ok)
	ExpectEq(newFile1, val)
	ExpectEq(2, trie.CountLeafs()) // Count should not change on overwrite
}

func (t *TrieTest) TestInsertDirAndPathExists() {
	trie := NewTrie()
	trie.InsertDir("/a/b")

	// Directory should exist
	ExpectTrue(trie.PathExists("/a/b"))
	ExpectTrue(trie.PathExists("/a"))

	// It should not be a file
	_, ok := trie.Get("/a/b")
	ExpectFalse(ok)

	// File count should be zero
	ExpectEq(0, trie.CountLeafs())

	// Path that doesn't exist
	ExpectFalse(trie.PathExists("/a/c"))

	// Insert a file inside the directory
	trie.Insert("/a/b/c", &FileInfo{size: 1})
	ExpectTrue(trie.PathExists("/a/b/c"))
	ExpectEq(1, trie.CountLeafs())
}

func (t *TrieTest) TestDelete() {
	trie := NewTrie()
	trie.Insert("/a/b/c", &FileInfo{size: 1})
	trie.Insert("/a/b/d", &FileInfo{size: 2})
	trie.Insert("/a/e", &FileInfo{size: 3})
	ExpectEq(3, trie.CountLeafs())

	// Delete a leaf node, which should prune empty parents
	trie.Delete("/a/b/c")
	_, ok := trie.Get("/a/b/c")
	ExpectFalse(ok)
	ExpectEq(2, trie.CountLeafs())

	// Sibling should still exist
	_, ok = trie.Get("/a/b/d")
	ExpectTrue(ok)

	// Delete a path that is a prefix to another path (but not a leaf)
	trie.InsertDir("/a/b")
	trie.Delete("/a/b") // This should do nothing as it's not a leaf
	_, ok = trie.Get("/a/b/d")
	ExpectTrue(ok) // Child should still exist
	ExpectEq(2, trie.CountLeafs())

	// Delete the rest of the /a/b branch
	trie.Delete("/a/b/d")
	_, ok = trie.Get("/a/b/d")
	ExpectFalse(ok)
	ExpectEq(1, trie.CountLeafs())

	// Node /a/b should be pruned now
	ExpectFalse(trie.PathExists("/a/b"))

	// but /a should still exist because of /a/e
	ExpectTrue(trie.PathExists("/a"))
	_, ok = trie.Get("/a/e")
	ExpectTrue(ok)

	// Delete non-existent path
	trie.Delete("/x/y/z") // no-op, should not panic
	ExpectEq(1, trie.CountLeafs())
}

func (t *TrieTest) TestDeleteFile() {
	trie := NewTrie()
	fileInfo := &FileInfo{size: 123}
	trie.Insert("/a/b/c", fileInfo)
	trie.Insert("/a/b/d", &FileInfo{size: 456})
	ExpectEq(2, trie.CountLeafs())

	// Delete a file and check returned data
	deletedFile, ok := trie.DeleteFile("/a/b/c")
	ExpectTrue(ok)
	ExpectEq(fileInfo, deletedFile)
	ExpectEq(1, trie.CountLeafs())

	// Verify the file is gone
	_, ok = trie.Get("/a/b/c")
	ExpectFalse(ok)

	// Verify parent node is not pruned
	ExpectTrue(trie.PathExists("/a/b"))

	// Verify sibling is not affected
	_, ok = trie.Get("/a/b/d")
	ExpectTrue(ok)

	// Delete non-existent file
	deletedFile, ok = trie.DeleteFile("/x/y/z")
	ExpectFalse(ok)
	ExpectEq(nil, deletedFile)
	ExpectEq(1, trie.CountLeafs())
}

func (t *TrieTest) TestListPathsWithPrefix() {
	trie := NewTrie()
	trie.Insert("/a/b/c", &FileInfo{})
	trie.Insert("/a/b/d", &FileInfo{})
	trie.Insert("/a/e", &FileInfo{})
	trie.Insert("/a", &FileInfo{}) // Prefix itself is a file
	trie.Insert("/f/g", &FileInfo{})

	// List with prefix /a/b
	paths := trie.ListPathsWithPrefix("/a/b")
	sort.Strings(paths)
	expected := []string{"/a/b/c", "/a/b/d"}
	ExpectEq(strings.Join(expected, ","), strings.Join(paths, ","))

	// List with prefix /a
	paths = trie.ListPathsWithPrefix("/a")
	sort.Strings(paths)
	expected = []string{"/a", "/a/b/c", "/a/b/d", "/a/e"}
	ExpectEq(strings.Join(expected, ","), strings.Join(paths, ","))

	// List all paths
	paths = trie.ListPathsWithPrefix("")
	sort.Strings(paths)
	expected = []string{"/a", "/a/b/c", "/a/b/d", "/a/e", "/f/g"}
	ExpectEq(strings.Join(expected, ","), strings.Join(paths, ","))

	// List with non-existent prefix
	paths = trie.ListPathsWithPrefix("/x")
	ExpectEq(nil, paths)
}

func (t *TrieTest) TestConcurrentInsert() {
	trie := NewTrie()
	var wg sync.WaitGroup
	numRoutines := 1000

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			path := fmt.Sprintf("/path/%d", i)
			fileInfo := &FileInfo{size: int64(i)}
			trie.Insert(path, fileInfo)
		}(i)
	}

	wg.Wait()

	ExpectEq(numRoutines, trie.CountLeafs())
	for i := 0; i < numRoutines; i++ {
		path := fmt.Sprintf("/path/%d", i)
		val, ok := trie.Get(path)
		ExpectTrue(ok)
		ExpectEq(int64(i), val.size)
	}
}

func (t *TrieTest) TestConcurrentInsertDelete() {
	trie := NewTrie()
	var wg sync.WaitGroup
	numRoutines := 4

	// Insert all paths first
	for i := 0; i < numRoutines; i++ {
		path := fmt.Sprintf("/path/%d", i)
		trie.Insert(path, &FileInfo{size: int64(i)})
	}
	ExpectEq(numRoutines, trie.CountLeafs())

	// Concurrently delete half of them
	for i := 0; i < numRoutines/2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			path := fmt.Sprintf("/path/%d", i)
			trie.Delete(path)
		}(i)
	}
	wg.Wait()

	ExpectEq(numRoutines/2, trie.CountLeafs())

	// Concurrently delete the other half with DeleteFile
	for i := numRoutines / 2; i < numRoutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			path := fmt.Sprintf("/path/%d", i)
			trie.DeleteFile(path)
		}(i)
	}
	wg.Wait()

	ExpectEq(0, trie.CountLeafs())
	// Root's child "path" should be pruned by Delete but not DeleteFile
	// Since Delete was used, it should be gone.
	ExpectTrue(trie.PathExists("/path"))

	// Concurrently delete the other half with DeleteFile
	for i := numRoutines / 2; i < numRoutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			path := fmt.Sprintf("/path/%d", i)
			trie.Delete(path)
		}(i)
	}
	wg.Wait()

	// Root's child "path" should be pruned by Delete but not DeleteFile
	// Since Delete was used, it should be gone.
	ExpectFalse(trie.PathExists("/path"))

}

func (t *TrieTest) TestMove() {
	trie := NewTrie()

	trie.Insert("/a/b/c", &FileInfo{size: 1})
	trie.Insert("/a/b/d", &FileInfo{size: 2})
	trie.InsertDir("/x/y")
	ExpectEq(2, trie.CountLeafs())

	// Move /a/b to /x/y/z
	ok := trie.Move("/a/b", "/x/y/z")
	ExpectTrue(ok)
	ExpectEq(2, trie.CountLeafs()) // Count should not change

	// Check new paths
	ExpectTrue(trie.PathExists("/x/y/z"))
	_, ok = trie.Get("/x/y/z/c")
	ExpectTrue(ok)
	_, ok = trie.Get("/x/y/z/d")
	ExpectTrue(ok)

	// Check old paths are gone
	ExpectFalse(trie.PathExists("/a/b"))
	ExpectFalse(trie.PathExists("/a/b/c"))

	//a should be pruned
	ExpectFalse(trie.PathExists("/a"))

	// fmt.Println(trie.root.ToString())
	// fmt.Println(trie.root.children["x"].ToString())

	// Move a single file
	trie.Insert("/p/q", &FileInfo{size: 3})
	ExpectEq(3, trie.CountLeafs())
	ok = trie.Move("/p/q", "/p/r")
	ExpectTrue(ok)
	ExpectEq(3, trie.CountLeafs())
	_, ok = trie.Get("/p/r")
	ExpectTrue(ok)
	ExpectFalse(trie.PathExists("/p/q"))
	// /p/q's parent /p should not be pruned as /p/r exists
	ExpectTrue(trie.PathExists("/p"))

	// Invalid moves
	ExpectFalse(trie.Move("/x/y/z", "/x/y/z/c")) // move into self
	ExpectFalse(trie.Move("/non/existent", "/new/path"))
	ExpectFalse(trie.Move("/p/r", "/x/y/z/d")) // destination exists
}

func (t *TrieTest) TestConcurrentMove() {
	trie := NewTrie()
	var wg sync.WaitGroup
	numRoutines := 100

	// Insert initial paths
	for i := 0; i < numRoutines; i++ {
		trie.Insert(fmt.Sprintf("/src/%d", i), &FileInfo{size: int64(i)})
	}
	trie.InsertDir("/dest")
	ExpectEq(numRoutines, trie.CountLeafs())

	// Concurrently move them
	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sourcePath := fmt.Sprintf("/src/%d", i)
			destPath := fmt.Sprintf("/dest/%d", i)
			trie.Move(sourcePath, destPath)
		}(i)
	}
	wg.Wait()

	ExpectEq(numRoutines, trie.CountLeafs())
	ExpectFalse(trie.PathExists("/src")) // Should be pruned
	ExpectTrue(trie.PathExists("/dest"))

	for i := 0; i < numRoutines; i++ {
		_, ok := trie.Get(fmt.Sprintf("/dest/%d", i))
		ExpectTrue(ok)
	}
}

func BenchmarkInsertOverlappingPath(b *testing.B) {
	const numFolders = 100
	const minFiles = 100
	const maxFiles = 500

	// Generate the file paths once to avoid this in the benchmark loop.
	var paths []string
	var currentPath string
	for i := 0; i < numFolders; i++ {
		currentPath += fmt.Sprintf("/dir%d", i)
		numFiles := rand.Intn(maxFiles-minFiles+1) + minFiles
		for j := 0; j < numFiles; j++ {
			paths = append(paths, fmt.Sprintf("%s/file%d", currentPath, j))
		}
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		trie := NewTrie()
		fileInfo := &FileInfo{size: 1}

		for _, path := range paths {
			trie.Insert(path, fileInfo)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	trie := NewTrie()
	numFolders := 100
	numFilesPerFolder := 10000
	var paths []string
	fileInfo := &FileInfo{size: 1}

	// Setup: Create and populate the trie.
	for i := 0; i < numFolders; i++ {
		folderPath := fmt.Sprintf("/dir%d", i)
		for j := 0; j < numFilesPerFolder; j++ {
			path := fmt.Sprintf("%s/file%d", folderPath, j)
			paths = append(paths, path)
			trie.Insert(path, fileInfo)
		}
	}

	// Shuffle paths to make access random within the benchmark loop.
	rand.Shuffle(len(paths), func(i, j int) { paths[i], paths[j] = paths[j], paths[i] })

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		// In each iteration, get a random file.
		trie.Get(paths[n%len(paths)])
	}
}

func BenchmarkConcurrentInsertDeepNesting(b *testing.B) {
	// This benchmark is memory-intensive. Run it with care.
	const totalEntries = 1_000_000 //1 million
	const nestingLevel = 20
	const maxConcurrentRoutines = 24

	// Generate paths
	paths := make([]string, totalEntries)
	for i := 0; i < totalEntries; i++ {
		var p strings.Builder
		for j := 0; j < nestingLevel; j++ {
			// Create directory names based on the file index to spread them out.
			// This creates a deep but also wide structure.
			// e.g., for i=1000, j=0 -> /d0_1000
			// for i=1000, j=1 -> /d0_1000/d1_1000
			fmt.Fprintf(&p, "/sub_dir_long_enough_to_crib%d", j)
		}
		fmt.Fprintf(&p, "/file_%d.txt", i)
		paths[i] = p.String()
	}

	fileInfo := &FileInfo{size: 1}

	b.ResetTimer()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	b.Logf("Initial HeapAlloc = %d MB", memStats.HeapAlloc/1024/1024)

	for n := 0; n < b.N; n++ {
		trie := NewTrie()
		var wg sync.WaitGroup
		wg.Add(totalEntries)
		sem := make(chan struct{}, maxConcurrentRoutines)

		for i := 0; i < totalEntries; i++ {
			sem <- struct{}{}
			go func(i int) {
				defer wg.Done()
				defer func() { <-sem }()
				trie.Insert(paths[i], fileInfo)
			}(i)
		}
		wg.Wait()

		if n == 0 {
			runtime.ReadMemStats(&memStats)
			b.Logf("HeapAlloc after trie creation = %d MB", memStats.HeapAlloc/1024/1024)
			b.ReportMetric(float64(memStats.HeapAlloc)/1024/1024, "MB/op")
		}
	}
}

func BenchmarkTrieMemory(b *testing.B) {
	// This benchmark is memory-intensive. Run it with care.
	const totalEntries = 15_000_000 //15 million
	const nestingLevel = 20

	{
		var memStats2 runtime.MemStats
		runtime.ReadMemStats(&memStats2)
		b.Logf("Stats before start heap: %d MB and sys: %d MB",
			memStats2.HeapAlloc/1024/1024, memStats2.Sys/1024/1024)
	}

	// Generate paths
	paths := make([]string, totalEntries)
	for i := range totalEntries {
		var p strings.Builder
		for j := range nestingLevel {
			// Create directory names based on the file index to spread them out.
			// This creates a deep but also wide structure.
			fmt.Fprintf(&p, "/sub_dir_long_enough_to_crib%d", j)
		}
		fmt.Fprintf(&p, "/file_%d.txt", i)
		paths[i] = p.String()
	}

	fileInfo := &FileInfo{size: 1}

	b.ResetTimer()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	{
		var memStats2 runtime.MemStats
		runtime.ReadMemStats(&memStats2)
		b.Logf("Stats after initialise data : %d MB and sys: %d MB",
			memStats2.HeapAlloc/1024/1024, memStats2.Sys/1024/1024)
	}

	trie := NewTrie()
	for i := 0; i < totalEntries; i++ {
		trie.InsertDirect(&paths[i], fileInfo)
	}

	paths = nil
	runtime.GC()
	var endStats runtime.MemStats
	runtime.ReadMemStats(&endStats)

	{
		var memStats2 runtime.MemStats
		runtime.ReadMemStats(&memStats2)
		b.Logf("Stats after setting data : %d MB and sys: %d MB ",
			memStats2.HeapAlloc/1024/1024, memStats2.Sys/1024/1024)
	}

	b.Logf("HeapAlloc after trie creation = %d MB (count: %d heap: %d MB))",
		int64((endStats.HeapAlloc-memStats.HeapAlloc)/1024/1024), trie.CountLeafs(),
		endStats.Sys/1024/1024)
	b.ReportMetric(float64(endStats.HeapAlloc)/1024/1024, "MB/op")

}
