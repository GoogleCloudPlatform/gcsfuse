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

package cmd

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRationalizeMetadataCache(t *testing.T) {
	// t.Parallel()
	testCases := []struct {
		name                  string
		args                  []string
		expectedTTLSecs       int64
		expectedStatCacheSize int64
	}{
		{
			name:                  "new_ttl_flag_set",
			args:                  []string{"--metadata-cache-ttl-secs=30"},
			expectedTTLSecs:       30,
			expectedStatCacheSize: 33, // default.
		},
		{
			name:                  "old_ttl_flags_set",
			args:                  []string{"--stat-cache-ttl=10s", "--type-cache-ttl=5s"},
			expectedTTLSecs:       5,
			expectedStatCacheSize: 33, // default.
		},
		{
			name:                  "new_stat-cache-size-mb_flag_set",
			args:                  []string{"--stat-cache-max-size-mb=20"},
			expectedTTLSecs:       60, // default.
			expectedStatCacheSize: 20,
		},
		{
			name:                  "old_stat-cache-capacity_flag_set",
			args:                  []string{"--stat-cache-capacity=1000"},
			expectedTTLSecs:       60, // default.
			expectedStatCacheSize: 2,
		},
		{
			name:                  "no_relevant_flags_set",
			args:                  []string{""},
			expectedTTLSecs:       60, // default.
			expectedStatCacheSize: 33, //default.
		},
		{
			name:                  "both_new_and_old_flags_set",
			args:                  []string{"--metadata-cache-ttl-secs=30", "--stat-cache-ttl=10s", "--type-cache-ttl=5s", "--stat-cache-capacity=1000", "--stat-cache-max-size-mb=20"},
			expectedTTLSecs:       30,
			expectedStatCacheSize: 20,
		},
		{
			name:                  "ttl_and_stat_cache_size_set_to_-1",
			args:                  []string{"--metadata-cache-ttl-secs=-1", "--stat-cache-max-size-mb=-1"},
			expectedTTLSecs:       math.MaxInt64 / int64(time.Second), // Max supported ttl in seconds.
			expectedStatCacheSize: math.MaxUint64 >> 20,               // Max supported cache size in MiB.
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := getConfigObject(t, tc.args)

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expectedTTLSecs, c.MetadataCache.TtlSecs)
				assert.Equal(t, tc.expectedStatCacheSize, c.MetadataCache.StatCacheMaxSizeMb)
			}
		})
	}
}

func TestRationalizeCloudMetricsExportIntervalSecs(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected int64
	}{
		{
			name:     "stackdriver-export-interval-set",
			args:     []string{"--stackdriver-export-interval=30h"},
			expected: 30 * 3600,
		},
		{
			name:     "cloud-metrics-export-interval-set",
			args:     []string{"--cloud-metrics-export-interval-secs=3200"},
			expected: 3200,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := getConfigObject(t, tc.args)

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, c.Metrics.CloudMetricsExportIntervalSecs)
			}
		})
	}
}
