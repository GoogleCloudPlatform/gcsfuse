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
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/util"
)

var testBucket = flag.String("testbucket", "", "The GCS bucket used for the test.")

var (
	binFile string
	logFile string
	mntDir  string
	testDir string
	tmpDir  string
)

func setUpTestDir() error {
	var err error
	testDir, err = ioutil.TempDir("", "gcsfuse_readwrite_test_")
	if err != nil {
		return fmt.Errorf("TempDir: %w\n", err)
	}

	err = util.BuildGcsfuse(testDir)
	if err != nil {
		return fmt.Errorf("BuildGcsfuse(%q): %w\n", testDir, err)
	}

	binFile = path.Join(testDir, "bin/gcsfuse")
	logFile = path.Join(testDir, "gcsfuse.log")
	mntDir = path.Join(testDir, "mnt")

	err = os.Mkdir(mntDir, 0755)
	if err != nil {
		return fmt.Errorf("Mkdir(%q): %v\n", mntDir, err)
	}
	return nil
}

func mountGcsfuse(flag string) error {
	mountCmd := exec.Command(
		binFile,
		"--debug_gcs",
		"--debug_fs",
		"--debug_fuse",
		"--log-file="+logFile,
		"--log-format=text",
		flag,
		*testBucket,
		mntDir,
	)
	output, err := mountCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot mount gcsfuse: %w\n", err)
	}
	if lines := bytes.Count(output, []byte{'\n'}); lines > 1 {
		return fmt.Errorf("mount output: %q\n", output)
	}
	return nil
}

func unMount() error {
	fusermount, err := exec.LookPath("fusermount")
	if err != nil {
		return fmt.Errorf("cannot find fusermount: %w", err)
	}
	cmd := exec.Command(fusermount, "-uz", mntDir)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fusermount error: %w", err)
	}
	return nil
}

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

func logAndExit(s string) {
	log.Print(s)
	os.Exit(1)
}

func createTempFile() string {
	// A temporary file is created and some lines are added
	// to it for testing purposes.
	fileName := path.Join(tmpDir, "tmpFile")
	err := os.WriteFile(fileName, []byte("line 1\nline 2\n"), 0666)
	if err != nil {
		logAndExit(fmt.Sprintf("Temporary file at %v", err))
	}
	return fileName
}

// executing test
func executeTest(flags []string, m *testing.M) (successCode int, err error) {
	for i := 0; i < len(flags); i++ {
		if err := mountGcsfuse(flags[i]); err != nil {
			log.Printf("mountGcsfuse: %v\n", err)
			return
		}

		log.Printf("Test log: %s\n", logFile)

		// Creating a temporary directory to store files
		// to be used for testing.
		var err error
		tmpDir, err = os.MkdirTemp(mntDir, "tmpDir")
		if err != nil {
			logAndExit(fmt.Sprintf("Mkdir at %q: %v", mntDir, err))
		}

		successCode = m.Run()

		os.RemoveAll(mntDir)
		err = unMount()
		if err != nil {
			return
		}
	}
	return
}

func TestMain(m *testing.M) {
	flag.Parse()

	if *testBucket == "" {
		log.Printf("--testbucket must be specified")
		os.Exit(0)
	}

	if err := setUpTestDir(); err != nil {
		log.Printf("setUpTestDir: %v\n", err)
		os.Exit(1)
	}

	flags := []string{"--experimental-enable-storage-client-library=true",
		"--experimental-enable-storage-client-library=false",
		"--implicit-dirs=true",
		"--implicit-dirs=false"}
	successCode, err := executeTest(flags, m)
	if err != nil {
		return
	}

	os.Exit(successCode)
}
