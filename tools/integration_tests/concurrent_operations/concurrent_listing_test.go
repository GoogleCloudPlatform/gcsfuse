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
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
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
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *concurrentListingTest) SetupTest() {
	// Do not use shared setup for parallel tests involving unique directories
}

func (s *concurrentListingTest) TearDownTest() {
	// Note: s.T() in TearDown might also be flaky in parallel suites,
	// but usually TearDown runs after the test body finishes.
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *concurrentListingTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *concurrentListingTest) SetupSuite() {
	setup.SetUpLogFilePath(s.baseTestName, GKETempDir, OldGKElogFilePath, testEnv.cfg)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

// createDirectoryStructureForTestCase creates initial directory structure in the
// given testCaseDir.
func createDirectoryStructureForTestCase(t *testing.T, testCaseDir string) {
	// Create explicitDir structure.
	explicitDir := path.Join(testCaseDir, "explicitDir")
	// fmt.Println("explicitDir", explicitDir)
	operations.CreateDirectory(explicitDir, t)
	// Use os.WriteFile to avoid issues with O_DIRECT in operations.CreateFileOfSize for small files.
	err := os.WriteFile(path.Join(explicitDir, "file1.txt"), make([]byte, 5), setup.FilePermission_0600)
	require.NoError(t, err)
	err = os.WriteFile(path.Join(explicitDir, "file2.txt"), make([]byte, 10), setup.FilePermission_0600)
	require.NoError(t, err)
}

// FIXED: Accept t *testing.T explicitly
func (s *concurrentListingTest) setupLocalTestDir(t *testing.T) string {
	// Use the passed 't', NOT s.T()
	mountedTestDirPath := client.SetupTestDirectory(s.ctx, s.storageClient, path.Join(testDirName, t.Name()))
	err := os.MkdirAll(mountedTestDirPath, setup.DirPermission_0755)
	require.NoError(t, err)
	return mountedTestDirPath
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// Test_OpenDirAndLookUp helps in detecting the deadlock when
// OpenDir() and LookUpInode() request for same directory comes in parallel.
func (s *concurrentListingTest) Test_OpenDirAndLookUp() {
	t := s.T() // Capture T
	t.Parallel()

	localTestDirPath := s.setupLocalTestDir(t) // Pass T

	createDirectoryStructureForTestCase(t, localTestDirPath) // Pass T
	targetDir := path.Join(localTestDirPath, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 40 * time.Second

	go func() {
		defer wg.Done()
		for range iterationsForLightOperations {
			f, err := os.Open(targetDir)
			require.Nil(t, err)

			err = f.Close()
			require.Nil(t, err) 
		}
	}()

	go func() {
		defer wg.Done()
		for range iterationsForLightOperations {
			_, err := os.Stat(targetDir)
			require.Nil(t, err) 
		}
	}()

	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock")
	}
}

// Test_Parallel_ReadDirAndLookUp tests for potential deadlocks or race conditions when
// ReadDir() is called concurrently with LookUp of same dir.
func (s *concurrentListingTest) Test_Parallel_ReadDirAndLookUp() {
    // 1. Capture T locally
    t := s.T()
    t.Parallel() 
    
    // 2. Pass T to helper
    localTestDirPath := s.setupLocalTestDir(t)

    // 3. Pass T to structure creator
    createDirectoryStructureForTestCase(t, localTestDirPath)
    targetDir := path.Join(localTestDirPath, "explicitDir")
    
    var wg sync.WaitGroup
    wg.Add(2)
    timeout := 200 * time.Second

    // Goroutine 1: Repeatedly calls Readdir
    go func() {
        defer wg.Done()
        for range iterationsForMediumOperations {
            f, err := os.Open(targetDir)
            require.Nil(t, err)

            _, err = f.Readdirnames(-1)
            require.Nil(t, err)

            err = f.Close()
            require.Nil(t, err)
        }
    }()

    // Goroutine 2: Repeatedly stats
    go func() {
        defer wg.Done()
        for range iterationsForLightOperations {
            _, err := os.Stat(targetDir)
            require.Nil(t, err)
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
// Test_MultipleConcurrentReadDir tests for potential deadlocks or race conditions
// when multiple goroutines call Readdir() concurrently on the same directory.
func (s *concurrentListingTest) Test_MultipleConcurrentReadDir() {
	t := s.T()
	t.Parallel()
	localTestDirPath := s.setupLocalTestDir(t)

	createDirectoryStructureForTestCase(t, localTestDirPath)
	targetDir := path.Join(localTestDirPath, "explicitDir")
	var wg sync.WaitGroup
	goroutineCount := 10
	wg.Add(goroutineCount)
	timeout := 600 * time.Second

	for range goroutineCount {
		go func() {
			defer wg.Done()

			for range iterationsForMediumOperations {
				f, err := os.Open(targetDir)
				require.Nil(t, err)

				_, err = f.Readdirnames(-1)
				require.Nil(t, err)

				err = f.Close()
				require.Nil(t, err)
			}
		}()
	}

	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected during concurrent Readdir calls")
	}
}

// Test_Parallel_ReadDirAndFileOperations detects race conditions and deadlocks when one goroutine
// performs Readdir() while another concurrently creates and deletes files in the same directory.
func (s *concurrentListingTest) Test_Parallel_ReadDirAndFileOperations() {
	t := s.T()
	t.Parallel()
	localTestDirPath := s.setupLocalTestDir(t)

	createDirectoryStructureForTestCase(t, localTestDirPath)
	targetDir := path.Join(localTestDirPath, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 400 * time.Second

	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations {
			f, err := os.Open(targetDir)
			require.Nil(t, err)

			_, err = f.Readdirnames(-1)
			require.Nil(t, err)

			err = f.Close()
			require.Nil(t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for range iterationsForHeavyOperations {
			filePath := path.Join(targetDir, "tmp_file.txt")
			renamedFilePath := path.Join(targetDir, "renamed_tmp_file.txt")

			f, err := os.Create(filePath)
			require.Nil(t, err)
			err = f.Close()
			require.Nil(t, err)

			err = os.Rename(filePath, renamedFilePath)
			require.Nil(t, err)

			err = os.Remove(renamedFilePath)
			require.Nil(t, err)
		}
	}()

	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected")
	}
}

// Test_Parallel_ReadDirAndDirOperations tests for potential deadlocks or race conditions when
// ReadDir() is called concurrently with directory creation and deletion operations.
func (s *concurrentListingTest) Test_Parallel_ReadDirAndDirOperations() {
	t := s.T()
	t.Parallel()
	localTestDirPath := s.setupLocalTestDir(t)

	createDirectoryStructureForTestCase(t, localTestDirPath)
	targetDir := path.Join(localTestDirPath, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 200 * time.Second

	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations {
			f, err := os.Open(targetDir)
			require.Nil(t, err)

			_, err = f.Readdirnames(-1)
			require.Nil(t, err)

			err = f.Close()
			require.Nil(t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for range iterationsForHeavyOperations {
			dirPath := path.Join(targetDir, "test_dir")
			renamedDirPath := path.Join(targetDir, "renamed_test_dir")

			err := os.Mkdir(dirPath, 0755)
			require.Nil(t, err)

			err = os.Rename(dirPath, renamedDirPath)
			require.Nil(t, err)

			err = os.Remove(renamedDirPath)
			require.Nil(t, err)
		}
	}()

	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected during Readdir and directory operations")
	}
}

// Test_Parallel_ReadDirAndFileEdit tests for potential deadlocks or race conditions when
// ReadDir() is called concurrently with modification of underneath file.
func (s *concurrentListingTest) Test_Parallel_ReadDirAndFileEdit() {
	t := s.T()
	t.Parallel()
	localTestDirPath := s.setupLocalTestDir(t)

	createDirectoryStructureForTestCase(t, localTestDirPath)
	targetDir := path.Join(localTestDirPath, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 400 * time.Second

	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations {
			f, err := os.Open(targetDir)
			require.Nil(t, err)

			_, err = f.Readdirnames(-1)
			require.Nil(t, err)

			err = f.Close()
			require.Nil(t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for i := range iterationsForHeavyOperations {
			filePath := path.Join(targetDir, fmt.Sprintf("test_file_%d.txt", i))

			err := os.WriteFile(filePath, []byte("Hello, world!"), setup.FilePermission_0600)
			require.Nil(t, err)
			time.Sleep(time.Second)

			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, setup.FilePermission_0600)
			require.Nil(t, err)
			_, err = f.Write([]byte("This is an edit."))
			require.Nil(t, err)
			err = f.Close()
			require.Nil(t, err)
		}
	}()

	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected during Readdir and directory operations")
	}
}

// Test_MultipleConcurrentOperations tests for potential deadlocks or race conditions when
// listing, file or folder operations, stat, opendir, file modifications happening concurrently.
func (s *concurrentListingTest) Test_MultipleConcurrentOperations() {
	t := s.T()
	t.Parallel()
	localTestDirPath := s.setupLocalTestDir(t)

	createDirectoryStructureForTestCase(t, localTestDirPath)
	targetDir := path.Join(localTestDirPath, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(5)
	timeout := 400 * time.Second

	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations {
			f, err := os.Open(targetDir)
			require.Nil(t, err)

			_, err = f.Readdirnames(-1)
			require.Nil(t, err)

			err = f.Close()
			require.Nil(t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for i := range iterationsForHeavyOperations {
			filePath := path.Join(targetDir, fmt.Sprintf("test_file_%d.txt", i))

			err := os.WriteFile(filePath, []byte("Hello, world!"), setup.FilePermission_0600)
			require.Nil(t, err)
			time.Sleep(time.Second)

			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, setup.FilePermission_0600)
			require.Nil(t, err)
			_, err = f.Write([]byte("This is an edit."))
			require.Nil(t, err)
			err = f.Close()
			require.Nil(t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for range iterationsForHeavyOperations {
			dirPath := path.Join(targetDir, "test_dir")
			renamedDirPath := path.Join(targetDir, "renamed_test_dir")

			err := os.Mkdir(dirPath, 0755)
			require.Nil(t, err)

			err = os.Rename(dirPath, renamedDirPath)
			require.Nil(t, err)

			err = os.Remove(renamedDirPath)
			require.Nil(t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for range iterationsForLightOperations {
			_, err := os.Stat(targetDir)
			require.Nil(t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for range iterationsForLightOperations {
			f, err := os.Open(targetDir)
			require.Nil(t, err)

			err = f.Close()
			require.Nil(t, err)
		}
	}()

	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected during Readdir and directory operations")
	}
}

// Test_ListWithMoveFile tests for potential deadlocks or race conditions when
// listing, file or folder operations, move file happening concurrently.
func (s *concurrentListingTest) Test_ListWithMoveFile() {
	t := s.T()
	t.Parallel()
	localTestDirPath := s.setupLocalTestDir(t)

	createDirectoryStructureForTestCase(t, localTestDirPath)
	targetDir := path.Join(localTestDirPath, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 400 * time.Second

	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations {
			f, err := os.Open(targetDir)
			require.NoError(t, err)

			_, err = f.Readdirnames(-1)
			require.Nil(t, err)

			require.NoError(t, f.Close())
		}
	}()

	err := os.WriteFile(path.Join(localTestDirPath, "move_file.txt"), []byte("Hello, world!"), setup.FilePermission_0600)
	require.NoError(t, err)

	go func() {
		defer wg.Done()
		for range iterationsForHeavyOperations {
			err = operations.Move(path.Join(localTestDirPath, "move_file.txt"), path.Join(targetDir, "move_file.txt"))
			require.NoError(t, err)
			err = operations.Move(path.Join(targetDir, "move_file.txt"), path.Join(localTestDirPath, "move_file.txt"))
			require.NoError(t, err)
		}
	}()

	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected")
	}
}

// Test_ListWithMoveDir tests for potential deadlocks or race conditions when
// listing, file or folder operations, move dir happening concurrently.
func (s *concurrentListingTest) Test_ListWithMoveDir() {
	t := s.T()
	t.Parallel()
	localTestDirPath := s.setupLocalTestDir(t)

	createDirectoryStructureForTestCase(t, localTestDirPath)
	targetDir := path.Join(localTestDirPath, "explicitDir")
	var wg sync.WaitGroup
	wg.Add(2)
	timeout := 400 * time.Second

	go func() {
		defer wg.Done()
		for range iterationsForMediumOperations {
			f, err := os.Open(targetDir)
			require.NoError(t, err)

			_, err = f.Readdirnames(-1)
			require.Nil(t, err)

			require.NoError(t, f.Close())
		}
	}()

	err := os.Mkdir(path.Join(localTestDirPath, "move_dir"), setup.DirPermission_0755)
	require.NoError(t, err)

	go func() {
		defer wg.Done()
		for range iterationsForHeavyOperations {
			err = operations.Move(path.Join(localTestDirPath, "move_dir"), path.Join(targetDir, "move_dir"))
			require.NoError(t, err)
			err = operations.Move(path.Join(targetDir, "move_dir"), path.Join(localTestDirPath, "move_dir"))
			require.NoError(t, err)
		}
	}()

	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		assert.FailNow(t, "Possible deadlock or race condition detected")
	}
}

// Test_StatWithNewFileWrite tests for potential deadlocks or race conditions when
// statting and creating a new file happen concurrently.
func (s *concurrentListingTest) Test_StatWithNewFileWrite() {
    // 1. Capture T locally
    t := s.T()
    t.Parallel()
    
    // 2. Pass T to helper
    localTestDirPath := s.setupLocalTestDir(t)

    // 3. Pass T to structure creator
    createDirectoryStructureForTestCase(t, localTestDirPath)
    targetDir := path.Join(localTestDirPath, "explicitDir")
    fmt.Println("targetDir", targetDir)
    
    var wg sync.WaitGroup
    wg.Add(2)
    timeout := 400 * time.Second // Adjust timeout as needed

    // Goroutine 1: Repeatedly calls Stat
    go func() {
        defer wg.Done()
        for range iterationsForMediumOperations {
            _, err := os.Stat(targetDir)
            require.NoError(t, err)
        }
    }()

    // Goroutine 2: Repeatedly create a file.
    go func() {
        defer wg.Done()
        for i := range iterationsForMediumOperations {
            // Create file
            filePath := path.Join(targetDir, fmt.Sprintf("tmp_file_%d.txt", i))
            err := os.WriteFile(filePath, []byte("Hello, world!"), setup.FilePermission_0600)

            require.NoError(t, err)
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
        assert.FailNow(t, "Possible deadlock or race condition detected")
    }
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestConcurrentListing(t *testing.T) {
	ts := &concurrentListingTest{ctx: context.Background(), storageClient: testEnv.storageClient, baseTestName: t.Name()}

	// Run tests for mounted directory if the flag is set. This assumes that run flag is properly passed by GKE team as per the config.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
