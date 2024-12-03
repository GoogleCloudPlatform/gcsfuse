// Copyright 2024 Google LLC
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

package proxy_server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfigFile(t *testing.T) {
	// Parse the config file
	config, err := parseConfigFile("testdata/config.yaml")
	assert.NoError(t, err)

	// Assert config values
	assert.Equal(t, "localhost:8080", config.TargetHost)
	assert.Len(t, config.RetryConfig, 2)

	assert.Equal(t, "JsonCreate", config.RetryConfig[0].Method)
	assert.Equal(t, "return-503", config.RetryConfig[0].RetryInstruction)
	assert.Equal(t, 3, config.RetryConfig[0].RetryCount)
	assert.Equal(t, 1, config.RetryConfig[0].SkipCount)

	assert.Equal(t, "JsonStat", config.RetryConfig[1].Method)
	assert.Equal(t, "stall-33s-after-20K", config.RetryConfig[1].RetryInstruction)
	assert.Equal(t, 5, config.RetryConfig[1].RetryCount)
	assert.Equal(t, 2, config.RetryConfig[1].SkipCount)
}
