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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverrideWithLoggingFlags(t *testing.T) {
	testCases := []struct {
		name        string
		mountConfig *Config
		debugFuse   bool
		debugGCS    bool
		debugMutex  bool
		expected    LogSeverity
	}{
		{
			name:        "No debug flags",
			mountConfig: &Config{Logging: LoggingConfig{Severity: "DEBUG"}}, // Initial severity
			expected:    "DEBUG",                                            // Should remain unchanged
		},
		{
			name:        "debugFuse true",
			mountConfig: &Config{Logging: LoggingConfig{Severity: "INFO"}},
			debugFuse:   true,
			expected:    "TRACE",
		},
		{
			name:        "debugGCS true",
			mountConfig: &Config{Logging: LoggingConfig{Severity: "WARNING"}},
			debugGCS:    true,
			expected:    "TRACE",
		},
		{
			name:        "debugMutex true",
			mountConfig: &Config{Logging: LoggingConfig{Severity: "OFF"}},
			debugMutex:  true,
			expected:    "TRACE",
		},
		{
			name:        "Multiple debug flags true",
			mountConfig: &Config{Logging: LoggingConfig{Severity: "INFO"}},
			debugFuse:   true,
			debugGCS:    true,
			expected:    "TRACE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			OverrideWithLoggingFlags(tc.mountConfig, tc.debugFuse, tc.debugGCS, tc.debugMutex)

			assert.Equal(t, tc.expected, tc.mountConfig.Logging.Severity)
		})
	}
}
