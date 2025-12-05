// Copyright 2025 Google LLC
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

	"github.com/stretchr/testify/assert"
)

// mockOpenFlags is a mock implementation of OpenFlagAttributes for testing.
type mockOpenFlags struct {
	fReadOnly  bool
	fWriteOnly bool
	fReadWrite bool
	fAppend    bool
	fODirect   bool
}

func (f *mockOpenFlags) IsReadOnly() bool  { return f.fReadOnly }
func (f *mockOpenFlags) IsWriteOnly() bool { return f.fWriteOnly }
func (f *mockOpenFlags) IsReadWrite() bool { return f.fReadWrite }
func (f *mockOpenFlags) IsAppend() bool    { return f.fAppend }
func (f *mockOpenFlags) IsDirect() bool    { return f.fODirect }

func TestFileOpenMode(t *testing.T) {
	testCases := []struct {
		name         string
		flags        OpenFlagAttributes
		expectedMode OpenMode
	}{
		{
			name: "ReadOnly",
			flags: &mockOpenFlags{
				fReadOnly: true,
			},
			expectedMode: NewOpenMode(ReadOnly, 0),
		},
		{
			name: "WriteOnly",
			flags: &mockOpenFlags{
				fWriteOnly: true,
			},
			expectedMode: NewOpenMode(WriteOnly, 0),
		},
		{
			name: "ReadWrite",
			flags: &mockOpenFlags{
				fReadWrite: true,
			},
			expectedMode: NewOpenMode(ReadWrite, 0),
		},
		{
			name: "ReadWrite with Append",
			flags: &mockOpenFlags{
				fReadWrite: true,
				fAppend:    true,
			},
			expectedMode: NewOpenMode(ReadWrite, O_APPEND),
		},
		{
			name: "WriteOnly with Append and O_DIRECT",
			flags: &mockOpenFlags{
				fWriteOnly: true,
				fAppend:    true,
				fODirect:   true,
			},
			expectedMode: NewOpenMode(WriteOnly, O_APPEND|O_DIRECT),
		},
		{
			name: "ReadOnly with O_DIRECT",
			flags: &mockOpenFlags{
				fReadOnly: true,
				fODirect:  true,
			},
			expectedMode: NewOpenMode(ReadOnly, O_DIRECT),
		},
		{
			name: "ReadWrite with all behavioural flags",
			flags: &mockOpenFlags{
				fReadWrite: true,
				fAppend:    true,
				fODirect:   true,
			},
			expectedMode: NewOpenMode(ReadWrite, O_APPEND|O_DIRECT),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mode := FileOpenMode(tc.flags)
			assert.Equal(t, tc.expectedMode, mode)
		})
	}
}
