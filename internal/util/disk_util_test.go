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
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type DiskUtilTest struct {
	suite.Suite
}

func TestDiskUtilSuite(t *testing.T) {
	suite.Run(t, new(DiskUtilTest))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (ts *DiskUtilTest) TestSpectulativeSizeOnDisk() {
	testcases := []struct {
		name              string
		input_filesize    uint64
		expected_disksize uint64
	}{
		{
			name:              "zero_size",
			input_filesize:    0,
			expected_disksize: 0,
		},
		{
			name:              "small_file",
			input_filesize:    1,
			expected_disksize: 4096,
		},
		{
			name:              "one_block_size",
			input_filesize:    4096,
			expected_disksize: 4096,
		},
		{
			name:              "more_than_one_block_but_less_than_two",
			input_filesize:    4097,
			expected_disksize: 8192,
		},
	}
	for _, tc := range testcases {
		ts.T().Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected_disksize, GetSpeculativeFileSizeOnDisk(tc.input_filesize))
		})
	}
}
