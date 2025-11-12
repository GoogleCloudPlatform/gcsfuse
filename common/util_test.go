// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package common

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsKLCacheEvictionUnSupported(t *testing.T) {
	testCases := []struct {
		name              string
		mockKernelVersion string
		expectedSkip      bool
	}{
		{
			name:              "Cloudtop_Supported",
			mockKernelVersion: "4.19.0-17-amd64",
			expectedSkip:      false,
		},
		{
			name:              "Cloudtop_Unsupported",
			mockKernelVersion: "6.10.11-1rodete2-amd64",
			expectedSkip:      true,
		},
		{
			name:              "GCP_Supported",
			mockKernelVersion: "6.8.0-1020-gcp",
			expectedSkip:      false,
		},
		{
			name:              "GCP_Unsupported_6.9.x",
			mockKernelVersion: "6.9.0-1020-gcp",
			expectedSkip:      true,
		},
		{
			name:              "GCP_Unsupported_6.10.x",
			mockKernelVersion: "6.10.0-1020-gcp",
			expectedSkip:      true,
		},
		{
			name:              "GCP_Unsupported_6.11.x",
			mockKernelVersion: "6.11.0-1020-gcp",
			expectedSkip:      true,
		},
		{
			name:              "GCP_Unsupported_6.12.x",
			mockKernelVersion: "6.12.0-1020-gcp",
			expectedSkip:      true,
		},
		{
			name:              "Amd64_Unsupported_6.10.x",
			mockKernelVersion: "6.10.0-1-amd64",
			expectedSkip:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalKernelVersion := kernelVersionToTest
			kernelVersionToTest = func() (string, error) { return tc.mockKernelVersion, nil }
			defer func() { kernelVersionToTest = originalKernelVersion }()

			skip, err := IsKLCacheEvictionUnSupported()
			require.NoError(t, err)
			assert.Equal(t, tc.expectedSkip, skip)
		})
	}
}

func TestJoinShutdownFunc(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		fns          []ShutdownFn
		expectedErrs []string
	}{
		{
			name:         "normal",
			fns:          []ShutdownFn{func(_ context.Context) error { return nil }},
			expectedErrs: nil,
		},
		{
			name:         "one_err",
			fns:          []ShutdownFn{func(_ context.Context) error { return fmt.Errorf("err") }},
			expectedErrs: []string{"err"},
		},
		{
			name: "two_err",
			fns: []ShutdownFn{
				func(_ context.Context) error { return fmt.Errorf("err1") },
				func(_ context.Context) error { return fmt.Errorf("err2") },
			},
			expectedErrs: []string{"err1", "err2"},
		},
		{
			name: "two_err_one_normal",
			fns: []ShutdownFn{
				func(_ context.Context) error { return fmt.Errorf("err1") },
				func(_ context.Context) error { return nil },
				func(_ context.Context) error { return fmt.Errorf("err2") },
			},
			expectedErrs: []string{"err1", "err2"},
		},
		{
			name: "nil",
			fns: []ShutdownFn{
				func(_ context.Context) error { return fmt.Errorf("err1") },
				nil,
				func(_ context.Context) error { return fmt.Errorf("err2") },
			},
			expectedErrs: []string{"err1", "err2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := JoinShutdownFunc(tc.fns...)(context.Background())

			if len(tc.expectedErrs) == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				for _, e := range tc.expectedErrs {
					assert.ErrorContains(t, err, e)
				}
			}
		})
	}
}

func TestCloseFile(t *testing.T) {
	// Setup
	f, err := os.CreateTemp("", "testFile-*")
	require.NoError(t, err)

	// Close file and assert
	assert.NotPanics(t, func() { CloseFile(f) })
}

func TestWriteFile(t *testing.T) {
	// Setup
	tmpFile, err := os.CreateTemp("", "testFile-*")
	require.NoError(t, err)
	filePath := tmpFile.Name()
	defer os.Remove(filePath)
	require.NoError(t, tmpFile.Close())

	// Call WriteFile
	err = WriteFile(filePath, "content")

	// Assertions
	assert.NoError(t, err)
	data, err := ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "content", string(data))
}

func TestReadFile(t *testing.T) {
	// Setup
	file, err := os.CreateTemp("", "testFile-*")
	require.NoError(t, err)
	fileName := file.Name()
	defer os.Remove(fileName)
	_, err = file.WriteString("content")
	require.NoError(t, err)
	require.NoError(t, file.Close())

	// Call ReadFile
	content, err := ReadFile(fileName)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "content", string(content))
}
