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
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/util"
)

var testBucket = flag.String("testbucket", "", "The GCS bucket used for the test.")
var mountedDirectory = flag.String("mountedDirectory", "", "The GCSFuse mounted directory used for the test.")
var integrationTest = flag.Bool("integrationTest", false, "Run tests only when the flag value is true.")
var testInstalledPackage = flag.Bool("testInstalledPackage", false, "[Optional] Run tests on the package pre-installed on the host machine. By default, integration tests build a new package to run the tests.")

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

const (
	BufferSize          = 100
	FilePermission_0600 = 0600
	DirPermission_0755  = 0755
	Charset             = "abcdefghijklmnopqrstuvwxyz0123456789"
	PathEnvVariable     = "PATH"
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
	out, err := cmd.Output()
	if err != nil {
		log.Printf("Error: %s", out)
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
		// Setting PATH so that executable is found in test directory.
		err := os.Setenv(PathEnvVariable, path.Join(TestDir(), "bin")+string(filepath.ListSeparator)+os.Getenv(PathEnvVariable))
		if err != nil {
			log.Printf("Error in setting PATH environment variable: %v", err.Error())
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

func GenerateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = Charset[seededRand.Intn(len(Charset))]
	}
	return string(b)
}

func UnMountBucket() {
	err := UnMount()
	if err != nil {
		LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
	}
}

func SaveLogFileInCaseOfFailure(successCode int) {
	if successCode != 0 {
		// Logfile name will be gcsfuse-failed-integration-test-log-xxxxx
		failedlogsFileName := "gcsfuse-failed-integration-test-logs-" + GenerateRandomString(5)
		log.Printf("log file is available on kokoro artifacts with file name: %s", failedlogsFileName)
		logFileInKokoroArtifact := path.Join(os.Getenv("KOKORO_ARTIFACTS_DIR"), failedlogsFileName)
		err := operations.CopyFile(logFile, logFileInKokoroArtifact)
		if err != nil {
			log.Fatalf("Error in coping logfile in kokoro artifact: %v", err)
		}
	}
}

func UnMountAndThrowErrorInFailure(flags []string, successCode int) {
	UnMountBucket()
	if successCode != 0 {
		// Print flag on which test fails
		f := strings.Join(flags, " ")
		log.Print("Test Fails on " + f)
		SaveLogFileInCaseOfFailure(successCode)
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

func IgnoreTestIfIntegrationTestFlagIsNotSet(t *testing.T) {
	flag.Parse()

	if !*integrationTest {
		t.SkipNow()
	}
}

func IgnoreTestIfIntegrationTestFlagIsSet(t *testing.T) {
	flag.Parse()

	if *integrationTest {
		t.SkipNow()
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
	if TestBucket() == "" {
		log.Fatal("Not running TestBucket tests as --testBucket flag is not set.")
	}
	if err := SetUpTestDir(); err != nil {
		log.Printf("setUpTestDir: %v\n", err)
		os.Exit(1)
	}
}

func SetUpLogDirForTestDirTests(logDirName string) (logDir string) {
	logDir = path.Join(TestDir(), logDirName)
	err := os.Mkdir(logDir, DirPermission_0755)
	if err != nil {
		log.Printf("os.Mkdir %s: %v\n", logDir, err)
		os.Exit(1)
	}
	return
}

func ValidateLogDirForMountedDirTests(logDirName string) (logDir string) {
	if *mountedDirectory == "" {
		return ""
	}
	logDir = path.Join(os.TempDir(), logDirName)
	_, err := os.Stat(logDir)
	if err != nil {
		log.Printf("validateLogDirForMountedDirTests %s: %v\n", logDir, err)
		os.Exit(1)
	}
	return
}

func LogAndExit(s string) {
	log.Print(s)
	log.Print(string(debug.Stack()))
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
		if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
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
	_, err := operations.ExecuteGcloudCommandf("storage rm -r gs://%s", directoryPathOnGCS)
	if err != nil {
		log.Printf("Error while cleaning up directory %s from GCS: %v",
			directoryPathOnGCS, err)
	}
}

func AreBothMountedDirectoryAndTestBucketFlagsSet() bool {
	if MountedDirectory() != "" && TestBucket() != "" {
		return true
	}
	log.Print("Not running mounted directory tests as both --mountedDirectory and --testBucket flags are not set.")
	return false
}

func separateBucketAndObjectName(bucket, object string) (string, string) {
	bucketAndObjectPath := strings.SplitN(bucket, "/", 2)
	bucket = bucketAndObjectPath[0]
	object = path.Join(bucketAndObjectPath[1], object)
	return bucket, object
}

func GetBucketAndObjectBasedOnTypeOfMount(object string) (string, string) {
	bucket := TestBucket()
	if strings.Contains(TestBucket(), "/") {
		// This case arises when we run tests on mounted directory and pass
		// bucket/directory in testbucket flag.
		bucket, object = separateBucketAndObjectName(bucket, object)
	}
	if dynamicBucketMounted != "" {
		bucket = dynamicBucketMounted
	}
	if OnlyDirMounted() != "" {
		var suffix string
		if strings.HasSuffix(object, "/") {
			suffix = "/"
		}
		object = path.Join(OnlyDirMounted(), object) + suffix
	}
	return bucket, object
}

func MountGCSFuseWithGivenMountFunc(flags []string, mountFunc func([]string) error) {
	if *mountedDirectory == "" {
		// Mount GCSFuse only when tests are not running on mounted directory.
		if err := mountFunc(flags); err != nil {
			LogAndExit(fmt.Sprintf("Failed to mount GCSFuse: %v", err))
		}
	}
}

func UnmountGCSFuseAndDeleteLogFile(rootDir string) {
	SetMntDir(rootDir)
	if *mountedDirectory == "" {
		// Unmount GCSFuse only when tests are not running on mounted directory.
		err := UnMount()
		if err != nil {
			LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
		}
		// delete log file created
		err = os.Remove(LogFile())
		if err != nil {
			LogAndExit(fmt.Sprintf("Error in deleting log file: %v", err))
		}
	}
}

func RunTestsOnlyForStaticMount(mountDir string, t *testing.T) {
	if strings.Contains(mountDir, *testBucket) || OnlyDirMounted() != "" {
		log.Println("This test will run only for static mounting...")
		t.SkipNow()
	}
}

// AppendFlagsToAllFlagsInTheFlagsSet appends each flag in newFlags to every flags present in the
// flagsSet.
// Input flagsSet: [][]string{{"--x", "--y"}, {"--x", "--z"}}
// Input newFlags: {"--a", "--b", ""}
// Output modified flagsSet: [][]string{{"--x", "--y", "--a"}, {"--x", "--z", "--a"},{"--x", "--y", "--b"},{"--x", "--z", "--b"},{"--x", "--y"}, {"--x", "--z"}}
func AppendFlagsToAllFlagsInTheFlagsSet(flagsSet *[][]string, newFlags ...string) {
	var resultFlagsSet [][]string
	for _, flags := range *flagsSet {
		for _, newFlag := range newFlags {
			f := flags
			if strings.Compare(newFlag, "") != 0 {
				f = append(flags, newFlag)
			}
			resultFlagsSet = append(resultFlagsSet, f)
		}
	}
	*flagsSet = resultFlagsSet
}
