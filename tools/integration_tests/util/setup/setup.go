// Copyright 2024 Google LLC
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
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"cloud.google.com/go/storage/experimental"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/util"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/api/iterator"
)

var isPresubmitRun = flag.Bool("presubmit", false, "Boolean flag to indicate if test-run is a presubmit run.")
var isZonalBucketRun = flag.Bool("zonal", false, "Boolean flag to indicate if test-run should use a zonal bucket.")
var testBucket = flag.String("testbucket", "", "The GCS bucket used for the test.")
var mountedDirectory = flag.String("mountedDirectory", "", "The GCSFuse mounted directory used for the test.")
var integrationTest = flag.Bool("integrationTest", false, "Run tests only when the flag value is true.")
var testInstalledPackage = flag.Bool("testInstalledPackage", false, "[Optional] Run tests on the package pre-installed on the host machine. By default, integration tests build a new package to run the tests.")
var testOnTPCEndPoint = flag.Bool("testOnTPCEndPoint", false, "Run tests on TPC endpoint only when the flag value is true.")
var gcsfusePreBuiltDir = flag.String("gcsfuse_prebuilt_dir", "", "Path to the pre-built GCSFuse directory containing bin/gcsfuse and sbin/mount.gcsfuse.")
var configFile = flag.String("config-file", "", "Common GCSFuse config file to run tests with.")

const (
	FilePermission_0600               = 0600
	DirPermission_0755                = 0755
	Charset                           = "abcdefghijklmnopqrstuvwxyz0123456789"
	PathEnvVariable                   = "PATH"
	GCSFuseLogFilePrefix              = "gcsfuse-failed-integration-test-logs-"
	ProxyServerLogFilePrefix          = "proxy-server-failed-integration-test-logs-"
	zoneMatcherRegex                  = "^[a-z]+-[a-z0-9]+-[a-z]$"
	regionMatcherRegex                = "^[a-z]+-[a-z0-9]+$"
	unsupportedCharactersInTestBucket = " "
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
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error: %s", out)
		panic(err)
	}
}

func IsPresubmitRun() bool {
	return *isPresubmitRun
}

func IsZonalBucketRun() bool {
	return *isZonalBucketRun
}

func SetIsZonalBucketRun(val bool) {
	*isZonalBucketRun = val
}

func TestBucket() string {
	return *testBucket
}

func TestInstalledPackage() bool {
	return *testInstalledPackage
}

