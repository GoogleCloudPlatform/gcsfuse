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
	FReadOnly  bool
	FWriteOnly bool
	FReadWrite bool
	FAppend    bool
	FODirect   bool
}

func (f *mockOpenFlags) IsReadOnly() bool  { return f.FReadOnly }
func (f *mockOpenFlags) IsWriteOnly() bool { return f.FWriteOnly }
func (f *mockOpenFlags) IsReadWrite() bool { return f.FReadWrite }
func (f *mockOpenFlags) IsAppend() bool    { return f.FAppend }
func (f *mockOpenFlags) IsDirect() bool    { return f.FODirect }

func TestFileOpenMode(t *testing.T) {
	testCases := []struct {
		name         string
		flags        OpenFlagAttributes
		expectedMode OpenMode
	}{
		{
			name: "ReadOnly",
			flags: &mockOpenFlags{
				FReadOnly: true,
			},
			expectedMode: OpenMode{
				AccessMode: ReadOnly,
				FileFlags:  0,
			},
		},
		{
			name: "WriteOnly",
			flags: &mockOpenFlags{
				FWriteOnly: true,
			},
			expectedMode: OpenMode{
				AccessMode: WriteOnly,
				FileFlags:  0,
			},
		},
		{
			name: "ReadWrite",
			flags: &mockOpenFlags{
				FReadWrite: true,
			},
			expectedMode: OpenMode{
				AccessMode: ReadWrite,
				FileFlags:  0,
			},
		},
		{
			name: "ReadWrite with Append",
			flags: &mockOpenFlags{
				FReadWrite: true,
				FAppend:    true,
			},
			expectedMode: OpenMode{
				AccessMode: ReadWrite,
				FileFlags:  O_APPEND,
			},
		},
		{
			name: "WriteOnly with Append and O_DIRECT",
			flags: &mockOpenFlags{
				FWriteOnly: true,
				FAppend:    true,
				FODirect:   true,
			},
			expectedMode: OpenMode{
				AccessMode: WriteOnly,
				FileFlags:  O_APPEND | O_DIRECT,
			},
		},
		{
			name: "ReadOnly with O_DIRECT",
			flags: &mockOpenFlags{
				FReadOnly: true,
				FODirect:  true,
			},
			expectedMode: OpenMode{
				AccessMode: ReadOnly,
				FileFlags:  O_DIRECT,
			},
		},
		{
			name: "ReadWrite with all behavioural flags",
			flags: &mockOpenFlags{
				FReadWrite: true,
				FAppend:    true,
				FODirect:   true,
			},
			expectedMode: OpenMode{
				AccessMode: ReadWrite,
				FileFlags:  O_APPEND | O_DIRECT,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mode := FileOpenMode(tc.flags)
			assert.Equal(t, tc.expectedMode, mode)
		})
	}
}
