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

package proxy_server

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfigFile(t *testing.T) {
	// Create a temporary config file
	configContent := `
targetHost: "http://localhost:8080"
retryConfig:
  - method: "JsonCreate"
    retryInstruction: "return-503"
    retryCount: 5
    skipCount: 1
  - method: "JsonStat"
    retryInstruction: "stall-33s-after-20K"
    retryCount: 3
    skipCount: 0
`
	tempFile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err, "failed to create temporary file")
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write([]byte(configContent))
	assert.NoError(t, err, "failed to write to temporary file")
	tempFile.Close()

	// Test parseConfigFile function
	config, err := parseConfigFile(tempFile.Name())
	assert.NoError(t, err, "failed to parse configuration file")
	assert.NotNil(t, config)

	// Assertions for the parsed configuration
	assert.Equal(t, "http://localhost:8080", config.TargetHost, "unexpected TargetHost value")

	assert.Len(t, config.RetryConfig, 2, "unexpected number of RetryConfig entries")
	assert.Equal(t, "JsonCreate", config.RetryConfig[0].Method, "unexpected method in first RetryConfig entry")
	assert.Equal(t, "return-503", config.RetryConfig[0].RetryInstruction, "unexpected retryInstruction in first RetryConfig entry")
	assert.Equal(t, 5, config.RetryConfig[0].RetryCount, "unexpected retryCount in first RetryConfig entry")
	assert.Equal(t, 1, config.RetryConfig[0].SkipCount, "unexpected skipCount in first RetryConfig entry")

	assert.Equal(t, "JsonStat", config.RetryConfig[1].Method, "unexpected method in second RetryConfig entry")
	assert.Equal(t, "stall-33s-after-20K", config.RetryConfig[1].RetryInstruction, "unexpected retryInstruction in second RetryConfig entry")
	assert.Equal(t, 3, config.RetryConfig[1].RetryCount, "unexpected retryCount in second RetryConfig entry")
	assert.Equal(t, 0, config.RetryConfig[1].SkipCount, "unexpected skipCount in second RetryConfig entry")
}
