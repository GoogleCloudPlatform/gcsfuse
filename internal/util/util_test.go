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
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func (t *UtilTest) TestRoundDurationToNextMultiple() {
	cases := []struct {
		input  time.Duration
		factor time.Duration
		output time.Duration
	}{
		// Following are test cases for rounding up to the next multiple of a second.
		{
			input:  -time.Second - time.Nanosecond,
			factor: time.Second,
			output: -time.Second,
		},
		{
			input:  -time.Second,
			factor: time.Second,
			output: -time.Second,
		},
		{
			input:  -time.Second + time.Nanosecond,
			factor: time.Second,
			output: 0,
		},
		{
			input:  -time.Nanosecond,
			factor: time.Second,
			output: 0,
		},
		{
			input:  0,
			factor: time.Second,
			output: 0,
		},
		{
			input:  time.Nanosecond,
			factor: time.Second,
			output: time.Second,
		},
		{
			input:  time.Second - time.Nanosecond,
			factor: time.Second,
			output: time.Second,
		},
		// Following are test cases for rounding up to the next multiple of a fraction of a second.
		{
			input:  -time.Second,
			factor: 300 * time.Millisecond,
			output: -900 * time.Millisecond,
		},
		{
			input:  time.Second,
			factor: 300 * time.Millisecond,
			output: 1200 * time.Millisecond,
		},
		{
			input:  500 * time.Millisecond,
			factor: 250 * time.Millisecond,
			output: 500 * time.Millisecond,
		},
		{
			input:  -500 * time.Millisecond,
			factor: 250 * time.Millisecond,
			output: -500 * time.Millisecond,
		},
		// Following are test cases for rounding up to the next multiple of larger than a second.
		{
			input:  -time.Second,
			factor: 1300 * time.Millisecond,
			output: 0,
		},
		{
			input:  -time.Second - 300*time.Millisecond,
			factor: 1300 * time.Millisecond,
			output: -1300 * time.Millisecond,
		},
		{
			input:  time.Second,
			factor: 1300 * time.Millisecond,
			output: 1300 * time.Millisecond,
		},
		{
			input:  time.Second + 300*time.Millisecond,
			factor: 1300 * time.Millisecond,
			output: 1300 * time.Millisecond,
		},
		{
			input:  time.Second + 300*time.Millisecond + time.Nanosecond,
			factor: 1300 * time.Millisecond,
			output: 2*time.Second + 600*time.Millisecond,
		},
	}

	for _, tc := range cases {
		AssertEq(tc.output, RoundDurationToNextMultiple(tc.input, tc.factor))
	}
}

func (t *UtilTest) TestMinInt64() {
	cases := []struct {
		input1, input2 int64
		output         int64
	}{
		{
			input1: math.MinInt64,
			input2: math.MinInt64,
			output: math.MinInt64,
		},
		{
			input1: math.MinInt64,
			input2: math.MinInt64 + 1,
			output: math.MinInt64,
		},
		{
			input1: math.MinInt64,
			input2: math.MaxInt64,
			output: math.MinInt64,
		},
		{
			input1: math.MaxInt64,
			input2: math.MaxInt64,
			output: math.MaxInt64,
		},
		{
			input1: math.MaxInt64,
			input2: math.MaxInt64 - 1,
			output: math.MaxInt64 - 1,
		},
	}

	for _, tc := range cases {
		AssertEq(tc.output, minInt64(tc.input1, tc.input2))
	}
}

func (t *UtilTest) TestMinDuration() {
	cases := []struct {
		input1, input2 time.Duration
		output         time.Duration
	}{
		{
			input1: 0,
			input2: 0,
			output: 0,
		},
		{
			input1: 0,
			input2: time.Nanosecond,
			output: 0,
		},
		{
			input1: 0,
			input2: -time.Nanosecond,
			output: -time.Nanosecond,
		},
		{
			input1: time.Nanosecond,
			input2: time.Nanosecond,
			output: time.Nanosecond,
		},
		{
			input1: -time.Nanosecond,
			input2: -time.Nanosecond,
			output: -time.Nanosecond,
		},
		{
			input1: time.Second,
			input2: time.Millisecond,
			output: time.Millisecond,
		},
		{
			input1: time.Millisecond,
			input2: time.Microsecond,
			output: time.Microsecond,
		},
	}

	for _, tc := range cases {
		AssertEq(tc.output, MinDuration(tc.input1, tc.input2))
	}
}
