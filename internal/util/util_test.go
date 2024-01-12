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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
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

func (t *UtilTest) TestStringifyShouldReturnAllFieldsPassedInCustomObjectAsMarshalledString() {
	sampleMap := map[string]int{
		"1": 1,
		"2": 2,
		"3": 3,
	}
	sampleNestedValue := nestedCustomType{
		SomeField: 10,
		SomeOther: sampleMap,
	}
	customObject := &customTypeForSuccess{
		Value:       "test_value",
		NestedValue: sampleNestedValue,
	}

	actual := Stringify(customObject)

	expected := "{\"Value\":\"test_value\",\"NestedValue\":{\"SomeField\":10,\"SomeOther\":{\"1\":1,\"2\":2,\"3\":3}}}"
	AssertEq(expected, actual)
}

func (t *UtilTest) TestStringifyShouldReturnEmptyStringWhenMarshalErrorsOut() {
	customInstance := customTypeForError{
		value: "example",
	}

	actual := Stringify(customInstance)

	expected := ""
	AssertEq(expected, actual)
}

type customTypeForSuccess struct {
	Value       string
	NestedValue nestedCustomType
}
type nestedCustomType struct {
	SomeField int
	SomeOther map[string]int
}
type customTypeForError struct {
	value string
}

// MarshalJSON returns an error to simulate a failure during JSON marshaling
func (c customTypeForError) MarshalJSON() ([]byte, error) {
	return nil, errors.New("intentional error during JSON marshaling")
}

func (t *UtilTest) TestShouldReturnFalseIfDirectoryOnGivenPathIsNotWritable() {
	path := "some-path-1"
	err := os.Mkdir(path, 0444)
	AssertEq(nil, err, "Error while creating directory")
	defer func(dirPath string) {
		err := deleteDirectory(dirPath)
		if err != nil {
			AssertEq(nil, err, "Error while deleting temporary test directory")
		}
	}(path)

	result, _ := HasReadWritePerms(path)

	AssertFalse(result)
}

func (t *UtilTest) TestShouldReturnTrueIfDirectoryOnGivenPathHasAllPermissions() {
	path := "some-path-1"
	err := os.Mkdir(path, 0444)
	AssertEq(nil, err, "Error while creating directory")
	// As all systems have 0022 umask, permissions will be overridden, need to modify again with chmod
	err = os.Chmod(path, 0777)
	AssertEq(nil, err, "Error while modifying directory permissions for test")
	defer func(dirPath string) {
		err := deleteDirectory(dirPath)
		if err != nil {
			AssertEq(nil, err, "Error while deleting temporary test directory")
		}
	}(path)

	result, _ := HasReadWritePerms(path)

	AssertTrue(result)
}

func (t *UtilTest) TestShouldReturnTrueIfDirectoryOnGivenPathIsReadableAndWritable() {
	path := "some-path-1"
	err := os.Mkdir(path, 0444)
	AssertEq(nil, err, "Error while creating directory")
	// As all systems have 0022 umask, permissions will be overridden, need to modify again with chmod
	err = os.Chmod(path, 0666)
	AssertEq(nil, err, "Error while modifying directory permissions for test")
	defer func(dirPath string) {
		err := deleteDirectory(dirPath)
		if err != nil {
			AssertEq(nil, err, "Error while deleting temporary test directory")
		}
	}(path)

	result, _ := HasReadWritePerms(path)

	AssertTrue(result)
}

func (t *UtilTest) TestShouldReturnErrorIfDirectoryOnGivenPathIsNotPresent() {
	path := "some-incorrect-path"

	result, err := HasReadWritePerms(path)

	AssertNe(nil, err)
	AssertFalse(result)

}

func deleteDirectory(dirPath string) error {
	err := os.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("Error while deleting test directory %s", err)
	}
	return nil
}