func TestOnTPCEndPoint() bool {
	return *testOnTPCEndPoint
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

func BinFile() string {
	return binFile
}

func SbinFile() string {
	return sbinFile
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

// SetUpTestDir creates a test directory, builds GCSFuse into it and returns it's path.
// This function also creates the mountDir at path testDir/mnt.
func SetUpTestDir() (string, error) {
	var err error
	testDir, err = os.MkdirTemp("", "gcsfuse_readwrite_test_")
	if err != nil {
		return "", fmt.Errorf("TempDir: %w", err)
	}

	// Order of priority to choose GCSFuse installation to run the tests
	// 1. Installed package if explicitly asked to
	// 2. Prebuilt GCSFuse dir if the said flag is passed
	// 3. Build it yourself
	if TestInstalledPackage() {
		// when testInstalledPackage flag is set, gcsfuse is preinstalled on the
		// machine. Hence, here we are overwriting binFile to gcsfuse.
		log.Printf("Using GCSFuse installed on the target machine")
		binFile = "gcsfuse"
		sbinFile = "mount.gcsfuse"
	} else if *gcsfusePreBuiltDir != "" {
		prebuiltDir := *gcsfusePreBuiltDir
		log.Printf("Using GCSFuse from pre-built directory specified by --gcsfuse_prebuilt_dir flag: %s", prebuiltDir)
		binFile = filepath.Join(prebuiltDir, "bin/gcsfuse")
		sbinFile = filepath.Join(prebuiltDir, "sbin/mount.gcsfuse")

		if _, statErr := os.Stat(binFile); statErr != nil {
			return "", fmt.Errorf("gcsfuse binary from --gcsfuse_prebuilt_dir not found at %s: %w", binFile, statErr)
		}
		if _, statErr := os.Stat(sbinFile); statErr != nil {
			return "", fmt.Errorf("mount helper from --gcsfuse_prebuilt_dir not found at %s: %w", sbinFile, statErr)
		}
		// Set PATH to include the bin directory of the pre-built gcsfuse
		err = os.Setenv(PathEnvVariable, filepath.Dir(binFile)+string(filepath.ListSeparator)+os.Getenv(PathEnvVariable))
		if err != nil {
			return "", fmt.Errorf("error setting PATH for --gcsfuse_prebuilt_dir: %v", err.Error())
		}
	} else {
		log.Printf("Building GCSFuse from source in the dir: %s ...", testDir)
		err = util.BuildGcsfuse(testDir)
		if err != nil {
			return "", fmt.Errorf("BuildGcsfuse(%q): %w", TestDir(), err)
		}
		binFile = path.Join(TestDir(), "bin/gcsfuse")
		sbinFile = path.Join(TestDir(), "sbin/mount.gcsfuse")

		// mount.gcsfuse will find gcsfuse executable in mentioned locations.
		// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/tools/mount_gcsfuse/find.go#L59
		// Setting PATH so that executable is found in test directory.
		err := os.Setenv(PathEnvVariable, path.Join(TestDir(), "bin")+string(filepath.ListSeparator)+os.Getenv(PathEnvVariable))
		if err != nil {
			return "", fmt.Errorf("error in setting PATH environment variable: %v", err.Error())
		}
	}

	mntDir = path.Join(TestDir(), "mnt")
	err = os.Mkdir(mntDir, 0755)
	if err != nil {
		return "", fmt.Errorf("Mkdir(%q): %v", MntDir(), err)
	}
	return TestDir(), nil
}

func UnMount(dir string) error {
	fusermount, err := exec.LookPath("fusermount")
	if err != nil {
		return fmt.Errorf("cannot find fusermount: %w", err)
	}
	cmd := exec.Command(fusermount, "-uz", dir)
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
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = Charset[seededRand.Intn(len(Charset))]
	}
	return string(b)
}

func UnMountBucket() {
	err := UnMount(mntDir)
	if err != nil {
		LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
	}
}

func SaveLogFileInCaseOfFailure(successCode int) {
	if successCode != 0 {
		SaveLogFileAsArtifact(LogFile(), GCSFuseLogFilePrefix+GenerateRandomString(5))
	}
}

// Saves logFile as given artifactName in KOKORO or
// TestDir based on where the test is ran.
func SaveLogFileAsArtifact(logFile, artifactName string) {
	logDir := os.Getenv("KOKORO_ARTIFACTS_DIR")
	if logDir == "" {
		// Save log files in TestDir as this run is not on KOKORO.
		logDir = TestDir()
	}
	artifactPath := path.Join(logDir, artifactName)
	logFileData, err := os.ReadFile(logFile)
	if err != nil {
		log.Fatalf("Error reading log file: %v", err)
	}
	err = os.WriteFile(artifactPath, logFileData, 0600)
	if err != nil {
		log.Fatalf("Error in writing log file to artifacts directory: %v", err)
	}
	log.Printf("Log file saved at %v", artifactPath)
}

// In case of test failure saves GCSFuse log file to
// KOKORO artifacts directory if test ran on KOKORO
// or saves to TestDir if test ran on local.
func SaveGCSFuseLogFileInCaseOfFailure(tb testing.TB) {
	if !tb.Failed() || MountedDirectory() != "" {
		return
	}
	SaveLogFileAsArtifact(LogFile(), GCSFuseLogFilePrefix+strings.ReplaceAll(tb.Name(), "/", "_")+GenerateRandomString(5))
}

// In case of test failure saves ProxyServerLogFile to
// KOKORO artifacts directory if test ran on KOKORO
// or saves to TestDir if test ran on local.
func SaveProxyServerLogFileInCaseOfFailure(proxyServerLogFile string, tb testing.TB) {
	if !tb.Failed() {
		return
	}
	SaveLogFileAsArtifact(proxyServerLogFile, ProxyServerLogFilePrefix+strings.ReplaceAll(tb.Name(), "/", "_")+GenerateRandomString(5))
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

func ConfigFile() string {
	if *configFile == "" {
		return ""
	}

	absPath, err := filepath.Abs(*configFile)
	if err != nil {
		log.Fatalf("error decoding config file path: %v", err)
	}
	return absPath
}

func IgnoreTestIfIntegrationTestFlagIsSet(t *testing.T) {
	flag.Parse()

	if *integrationTest {
		t.SkipNow()
	}
}

// IgnoreTestIfIntegrationTestFlagIsNotSet helps skip a test if --integrationTest flag is not set.
// If the test uses TestMain, then one usually calls os.Exit() to skip the test,
// but for non-TestMain tests, this helps skip integration tests if --integrationTest has not been passed.
func IgnoreTestIfIntegrationTestFlagIsNotSet(t *testing.T) {
	flag.Parse()

	if !*integrationTest {
		t.SkipNow()
	}
}

func IgnoreTestIfPresubmitFlagIsSet(b *testing.B) {
	flag.Parse()

	if *isPresubmitRun {
		b.SkipNow()
	}
}

func ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet() {
	ParseSetUpFlags()

	if *testBucket == "" && *mountedDirectory == "" {
		log.Print("--testbucket or --mountedDirectory must be specified")
		os.Exit(1)
	}
}

// Deprecated: Use RunTestsForMountedDirectory instead.
// TODO(b/438068132): cleanup deprecated methods after migration is complete.
func RunTestsForMountedDirectoryFlag(m *testing.M) {
	// Execute tests for the mounted directory.
	if *mountedDirectory != "" {
		os.Exit(RunTestsForMountedDirectory(*mountedDirectory, m))
	}
}

// RunTestsForMountedDirectory executes tests for the mounted directory.
// User is expected to ensure that this function is called when mounted directory is set.
// Returns exit code.
func RunTestsForMountedDirectory(mountedDirectory string, m *testing.M) int {
	// Execute tests for the mounted directory.
	if mountedDirectory == "" {
		log.Println("RunTestsForMountedDirectory failed: Mounted directory is not set.")
		return 1
	}
	mntDir = mountedDirectory
	return ExecuteTest(m)
}

// SetUpTestDirForTestBucketFlag is Deprecated: Use SetUpTestDirForTestBucket instead.
// TODO(b/438068132): cleanup deprecated methods after migration is complete.
func SetUpTestDirForTestBucketFlag() {
	cfg := &test_suite.TestConfig{
		GKEMountedDirectory:     MountedDirectory(),
		GCSFuseMountedDirectory: MntDir(),
		TestBucket:              TestBucket(),
		LogFile:                 LogFile(),
	}
	SetUpTestDirForTestBucket(cfg)
}

// SetUpTestDirForTestBucket creates a test directory with GCSFuse binaries, mount point, log file, etc.
// Test config is passed by reference so it can set the LogFile, mountPath variables in the config
func SetUpTestDirForTestBucket(cfg *test_suite.TestConfig) {
	if cfg.TestBucket == "" {
		log.Fatal("Not running TestBucket tests as --testBucket flag is not set.")
	}
	if strings.ContainsAny(cfg.TestBucket, unsupportedCharactersInTestBucket) {
		log.Fatalf("Passed testBucket %q contains one or more of the following unsupported character(s): %q", cfg.TestBucket, unsupportedCharactersInTestBucket)
	}
	testDirPath, err := SetUpTestDir()
	if err != nil {
		log.Printf("setUpTestDir: %v\n", err)
		os.Exit(1)
	}

	cfg.GCSFuseMountedDirectory = path.Join(testDirPath, "mnt")
	cfg.LogFile = path.Join(TestDir(), "gcsfuse.log")
	// TODO: clean up this global variable up after migration is complete.
	SetLogFile(cfg.LogFile)
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

// SetupTestDirectory creates a test directory hierarchy in the mounted directory,
// cleaning up any content present. It takes a testDirName which can include
// slashes to create nested directories (e.g., "a/b/c").
func SetupTestDirectory(testDirName string) string {
	testDirPath := path.Join(MntDir(), testDirName)
	err := os.MkdirAll(testDirPath, DirPermission_0755)
	if err != nil && !strings.Contains(err.Error(), "file exists") {
		log.Printf("Error while setting up directory %s for testing: %v", testDirPath, err)
	}
	CleanUpDir(testDirPath)
	return testDirPath
}

// SetupTestDirectoryRecursive recursively creates a testDirectory in the mounted directory and cleans up
// any content present in it.
func SetupTestDirectoryRecursive(testDirName string) string {
	testDirPath := path.Join(MntDir(), testDirName)
	err := os.MkdirAll(testDirPath, DirPermission_0755)
	if err != nil && !strings.Contains(err.Error(), "file exists") {
		log.Printf("Error while setting up directory %s for testing: %v", testDirPath, err)
	}
	CleanUpDir(testDirPath)
	return testDirPath
}

// CleanupDirectoryOnGCS cleans up the object/directory path passed in parameter.
func CleanupDirectoryOnGCS(ctx context.Context, client *storage.Client, directoryPathOnGCS string) {
	bucketAndDirPath := strings.Split(directoryPathOnGCS, "/")
	bucket, dirPath := bucketAndDirPath[0], bucketAndDirPath[1]
	bucketHandle := client.Bucket(bucket)

	it := bucketHandle.Objects(ctx, &storage.Query{Prefix: dirPath + "/"})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break // No more objects found
		}
		if err != nil {
			log.Fatalf("Error iterating objects: %v", err)
		}
		if err := bucketHandle.Object(attrs.Name).Delete(ctx); err != nil {
			log.Printf("Error deleting object %s: %v", attrs.Name, err)
		}
	}
}

func AreBothMountedDirectoryAndTestBucketFlagsSet() bool {
	if MountedDirectory() != "" && TestBucket() != "" {
		return true
	}
	log.Print("Not running mounted directory tests as both --mountedDirectory and --testBucket flags are not set.")
	return false
}

// Deprecated: use ResolveIsHierarchicalBucket instead.
// TODO(b/438068132): cleanup deprecated methods after migration is complete.
func IsHierarchicalBucket(ctx context.Context, storageClient *storage.Client) bool {
	return ResolveIsHierarchicalBucket(ctx, TestBucket(), storageClient)
}

func ResolveIsHierarchicalBucket(ctx context.Context, testBucket string, storageClient *storage.Client) bool {
	attrs, err := storageClient.Bucket(testBucket).Attrs(ctx)
	if err != nil {
		return false
	}
	if attrs.HierarchicalNamespace != nil && attrs.HierarchicalNamespace.Enabled {
		return true
	}

	return false
}

// TestEnvironment sets the global variables like test bucket, mount point, log file and isZonalBucket variable
// based on the bucket type. Also returns the bucket type.
func TestEnvironment(ctx context.Context, cfg *test_suite.TestConfig) string {
	// TODO: clean up SetGlobalVars after migration completes.
	SetGlobalVars(cfg)
	bucketType, err := BucketType(ctx, cfg.TestBucket)
	if err != nil {
		log.Fatalf("BucketType failed: %v", err)
	}
	if bucketType == ZonalBucket {
		SetIsZonalBucketRun(true)
	}

	return bucketType
}

const FlatBucket = "flat"
const HNSBucket = "hns"
const ZonalBucket = "zonal"

func BucketType(ctx context.Context, testBucket string) (bucketType string, err error) {
	// For only-dir mounts bucket name is passed as <test_bucket>/<only_dir> by GKE.
	testBucket = strings.Split(testBucket, "/")[0]
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	storageClient, err := storage.NewGRPCClient(ctx, storage.WithDisabledClientMetrics(), experimental.WithGRPCBidiReads())
	if err != nil {
		return "", fmt.Errorf("failed to create storage client: %w", err)
	}
	attrs, err := storageClient.Bucket(testBucket).Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket attributes: %w", err)
	}
	if attrs.LocationType == "zone" {
		return ZonalBucket, nil
	}
	if attrs.HierarchicalNamespace != nil && attrs.HierarchicalNamespace.Enabled {
		return HNSBucket, nil
	}
	return FlatBucket, nil
}

