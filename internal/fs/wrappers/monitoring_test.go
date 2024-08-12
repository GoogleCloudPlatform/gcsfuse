// Copyright 2024 Google LLC
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

package wrappers

import (
	"fmt"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFsErrStrAndGroup(t *testing.T) {
	t.Parallel()
	tests := []struct {
		fsErr          error
		expectedErrStr string
		expectedErrGrp string
	}{
		{
			fsErr:          fmt.Errorf("some random error"),
			expectedErrStr: "input/output error",
			expectedErrGrp: "input/output error",
		},
		{
			fsErr:          syscall.ENOTEMPTY,
			expectedErrStr: "directory not empty",
			expectedErrGrp: "directory not empty",
		},
		{
			fsErr:          syscall.EEXIST,
			expectedErrStr: "file exists",
			expectedErrGrp: "file exists",
		},
		{
			fsErr:          syscall.EINVAL,
			expectedErrStr: "invalid argument",
			expectedErrGrp: "invalid argument",
		},
		{
			fsErr:          syscall.EINTR,
			expectedErrStr: "interrupted system call",
			expectedErrGrp: "interrupt errors",
		},
		{
			fsErr:          syscall.ENOSYS,
			expectedErrStr: "function not implemented",
			expectedErrGrp: "function not implemented",
		},
		{
			fsErr:          syscall.ENOSPC,
			expectedErrStr: "no space left on device",
			expectedErrGrp: "process/resource management errors",
		},
		{
			fsErr:          syscall.E2BIG,
			expectedErrStr: "argument list too long",
			expectedErrGrp: "invalid operation",
		},
		{
			fsErr:          syscall.EHOSTDOWN,
			expectedErrStr: "host is down",
			expectedErrGrp: "network errors",
		},
		{
			fsErr:          syscall.ENODATA,
			expectedErrStr: "no data available",
			expectedErrGrp: "miscellaneous errors",
		},
		{
			fsErr:          syscall.ENODEV,
			expectedErrStr: "no such device",
			expectedErrGrp: "device errors",
		},
		{
			fsErr:          syscall.EISDIR,
			expectedErrStr: "is a directory",
			expectedErrGrp: "file/directory errors",
		},
		{
			fsErr:          syscall.ENOSYS,
			expectedErrStr: "function not implemented",
			expectedErrGrp: "function not implemented",
		},
		{
			fsErr:          syscall.ENFILE,
			expectedErrStr: "too many open files in system",
			expectedErrGrp: "too many open files",
		},
		{
			fsErr:          syscall.EPERM,
			expectedErrStr: "operation not permitted",
			expectedErrGrp: "permission errors",
		},
	}

	for idx, tc := range tests {
		t.Run(fmt.Sprintf("fsErrStrAndGroup - case: %d", idx), func(t *testing.T) {
			t.Parallel()
			actualErrStr, actualErrGrp := fsErrStrAndGroup(tc.fsErr)

			assert.Equal(t, tc.expectedErrStr, actualErrStr)
			assert.Equal(t, tc.expectedErrGrp, actualErrGrp)
		})
	}
}
