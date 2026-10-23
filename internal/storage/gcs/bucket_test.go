// Copyright 2026 Google LLC
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

package gcs

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/assert"
)

func TestBucketType_IsRapid(t *testing.T) {
	testCases := []struct {
		name     string
		zonal    bool
		rcu      bool
		expected bool
	}{
		{
			name:     "Neither Zonal nor Rapid Cache Ultra",
			zonal:    false,
			rcu:      false,
			expected: false,
		},
		{
			name:     "Only Zonal is true",
			zonal:    true,
			rcu:      false,
			expected: true,
		},
		{
			name:     "Only Rapid Cache Ultra is true",
			zonal:    false,
			rcu:      true,
			expected: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bt := BucketType{
				Zonal: tc.zonal,
				RCU:   tc.rcu,
			}

			assert.Equal(t, tc.expected, bt.IsRapid())
		})
	}
}

func TestBucketType_WriteStrategy(t *testing.T) {
	testCases := []struct {
		name        string
		zonal       bool
		rcu         bool
		writeConfig *cfg.WriteConfig
		expected    WriteStrategy
	}{
		{
			name:        "Standard bucket",
			zonal:       false,
			rcu:         false,
			writeConfig: &cfg.WriteConfig{RapidWrite: cfg.RapidWriteStrategyEnabled},
			expected:    WriteStrategyResumable,
		},
		{
			name:        "Zonal bucket",
			zonal:       true,
			rcu:         false,
			writeConfig: nil,
			expected:    WriteStrategyAppendable,
		},
		{
			name:        "Rapid Cache Ultra, Rapid Writes Enabled",
			zonal:       false,
			rcu:         true,
			writeConfig: &cfg.WriteConfig{RapidWrite: cfg.RapidWriteStrategyEnabled},
			expected:    WriteStrategyAppendable,
		},
		{
			name:        "Rapid Cache Ultra, Rapid Writes Disabled",
			zonal:       false,
			rcu:         true,
			writeConfig: &cfg.WriteConfig{RapidWrite: cfg.RapidWriteStrategyDisabled},
			expected:    WriteStrategyResumable,
		},
		{
			name:        "Rapid Cache Ultra, nil WriteConfig",
			zonal:       false,
			rcu:         true,
			writeConfig: nil,
			expected:    WriteStrategyResumable,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bt := BucketType{
				Zonal: tc.zonal,
				RCU:   tc.rcu,
			}

			assert.Equal(t, tc.expected, bt.WriteStrategy(tc.writeConfig))
		})
	}
}