// AddCacheDirToFlags iterates over a set of flag slices and updates any empty "--cache-dir" flags.
func AddCacheDirToFlags(flagSets [][]string, testname string) [][]string {
	for i := range flagSets {
		for j := range flagSets[i] {
			if flagSets[i][j] == "--cache-dir=" {
				flagSets[i][j] = fmt.Sprintf("--cache-dir=%s/cache-dir-%s-%s", os.TempDir(), testname, GenerateRandomString(4))
			}
		}
	}
	return flagSets
}

// BuildFlagSets dynamically builds a list of flag sets based on bucket compatibility.
// bucketType should be "flat", "hns", or "zonal".
// The run parameter filters flag sets based on the 'Run' field in the test
// configuration, which typically corresponds to a specific test name. If run is
// an empty string, all flag sets for the package are returned.
func BuildFlagSets(cfg test_suite.TestConfig, bucketType string, run string) [][]string {
	// In case of mounted-directory, no need to
	// parse flags. Just return a single
	// set of empty flags to run only one test case
	// for a single `go test` command.
	if cfg.GKEMountedDirectory != "" {
		return [][]string{{""}}
	}

	var dynamicFlags [][]string

	// 1. Iterate through each defined test configuration (e.g., HTTP, gRPC).
	for _, testCase := range cfg.Configs {
		// 2. Check if the current test case is compatible with the bucket type.
		// This is a safe and concise way to check the map.
		isCompatible, ok := testCase.Compatible[bucketType]
		if ok && isCompatible && (run == "" || run == testCase.Run) {
			// 3. If compatible, process its flags and add them to the result.
			for _, flagString := range testCase.Flags {
				flagString = strings.ReplaceAll(flagString, ",", " ")
				dynamicFlags = append(dynamicFlags, strings.Fields(flagString))
			}
		}
	}
	return dynamicFlags
}

