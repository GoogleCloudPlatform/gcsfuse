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

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/util"
)

var testBucket = flag.String("testbucket", "", "The GCS bucket used for the test.")
var mountedDirectory = flag.String("mountedDirectory", "", "The GCSFuse mounted directory used for the test.")
var integrationTest = flag.Bool("integrationTest", false, "Run tests only when the flag value is true.")
var testInstalledPackage = flag.Bool("testInstalledPackage", false, "[Optional] Run tests on the package pre-installed on the host machine. By default, integration tests build a new package to run the tests.")

const (
	BufferSize          = 100
	FilePermission_0600 = 0600
	DirPermission_0755  = 0755
)

var (
	binFile              string
	logFile              string
	testDir              string
	mntDir               string
	sbinFile             string
	onlyDirMounted       string
	dynamicBucketMounted string
)

// Run the shell script to prepare the testData in the specified bucket.
// First argument will be name of scipt script
func RunScriptForTestData(args ...string) {
	cmd := exec.Command("/bin/bash", args...)
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

func SbinFile() string {
	return sbinFile
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

// OnlyDirMounted returns the name of the directory mounted in case of only dir mount.
func OnlyDirMounted() string {
	return onlyDirMounted
}

// SetOnlyDirMounted sets the name of the directory mounted in case of only dir mount.
func SetOnlyDirMounted(onlyDirValue string) {
	onlyDirMounted = onlyDirValue
}

// DynamicBucketMounted returns the name of the bucket in case of dynamic mount.
func DynamicBucketMounted() string {
	return dynamicBucketMounted
}

// SetDynamicBucketMounted sets the name of the bucket in case of dynamic mount.
func SetDynamicBucketMounted(dynamicBucketValue string) {
	dynamicBucketMounted = dynamicBucketValue
}

func CompareFileContents(t *testing.T, fileName string, fileContent string) {
	content, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Read: %v", err)
	}

	if got := string(content); got != fileContent {
		t.Errorf("File content doesn't match. Expected: %q, Actual: %q", fileContent, got)
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

	defer operations.CloseFile(file)

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
		sbinFile = path.Join(TestDir(), "sbin/mount.gcsfuse")

		// mount.gcsfuse will find gcsfuse executable in mentioned locations.
		// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/tools/mount_gcsfuse/find.go#L59
		// Copying the executable to /usr/local/bin
		err := operations.CopyDirWithRootPermission(binFile, "/usr/local/bin")
		if err != nil {
			log.Printf("Error in copying bin file:%v", err)
		}
	} else {
		// when testInstalledPackage flag is set, gcsfuse is preinstalled on the
		// machine. Hence, here we are overwriting binFile to gcsfuse.
		binFile = "gcsfuse"
		sbinFile = "mount.gcsfuse"
	}
	logFile = path.Join(TestDir(), "gcsfuse.log")
	mntDir = path.Join(TestDir(), "mnt")

	err = os.Mkdir(mntDir, 0755)
	if err != nil {
		return fmt.Errorf("Mkdir(%q): %v\n", MntDir(), err)
	}
	return nil
}

// Removing bin file after testing.
func RemoveBinFileCopiedForTesting() {
	if !TestInstalledPackage() {
		cmd := exec.Command("sudo", "rm", "/usr/local/bin/gcsfuse")
		err := cmd.Run()
		if err != nil {
			log.Printf("Error in removing file:%v", err)
		}
	}
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

func ExecuteTest(m *testing.M) (successCode int) {
	successCode = m.Run()

	return successCode
}

func UnMountAndThrowErrorInFailure(flags []string, successCode int) {
	err := UnMount()
	if err != nil {
		LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
	}

	// Print flag on which test fails
	if successCode != 0 {
		f := strings.Join(flags, " ")
		log.Print("Test Fails on " + f)
		return
	}
}

func ExecuteTestForFlagsSet(flags []string, m *testing.M) (successCode int) {
	successCode = ExecuteTest(m)

	UnMountAndThrowErrorInFailure(flags, successCode)

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
		successCode := ExecuteTest(m)
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

// CleanUpDir cleans up the content in given directory.
func CleanUpDir(directoryPath string) {
	dir, err := os.ReadDir(directoryPath)
	if err != nil {
		log.Printf("Error in reading directory: %v", err)
	}

	for _, d := range dir {
		err := os.RemoveAll(path.Join([]string{directoryPath, d.Name()}...))
		if err != nil {
			log.Printf("Error in removing directory: %v", err)
		}
	}
}

// CleanMntDir cleans the mounted directory.
func CleanMntDir() {
	CleanUpDir(mntDir)
}

// SetupTestDirectory creates a testDirectory in the mounted directory and cleans up
// any content present in it.
func SetupTestDirectory(testDirName string) string {
	testDirPath := path.Join(MntDir(), testDirName)
	err := os.Mkdir(testDirPath, DirPermission_0755)
	if err != nil && !strings.Contains(err.Error(), "file exists") {
		log.Printf("Error while setting up directory %s for testing: %v", testDirPath, err)
	}
	CleanUpDir(testDirPath)
	return testDirPath
}

// CleanupDirectoryOnGCS cleans up the object/directory path passed in parameter.
func CleanupDirectoryOnGCS(directoryPathOnGCS string) {
	_, err := operations.ExecuteGsutilCommandf("rm -rf gs://%s", directoryPathOnGCS)
	if err != nil {
		log.Printf("Error while cleaning up directory %s from GCS: %v",
			directoryPathOnGCS, err)
	}
}
