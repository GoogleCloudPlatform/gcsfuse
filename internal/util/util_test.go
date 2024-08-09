// Copyright 2023 Google LLC
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
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const gcsFuseParentProcessDir = "/var/generic/google"

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type UtilTest struct {
	suite.Suite
}

func TestUtilSuite(t *testing.T) {
	suite.Run(t, new(UtilTest))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (ts *UtilTest) TestResolveWhenParentProcDirEnvNotSetAndFilePathStartsWithTilda() {
	resolvedPath, err := GetResolvedPath("~/test.txt")

	assert.Equal(ts.T(), nil, err)
	homeDir, err := os.UserHomeDir()
	assert.Equal(ts.T(), nil, err)
	assert.Equal(ts.T(), filepath.Join(homeDir, "test.txt"), resolvedPath)
}

func (ts *UtilTest) TestResolveWhenParentProcDirEnvNotSetAndFilePathStartsWithDot() {
	resolvedPath, err := GetResolvedPath("./test.txt")

	assert.Equal(ts.T(), nil, err)
	currentWorkingDir, err := os.Getwd()
	assert.Equal(ts.T(), nil, err)
	assert.Equal(ts.T(), filepath.Join(currentWorkingDir, "./test.txt"), resolvedPath)
}

func (ts *UtilTest) TestResolveWhenParentProcDirEnvNotSetAndFilePathStartsWithDoubleDot() {
	resolvedPath, err := GetResolvedPath("../test.txt")

	assert.Equal(ts.T(), nil, err)
	currentWorkingDir, err := os.Getwd()
	assert.Equal(ts.T(), nil, err)
	assert.Equal(ts.T(), filepath.Join(currentWorkingDir, "../test.txt"), resolvedPath)
}

func (ts *UtilTest) TestResolveWhenParentProcDirEnvNotSetAndRelativePath() {
	resolvedPath, err := GetResolvedPath("test.txt")

	assert.Equal(ts.T(), nil, err)
	currentWorkingDir, err := os.Getwd()
	assert.Equal(ts.T(), nil, err)
	assert.Equal(ts.T(), filepath.Join(currentWorkingDir, "test.txt"), resolvedPath)
}

func (ts *UtilTest) TestResolveWhenParentProcDirEnvNotSetAndAbsoluteFilePath() {
	resolvedPath, err := GetResolvedPath("/var/dir/test.txt")

	assert.Equal(ts.T(), nil, err)
	assert.Equal(ts.T(), "/var/dir/test.txt", resolvedPath)
}

func (ts *UtilTest) ResolveEmptyFilePath() {
	resolvedPath, err := GetResolvedPath("")

	assert.Equal(ts.T(), nil, err)
	assert.Equal(ts.T(), "", resolvedPath)
}

// Below all tests when GCSFUSE_PARENT_PROCESS_DIR env variable is set.
// By setting this environment variable, resolve will work for child process.
func (ts *UtilTest) ResolveWhenParentProcDirEnvSetAndFilePathStartsWithTilda() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := GetResolvedPath("~/test.txt")

	assert.Equal(ts.T(), nil, err)
	homeDir, err := os.UserHomeDir()
	assert.Equal(ts.T(), nil, err)
	assert.Equal(ts.T(), filepath.Join(homeDir, "test.txt"), resolvedPath)
}

func (ts *UtilTest) ResolveWhenParentProcDirEnvSetAndFilePathStartsWithDot() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := GetResolvedPath("./test.txt")

	assert.Equal(ts.T(), nil, err)
	assert.Equal(ts.T(), filepath.Join(gcsFuseParentProcessDir, "./test.txt"), resolvedPath)
}

func (ts *UtilTest) ResolveWhenParentProcDirEnvSetAndFilePathStartsWithDoubleDot() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := GetResolvedPath("../test.txt")

	assert.Equal(ts.T(), nil, err)
	assert.Equal(ts.T(), filepath.Join(gcsFuseParentProcessDir, "../test.txt"), resolvedPath)
}

func (ts *UtilTest) ResolveWhenParentProcDirEnvSetAndRelativePath() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := GetResolvedPath("test.txt")

	assert.Equal(ts.T(), nil, err)
	assert.Equal(ts.T(), filepath.Join(gcsFuseParentProcessDir, "test.txt"), resolvedPath)
}

func (ts *UtilTest) ResolveWhenParentProcDirEnvSetAndAbsoluteFilePath() {
	os.Setenv(GCSFUSE_PARENT_PROCESS_DIR, gcsFuseParentProcessDir)
	defer os.Unsetenv(GCSFUSE_PARENT_PROCESS_DIR)

	resolvedPath, err := GetResolvedPath("/var/dir/test.txt")

	assert.Equal(ts.T(), nil, err)
	assert.Equal(ts.T(), "/var/dir/test.txt", resolvedPath)
}

func (ts *UtilTest) TestMiBsToBytes() {
	cases := []struct {
		mib   uint64
		bytes uint64
	}{
		{
			mib:   0,
			bytes: 0,
		},
		{
			mib:   1,
			bytes: 1048576,
		},
		{
			mib:   5,
			bytes: 5242880,
		},
		{
			mib:   1024,
			bytes: 1073741824,
		},
		{
			mib:   0xFFFFFFFFFFF,      // 2^44 - 1
			bytes: 0xFFFFFFFFFFF00000, // 2^20 * (2^44 - 1)
		},
	}

	for _, tc := range cases {
		assert.Equal(ts.T(), tc.bytes, MiBsToBytes(tc.mib))
	}
}

func (ts *UtilTest) TestBytesToHigherMiBs() {
	cases := []struct {
		bytes uint64
		mib   uint64
	}{
		{
			bytes: 0,
			mib:   0,
		},
		{
			bytes: 1048576,
			mib:   1,
		},
		{
			bytes: 1,
			mib:   1,
		},
		{
			bytes: 0xFFFFFFFFFFF00000, // 2^20 * (2^44 - 1)
			mib:   0xFFFFFFFFFFF,      // 2^44 - 1
		},
		{
			bytes: 0xFFFFFFFFFFF00001, // 2^20 * (2^44 - 1) + 1
			mib:   0x100000000000,     // 2^44
		},
		{
			bytes: math.MaxUint64, // (2^64 - 1)
			mib:   0x100000000000, // 2^44
		},
	}

	for _, tc := range cases {
		assert.Equal(ts.T(), tc.mib, BytesToHigherMiBs(tc.bytes))
	}
}

func (ts *UtilTest) TestIsolateContextFromParentContext() {
	parentCtx, parentCtxCancel := context.WithCancel(context.Background())

	// Call the method and cancel the parent context.
	newCtx, newCtxCancel := IsolateContextFromParentContext(parentCtx)
	parentCtxCancel()

	// Validate new context is not cancelled after parent's cancellation.
	assert.NoError(ts.T(), newCtx.Err())
	// Cancel the new context and validate.
	newCtxCancel()
	assert.ErrorIs(ts.T(), newCtx.Err(), context.Canceled)
}
