// Copyright 2024 Google Inc. All Rights Reserved.
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

package concurrent_operations

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

// Test cases of this test suites are parallelizable, use -parallel n to set
// the parallelism in the go test command.
type concurrentListingTest struct{}

func (s *concurrentListingTest) Setup(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *concurrentListingTest) Teardown(t *testing.T) {}

// Create initial directory structure for kernel-list-cache test in the
// given testCaseDir.
// bucket
//
//		file1.txt
//		file2.txt
//	  explicitDir/
//		explicitDir/file1.txt
//		explicitDir/file2.txt
func createDirectoryStructureForTestCase(t *testing.T, testCaseDir string) {
	operations.CreateDirectory(path.Join(testDirPath, testCaseDir), t)

	// Create explicitDir structure
	explicitDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	operations.CreateDirectory(explicitDir, t)
	operations.CreateFileOfSize(5, path.Join(explicitDir, "file1.txt"), t)
	operations.CreateFileOfSize(10, path.Join(explicitDir, "file2.txt"), t)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// Test_OpenDirAndLookUp helps in detecting the deadlock when
// OpenDir() and LookUpInode() request for same directory comes in parallel.
func (s *concurrentListingTest) Test_OpenDirAndLookUp(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_OpenDirAndLookUp"
	createDirectoryStructureForTestCase(t, testCaseDir)

	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")

	var wg sync.WaitGroup
	wg.Add(2)
	// Fail if the operation takes more than timeout.
	timeout := 5 * time.Second
	iterationsPerGoroutine := 100

	// Goroutine 1: Repeatedly calls OpenDir.
	go func() {
		defer wg.Done()
		for i := 0; i < iterationsPerGoroutine; i++ {
			f, err := os.Open(targetDir)
			assert.Nil(t, err)

			err = f.Close()
			assert.Nil(t, err)
		}
	}()

	// Goroutine 1: Repeatedly calls Stat.
	go func() {
		defer wg.Done()
		for i := 0; i < iterationsPerGoroutine; i++ {
			_, err := os.Stat(targetDir)
			assert.Nil(t, err)
		}
	}()

	// Wait for goroutines or timeout.
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()
	select {
	case <-done:
		// Operation completed successfully before timeout.
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock")
	}
}

// Test_Parallel_ReadDirAndDirOperations tests for potential deadlocks or race conditions when
// ReadDir() is called concurrently with directory creation and deletion operations.
func (s *concurrentListingTest) Test_Parallel_ReadDirAndLookUp(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.

	testCaseDir := "Test_Parallel_ReadDirAndLookUp"
	createDirectoryStructureForTestCase(t, testCaseDir)

	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")

	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 200 * time.Second
	iterationsPerGoroutine := 40

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for i := 0; i < iterationsPerGoroutine; i++ {
			f, err := os.Open(targetDir)
			assert.Nil(t, err)

			_, err = f.Readdirnames(0)
			assert.Nil(t, err)

			err = f.Close()
			assert.Nil(t, err)
		}
	}()

	// Goroutine 2: Creates and deletes directories
	go func() {
		defer wg.Done()
		for i := 0; i < iterationsPerGoroutine; i++ {
			_, err := os.Stat(targetDir)
			assert.Nil(t, err)
		}
	}()

	// Wait for goroutines or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success: Both operations finished before timeout
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected during Readdir and directory operations")
	}
}

// Test_Concurrent_ReadDir tests for potential deadlocks or race conditions
// when multiple goroutines call Readdir() concurrently on the same directory.
func (s *concurrentListingTest) Test_MultipleConcurrentReadDir(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.

	testCaseDir := "Test_MultipleConcurrentReadDir"
	createDirectoryStructureForTestCase(t, testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")

	var wg sync.WaitGroup
	goroutineCount := 10          // Number of concurrent goroutines
	iterationsPerGoroutine := 100 // Number of iterations per goroutine

	wg.Add(goroutineCount)
	timeout := 50 * time.Second

	for i := 0; i < goroutineCount; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < iterationsPerGoroutine; j++ {
				f, err := os.Open(targetDir)
				assert.Nil(t, err)

				_, err = f.Readdirnames(-1) // Read all directory entries
				assert.Nil(t, err)

				err = f.Close()
				assert.Nil(t, err)
			}
		}()
	}

	// Wait for goroutines or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success: All Readdir operations finished before timeout
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected during concurrent Readdir calls")
	}
}

