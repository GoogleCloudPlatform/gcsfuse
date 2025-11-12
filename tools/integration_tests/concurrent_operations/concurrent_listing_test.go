// Copyright 2024 Google LLC
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
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (

	// Used for operation like, creation, deletion, rename or edit of files/folders.
	iterationsForHeavyOperations = 50

	// Used for listing of directories.
	iterationsForMediumOperations = 200

	// Used for Open of Stat.
	iterationsForLightOperations = 1000
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

// This test-suite contains parallelizable test-case. Use "-parallel n" to limit
// the degree of parallelism. By default it uses GOMAXPROCS.
// Ref: https://stackoverflow.com/questions/24375966/does-go-test-run-unit-tests-concurrently
type concurrentListingTest struct {
	suite.Suite
}

func (s *concurrentListingTest) SetupTest() {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *concurrentListingTest) TearDownTest() {}

// createDirectoryStructureForTestCase creates initial directory structure in the
// given testCaseDir.
// bucket
//
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
func (s *concurrentListingTest) Test_OpenDirAndLookUp() {
	s.T().Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_OpenDirAndLookUp"
	createDirectoryStructureForTestCase(s.T(), testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	// Fails if the operation takes more than timeout.
	timeout := 40 * time.Second

	// Goroutine 1: Repeatedly calls OpenDir.
	go func() {
		defer wg.Done()
		for range iterationsForLightOperations {
			f, err := os.Open(targetDir)
			require.Nil(s.T(), err)

			err = f.Close()
			require.Nil(s.T(), err)
		}
	}()

	// Goroutine 1: Repeatedly calls Stat.
	go func() {
		defer wg.Done()
		for range iterationsForLightOperations {
			_, err := os.Stat(targetDir)
			require.Nil(s.T(), err)
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
		assert.FailNow(s.T(), "Possible deadlock")
	}
}

// Test_Parallel_ReadDirAndLookUp tests for potential deadlocks or race conditions when
// ReadDir() is called concurrently with LookUp of same dir.
func (s *concurrentListingTest) Test_Parallel_ReadDirAndLookUp() {
	s.T().Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_Parallel_ReadDirAndLookUp"
	createDirectoryStructureForTestCase(s.T(), testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 200 * time.Second

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations {
			f, err := os.Open(targetDir)
			require.Nil(s.T(), err)

			_, err = f.Readdirnames(-1)
			require.Nil(s.T(), err)

			err = f.Close()
			require.Nil(s.T(), err)
		}
	}()

	// Goroutine 2: Repeatedly stats
	go func() {
		defer wg.Done()
		for range iterationsForLightOperations {
			_, err := os.Stat(targetDir)
			require.Nil(s.T(), err)
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
		assert.FailNow(s.T(), "Possible deadlock or race condition detected during Readdir and directory operations")
	}
}

// Test_MultipleConcurrentReadDir tests for potential deadlocks or race conditions
// when multiple goroutines call Readdir() concurrently on the same directory.
func (s *concurrentListingTest) Test_MultipleConcurrentReadDir() {
	s.T().Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_MultipleConcurrentReadDir"
	createDirectoryStructureForTestCase(s.T(), testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	var wg sync.WaitGroup
	goroutineCount := 10 // Number of concurrent goroutines
	wg.Add(goroutineCount)
	timeout := 600 * time.Second // More timeout to accommodate the high listing time without kernel-list-cache.

	// Create multiple go routines to listing concurrently.
	for range goroutineCount {
		go func() {
			defer wg.Done()

			for range iterationsForMediumOperations {
				f, err := os.Open(targetDir)
				require.Nil(s.T(), err)

				_, err = f.Readdirnames(-1) // Read all directory entries
				require.Nil(s.T(), err)

				err = f.Close()
				require.Nil(s.T(), err)
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
		assert.FailNow(s.T(), "Possible deadlock or race condition detected during concurrent Readdir calls")
	}
}

// Test_Parallel_ReadDirAndFileOperations detects race conditions and deadlocks when one goroutine
// performs Readdir() while another concurrently creates and deletes files in the same directory.
func (s *concurrentListingTest) Test_Parallel_ReadDirAndFileOperations() {
	s.T().Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_Parallel_ReadDirAndFileOperations"
	createDirectoryStructureForTestCase(s.T(), testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 400 * time.Second // Adjust timeout as needed

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations { // Adjust iteration count if needed
			f, err := os.Open(targetDir)
			require.Nil(s.T(), err)

			_, err = f.Readdirnames(-1)
			require.Nil(s.T(), err)

			err = f.Close()
			require.Nil(s.T(), err)
		}
	}()

	// Goroutine 2: Creates and deletes files
	go func() {
		defer wg.Done()
		for range iterationsForHeavyOperations { // Adjust iteration count if needed
			filePath := path.Join(targetDir, "tmp_file.txt")
			renamedFilePath := path.Join(targetDir, "renamed_tmp_file.txt")

			// Create
			f, err := os.Create(filePath)
			require.Nil(s.T(), err)

			err = f.Close()
			require.Nil(s.T(), err)

			// Rename
			err = os.Rename(filePath, renamedFilePath)
			require.Nil(s.T(), err)

			// Delete
			err = os.Remove(renamedFilePath)
			require.Nil(s.T(), err)
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
		assert.FailNow(s.T(), "Possible deadlock or race condition detected")
	}
}

// Test_Parallel_ReadDirAndDirOperations tests for potential deadlocks or race conditions when
// ReadDir() is called concurrently with directory creation and deletion operations.
func (s *concurrentListingTest) Test_Parallel_ReadDirAndDirOperations() {
	s.T().Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_Parallel_ReadDirAndDirOperations"
	createDirectoryStructureForTestCase(s.T(), testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 200 * time.Second

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations {
			f, err := os.Open(targetDir)
			require.Nil(s.T(), err)

			_, err = f.Readdirnames(-1)
			require.Nil(s.T(), err)

			err = f.Close()
			require.Nil(s.T(), err)
		}
	}()

	// Goroutine 2: Creates and deletes directories
	go func() {
		defer wg.Done()
		for range iterationsForHeavyOperations {
			dirPath := path.Join(targetDir, "test_dir")
			renamedDirPath := path.Join(targetDir, "renamed_test_dir")

			// Create
			err := os.Mkdir(dirPath, 0755)
			require.Nil(s.T(), err)

			// Rename
			err = os.Rename(dirPath, renamedDirPath)
			require.Nil(s.T(), err)

			// Delete
			err = os.Remove(renamedDirPath)
			require.Nil(s.T(), err)
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
		assert.FailNow(s.T(), "Possible deadlock or race condition detected during Readdir and directory operations")
	}
}

// Test_Parallel_ReadDirAndFileEdit tests for potential deadlocks or race conditions when
// ReadDir() is called concurrently with modification of underneath file.
func (s *concurrentListingTest) Test_Parallel_ReadDirAndFileEdit() {
	s.T().Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_Parallel_ListDirAndFileEdit"
	createDirectoryStructureForTestCase(s.T(), testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 400 * time.Second

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations {
			f, err := os.Open(targetDir)
			require.Nil(s.T(), err)

			_, err = f.Readdirnames(-1)
			require.Nil(s.T(), err)

			err = f.Close()
			require.Nil(s.T(), err)
		}
	}()

	// Goroutine 2: Create and edit files
	go func() {
		defer wg.Done()
		for i := range iterationsForHeavyOperations {
			filePath := path.Join(targetDir, fmt.Sprintf("test_file_%d.txt", i))

			// Create file
			err := os.WriteFile(filePath, []byte("Hello, world!"), setup.FilePermission_0600)
			require.Nil(s.T(), err)
			time.Sleep(time.Second)

			// Edit file (append some data)
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, setup.FilePermission_0600)
			require.Nil(s.T(), err)
			_, err = f.Write([]byte("This is an edit."))
			require.Nil(s.T(), err)
			err = f.Close()
			require.Nil(s.T(), err)
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
		assert.FailNow(s.T(), "Possible deadlock or race condition detected during Readdir and directory operations")
	}
}

// Test_MultipleConcurrentOperations tests for potential deadlocks or race conditions when
// listing, file or folder operations, stat, opendir, file modifications happening concurrently.
func (s *concurrentListingTest) Test_MultipleConcurrentOperations() {
	s.T().Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_MultipleConcurrentOperations"
	createDirectoryStructureForTestCase(s.T(), testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(5)
	timeout := 400 * time.Second

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations { // Adjust iteration count if needed
			f, err := os.Open(targetDir)
			require.Nil(s.T(), err)

			_, err = f.Readdirnames(-1)
			require.Nil(s.T(), err)

			err = f.Close()
			require.Nil(s.T(), err)
		}
	}()

	// Goroutine 2: Create and edit files
	go func() {
		defer wg.Done()
		for i := range iterationsForHeavyOperations {
			filePath := path.Join(targetDir, fmt.Sprintf("test_file_%d.txt", i))

			// Create file
			err := os.WriteFile(filePath, []byte("Hello, world!"), setup.FilePermission_0600)
			require.Nil(s.T(), err)
			time.Sleep(time.Second)

			// Edit file (append some data)
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, setup.FilePermission_0600)
			require.Nil(s.T(), err)
			_, err = f.Write([]byte("This is an edit."))
			require.Nil(s.T(), err)
			err = f.Close()
			require.Nil(s.T(), err)
		}
	}()

	// Goroutine 3: Creates and deletes directories
	go func() {
		defer wg.Done()
		for range iterationsForHeavyOperations {
			dirPath := path.Join(targetDir, "test_dir")
			renamedDirPath := path.Join(targetDir, "renamed_test_dir")

			// Create
			err := os.Mkdir(dirPath, 0755)
			require.Nil(s.T(), err)

			// Rename
			err = os.Rename(dirPath, renamedDirPath)
			require.Nil(s.T(), err)

			// Delete
			err = os.Remove(renamedDirPath)
			require.Nil(s.T(), err)
		}
	}()

	// Goroutine 4: Repeatedly stats
	go func() {
		defer wg.Done()
		for range iterationsForLightOperations {
			_, err := os.Stat(targetDir)
			require.Nil(s.T(), err)
		}
	}()

	// Goroutine 5: Repeatedly calls OpenDir.
	go func() {
		defer wg.Done()
		for range iterationsForLightOperations {
			f, err := os.Open(targetDir)
			require.Nil(s.T(), err)

			err = f.Close()
			require.Nil(s.T(), err)
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
		assert.FailNow(s.T(), "Possible deadlock or race condition detected during Readdir and directory operations")
	}
}

// Test_ListWithMoveFile tests for potential deadlocks or race conditions when
// listing, file or folder operations, move file happening concurrently.
func (s *concurrentListingTest) Test_ListWithMoveFile() {
	s.T().Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_ListWithMoveFile"
	createDirectoryStructureForTestCase(s.T(), testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 400 * time.Second // Adjust timeout as needed

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations { // Adjust iteration count if needed
			f, err := os.Open(targetDir)
			require.NoError(s.T(), err)

			_, err = f.Readdirnames(-1)
			require.Nil(s.T(), err)

			require.NoError(s.T(), f.Close())
		}
	}()

	// Create file
	err := os.WriteFile(path.Join(testDirPath, "move_file.txt"), []byte("Hello, world!"), setup.FilePermission_0600)
	require.NoError(s.T(), err)

	// Goroutine 2: Move file
	go func() {
		defer wg.Done()
		for range iterationsForHeavyOperations { // Adjust iteration count if needed
			// Move File in the target directory
			err = operations.Move(path.Join(testDirPath, "move_file.txt"), path.Join(targetDir, "move_file.txt"))
			require.NoError(s.T(), err)
			// Move File out of the target directory
			err = operations.Move(path.Join(targetDir, "move_file.txt"), path.Join(testDirPath, "move_file.txt"))
			require.NoError(s.T(), err)
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
		assert.FailNow(s.T(), "Possible deadlock or race condition detected")
	}
}

// Test_ListWithMoveDir tests for potential deadlocks or race conditions when
// listing, file or folder operations, move dir happening concurrently.
func (s *concurrentListingTest) Test_ListWithMoveDir() {
	s.T().Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_ListWithMoveDir"
	createDirectoryStructureForTestCase(s.T(), testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 400 * time.Second // Adjust timeout as needed

	// Goroutine 1: Repeatedly calls Readdir
	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations { // Adjust iteration count if needed
			f, err := os.Open(targetDir)
			require.NoError(s.T(), err)

			_, err = f.Readdirnames(-1)
			require.Nil(s.T(), err)

			require.NoError(s.T(), f.Close())
		}
	}()
	// Create Dir
	err := os.Mkdir(path.Join(testDirPath, "move_dir"), setup.DirPermission_0755)
	require.NoError(s.T(), err)

	// Goroutine 2: Move Dir
	go func() {
		defer wg.Done()
		for range iterationsForHeavyOperations { // Adjust iteration count if needed
			// Move Dir in the target dir
			err = operations.Move(path.Join(testDirPath, "move_dir"), path.Join(targetDir, "move_dir"))
			require.NoError(s.T(), err)
			// Move Dir out of the target dir
			err = operations.Move(path.Join(targetDir, "move_dir"), path.Join(testDirPath, "move_dir"))
			require.NoError(s.T(), err)
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
		assert.FailNow(s.T(), "Possible deadlock or race condition detected")
	}
}

// Test_StatWithNewFileWrite tests for potential deadlocks or race conditions when
// statting and creating a new file happen concurrently.
func (s *concurrentListingTest) Test_StatWithNewFileWrite() {
	s.T().Parallel()
	testCaseDir := "Test_StatWithNewFileWrite"
	createDirectoryStructureForTestCase(s.T(), testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 400 * time.Second // Adjust timeout as needed

	// Goroutine 1: Repeatedly calls Stat
	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations {
			_, err := os.Stat(targetDir)

			require.NoError(s.T(), err)
		}
	}()

	// Goroutine 2: Repeatedly create a file.
	go func() {
		defer wg.Done()
		for i := range iterationsForMediumOperations {
			// Create file
			filePath := path.Join(targetDir, fmt.Sprintf("tmp_file_%d.txt", i))
			err := os.WriteFile(filePath, []byte("Hello, world!"), setup.FilePermission_0600)

			require.NoError(s.T(), err)
		}
	}()

	// Wait for goroutines or timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success: Both operations finished before timeout
	case <-time.After(timeout):
		assert.FailNow(s.T(), "Possible deadlock or race condition detected")
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestConcurrentListing(t *testing.T) {
	ts := &concurrentListingTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	flagsSet := [][]string{
		{"--kernel-list-cache-ttl-secs=0"}, {"--kernel-list-cache-ttl-secs=-1"},
	}

	if !testing.Short() {
		setup.AppendFlagsToAllFlagsInTheFlagsSet(&flagsSet, "", "--client-protocol=grpc")
	}

	for _, flags := range flagsSet {
		mountGCSFuseAndSetupTestDir(flags, testDirName, t)
		// Parallel subtest execution is suspended until its calling test function has
		// returned. Hence invoking RunTest inside another test, otherwise unmount will
		// happen before the subtest execution starts.
		t.Run(fmt.Sprintf("Flags_%v", flags), func(t *testing.T) {
			suite.Run(t, ts)
		})
		setup.UnmountGCSFuse(setup.MntDir())
	}
}
