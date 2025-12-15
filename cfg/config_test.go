// Copyright 2025 Google LLC
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

// GENERATED CODE - DO NOT EDIT MANUALLY.

package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyOptimizations(t *testing.T) {
	// Tests for file-system.async-read
	t.Run("file-system.async-read", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name:   "user_set",
				config: Config{},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"async-read":   true,
						"machine-type": true,
					},
				},
				expectOptimized: false,
				expectedValue:   !(false),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.FileSystem.AsyncRead = tc.expectedValue.(bool)
				} else {
					c.FileSystem.AsyncRead = false
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "file-system.async-read")
				} else {
					assert.NotContains(t, optimizedFlags, "file-system.async-read")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.FileSystem.AsyncRead)
			})
		}
	})
	// Tests for file-system.congestion-threshold
	t.Run("file-system.congestion-threshold", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name:   "user_set",
				config: Config{},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"congestion-threshold": true,
						"machine-type":         true,
					},
				},
				expectOptimized: false,
				expectedValue:   int64(98765),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   0,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.FileSystem.CongestionThreshold = tc.expectedValue.(int64)
				} else {
					c.FileSystem.CongestionThreshold = 0
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "file-system.congestion-threshold")
				} else {
					assert.NotContains(t, optimizedFlags, "file-system.congestion-threshold")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.FileSystem.CongestionThreshold)
			})
		}
	})
	// Tests for file-cache.cache-file-for-range-read
	t.Run("file-cache.cache-file-for-range-read", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name: "user_set",
				config: Config{
					Profile: "aiml-serving",
				},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"file-cache-cache-file-for-range-read": true,
						"machine-type":                         true,
					},
				},
				expectOptimized: false,
				expectedValue:   !(false),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   false,
			},
			{
				name:            "profile_aiml-serving",
				config:          Config{Profile: "aiml-serving"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   true,
			},
			{
				name:            "profile_aiml-checkpointing",
				config:          Config{Profile: "aiml-checkpointing"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.FileCache.CacheFileForRangeRead = tc.expectedValue.(bool)
				} else {
					c.FileCache.CacheFileForRangeRead = false
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "file-cache.cache-file-for-range-read")
				} else {
					assert.NotContains(t, optimizedFlags, "file-cache.cache-file-for-range-read")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.FileCache.CacheFileForRangeRead)
			})
		}
	})
	// Tests for implicit-dirs
	t.Run("implicit-dirs", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name: "user_set",
				config: Config{
					Profile: "aiml-training",
				},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"implicit-dirs": true,
						"machine-type":  true,
					},
					stringFlags: map[string]string{
						"machine-type": "a2-megagpu-16g",
					},
				},
				expectOptimized: false,
				expectedValue:   !(false),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   false,
			},
			{
				name:            "profile_aiml-training",
				config:          Config{Profile: "aiml-training"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   true,
			},
			{
				name:            "profile_aiml-serving",
				config:          Config{Profile: "aiml-serving"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   true,
			},
			{
				name:            "profile_aiml-checkpointing",
				config:          Config{Profile: "aiml-checkpointing"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   true,
			},
			{
				name:   "machine_group_high-performance",
				config: Config{Profile: ""},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   true,
			},
			{
				name:   "profile_overrides_machine_type",
				config: Config{Profile: "aiml-training"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   true,
			}, {
				name:   "fallback_to_machine_type_with_non_existent_profile",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.ImplicitDirs = tc.expectedValue.(bool)
				} else {
					c.ImplicitDirs = false
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "implicit-dirs")
				} else {
					assert.NotContains(t, optimizedFlags, "implicit-dirs")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.ImplicitDirs)
			})
		}
	})
	// Tests for file-system.kernel-list-cache-ttl-secs
	t.Run("file-system.kernel-list-cache-ttl-secs", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name: "user_set",
				config: Config{
					Profile: "aiml-serving",
				},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"kernel-list-cache-ttl-secs": true,
						"machine-type":               true,
					},
				},
				expectOptimized: false,
				expectedValue:   int64(98765),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   0,
			},
			{
				name:            "profile_aiml-serving",
				config:          Config{Profile: "aiml-serving"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   -1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.FileSystem.KernelListCacheTtlSecs = tc.expectedValue.(int64)
				} else {
					c.FileSystem.KernelListCacheTtlSecs = 0
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "file-system.kernel-list-cache-ttl-secs")
				} else {
					assert.NotContains(t, optimizedFlags, "file-system.kernel-list-cache-ttl-secs")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.FileSystem.KernelListCacheTtlSecs)
			})
		}
	})
	// Tests for file-system.max-background
	t.Run("file-system.max-background", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name:   "user_set",
				config: Config{},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"max-background": true,
						"machine-type":   true,
					},
				},
				expectOptimized: false,
				expectedValue:   int64(98765),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   0,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.FileSystem.MaxBackground = tc.expectedValue.(int64)
				} else {
					c.FileSystem.MaxBackground = 0
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "file-system.max-background")
				} else {
					assert.NotContains(t, optimizedFlags, "file-system.max-background")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.FileSystem.MaxBackground)
			})
		}
	})
	// Tests for metadata-cache.negative-ttl-secs
	t.Run("metadata-cache.negative-ttl-secs", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name: "user_set",
				config: Config{
					Profile: "aiml-training",
				},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"metadata-cache-negative-ttl-secs": true,
						"machine-type":                     true,
					},
					stringFlags: map[string]string{
						"machine-type": "a2-megagpu-16g",
					},
				},
				expectOptimized: false,
				expectedValue:   int64(98765),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   5,
			},
			{
				name:            "profile_aiml-training",
				config:          Config{Profile: "aiml-training"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   0,
			},
			{
				name:            "profile_aiml-serving",
				config:          Config{Profile: "aiml-serving"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   0,
			},
			{
				name:            "profile_aiml-checkpointing",
				config:          Config{Profile: "aiml-checkpointing"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   0,
			},
			{
				name:   "machine_group_high-performance",
				config: Config{Profile: ""},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   0,
			},
			{
				name:   "profile_overrides_machine_type",
				config: Config{Profile: "aiml-training"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   0,
			}, {
				name:   "fallback_to_machine_type_with_non_existent_profile",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   0,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.MetadataCache.NegativeTtlSecs = tc.expectedValue.(int64)
				} else {
					c.MetadataCache.NegativeTtlSecs = 5
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "metadata-cache.negative-ttl-secs")
				} else {
					assert.NotContains(t, optimizedFlags, "metadata-cache.negative-ttl-secs")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.MetadataCache.NegativeTtlSecs)
			})
		}
	})
	// Tests for metadata-cache.ttl-secs
	t.Run("metadata-cache.ttl-secs", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name: "user_set",
				config: Config{
					Profile: "aiml-training",
				},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"metadata-cache-ttl-secs": true,
						"machine-type":            true,
					},
					stringFlags: map[string]string{
						"machine-type": "a2-megagpu-16g",
					},
				},
				expectOptimized: false,
				expectedValue:   int64(98765),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   60,
			},
			{
				name:            "profile_aiml-training",
				config:          Config{Profile: "aiml-training"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   -1,
			},
			{
				name:            "profile_aiml-serving",
				config:          Config{Profile: "aiml-serving"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   -1,
			},
			{
				name:            "profile_aiml-checkpointing",
				config:          Config{Profile: "aiml-checkpointing"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   -1,
			},
			{
				name:   "machine_group_high-performance",
				config: Config{Profile: ""},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   -1,
			},
			{
				name:   "profile_overrides_machine_type",
				config: Config{Profile: "aiml-training"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   -1,
			}, {
				name:   "fallback_to_machine_type_with_non_existent_profile",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   -1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.MetadataCache.TtlSecs = tc.expectedValue.(int64)
				} else {
					c.MetadataCache.TtlSecs = 60
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "metadata-cache.ttl-secs")
				} else {
					assert.NotContains(t, optimizedFlags, "metadata-cache.ttl-secs")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.MetadataCache.TtlSecs)
			})
		}
	})
	// Tests for file-system.rename-dir-limit
	t.Run("file-system.rename-dir-limit", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name: "user_set",
				config: Config{
					Profile: "aiml-checkpointing",
				},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"rename-dir-limit": true,
						"machine-type":     true,
					},
					stringFlags: map[string]string{
						"machine-type": "a2-megagpu-16g",
					},
				},
				expectOptimized: false,
				expectedValue:   int64(98765),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   0,
			},
			{
				name:            "profile_aiml-checkpointing",
				config:          Config{Profile: "aiml-checkpointing"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   200000,
			},
			{
				name:   "machine_group_high-performance",
				config: Config{Profile: ""},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   200000,
			},
			{
				name:   "profile_overrides_machine_type",
				config: Config{Profile: "aiml-checkpointing"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   200000,
			}, {
				name:   "fallback_to_machine_type_with_non_existent_profile",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   200000,
			}, {
				name:   "fallback_to_machine_type_when_aiml-training_is_unrelated",
				config: Config{Profile: "aiml-training"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   200000,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.FileSystem.RenameDirLimit = tc.expectedValue.(int64)
				} else {
					c.FileSystem.RenameDirLimit = 0
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "file-system.rename-dir-limit")
				} else {
					assert.NotContains(t, optimizedFlags, "file-system.rename-dir-limit")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.FileSystem.RenameDirLimit)
			})
		}
	})
	// Tests for metadata-cache.stat-cache-max-size-mb
	t.Run("metadata-cache.stat-cache-max-size-mb", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name: "user_set",
				config: Config{
					Profile: "aiml-training",
				},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"stat-cache-max-size-mb": true,
						"machine-type":           true,
					},
					stringFlags: map[string]string{
						"machine-type": "a2-megagpu-16g",
					},
				},
				expectOptimized: false,
				expectedValue:   int64(98765),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   33,
			},
			{
				name:            "profile_aiml-training",
				config:          Config{Profile: "aiml-training"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   -1,
			},
			{
				name:            "profile_aiml-serving",
				config:          Config{Profile: "aiml-serving"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   -1,
			},
			{
				name:            "profile_aiml-checkpointing",
				config:          Config{Profile: "aiml-checkpointing"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   -1,
			},
			{
				name:   "machine_group_high-performance",
				config: Config{Profile: ""},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   1024,
			},
			{
				name:   "profile_overrides_machine_type",
				config: Config{Profile: "aiml-training"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   -1,
			}, {
				name:   "fallback_to_machine_type_with_non_existent_profile",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   1024,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.MetadataCache.StatCacheMaxSizeMb = tc.expectedValue.(int64)
				} else {
					c.MetadataCache.StatCacheMaxSizeMb = 33
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "metadata-cache.stat-cache-max-size-mb")
				} else {
					assert.NotContains(t, optimizedFlags, "metadata-cache.stat-cache-max-size-mb")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.MetadataCache.StatCacheMaxSizeMb)
			})
		}
	})
	// Tests for metadata-cache.type-cache-max-size-mb
	t.Run("metadata-cache.type-cache-max-size-mb", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name: "user_set",
				config: Config{
					Profile: "aiml-training",
				},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"type-cache-max-size-mb": true,
						"machine-type":           true,
					},
					stringFlags: map[string]string{
						"machine-type": "a2-megagpu-16g",
					},
				},
				expectOptimized: false,
				expectedValue:   int64(98765),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   4,
			},
			{
				name:            "profile_aiml-training",
				config:          Config{Profile: "aiml-training"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   -1,
			},
			{
				name:            "profile_aiml-serving",
				config:          Config{Profile: "aiml-serving"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   -1,
			},
			{
				name:            "profile_aiml-checkpointing",
				config:          Config{Profile: "aiml-checkpointing"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				expectOptimized: true,
				expectedValue:   -1,
			},
			{
				name:   "machine_group_high-performance",
				config: Config{Profile: ""},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   128,
			},
			{
				name:   "profile_overrides_machine_type",
				config: Config{Profile: "aiml-training"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   -1,
			}, {
				name:   "fallback_to_machine_type_with_non_existent_profile",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   128,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.MetadataCache.TypeCacheMaxSizeMb = tc.expectedValue.(int64)
				} else {
					c.MetadataCache.TypeCacheMaxSizeMb = 4
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "metadata-cache.type-cache-max-size-mb")
				} else {
					assert.NotContains(t, optimizedFlags, "metadata-cache.type-cache-max-size-mb")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.MetadataCache.TypeCacheMaxSizeMb)
			})
		}
	})
	// Tests for write.global-max-blocks
	t.Run("write.global-max-blocks", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			expectOptimized bool
			expectedValue   any
		}{
			{
				name:   "user_set",
				config: Config{},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"write-global-max-blocks": true,
						"machine-type":            true,
					},
					stringFlags: map[string]string{
						"machine-type": "a2-megagpu-16g",
					},
				},
				expectOptimized: false,
				expectedValue:   int64(98765),
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
				expectOptimized: false,
				expectedValue:   4,
			},
			{
				name:   "machine_group_high-performance",
				config: Config{Profile: ""},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   1600,
			}, {
				name:   "fallback_to_machine_type_with_non_existent_profile",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   1600,
			}, {
				name:   "fallback_to_machine_type_when_aiml-training_is_unrelated",
				config: Config{Profile: "aiml-training"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "a2-megagpu-16g"},
				},
				expectOptimized: true,
				expectedValue:   1600,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.Write.GlobalMaxBlocks = tc.expectedValue.(int64)
				} else {
					c.Write.GlobalMaxBlocks = 4
				}

				optimizedFlags := c.ApplyOptimizations(tc.isSet)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "write.global-max-blocks")
				} else {
					assert.NotContains(t, optimizedFlags, "write.global-max-blocks")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.Write.GlobalMaxBlocks)
			})
		}
	})
}