func SetGlobalVars(cfg *test_suite.TestConfig) {
	// TODO: clean global variables after test migration to config file completes.
	testBucket = &cfg.TestBucket
	logFile = cfg.LogFile
	mntDir = cfg.GKEMountedDirectory
	onlyDirMounted = cfg.OnlyDir
}

// Explicitly set the enable-hns config flag to true when running tests on the HNS bucket.
func AddHNSFlagForHierarchicalBucket(ctx context.Context, storageClient *storage.Client) ([]string, error) {
	if !IsHierarchicalBucket(ctx, storageClient) {
		return nil, fmt.Errorf("bucket is not Hierarchical")
	}

	var flags []string
	mountConfig4 := map[string]any{
		"enable-hns": true,
	}
	filePath4 := YAMLConfigFile(mountConfig4, "config_hns.yaml")
	flags = append(flags, "--config-file="+filePath4)
	return flags, nil
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

func MountGCSFuseWithGivenMountWithConfigFunc(config *test_suite.TestConfig, flags []string, mountFunc func(*test_suite.TestConfig, []string) error) {
	if config.GKEMountedDirectory == "" {
		// Mount GCSFuse only when tests are not running on mounted directory.
		if err := mountFunc(config, flags); err != nil {
			LogAndExit(fmt.Sprintf("Failed to mount GCSFuse: %v", err))
		}
	}
}

// MayMountGCSFuseWithGivenMountWithConfigFunc is similar to MountGCSFuseWithGivenMountWithConfigFunc,
// except that it returns error on failure, instead of panic'ing.
func MayMountGCSFuseWithGivenMountWithConfigFunc(config *test_suite.TestConfig, flags []string, mountFunc func(*test_suite.TestConfig, []string) error) error {
	if config.GKEMountedDirectory == "" {
		// Mount GCSFuse only when tests are not running on mounted directory.
		if err := mountFunc(config, flags); err != nil {
			return fmt.Errorf("Failed to mount GCSFuse: %w", err)
		}
	}
	return nil
}

func UnmountGCSFuseAndDeleteLogFile(rootDir string) {
	UnmountGCSFuse(rootDir)
	// delete log file created
	if *mountedDirectory == "" {
		err := os.Remove(LogFile())
		if err != nil {
			LogAndExit(fmt.Sprintf("Error in deleting log file: %v", err))
		}
	}
}

func UnmountGCSFuse(rootDir string) {
	SetMntDir(rootDir)
	if *mountedDirectory == "" {
		// Unmount GCSFuse only when tests are not running on mounted directory.
		err := UnMount(mntDir)
		if err != nil {
			LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
		}
	}
}

func UnmountGCSFuseWithConfig(cfg *test_suite.TestConfig) {
	if cfg.GKEMountedDirectory == "" {
		// Unmount GCSFuse only when tests are not running on mounted directory.
		err := UnMount(cfg.GCSFuseMountedDirectory)
		if err != nil {
			LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
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

func CreateProxyServerLogFile(t *testing.T) string {
	proxyServerLogFile := path.Join(TestDir(), "proxy-server-log-"+GenerateRandomString(5))
	_, err := os.Create(proxyServerLogFile)
	if err != nil {
		t.Fatalf("Error in creating log file for proxy server: %v", err)
	}
	return proxyServerLogFile
}

func AppendProxyEndpointToFlagSet(flagSet *[]string, port int) {
	*flagSet = append(*flagSet, "--custom-endpoint="+fmt.Sprintf("http://localhost:%d/storage/v1/", port))
}

// GetGCEZone returns the GCE zone of the current machine from
// the GCP resource detector.
func GetGCEZone(ctx context.Context) (string, error) {
	detectedAttrs, err := resource.New(ctx, resource.WithDetectors(gcp.NewDetector()))
	if err != nil {
		return "", fmt.Errorf("failed to fetch GCP resource detector: %w", err)
	}
	attrs := detectedAttrs.Set()
	if zoneValue, exists := attrs.Value("cloud.availability_zone"); exists {
		zone := zoneValue.AsString()
		// Confirm that the zone string is in right format e.g. us-central1-a.
		if match, err := regexp.MatchString(zoneMatcherRegex, zone); !match || err != nil {
			return zone, fmt.Errorf("zone %q returned by GCP resource detector is not a valid zone-string: %w", zone, err)
		}
		return zone, nil
	}
	return "", fmt.Errorf("cloud.availability_zone not found in GCP resource detector")
}

// GetGCERegion return the GCE region for a given GCE zone.
// E.g. from us-central1-a, it returns us-central1.
func GetGCERegion(gceZone string) (string, error) {
	indexOfLastHyphen := strings.LastIndex(gceZone, "-")
	if indexOfLastHyphen < 0 {
		return "", fmt.Errorf("input gceZone %q is not proper. It is expected to be of the form <country>-<region>-<zone> e.g. us-central1-a.", gceZone)
	}
	region := gceZone[:indexOfLastHyphen]

	// Confirm that the region string is in right format e.g. us-central1.
	if match, err := regexp.MatchString(regionMatcherRegex, region); !match || err != nil {
		return region, fmt.Errorf("zone %q returned by GCE metadata server is not a valid zone-string: %w", region, err)
	}
	return region, nil
}

// IsDynamicMount returns true if the mount is dynamic.
// In dynamic mounts, rootDir contains all buckets, and mountDir is the specific bucket directory.
func IsDynamicMount(mountDir, rootDir string) bool {
	return mountDir != rootDir
}

// ExtractServiceVersionFromFlags parses the cloud-profiler-label from a slice of flag strings.
func ExtractServiceVersionFromFlags(flags []string) string {
	// Regex to find --cloud-profiler-label=some_value or --cloud-profiler-label some_value
	re := regexp.MustCompile(`--cloud-profiler-label[=\s]([^\s]+)`)
	for _, flagSet := range flags {
		matches := re.FindStringSubmatch(flagSet)
		// matches[0] is the full match, e.g., "--cloud-profiler-label=v1"
		// matches[1] is the first capturing group, e.g., "v1"
		if len(matches) > 1 {
			return matches[1]
		}
	}
	log.Fatal("Profile label should have been provided for mounted directory test.")
	return ""
}

func OverrideFilePathsInFlagSet(t *test_suite.TestConfig, GCSFuseTempDirPath string) {
	for _, flags := range t.Configs {
		for i := range flags.Flags {
			// Iterate over the indices of the flags slice
			flags.Flags[i] = strings.ReplaceAll(flags.Flags[i], "/gcsfuse-tmp", path.Join(GCSFuseTempDirPath, "gcsfuse-tmp"))
		}
		for i := range flags.SecondaryFlags {
			// Iterate over the indices of the secondary flags slice
			flags.SecondaryFlags[i] = strings.ReplaceAll(flags.SecondaryFlags[i], "/gcsfuse-tmp", path.Join(GCSFuseTempDirPath, "gcsfuse-tmp"))
		}
	}
}

func SetUpLogFilePath(testName string, GKETempDir string, OldGKElogFilePath string, cfg *test_suite.TestConfig) {
	var logFilePath string
	if cfg.GKEMountedDirectory != "" { // GKE path
		logFilePath = path.Join(GKETempDir, testName) + ".log"
		if ConfigFile() == "" {
			// TODO: clean this up when GKE test migration completes.
			logFilePath = OldGKElogFilePath
		}
	} else {
		logFilePath = path.Join(TestDir(), GKETempDir, testName) + ".log"
	}
	cfg.LogFile = logFilePath
	SetLogFile(logFilePath)
}
