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

package cfg

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOctalTypeInConfigMarshalling(t *testing.T) {
	c := Config{
		FileSystem: FileSystemConfig{
			DirMode: 0755,
		},
	}
	expected :=
		`file-system:
    dir-mode: "755"
`

	str, err := util.YAMLStringify(&c)

	if assert.NoError(t, err) {
		assert.Equal(t, expected, str)
	}
}
func TestOctalMarshalling(t *testing.T) {
	o := Octal(0765)

	str, err := util.YAMLStringify(&o)

	if assert.NoError(t, err) {
		assert.Equal(t, "\"765\"\n", str)
	}
}

func TestOctalUnmarshalling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		str      string
		expected Octal
		wantErr  bool
	}{
		{
			str:      "753",
			expected: 0753,
			wantErr:  false,
		},
		{
			str:      "644",
			expected: 0644,
			wantErr:  false,
		},
		{
			str:     "945",
			wantErr: true,
		},
		{
			str:     "abc",
			wantErr: true,
		},
	}

	for idx, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("octal-unmarshalling: %d", idx), func(t *testing.T) {
			t.Parallel()
			var o Octal

			err := (&o).UnmarshalText([]byte(tc.str))

			if tc.wantErr {
				assert.Error(t, err)
			} else if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, o)
			}
		})
	}
}

func TestLogSeverityUnmarshalling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		str      string
		expected LogSeverity
		wantErr  bool
	}{
		{
			str:      "TRACE",
			expected: "TRACE",
			wantErr:  false,
		},
		{
			str:      "info",
			expected: "INFO",
			wantErr:  false,
		},
		{
			str:      "debUG",
			expected: "DEBUG",
			wantErr:  false,
		},
		{
			str:      "waRniNg",
			expected: "WARNING",
			wantErr:  false,
		},
		{
			str:      "OFF",
			expected: "OFF",
			wantErr:  false,
		},
		{
			str:      "ERROR",
			expected: "ERROR",
			wantErr:  false,
		},
		{
			str:     "EMPEROR",
			wantErr: true,
		},
	}

	for idx, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("log-severity-unmarshalling: %d", idx), func(t *testing.T) {
			t.Parallel()
			var l LogSeverity

			err := (&l).UnmarshalText([]byte(tc.str))

			if tc.wantErr {
				assert.Error(t, err)
			} else if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, l)
			}
		})
	}
}

func TestProtocolUnmarshalling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		str      string
		expected Protocol
		wantErr  bool
	}{
		{
			str:      "http1",
			expected: "http1",
			wantErr:  false,
		},
		{
			str:      "HTtp1",
			expected: "http1",
			wantErr:  false,
		},
		{
			str:      "gRPC",
			expected: "grpc",
			wantErr:  false,
		},
		{
			str:      "HTTP2",
			expected: "http2",
			wantErr:  false,
		},
		{
			str:     "http100",
			wantErr: true,
		},
	}

	for idx, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("protocol-unmarshalling: %d", idx), func(t *testing.T) {
			t.Parallel()
			var p Protocol

			err := (&p).UnmarshalText([]byte(tc.str))

			if tc.wantErr {
				assert.Error(t, err)
			} else if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, p)
			}
		})
	}
}

func TestResolvedPathUnmarshalling(t *testing.T) {
	t.Parallel()
	h, err := os.UserHomeDir()
	require.NoError(t, err)
	tests := []struct {
		str      string
		expected ResolvedPath
	}{
		{
			str:      "~/test.txt",
			expected: ResolvedPath(path.Join(h, "test.txt")),
		},
		{
			str:      "/a/test.txt",
			expected: "/a/test.txt",
		},
	}

	for idx, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("resolved-path-unmarshalling: %d", idx), func(t *testing.T) {
			t.Parallel()
			var p ResolvedPath

			err := (&p).UnmarshalText([]byte(tc.str))

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, p)
			}
		})
	}
}

func TestConfigMarshalling(t *testing.T) {
	t.Parallel()
	c := Config{
		FileSystem: FileSystemConfig{
			FileMode: 0732,
		},
		GcsConnection: GcsConnectionConfig{
			ClientProtocol: "grpc",
			BillingProject: "abc",
		},
		GcsRetries: GcsRetriesConfig{
			MaxRetryAttempts: 45,
		},
	}

	actual, err := util.YAMLStringify(c)

	expected :=
		`file-system:
    file-mode: "732"
gcs-connection:
    billing-project: abc
    client-protocol: grpc
gcs-retries:
    max-retry-attempts: 45
`
	if assert.NoError(t, err) {
		assert.Equal(t, expected, actual)
	}
}
