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
	"os"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
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

func (t *UtilTest) TestDeepSizeof() {
	// tests for pointers
	var i int
	// var nilPtr *int
	// emptyStr := ""
	helloStr := "hello"
	// emptyIntSlice := []int{}
	// intSlice := []int{1, 2, 3}

	for _, input := range []struct {
		val          any
		expectedSize int
	}{
		// // tests for built-in types
		// {expectedSize: int(unsafe.Sizeof(int(0))), val: int(0)},
		// {expectedSize: int(unsafe.Sizeof(uint(0))), val: uint(0)},
		// {expectedSize: 1, val: int8(0)},
		// {expectedSize: 1, val: uint8(0)},
		// {expectedSize: 2, val: int16(0)},
		// {expectedSize: 2, val: uint16(0)},
		// {expectedSize: 4, val: int32(0)},
		// {expectedSize: 4, val: uint32(0)},
		// {expectedSize: 8, val: int64(0)},
		// {expectedSize: 8, val: uint64(0)},
		// {expectedSize: 1, val: true},
		// {expectedSize: 4, val: float32(0)},
		// {expectedSize: 8, val: float64(0)},
		// // tests for strings
		// {
		// 	expectedSize: int(unsafe.Sizeof(emptyStr)), val: "",
		// },
		// {
		// 	expectedSize: int(unsafe.Sizeof(emptyStr)) + len(helloStr), val: helloStr,
		// },
		// // tests for pointers
		// {
		// 	expectedSize: int(unsafe.Sizeof(&i)) + DeepSizeof(i), val: &i,
		// },
		// {
		// 	expectedSize: int(unsafe.Sizeof(&emptyStr)) + DeepSizeof(emptyStr), val: &emptyStr,
		// },
		// {
		// 	// expectedSize: DeepSizeof(&emptyStr) + DeepSizeof(helloStr), val: &helloStr,
		// 	expectedSize: 8 + 21, val: &helloStr,
		// },
		// // tests for structs
		// // empty struct
		// {
		// 	expectedSize: 0, val: (struct{}{}),
		// },
		// // struct with int
		// {
		// 	expectedSize: DeepSizeof(i), val: (struct{ x int }{x: 0}),
		// },
		// // struct with int-pointer
		// {
		// 	expectedSize: DeepSizeof(nilPtr), val: (struct{ x *int }{x: &i}),
		// },
		// struct with int-pointer and string
		{
			// expectedSize: DeepSizeof(nilPtr) + DeepSizeof(helloStr),
			expectedSize: 29,
			val: (struct {
				x *int
				s string
			}{x: &i, s: helloStr}),
		},
		// // tests for slices
		// {
		// 	expectedSize: int(unsafe.Sizeof(emptyIntSlice)), val: emptyIntSlice,
		// },
		// {
		// 	expectedSize: len(intSlice)*DeepSizeof(i) + DeepSizeof(emptyIntSlice), val: intSlice,
		// },
	} {
		actualSize := DeepSizeof(input.val)
		logger.Infof("input-type: %T, input: %+v, actual-size: %v", input.val, input, actualSize)
		AssertEq(input.expectedSize, actualSize)
	}
}
