// Copyright 2021 Google Inc. All Rights Reserved.
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

// Provides integration tests when implicit_dir flag is set.
package implicitdir_test

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func clearKernelCache() error {
	if _, err := os.Stat("/proc/sys/vm/drop_caches"); err != nil {
		log.Printf("Kernel cache file not found: %v", err)
		// No need to stop the test execution if cache file is not found. Further
		// reads will be served from kernel cache.
		return nil
	}

	// sudo permission is required to clear kernel page cache.
	cmd := exec.Command("sudo", "sh", "-c", "echo 3 > /proc/sys/vm/drop_caches")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clear kernel cache failed with error: %w", err)
	}
	return nil
}

func compareFileContents(t *testing.T, fileName string, fileContent string) {
	// After write, data will be cached by kernel. So subsequent read will be
	// served using cached data by kernel instead of calling gcsfuse.
	// Clearing kernel cache to ensure that gcsfuse is invoked during read operation.
	err := clearKernelCache()
	if err != nil {
		t.Errorf("Clear Kernel Cache: %v", err)
	}

	content, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Read: %v", err)
	}

	if got := string(content); got != fileContent {
		t.Errorf("File content doesn't match. Expected: %q, Actual: %q", got, fileContent)
	}
}

func createTempFile() string {
	// A temporary file is created and some lines are added
	// to it for testing purposes.
	fileName := path.Join(setup.TmpDir(), "tmpFile")
	err := os.WriteFile(fileName, []byte("line 1\nline 2\n"), 0666)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Temporary file at %v", err))
	}
	return fileName
}

func executeTest(m *testing.M) (successCode int) {
	// Creating a temporary directory to store files
	// to be used for testing.
	var err error
	tmpDir, err := os.MkdirTemp(setup.MntDir(), "tmpDir")
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Mkdir at %q: %v", setup.MntDir(), err))
	}
	setup.SetTmpDir(tmpDir)
	successCode = m.Run()

	os.RemoveAll(setup.MntDir())

	return successCode
}

func executeTestForFlags(flags [][]string, m *testing.M) (successCode int) {
	var err error

	for i := 0; i < len(flags); i++ {
		if err = setup.MountGcsfuse(flags[i]); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}

		successCode = executeTest(m)

		err = setup.UnMount()
		if err != nil {
			setup.LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
		}

		// Print flag on which test fails
		if successCode != 0 {
			var f string
			for j := 0; j < len(flags[i]); j++ {
				f += flags[i][j]
			}
			log.Print("Test Fails on " + f)
			return
		}

	}
	return
}

func TestMain(m *testing.M) {
	flag.Parse()

	if setup.TestBucket() == "" && setup.MountedDirectory() == "" {
		log.Printf("--testbucket or --mountedDirectory must be specified")
		os.Exit(0)
	} else if setup.TestBucket() != "" && setup.MountedDirectory() != "" {
		log.Printf("Both --testbucket and --mountedDirectory can't be specified at the same time.")
		os.Exit(0)
	}

	if setup.MountedDirectory() != "" {
		setup.SetMntDir(setup.MountedDirectory())
		successCode := executeTest(m)
		os.Exit(successCode)
	}

	if err := setup.SetUpTestDir(); err != nil {
		log.Printf("setUpTestDir: %v\n", err)
		os.Exit(1)
	}

	flags := [][]string{{"--enable-storage-client-library=true"},
		{"--enable-storage-client-library=false"},
		{"--implicit-dirs=true"},
		{"--implicit-dirs=false"}}

	successCode := executeTestForFlags(flags, m)

	log.Printf("Test log: %s\n", setup.LogFile())
	os.Exit(successCode)
}
