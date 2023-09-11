package main

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/jacobsa/ogletest"
)

const gcsFuseParentProcessDir = "/var/generic/google"

func TestUtil(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type UtilTest struct {
}

func init() { RegisterTestSuite(&UtilTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *UtilTest) ResolveWhenParentProcDirEnvNotSetAndFilePathStartsWithTilda() {
	resolvedPath, err := getResolvedPath("~/test.txt")

	AssertEq(nil, err)
	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(homeDir, "test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvNotSetAndFilePathStartsWithDot() {
	resolvedPath, err := getResolvedPath("./test.txt")

	AssertEq(nil, err)
	currentWorkingDir, err := os.Getwd()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(currentWorkingDir, "./test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvNotSetAndFilePathStartsWithDoubleDot() {
	resolvedPath, err := getResolvedPath("../test.txt")

	AssertEq(nil, err)
	currentWorkingDir, err := os.Getwd()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(currentWorkingDir, "../test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvNotSetAndRelativePath() {
	resolvedPath, err := getResolvedPath("test.txt")

	AssertEq(nil, err)
	currentWorkingDir, err := os.Getwd()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(currentWorkingDir, "test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvNotSetAndAbsoluteFilePath() {
	resolvedPath, err := getResolvedPath("/var/dir/test.txt")

	AssertEq(nil, err)
	ExpectEq("/var/dir/test.txt", resolvedPath)
}

func (t *UtilTest) ResolveEmptyFilePath() {
	resolvedPath, err := getResolvedPath("")

	AssertEq(nil, err)
	ExpectEq("", resolvedPath)
}

// Below all tests when GCSFUSE_PARENT_PROCESS_DIR env variable is set.
// By setting this environment variable, resolve will work for child process.
func (t *UtilTest) ResolveWhenParentProcDirEnvSetAndFilePathStartsWithTilda() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := getResolvedPath("~/test.txt")

	AssertEq(nil, err)
	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(homeDir, "test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvSetAndFilePathStartsWithDot() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := getResolvedPath("./test.txt")

	AssertEq(nil, err)
	ExpectEq(filepath.Join(gcsFuseParentProcessDir, "./test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvSetAndFilePathStartsWithDoubleDot() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := getResolvedPath("../test.txt")

	AssertEq(nil, err)
	ExpectEq(filepath.Join(gcsFuseParentProcessDir, "../test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvSetAndRelativePath() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := getResolvedPath("test.txt")

	AssertEq(nil, err)
	ExpectEq(filepath.Join(gcsFuseParentProcessDir, "test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvSetAndAbsoluteFilePath() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := getResolvedPath("/var/dir/test.txt")

	AssertEq(nil, err)
	ExpectEq("/var/dir/test.txt", resolvedPath)
}
