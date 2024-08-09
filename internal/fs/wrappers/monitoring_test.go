// Copyright 2024 Google Inc. All Rights Reserved.
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

func TestFsErrStrAndType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		fsErr           error
		expectedErrStr  string
		expectedErrType string
	}{
		{
			fsErr:           fmt.Errorf("some random error"),
			expectedErrStr:  "input/output error",
			expectedErrType: "input/output error",
		},
		{
			fsErr:           syscall.ENOTEMPTY,
			expectedErrStr:  "directory not empty",
			expectedErrType: "directory not empty",
		},
		{
			fsErr:           syscall.EEXIST,
			expectedErrStr:  "file exists",
			expectedErrType: "file exists",
		},
		{
			fsErr:           syscall.EINVAL,
			expectedErrStr:  "invalid argument",
			expectedErrType: "invalid argument",
		},
		{
			fsErr:           syscall.EINTR,
			expectedErrStr:  "interrupted system call",
			expectedErrType: "interrupt errors",
		},
		{
			fsErr:           syscall.ENOSYS,
			expectedErrStr:  "function not implemented",
			expectedErrType: "function not implemented",
		},
		{
			fsErr:           syscall.ENOSPC,
			expectedErrStr:  "no space left on device",
			expectedErrType: "process/resource management errors",
		},
		{
			fsErr:           syscall.E2BIG,
			expectedErrStr:  "argument list too long",
			expectedErrType: "invalid operation",
		},
		{
			fsErr:           syscall.EHOSTDOWN,
			expectedErrStr:  "host is down",
			expectedErrType: "network errors",
		},
		{
			fsErr:           syscall.ENODATA,
			expectedErrStr:  "no data available",
			expectedErrType: "miscellaneous errors",
		},
		{
			fsErr:           syscall.ENODEV,
			expectedErrStr:  "no such device",
			expectedErrType: "device errors",
		},
		{
			fsErr:           syscall.EISDIR,
			expectedErrStr:  "is a directory",
			expectedErrType: "file/directory errors",
		},
		{
			fsErr:           syscall.ENOSYS,
			expectedErrStr:  "function not implemented",
			expectedErrType: "function not implemented",
		},
		{
			fsErr:           syscall.ENFILE,
			expectedErrStr:  "too many open files in system",
			expectedErrType: "too many open files",
		},
		{
			fsErr:           syscall.EPERM,
			expectedErrStr:  "operation not permitted",
			expectedErrType: "permission errors",
		},
	}

	for idx, tc := range tests {
		t.Run(fmt.Sprintf("fsErrStrAndType - case: %d", idx), func(t *testing.T) {
			t.Parallel()
			actualErrStr, actualErrGrp := fsErrStrAndType(tc.fsErr)

			assert.Equal(t, tc.expectedErrStr, actualErrStr)
			assert.Equal(t, tc.expectedErrType, actualErrGrp)
		})
	}
}
