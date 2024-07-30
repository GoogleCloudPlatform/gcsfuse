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

func validLogRotateConfig() LogRotateLoggingConfig {
	return LogRotateLoggingConfig{
		BackupFileCount: 0,
		Compress:        false,
		MaxFileSizeMb:   1,
	}
}

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name                   string
		config                 *Config
		expectedCustomEndpoint string
		wantErr                bool
	}{
		{
			name: "Valid Config 1",
			config: &Config{
				Logging: LoggingConfig{LogRotate: validLogRotateConfig()},
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "https://bing.com/search?q=dotnet",
				},
			},
			expectedCustomEndpoint: "https://bing.com/search?q=dotnet",
			wantErr:                false,
		},
		{
			name: "Valid Config 2",
			config: &Config{
				Logging: LoggingConfig{LogRotate: validLogRotateConfig()},
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "https://j@ne:password@google.com",
				},
			},
			expectedCustomEndpoint: "https://j%40ne:password@google.com",
			wantErr:                false,
		},
		{
			name: "Invalid Config",
			config: &Config{
				Logging: LoggingConfig{LogRotate: validLogRotateConfig()},
				GcsConnection: GcsConnectionConfig{
					CustomEndpoint: "a_b://abc",
				},
			},
			expectedCustomEndpoint: "",
			wantErr:                true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := ValidateConfig(tc.config)

			assert.Equal(t, tc.config.GcsConnection.CustomEndpoint, tc.expectedCustomEndpoint)
			if tc.wantErr {
				assert.Error(t, actualErr)
			} else {
				assert.NoError(t, actualErr)
			}
		})
	}
}
