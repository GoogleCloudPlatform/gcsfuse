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

package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRationalizeCustomEndpointSuccessful(t *testing.T) {
	testCases := []struct {
		name                   string
		config                 *Config
		expectedCustomEndpoint string
	}{
		{
			name: "Valid Config where input and expected custom endpoint match.",
			config: &Config{
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "https://bing.com/search?q=dotnet",
				},
			},
			expectedCustomEndpoint: "https://bing.com/search?q=dotnet",
		},
		{
			name: "Valid Config where input and expected custom endpoint differ.",
			config: &Config{
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "https://j@ne:password@google.com",
				},
			},
			expectedCustomEndpoint: "https://j%40ne:password@google.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := Rationalize(tc.config)

			if assert.NoError(t, actualErr) {
				assert.Equal(t, tc.expectedCustomEndpoint, tc.config.GcsConnection.CustomEndpoint)
			}
		})
	}
}

func TestRationalizeCustomEndpointUnsuccessful(t *testing.T) {
	testCases := []struct {
		name   string
		config *Config
	}{
		{
			name: "Invalid Config",
			config: &Config{
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "a_b://abc",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, Rationalize(tc.config))
		})
	}
}

func TestLoggingSeverityRationalization(t *testing.T) {
	testcases := []struct {
		name       string
		cfgSev     string
		debugFuse  bool
		debugGCS   bool
		debugMutex bool
		expected   LogSeverity
	}{
		{
			name:       "no debug flags",
			cfgSev:     "INFO",
			debugFuse:  false,
			debugGCS:   false,
			debugMutex: false,
			expected:   "INFO",
		},
		{
			name:       "debugFuse true",
			cfgSev:     "INFO",
			debugFuse:  true,
			debugGCS:   false,
			debugMutex: false,
			expected:   "TRACE",
		},
		{
			name:       "debugGCS true",
			cfgSev:     "INFO",
			debugFuse:  false,
			debugGCS:   true,
			debugMutex: false,
			expected:   "TRACE",
		},
		{
			name:       "debugMutex true",
			cfgSev:     "INFO",
			debugFuse:  false,
			debugGCS:   false,
			debugMutex: true,
			expected:   "TRACE",
		},
		{
			name:       "multiple debug flags true",
			cfgSev:     "INFO",
			debugFuse:  true,
			debugGCS:   false,
			debugMutex: true,
			expected:   "TRACE",
		},
	}

	for _, tc := range testcases {
		c := Config{
			Logging: LoggingConfig{
				Severity: LogSeverity(tc.cfgSev),
			},
			Debug: DebugConfig{
				Fuse:     tc.debugFuse,
				Gcs:      tc.debugGCS,
				LogMutex: tc.debugMutex,
			},
		}

		err := Rationalize(&c)

		if assert.NoError(t, err) {
			assert.Equal(t, tc.expected, c.Logging.Severity)
		}
	}
}

func TestRationalize_TokenURLSuccessful(t *testing.T) {
	testCases := []struct {
		name             string
		config           *Config
		expectedTokenURL string
	}{
		{
			name: "Valid Config where input and expected token url match.",
			config: &Config{
				GcsAuth: GcsAuthConfig{
					TokenUrl: "https://bing.com/search?q=dotnet",
				},
			},
			expectedTokenURL: "https://bing.com/search?q=dotnet",
		},
		{
			name: "Valid Config where input and expected token url differ.",
			config: &Config{
				GcsAuth: GcsAuthConfig{
					TokenUrl: "https://j@ne:password@google.com",
				},
			},
			expectedTokenURL: "https://j%40ne:password@google.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := Rationalize(tc.config)

			if assert.NoError(t, actualErr) {
				assert.Equal(t, tc.expectedTokenURL, tc.config.GcsAuth.TokenUrl)
			}
		})
	}
}

func TestRationalize_TokenURLUnsuccessful(t *testing.T) {
	testCases := []struct {
		name   string
		config *Config
	}{
		{
			name: "Invalid Config",
			config: &Config{
				GcsAuth: GcsAuthConfig{
					TokenUrl: "a_b://abc",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, Rationalize(tc.config))
		})
	}
}
