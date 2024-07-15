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
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
)

const (
	requiredCpuCount = 30
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

// This test-suite contains parallelizable test-case. Use "-parallel n" to limit
// the degree of parallelism. By default it uses GOMAXPROCS.
// Ref: https://stackoverflow.com/questions/24375966/does-go-test-run-unit-tests-concurrently
type highCpuConcurrentListingTest struct{}

func (s *highCpuConcurrentListingTest) Setup(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *highCpuConcurrentListingTest) Teardown(t *testing.T) {}

func getRelativePathFromDirectory(t *testing.T, filePath, targetDir string) string {
	// Split the file path into its components
	parts := strings.Split(filepath.Clean(filePath), string(filepath.Separator))

	// Find the index of the target directory
	targetIndex := -1
	for i, part := range parts {
		if part == targetDir {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		t.Errorf("Target directory not found.")
	}

	// Construct the relative path by joining the components after the target directory
	return filepath.Join(parts[targetIndex+1:]...)
}

func createDirectoryStructureForTestCaseParallel(t *testing.T, testCaseDir string) {
	operations.CreateDirectory(path.Join(testDirPath, testCaseDir), t)

	explicitDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	operations.CreateDirectory(explicitDir, t)
	numFiles := 2
	numLevel := 2

	var globalWG sync.WaitGroup

	createFilesInGivenDir := func(dir string) {
		var wg sync.WaitGroup
		for i := 0; i < numFiles; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				fileName := fmt.Sprintf("file%d.txt", i+1)
				client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName, path.Join(getRelativePathFromDirectory(t, dir, testDirName), fileName), "", t)
				operations.CreateFileOfSize(5, path.Join(dir, fileName), t)
			}(i)
		}

		wg.Wait()
	}

	lastLevel := explicitDir
	for level := 0; level < numLevel; level++ {
		currLevel := path.Join(lastLevel, fmt.Sprintf("level%d", level+1))
		lastLevel = currLevel
		operations.CreateDirectory(currLevel, t)
		globalWG.Add(1)
		// Create 100 files at the current level.
		go func() {
			defer globalWG.Done()
			createFilesInGivenDir(currLevel)
		}()
	}
	globalWG.Wait()
}

func listDirectoryRecursivelyWithCmd(t *testing.T, root string) {
	cmd := exec.Command("ls", "-R", root)
	_, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Error in listing recursively: %v", err)
	}
}

func listDirectoryRecursively(t *testing.T, root string) {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // Handle errors during the walk
		}

		return nil
	})
	if err != nil {
		t.Errorf("Error in listing recursively: %v", err)
	}
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// Test_AllReadOperationTogether tests for potential deadlocks or race conditions
// when multiple goroutines performs recursive listing .
func (s *highCpuConcurrentListingTest) Test_AllReadOperationTogether(t *testing.T) {
	if runtime.NumCPU() < requiredCpuCount {
		t.SkipNow()
	}
	t.Parallel() // Mark the test parallelizable.
	testCaseDir := "Test_MultipleConcurrentRecursiveListing"
	createDirectoryStructureForTestCaseParallel(t, testCaseDir)
	targetDir := path.Join(testDirPath, testCaseDir, "explicitDir")
	var wg sync.WaitGroup
	goRoutineCountPerOperation := 5
	timeout := 200 * time.Second

	// Create multiple go routines to listing concurrently.
	for r := 0; r < goRoutineCountPerOperation; r++ {
		wg.Add(3)

		// Repeatedly do recursive listing.
		go func() {
			defer wg.Done()

			for j := 0; j < iterationsForMediumOperations; j++ {
				listDirectoryRecursivelyWithCmd(t, targetDir)
			}
		}()

		// Repeatedly stats
		go func() {
			defer wg.Done()
			for i := 0; i < iterationsForLightOperations; i++ {
				_, err := os.Stat(targetDir)
				assert.Nil(t, err)
			}
		}()

		// Repeatedly calls OpenDir.
		go func() {
			defer wg.Done()
			for i := 0; i < iterationsForLightOperations; i++ {
				f, err := os.Open(targetDir)
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

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestHighCpuConcurrentListing(t *testing.T) {
	ts := &highCpuConcurrentListingTest{}
	test_setup.RunTests(t, ts)
}