// Test_Parallel_ReadDirAndFileOperations detects race conditions and deadlocks when one goroutine
// performs Readdir() while another concurrently creates and deletes files in the same directory.
func (s *concurrentListingTest) Test_Parallel_ReadDirAndFileOperations(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.

	testCaseDir := "Test_Parallel_ReadDirAndFileOperations"
	createDirectoryStructureForTestCase(t, testCaseDir)

	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")

	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 400 * time.Second // Adjust timeout as needed
	iterationsPerGoroutine := 100

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for i := 0; i < iterationsPerGoroutine; i++ { // Adjust iteration count if needed
			f, err := os.Open(targetDir)
			assert.Nil(t, err)

			_, err = f.Readdirnames(-1)
			if err != nil {
				// This is expected, see the documentation for fixConflictingNames() call in dir_handle.go.
				assert.True(t, strings.Contains(err.Error(), "input/output error"))
			}

			err = f.Close()
			assert.Nil(t, err)
		}
	}()

	// Goroutine 2: Creates and deletes files
	go func() {
		defer wg.Done()
		for i := 0; i < iterationsPerGoroutine; i++ { // Adjust iteration count if needed
			filePath := path.Join(targetDir, "tmp_file.txt")
			renamedFilePath := path.Join(targetDir, "renamed_tmp_file.txt")

			// Create
			f, err := os.Create(filePath)
			assert.Nil(t, err)

			err = f.Close()
			assert.Nil(t, err)

			// Rename
			err = os.Rename(filePath, renamedFilePath)
			assert.Nil(t, err)

			// Delete
			err = os.Remove(renamedFilePath)
			assert.Nil(t, err)
		}
	}()

	// Wait for goroutines or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success: Both operations finished before timeout
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected")
	}
}

// Test_Parallel_ReadDirAndDirOperations tests for potential deadlocks or race conditions when
// ReadDir() is called concurrently with directory creation and deletion operations.
func (s *concurrentListingTest) Test_Parallel_ReadDirAndDirOperations(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.

	testCaseDir := "Test_Parallel_ReadDirAndDirOperations"
	createDirectoryStructureForTestCase(t, testCaseDir)

	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")

	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 200 * time.Second
	iterationsPerGoroutine := 40

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for i := 0; i < iterationsPerGoroutine; i++ {
			f, err := os.Open(targetDir)
			assert.Nil(t, err)

			_, err = f.Readdirnames(0)
			assert.Nil(t, err)

			err = f.Close()
			assert.Nil(t, err)
		}
	}()

	// Goroutine 2: Creates and deletes directories
	go func() {
		defer wg.Done()
		for i := 0; i < iterationsPerGoroutine; i++ {
			dirPath := path.Join(targetDir, "test_dir")
			renamedDirPath := path.Join(targetDir, "renamed_test_dir")

			// Create
			err := os.Mkdir(dirPath, 0755)
			assert.Nil(t, err)

			// Rename
			err = os.Rename(dirPath, renamedDirPath)
			assert.Nil(t, err)

			// Delete
			err = os.Remove(renamedDirPath)
			assert.Nil(t, err)
		}
	}()

	// Wait for goroutines or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success: Both operations finished before timeout
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected during Readdir and directory operations")
	}
}

func (s *concurrentListingTest) Test_Parallel_ListDirAndFileEdit(t *testing.T) {
	t.Parallel() // Mark the test parallelizable.

	testCaseDir := "Test_Parallel_ListDirAndFileEdit"
	createDirectoryStructureForTestCase(t, testCaseDir)

	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")

	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 400 * time.Second
	iterationsPerGoroutine := 100

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for i := 0; i < iterationsPerGoroutine; i++ {
			f, err := os.Open(targetDir)
			assert.Nil(t, err)

			_, err = f.Readdirnames(0)
			assert.Nil(t, err)

			err = f.Close()
			assert.Nil(t, err)
		}
	}()

	// Goroutine 2: Create and edit files
	go func() {
		defer wg.Done()
		for i := 0; i < iterationsPerGoroutine; i++ {
			filePath := path.Join(targetDir, fmt.Sprintf("test_file_%d.txt", i))

			// Create file
			err := os.WriteFile(filePath, []byte("Hello, world!"), setup.FilePermission_0600)
			assert.Nil(t, err)

			// Edit file (append some data)
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, setup.FilePermission_0600)
			assert.Nil(t, err)
			_, err = f.Write([]byte("This is an edit."))
			assert.Nil(t, err)
			err = f.Close()
			assert.Nil(t, err)
		}
	}()

	// Wait for goroutines or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success: Both operations finished before timeout
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected during Readdir and directory operations")
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestConcurrentListing(t *testing.T) {
	ts := &concurrentListingTest{}
	test_setup.RunTests(t, ts)
}
