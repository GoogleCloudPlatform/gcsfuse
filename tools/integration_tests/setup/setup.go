package setup

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/util"
)

var testBucket = flag.String("testbucket", "", "The GCS bucket used for the test.")
var mountedDirectory = flag.String("mountedDirectory", "", "The GCSFuse mounted directory used for the test.")

var (
	binFile string
	logFile string
	testDir string
	mntDir  string
)

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

func ClearKernelCache() error {
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

func CompareFileContents(t *testing.T, fileName string, fileContent string) {
	// After write, data will be cached by kernel. So subsequent read will be
	// served using cached data by kernel instead of calling gcsfuse.
	// Clearing kernel cache to ensure that gcsfuse is invoked during read operation.
	err := ClearKernelCache()
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

func CreateTempFile() string {
	// A temporary file is created and some lines are added
	// to it for testing purposes.
	fileName := path.Join(mntDir, "tmpFile")
	err := os.WriteFile(fileName, []byte("line 1\nline 2\n"), 0666)
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

	output, err := mountCmd.CombinedOutput()
	if err != nil {
		log.Println(mountCmd.String())
		return fmt.Errorf("cannot mount gcsfuse: %w\n", err)
	}
	if lines := bytes.Count(output, []byte{'\n'}); lines > 1 {
		return fmt.Errorf("mount output: %q\n", output)
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
			LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}

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

func LogAndExit(s string) {
	log.Print(s)
	os.Exit(1)
}
