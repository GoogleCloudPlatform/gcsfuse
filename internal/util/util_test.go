// Copyright 2023 Google Inc. All Rights Reserved.
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

package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/internal/flag"
	. "github.com/jacobsa/ogletest"
	"github.com/urfave/cli"
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
	resolvedPath, err := GetResolvedPath("~/test.txt")

	AssertEq(nil, err)
	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(homeDir, "test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvNotSetAndFilePathStartsWithDot() {
	resolvedPath, err := GetResolvedPath("./test.txt")

	AssertEq(nil, err)
	currentWorkingDir, err := os.Getwd()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(currentWorkingDir, "./test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvNotSetAndFilePathStartsWithDoubleDot() {
	resolvedPath, err := GetResolvedPath("../test.txt")

	AssertEq(nil, err)
	currentWorkingDir, err := os.Getwd()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(currentWorkingDir, "../test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvNotSetAndRelativePath() {
	resolvedPath, err := GetResolvedPath("test.txt")

	AssertEq(nil, err)
	currentWorkingDir, err := os.Getwd()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(currentWorkingDir, "test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvNotSetAndAbsoluteFilePath() {
	resolvedPath, err := GetResolvedPath("/var/dir/test.txt")

	AssertEq(nil, err)
	ExpectEq("/var/dir/test.txt", resolvedPath)
}

func (t *UtilTest) ResolveEmptyFilePath() {
	resolvedPath, err := GetResolvedPath("")

	AssertEq(nil, err)
	ExpectEq("", resolvedPath)
}

// Below all tests when GCSFUSE_PARENT_PROCESS_DIR env variable is set.
// By setting this environment variable, resolve will work for child process.
func (t *UtilTest) ResolveWhenParentProcDirEnvSetAndFilePathStartsWithTilda() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := GetResolvedPath("~/test.txt")

	AssertEq(nil, err)
	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(homeDir, "test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvSetAndFilePathStartsWithDot() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := GetResolvedPath("./test.txt")

	AssertEq(nil, err)
	ExpectEq(filepath.Join(gcsFuseParentProcessDir, "./test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvSetAndFilePathStartsWithDoubleDot() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := GetResolvedPath("../test.txt")

	AssertEq(nil, err)
	ExpectEq(filepath.Join(gcsFuseParentProcessDir, "../test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvSetAndRelativePath() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := GetResolvedPath("test.txt")

	AssertEq(nil, err)
	ExpectEq(filepath.Join(gcsFuseParentProcessDir, "test.txt"), resolvedPath)
}

func (t *UtilTest) ResolveWhenParentProcDirEnvSetAndAbsoluteFilePath() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := GetResolvedPath("/var/dir/test.txt")

	AssertEq(nil, err)
	ExpectEq("/var/dir/test.txt", resolvedPath)
}

func (t *UtilTest) TestResolveConfigFilePaths() {
	mountConfig := &config.MountConfig{}
	mountConfig.LogConfig = config.LogConfig{
		FilePath: "~/test.txt",
	}

	err := ResolveConfigFilePaths(mountConfig)

	AssertEq(nil, err)
	homeDir, err := os.UserHomeDir()
	AssertEq(nil, err)
	ExpectEq(filepath.Join(homeDir, "test.txt"), mountConfig.LogConfig.FilePath)
}

func (t *UtilTest) TestResolvePathForTheFlagInContext() {
	app := flag.NewApp("")
	currentWorkingDir, err := os.Getwd()
	AssertEq(nil, err)
	app.Action = func(appCtx *cli.Context) {
		err = resolvePathForTheFlagInContext("log-file", appCtx)
		AssertEq(nil, err)
		err = resolvePathForTheFlagInContext("key-file", appCtx)
		AssertEq(nil, err)
		err = resolvePathForTheFlagInContext("config-file", appCtx)
		AssertEq(nil, err)

		ExpectEq(filepath.Join(currentWorkingDir, "test.txt"),
			appCtx.String("log-file"))
		ExpectEq(filepath.Join(currentWorkingDir, "test.txt"),
			appCtx.String("key-file"))
		ExpectEq(filepath.Join(currentWorkingDir, "config.yaml"),
			appCtx.String("config-file"))
	}
	// Simulate argv.
	fullArgs := []string{"some_app", "--log-file=test.txt",
		"--key-file=test.txt", "--config-file=config.yaml"}

	err = app.Run(fullArgs)

	AssertEq(nil, err)
}

func (t *UtilTest) TestResolvePathForTheFlagsInContext() {
	app := flag.NewApp("")
	currentWorkingDir, err := os.Getwd()
	AssertEq(nil, err)
	app.Action = func(appCtx *cli.Context) {
		err = ResolvePathForTheFlagsInContext(appCtx)
		AssertEq(nil, err)

		ExpectEq(filepath.Join(currentWorkingDir, "test.txt"),
			appCtx.String("log-file"))
		ExpectEq(filepath.Join(currentWorkingDir, "test.txt"),
			appCtx.String("key-file"))
		ExpectEq(filepath.Join(currentWorkingDir, "config.yaml"),
			appCtx.String("config-file"))
	}
	// Simulate argv.
	fullArgs := []string{"some_app", "--log-file=test.txt",
		"--key-file=test.txt", "--config-file=config.yaml"}

	err = app.Run(fullArgs)

	AssertEq(nil, err)
}
