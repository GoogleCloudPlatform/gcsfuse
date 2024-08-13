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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_DefaultMaxParallelDownloads(t *testing.T) {
	assert.GreaterOrEqual(t, DefaultMaxParallelDownloads(), 16)
}

func TestIsFileCacheEnabled(t *testing.T) {
	testCases := []struct {
		name                       string
		config                     *Config
		expectedIsFileCacheEnabled bool
	}{
		{
			name: "Config with CacheDir set and cache size non zero.",
			config: &Config{
				CacheDir: "/tmp/folder/",
				FileCache: FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedIsFileCacheEnabled: true,
		},
		{
			name:                       "Empty Config.",
			config:                     &Config{},
			expectedIsFileCacheEnabled: false,
		},
		{
			name: "Config with CacheDir unset",
			config: &Config{
				CacheDir: "",
				FileCache: FileCacheConfig{
					MaxSizeMb: -1,
				},
			},
			expectedIsFileCacheEnabled: false,
		},
		{
			name: "Config with CacheDir set and cache size zero.",
			config: &Config{
				CacheDir: "//tmp//folder//",
				FileCache: FileCacheConfig{
					MaxSizeMb: 0,
				},
			},
			expectedIsFileCacheEnabled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedIsFileCacheEnabled, IsFileCacheEnabled(tc.config))
		})
	}

}

func Test_IsTtlInSecsValid(t *testing.T) {
	var testCases = []struct {
		testName    string
		ttlInSecs   int64
		expectedErr error
	}{
		{"Negative", -5, fmt.Errorf(ttlInSecsInvalidValueError)},
		{"Valid negative", -1, nil},
		{"Positive", 8, nil},
		{"Unsupported Large positive", 9223372037, fmt.Errorf(ttlInSecsTooHighError)},
		{"Zero", 0, nil},
		{"Valid upper limit", 9223372036, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.Equal(t, tc.expectedErr, isTTLInSecsValid(tc.ttlInSecs))
		})
	}
}

func Test_ListCacheTtlSecsToDuration(t *testing.T) {
	var testCases = []struct {
		testName         string
		ttlInSecs        int64
		expectedDuration time.Duration
	}{
		{"-1", -1, maxSupportedTTL},
		{"0", 0, time.Duration(0)},
		{"Max supported positive", 9223372036, maxSupportedTTL},
		{"Positive", 1, time.Second},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.expectedDuration, ListCacheTTLSecsToDuration(tt.ttlInSecs))
		})
	}
}

func Test_ListCacheTtlSecsToDuration_InvalidCall(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	// Calling with invalid argument to trigger panic.
	ListCacheTTLSecsToDuration(-3)
}
