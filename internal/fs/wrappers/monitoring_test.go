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
	"github.com/vipnydav/gcsfuse/v3/metrics"
)

func TestFsErrStrAndCategory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		fsErr            error
		expectedCategory metrics.FsErrorCategory
	}{
		{
			fsErr:            fmt.Errorf("some random error"),
			expectedCategory: errIO,
		},
		{
			fsErr:            syscall.ENOTEMPTY,
			expectedCategory: errDirNotEmpty,
		},
		{
			fsErr:            syscall.EEXIST,
			expectedCategory: errFileExists,
		},
		{
			fsErr:            syscall.EINVAL,
			expectedCategory: errInvalidArg,
		},
		{
			fsErr:            syscall.EINTR,
			expectedCategory: errInterrupt,
		},
		{
			fsErr:            syscall.ENOSYS,
			expectedCategory: errNotImplemented,
		},
		{
			fsErr:            syscall.ENOSPC,
			expectedCategory: errProcessMgmt,
		},
		{
			fsErr:            syscall.E2BIG,
			expectedCategory: errInvalidOp,
		},
		{
			fsErr:            syscall.EHOSTDOWN,
			expectedCategory: errNetwork,
		},
		{
			fsErr:            syscall.ENODATA,
			expectedCategory: errMisc,
		},
		{
			fsErr:            syscall.ENODEV,
			expectedCategory: errDevice,
		},
		{
			fsErr:            syscall.EISDIR,
			expectedCategory: errFileDir,
		},
		{
			fsErr:            syscall.ENOSYS,
			expectedCategory: errNotImplemented,
		},
		{
			fsErr:            syscall.ENFILE,
			expectedCategory: errTooManyFiles,
		},
		{
			fsErr:            syscall.EPERM,
			expectedCategory: errPerm,
		},
	}

	for idx, tc := range tests {
		t.Run(fmt.Sprintf("fsErrStrAndCategor_case_%d", idx), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expectedCategory, categorize(tc.fsErr))
		})
	}
}
