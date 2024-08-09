// Copyright 2023 Google LLC
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

package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type flags struct {
	LogFile          string
	LogFormat        string
	DebugFuse        bool
	DebugGCS         bool
	DebugMutex       bool
	IgnoreInterrupts bool
	AnonymousAccess  bool
	MaxRetryAttempts int64
}
type ConfigTest struct {
	suite.Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTest))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

type TestCliContext struct {
	isSet bool
}

func (s *TestCliContext) IsSet(flag string) bool {
	return s.isSet
}

func Test_IsTtlInSecsValid(t *testing.T) {
	var testCases = []struct {
		testName    string
		ttlInSecs   int64
		expectedErr error
	}{
		{"Negative", -5, fmt.Errorf(TtlInSecsInvalidValueError)},
		{"Valid negative", -1, nil},
		{"Positive", 8, nil},
		{"Unsupported Large positive", 9223372037, fmt.Errorf(TtlInSecsTooHighError)},
		{"Zero", 0, nil},
		{"Valid upper limit", 9223372036, nil},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.expectedErr, IsTtlInSecsValid(tt.ttlInSecs))
		})
	}
}

func Test_ListCacheTtlSecsToDuration(t *testing.T) {
	var testCases = []struct {
		testName         string
		ttlInSecs        int64
		expectedDuration time.Duration
	}{
		{"-1", -1, MaxSupportedTtl},
		{"0", 0, time.Duration(0)},
		{"Max supported positive", 9223372036, MaxSupportedTtl},
		{"Positive", 1, time.Second},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.expectedDuration, ListCacheTtlSecsToDuration(tt.ttlInSecs))
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
	ListCacheTtlSecsToDuration(-3)
}
