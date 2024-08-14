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

func TestFsErrStrAndCategory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		fsErr            error
		expectedStr      string
		expectedCategory string
	}{
		{
			fsErr:            fmt.Errorf("some random error"),
			expectedStr:      "input/output error",
			expectedCategory: "input/output error",
		},
		{
			fsErr:            syscall.ENOTEMPTY,
			expectedStr:      "directory not empty",
			expectedCategory: "directory not empty",
		},
		{
			fsErr:            syscall.EEXIST,
			expectedStr:      "file exists",
			expectedCategory: "file exists",
		},
		{
			fsErr:            syscall.EINVAL,
			expectedStr:      "invalid argument",
			expectedCategory: "invalid argument",
		},
		{
			fsErr:            syscall.EINTR,
			expectedStr:      "interrupted system call",
			expectedCategory: "interrupt errors",
		},
		{
			fsErr:            syscall.ENOSYS,
			expectedStr:      "function not implemented",
			expectedCategory: "function not implemented",
		},
		{
			fsErr:            syscall.ENOSPC,
			expectedStr:      "no space left on device",
			expectedCategory: "process/resource management errors",
		},
		{
			fsErr:            syscall.E2BIG,
			expectedStr:      "argument list too long",
			expectedCategory: "invalid operation",
		},
		{
			fsErr:            syscall.EHOSTDOWN,
			expectedStr:      "host is down",
			expectedCategory: "network errors",
		},
		{
			fsErr:            syscall.ENODATA,
			expectedStr:      "no data available",
			expectedCategory: "miscellaneous errors",
		},
		{
			fsErr:            syscall.ENODEV,
			expectedStr:      "no such device",
			expectedCategory: "device errors",
		},
		{
			fsErr:            syscall.EISDIR,
			expectedStr:      "is a directory",
			expectedCategory: "file/directory errors",
		},
		{
			fsErr:            syscall.ENOSYS,
			expectedStr:      "function not implemented",
			expectedCategory: "function not implemented",
		},
		{
			fsErr:            syscall.ENFILE,
			expectedStr:      "too many open files in system",
			expectedCategory: "too many open files",
		},
		{
			fsErr:            syscall.EPERM,
			expectedStr:      "operation not permitted",
			expectedCategory: "permission errors",
		},
	}

	for idx, tc := range tests {
		t.Run(fmt.Sprintf("fsErrStrAndCategor_case_%d", idx), func(t *testing.T) {
			t.Parallel()

			actualErrStr, actualErrGrp := errStrAndCategory(tc.fsErr)

			assert.Equal(t, tc.expectedStr, actualErrStr)
			assert.Equal(t, tc.expectedCategory, actualErrGrp)
		})
	}
}
