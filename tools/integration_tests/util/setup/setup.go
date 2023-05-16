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

const BufferSize = 100
const FilePermission_0600 = 0600

var (
	binFile string
	logFile string
	testDir string
	mntDir  string
)

// Run shell script
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

	err = util.BuildGcsfuse(testDir)
	if err != nil {
		return fmt.Errorf("BuildGcsfuse(%q): %w\n", TestDir(), err)
	}

	binFile = path.Join(TestDir(), "bin/gcsfuse")
	logFile = path.Join(TestDir(), "gcsfuse.log")
	mntDir = path.Join(TestDir(), "mnt")

	err = os.Mkdir(mntDir, 0755)
	if err != nil {
		return fmt.Errorf("Mkdir(%q): %v\n", MntDir(), err)
	}
	return nil
}

func MountGcsfuse(flags []string) error {
	defaultArg := []string{"--debug_gcs",
		"--debug_fs",
		"--debug_fuse",
		"--log-file=" + LogFile(),
		"--log-format=text",
		*testBucket,
		mntDir}

	for i := 0; i < len(defaultArg); i++ {
		flags = append(flags, defaultArg[i])
	}

	mountCmd := exec.Command(
		binFile,
		flags...,
	)

	// Adding mount command in LogFile
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Could not open logfile")
	}
	defer file.Close()

	_, err = file.WriteString(mountCmd.String() + "\n")
	if err != nil {
		fmt.Println("Could not write cmd to logFile")
	}

	_, err = mountCmd.CombinedOutput()
	if err != nil {
		log.Println(mountCmd.String())
		return fmt.Errorf("cannot mount gcsfuse: %w\n", err)
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

	os.RemoveAll(mntDir)

	return successCode
}

func ExecuteTestForFlags(flags [][]string, m *testing.M) (successCode int) {
	var err error

	for i := 0; i < len(flags); i++ {
		if err = MountGcsfuse(flags[i]); err != nil {
			log.Print(fmt.Sprintf("mountGcsfuse: %v\n", err))
			return 1
		}

		// Clean the mountedDirectory before running any tests.
		os.RemoveAll(mntDir)

		successCode = ExecuteTest(m)

		err = UnMount()
		if err != nil {
			LogAndExit(fmt.Sprintf("Error in unmounting bucket: %v", err))
		}

		// Print flag on which test fails
		if successCode != 0 {
			f := strings.Join(flags[i], " ")
			log.Print("Test Fails on " + f)
			return
		}

	}
	return
}

func ParseSetUpFlags() {
	flag.Parse()

	if !*integrationTest {
		log.Printf("Pass --integrationTest flag to run the tests.")
		os.Exit(0)
	}
}

func RunTests(flags [][]string, m *testing.M) (successCode int) {
	// Setting gcloud credentials if removed because of previous run.
	//RunScriptForTestData("../util/key_file/testdata/set_gcloud_creds.sh", "")

	ParseSetUpFlags()

	if *testBucket == "" && *mountedDirectory == "" {
		log.Printf("--testbucket or --mountedDirectory must be specified")
		return 1
	}

	// Execute tests for the mounted directory.
	if *mountedDirectory != "" {
		mntDir = *mountedDirectory
		successCode := ExecuteTest(m)
		return successCode
	}

	// Execute tests for testBucket
	if err := SetUpTestDir(); err != nil {
		log.Printf("setUpTestDir: %v\n", err)
		os.Exit(1)
	}
	successCode = ExecuteTestForFlags(flags, m)

	log.Printf("Test log: %s\n", logFile)

	return successCode
}

func LogAndExit(s string) {
	log.Print(s)
	os.Exit(1)
}
