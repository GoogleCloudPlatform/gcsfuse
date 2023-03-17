package setup

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/googlecloudplatform/gcsfuse/tools/util"
)

var testBucket = flag.String("testbucket", "", "The GCS bucket used for the test.")
var mountedDirectory = flag.String("mountedDirectory", "", "The GCSFuse mounted directory used for the test.")

var (
	binFile string
	logFile string
	testDir string
	tmpDir  string
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

func SetTmpDir(tmpDirValue string) {
	tmpDir = tmpDirValue
}

func SetMntDir(mntDirValue string) {
	mntDir = mntDirValue
}

func MntDir() string {
	return mntDir
}

func TestDir() string {
	return testDir
}

func TmpDir() string {
	return tmpDir
}

func SetUpTestDir() error {
	var err error
	testDir, err := os.MkdirTemp("", "gcsfuse_readwrite_test_")
	if err != nil {
		return fmt.Errorf("TempDir: %w\n", err)
	}
	SetTestDir(testDir)

	err = util.BuildGcsfuse(TestDir())
	if err != nil {
		return fmt.Errorf("BuildGcsfuse(%q): %w\n", TestDir(), err)
	}

	SetBinFile(path.Join(TestDir(), "bin/gcsfuse"))
	SetLogFile(path.Join(TestDir(), "gcsfuse.log"))
	SetMntDir(path.Join(TestDir(), "mnt"))

	err = os.Mkdir(MntDir(), 0755)
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
		TestBucket(),
		MntDir()}

	for i := 0; i < len(defaultArg); i++ {
		flags = append(flags, defaultArg[i])
	}

	mountCmd := exec.Command(
		BinFile(),
		flags...,
	)

	// Adding mount command in LogFile
	file, err := os.OpenFile(LogFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
	cmd := exec.Command(fusermount, "-uz", MntDir())
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fusermount error: %w", err)
	}
	return nil
}

func LogAndExit(s string) {
	log.Print(s)
	os.Exit(1)
}
