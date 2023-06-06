// Copyright 2023 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package setup

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/util"
)

var testBucket = flag.String("testbucket", "", "The GCS bucket used for the test.")
var mountedDirectory = flag.String("mountedDirectory", "", "The GCSFuse mounted directory used for the test.")
var integrationTest = flag.Bool("integrationTest", false, "Run tests only when the flag value is true.")
var testInstalledPackage = flag.Bool("testInstalledPackage", false, "[Optional] Run tests on the package pre-installed on the host machine. By default, integration tests build a new package to run the tests.")

const BufferSize = 100
const FilePermission_0600 = 0600

var (
	binFile string
	logFile string
	testDir string
	mntDir  string
)

// Run the shell script to prepare the testData in the specified bucket.
func RunScriptForTestData(script string, testBucket string) {
	cmd := exec.Command("/bin/bash", script, testBucket)
	_, err := cmd.Output()
	if err != nil {
		panic(err)
	}
}

func TestBucket() string {
	return *testBucket
}

func TestInstalledPackage() bool {
	return *testInstalledPackage
}

func MountedDirectory() string {
	return *mountedDirectory
}

func SetLogFile(logFileValue string) {
	logFile = logFileValue
}

func LogFile() string {
	return logFile
}

func SetBinFile(binFileValue string) {
	binFile = binFileValue
}

func BinFile() string {
	return binFile
}

func SetTestDir(testDirValue string) {
	testDir = testDirValue
}

func TestDir() string {
	return testDir
}

func SetMntDir(mntDirValue string) {
	mntDir = mntDirValue
}

func MntDir() string {
	return mntDir
}

func SetTestBucket(testBucketValue string) {
	*testBucket = testBucketValue
}

func CompareFileContents(t *testing.T, fileName string, fileContent string) {
	content, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Read: %v", err)
	}

	if got := string(content); got != fileContent {
		t.Errorf("File content doesn't match. Expected: %q, Actual: %q", got, fileContent)
	}
}

func CreateTempFile() string {
	// A temporary file is created and some lines are added
	// to it for testing purposes.

	fileName := path.Join(mntDir, "tmpFile")
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_DIRECT, FilePermission_0600)
	if err != nil {
		LogAndExit(fmt.Sprintf("Error in the opening the file %v", err))
	}
	defer file.Close()

	_, err = file.WriteString("line 1\nline 2\n")
	if err != nil {
		LogAndExit(fmt.Sprintf("Temporary file at %v", err))
	}

	return fileName
}

func SetUpTestDir() error {
	var err error
	testDir, err = os.MkdirTemp("", "gcsfuse_readwrite_test_")
	if err != nil {
		return fmt.Errorf("TempDir: %w\n", err)
	}

	if !TestInstalledPackage() {
		err = util.BuildGcsfuse(testDir)
		if err != nil {
			return fmt.Errorf("BuildGcsfuse(%q): %w\n", TestDir(), err)
		}
		binFile = path.Join(TestDir(), "bin/gcsfuse")
	} else {
		// when testInstalledPackage flag is set, gcsfuse is preinstalled on the
		// machine. Hence, here we are overwriting binFile to gcsfuse.
		binFile = "gcsfuse"
	}
	logFile = path.Join(TestDir(), "gcsfuse.log")
	mntDir = path.Join(TestDir(), "mnt")

	err = os.Mkdir(mntDir, 0755)
	if err != nil {
		return fmt.Errorf("Mkdir(%q): %v\n", MntDir(), err)
	}
	return nil
}

func UnMount() error {
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

func executeTest(m *testing.M) (successCode int) {
	successCode = m.Run()

	os.RemoveAll(mntDir)

	return successCode
}

func ExecuteTestForFlagsSet(flags []string, m *testing.M) (successCode int) {
	var err error

	// Clean the mountedDirectory before running any tests.
	os.RemoveAll(mntDir)

	successCode = executeTest(m)

	err = UnMount()
	if err != nil {
		LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
	}

	// Print flag on which test fails
	if successCode != 0 {
		f := strings.Join(flags, " ")
		log.Print("Test Fails on " + f)
		return
	}
	return
}

func ParseSetUpFlags() {
	flag.Parse()

	if !*integrationTest {
		log.Print("Pass --integrationTest flag to run the tests.")
		os.Exit(0)
	}
}

func ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet() {
	ParseSetUpFlags()

	if *testBucket == "" && *mountedDirectory == "" {
		log.Print("--testbucket or --mountedDirectory must be specified")
		os.Exit(1)
	}
}

func RunTestsForMountedDirectoryFlag(m *testing.M) {
	// Execute tests for the mounted directory.
	if *mountedDirectory != "" {
		mntDir = *mountedDirectory
		successCode := executeTest(m)
		os.Exit(successCode)
	}
}

func SetUpTestDirForTestBucketFlag() {
	if err := SetUpTestDir(); err != nil {
		log.Printf("setUpTestDir: %v\n", err)
		os.Exit(1)
	}
}

func LogAndExit(s string) {
	log.Print(s)
	os.Exit(1)
}
