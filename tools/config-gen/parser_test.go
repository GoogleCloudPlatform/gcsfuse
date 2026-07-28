/*
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckFlagName_Valid(t *testing.T) {
	validNames := []string{"a", "abc", "ab-c", "ab-c-d", "a_b"}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			assert.NoError(t, checkFlagName(name))
		})
	}
}

func TestCheckFlagName_Invalid(t *testing.T) {
	invalidNames := []string{"", "a-", "-a", "a--b", "a-b-", "A-b", "a.b", "1-a"}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, checkFlagName(name))
		})
	}
}

func TestValidateMachineTypeGroups(t *testing.T) {
	testCases := []struct {
		name        string
		input       map[string][]string
		expectErr   bool
		errContains string
	}{
		{
			name: "Valid_groups",
			input: map[string][]string{
				"another-group":    {"gce-vm"},
				"high-performance": {"a2-megagpu-16g", "a3-highgpu-8g"},
			},
			expectErr: false,
		},
		{
			name:      "Empty_groups_map",
			input:     map[string][]string{},
			expectErr: false,
		},
		{
			name: "Invalid_group_name_format_(snake_case)",
			input: map[string][]string{
				"invalid_group": {"vm"},
			},
			expectErr:   true,
			errContains: "does not conform",
		},
		{
			name: "Invalid_group_name_format_(PascalCase)",
			input: map[string][]string{
				"InvalidGroup": {"vm"},
			},
			expectErr:   true,
			errContains: "does not conform",
		},
		{
			name: "Empty_machine_type_list",
			input: map[string][]string{
				"a-valid-group": {},
			},
			expectErr:   true,
			errContains: "must contain at least one machine type",
		},
		{
			name: "Unsorted_machine_types_in_a_group",
			input: map[string][]string{
				"a-valid-group": {"z-vm", "a-vm"},
			},
			expectErr:   true,
			errContains: "machine types in group \"a-valid-group\" are not sorted",
		},
		{
			name: "Duplicate_machine_types_in_a_group",
			input: map[string][]string{
				"a-valid-group": {"a-vm", "a-vm", "z-vm"},
			},
			expectErr:   true,
			errContains: "duplicate machine type found in group \"a-valid-group\"",
		},
		{
			name: "a_machine_type_in_multiple_groups",
			input: map[string][]string{
				"a-valid-group":       {"a-vm", "b-vm"},
				"another-valid-group": {"a-vm", "c-vm"},
			},
			expectErr:   true,
			errContains: "cannot be in multiple groups",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMachineTypeGroups(tc.input)

			if tc.expectErr {
				assert.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateForDuplicatesInSortedSlice(t *testing.T) {
	testCases := []struct {
		name      string
		input     []string
		expectErr bool
	}{
		{
			name:      "Slice_with_unique_strings",
			input:     []string{"a", "b", "c"},
			expectErr: false,
		},
		{
			name:      "Empty_slice",
			input:     []string{},
			expectErr: false,
		},
		{
			name:      "Slice_with_duplicate_strings",
			input:     []string{"a", "b", "b"},
			expectErr: true,
		},
		{
			name:      "Slice_with_an_empty_string",
			input:     []string{"", "c", "c"},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateForDuplicatesInSortedSlice(tc.input)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseParamsYAMLStr_Success(t *testing.T) {
	// ARRANGE
	yamlContent := `
retired-params: [6, 7]
machine-type-groups:
  high-performance:
    - "a2-megagpu-16g"
    - "a3-highgpu-8g"
  low-latency:
    - "c2-standard-4"
params:
  - config-path: "app-name"
    proto-tag: 1
    flag-name: "app-name"
    type: "string"
    default: "gcsfuse"
    "usage": "Application name"
  - config-path: "file-system.enable-kernel-reader"
    proto-tag: 2
    flag-name: "enable-kernel-reader"
    type: "bool"
    default: false
    "usage": "Whether to enable kernel-based reader"
    optimizations:
      bucket-type-optimization:
        - bucket-type: "zonal, flat, pirlo"
          value: true
        - bucket-type: "hierarchical"
          value: false
  - config-path: "file-system.max-read-ahead-kb"
    proto-tag: 3
    flag-name: "max-read-ahead-kb"
    type: "int"
    default: "128"
    "usage": "Maximum read ahead in KB"
    optimizations:
      bucket-type-optimization:
        - bucket-type: "zonal"
          value: 1024
        - bucket-type: "hierarchical, flat, pirlo"
          value: 2048
      machine-based-optimization:
        - group: high-performance
          value: 2048
      profiles:
        - name: aiml-training
          value: 4096
  - config-path: "implicit-dirs"
    proto-tag: 4
    flag-name: "implicit-dirs"
    type: "bool"
    default: false
    "usage": "Whether or not to enable implicit directories"
    optimizations:
      machine-based-optimization:
        - group: high-performance
          value: true
  - config-path: "metadata-cache.ttl-secs"
    proto-tag: 5
    flag-name: "metadata-cache-ttl-secs"
    type: "int"
    default: "60"
    "usage": "Metadata cache TTL in seconds"
    optimizations:
      machine-based-optimization:
        - group: high-performance
          value: -1
      profiles:
        - name: aiml-training
          value: -1
`

	// ACT
	parsedYAML, err := parseParamsYAMLStr(yamlContent)

	// ASSERT
	require.NoError(t, err)

	t.Run("TestMachineTypeGroupsParsing", func(t *testing.T) {
		expectedGroups := map[string][]string{
			"high-performance": {"a2-megagpu-16g", "a3-highgpu-8g"},
			"low-latency":      {"c2-standard-4"},
		}
		assert.Equal(t, expectedGroups, parsedYAML.MachineTypeGroups)
	})

	t.Run("TestParamWithOnlyBucketBasedOptimizations", func(t *testing.T) {
		param := parsedYAML.Params[1]
		require.NotNil(t, param.Optimizations)
		expected := &shared.OptimizationRules{
			BucketTypeOptimization: []shared.BucketTypeOptimization{
				{BucketTypes: shared.BucketTypeList{"zonal", "flat", "pirlo"}, Value: true},
				{BucketTypes: shared.BucketTypeList{"hierarchical"}, Value: false},
			},
		}
		assert.Equal(t, "file-system.enable-kernel-reader", param.ConfigPath)
		assert.Equal(t, expected.BucketTypeOptimization, param.Optimizations.BucketTypeOptimization)
		assert.Nil(t, param.Optimizations.Profiles)
		assert.Nil(t, param.Optimizations.MachineBasedOptimization)
	})

	t.Run("TestParamWithAllOptimizationTypes", func(t *testing.T) {
		param := parsedYAML.Params[2]
		require.NotNil(t, param.Optimizations)
		expected := &shared.OptimizationRules{
			BucketTypeOptimization: []shared.BucketTypeOptimization{
				{BucketTypes: shared.BucketTypeList{"zonal"}, Value: 1024},
				{BucketTypes: shared.BucketTypeList{"hierarchical", "flat", "pirlo"}, Value: 2048},
			},
			MachineBasedOptimization: []shared.MachineBasedOptimization{
				{Group: "high-performance", Value: 2048},
			},
			Profiles: []shared.ProfileOptimization{
				{Name: "aiml-training", Value: 4096},
			},
		}
		assert.Equal(t, "file-system.max-read-ahead-kb", param.ConfigPath)
		assert.Equal(t, expected.BucketTypeOptimization, param.Optimizations.BucketTypeOptimization)
		assert.Equal(t, expected.MachineBasedOptimization, param.Optimizations.MachineBasedOptimization)
		assert.Equal(t, expected.Profiles, param.Optimizations.Profiles)
	})

	t.Run("TestParamWithOnlyMachineBasedOptimizations", func(t *testing.T) {
		param := parsedYAML.Params[3]
		require.NotNil(t, param.Optimizations)
		expected := &shared.OptimizationRules{
			MachineBasedOptimization: []shared.MachineBasedOptimization{
				{Group: "high-performance", Value: true},
			},
		}
		assert.Equal(t, "implicit-dirs", param.ConfigPath)
		assert.Equal(t, expected.MachineBasedOptimization, param.Optimizations.MachineBasedOptimization)
		assert.Nil(t, param.Optimizations.Profiles)
	})

	t.Run("TestParamWithMixedOptimizations", func(t *testing.T) {
		param := parsedYAML.Params[4]
		require.NotNil(t, param.Optimizations)
		expected := &shared.OptimizationRules{
			MachineBasedOptimization: []shared.MachineBasedOptimization{
				{Group: "high-performance", Value: -1},
			},
			Profiles: []shared.ProfileOptimization{
				{
					Name:  "aiml-training",
					Value: -1,
				},
			},
		}
		assert.Equal(t, "metadata-cache.ttl-secs", param.ConfigPath)
		assert.Equal(t, expected, param.Optimizations)
	})

	t.Run("TestParamWithNoOptimizationsAndProtoMetadata", func(t *testing.T) {
		param := parsedYAML.Params[0]
		assert.Equal(t, "app-name", param.ConfigPath)
		assert.Nil(t, param.Optimizations)
		// Check proto metadata for a string field (privacy booleanization)
		assert.Equal(t, "bool", param.ProtoType)
		assert.Equal(t, "is_app_name_set", param.ProtoFieldName)
		assert.Equal(t, 1, param.ProtoTag)
	})

	t.Run("testParamProtoTag", func(t *testing.T) {
		assert.Equal(t, 1, parsedYAML.Params[0].ProtoTag)
		assert.Equal(t, 2, parsedYAML.Params[1].ProtoTag)
		assert.Equal(t, 3, parsedYAML.Params[2].ProtoTag)
		assert.Equal(t, 4, parsedYAML.Params[3].ProtoTag)
		assert.Equal(t, 5, parsedYAML.Params[4].ProtoTag)
		assert.Equal(t, []int{6, 7}, parsedYAML.RetiredParams)
	})
}

func TestValidateProtoTags(t *testing.T) {
	testCases := []struct {
		name                   string
		params                 []Param
		retiredTags            []int
		expectErr              bool
		expectedErrorSubstring string
	}{
		{
			name: "valid_sequential_tags_without_retired",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 1},
				{FlagName: "p2", ConfigPath: "p2", ProtoTag: 2},
				{FlagName: "p3", ConfigPath: "p3", ProtoTag: 3},
			},
			retiredTags: nil,
			expectErr:   false,
		},
		{
			name: "valid_with_retired_params_filling_gaps",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 1},
				{FlagName: "p3", ConfigPath: "p3", ProtoTag: 3},
			},
			retiredTags: []int{2},
			expectErr:   false,
		},
		{
			name: "valid_with_last_tag_retired",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 1},
				{FlagName: "p2", ConfigPath: "p2", ProtoTag: 2},
			},
			retiredTags: []int{3},
			expectErr:   false,
		},
		{
			name: "invalid_gap_in_retired_params",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 1},
				{FlagName: "p2", ConfigPath: "p2", ProtoTag: 2},
			},
			retiredTags:            []int{4},
			expectErr:              true,
			expectedErrorSubstring: "missing proto-tag 3 in sequence 1..4",
		},
		{
			name: "deprecated_without_config_path_skips_tag_validation",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 1},
				{FlagName: "deprecated-cli", ConfigPath: "", ProtoTag: 0},
			},
			retiredTags: nil,
			expectErr:   false,
		},
		{
			name: "invalid_zero_tag",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 0},
			},
			retiredTags:            nil,
			expectErr:              true,
			expectedErrorSubstring: "has invalid proto-tag 0 (must be > 0)",
		},
		{
			name: "invalid_negative_tag",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: -1},
			},
			retiredTags:            nil,
			expectErr:              true,
			expectedErrorSubstring: "has invalid proto-tag -1 (must be > 0)",
		},
		{
			name: "duplicate_active_tag",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 1},
				{FlagName: "p2", ConfigPath: "p2", ProtoTag: 1},
			},
			retiredTags:            nil,
			expectErr:              true,
			expectedErrorSubstring: "duplicate proto-tag 1 found for parameter",
		},
		{
			name: "active_tag_collides_with_retired_tag",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 2},
			},
			retiredTags:            []int{2},
			expectErr:              true,
			expectedErrorSubstring: "duplicate proto-tag 2 found for parameter \"p1\" and \"retired-tag 2\"",
		},
		{
			name: "duplicate_retired_tag",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 1},
			},
			retiredTags:            []int{2, 2},
			expectErr:              true,
			expectedErrorSubstring: "duplicate proto-tag 2 found for retired-tag",
		},
		{
			name: "retired_tag_non_positive",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 1},
			},
			retiredTags:            []int{0},
			expectErr:              true,
			expectedErrorSubstring: "retired proto-tag 0 is invalid (must be > 0)",
		},
		{
			name: "missing_tag_unrecorded_gap",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 1},
				{FlagName: "p3", ConfigPath: "p3", ProtoTag: 3},
			},
			retiredTags:            nil,
			expectErr:              true,
			expectedErrorSubstring: "missing proto-tag 2 in sequence 1..3",
		},
		{
			name: "deprecated_without_config_path_has_non_zero_tag",
			params: []Param{
				{FlagName: "p1", ConfigPath: "p1", ProtoTag: 1},
				{FlagName: "deprecated-cli", ConfigPath: "", ProtoTag: 2},
			},
			retiredTags:            nil,
			expectErr:              true,
			expectedErrorSubstring: "parameter \"deprecated-cli\" without config-path must not have proto-tag set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProtoTags(tc.params, tc.retiredTags)
			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorSubstring)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseParamsYAMLStr_Negative(t *testing.T) {
	testCases := []struct {
		name                   string
		yamlContent            string
		expectedErrorSubstring string
	}{
		{
			name: "MalformedYAML",
			yamlContent: `
params:
  - config-path: "a"
   - config-path: "b" # Bad indentation
`,
			expectedErrorSubstring: "did not find expected '-' indicator",
		},
		{
			name: "DuplicateFlagName",
			yamlContent: `
params:
  - flag-name: "my-flag"
    config-path: "a"
    proto-tag: 1
  - flag-name: "my-flag"
    config-path: "b"
    proto-tag: 2
`,
			expectedErrorSubstring: "duplicate",
		},
		{
			name: "InvalidGroupName",
			yamlContent: `
machine-type-groups:
  Invalid_Group_Name:
    - "a-machine"
`,
			expectedErrorSubstring: "group name \"Invalid_Group_Name\" does not conform",
		},
		{
			name: "UnsortedMachineTypesInGroup",
			yamlContent: `
machine-type-groups:
  my-group:
    - "z-machine"
    - "a-machine"
`,
			expectedErrorSubstring: "machine types in group \"my-group\" are not sorted alphabetically",
		},
		{
			name: "DuplicateMachineTypeInGroup",
			yamlContent: `
machine-type-groups:
  my-group:
    - "a-machine"
    - "a-machine"
`,
			expectedErrorSubstring: "duplicate machine type found in group \"my-group\"",
		},
		{
			name: "EmptyMachineTypeList",
			yamlContent: `
machine-type-groups:
  my-group: []
`,
			expectedErrorSubstring: "group \"my-group\" must contain at least one machine type",
		},
		{
			name: "UnsupportedBucketType",
			yamlContent: `
params:
  - config-path: "test-param"
    proto-tag: 1
    flag-name: "test-flag"
    type: "bool"
    default: false
    usage: "Test flag for bucket type validation"
    optimizations:
      bucket-type-optimization:
        - bucket-type: "invalid-bucket-type"
          value: true
`,
			expectedErrorSubstring: "invalid bucket-type",
		},
		{
			name: "DuplicateBucketType",
			yamlContent: `
params:
  - config-path: "test-param"
    proto-tag: 1
    flag-name: "test-flag"
    type: "bool"
    default: false
    usage: "Test flag for duplicate bucket type validation"
    optimizations:
      bucket-type-optimization:
        - bucket-type: "zonal, zonal"
          value: true
`,
			expectedErrorSubstring: "duplicate bucket-type \"zonal\"",
		},
		{
			name: "EmptyBucketTypeList",
			yamlContent: `
params:
  - config-path: "test-param"
    proto-tag: 1
    flag-name: "test-flag"
    type: "bool"
    default: false
    usage: "Test flag for empty bucket type validation"
    optimizations:
      bucket-type-optimization:
        - bucket-type: ""
          value: true
`,
			expectedErrorSubstring: "bucket-type list is empty",
		},
		{
			name: "UnknownConfigurationType",
			yamlContent: `
params:
  - config-path: "test-param"
    proto-tag: 1
    flag-name: "test-flag"
    type: "unknownTypeForGoAndProto"
    default: 500
    usage: "Test flag for unknown type validation"
`,
			expectedErrorSubstring: "unsupported datatype: unknownTypeForGoAndProto",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseParamsYAMLStr(tc.yamlContent)
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), tc.expectedErrorSubstring), "Expected error to contain %q, but got: %q", tc.expectedErrorSubstring, err.Error())
		})
	}
}

func TestPopulateProtoMetadataExhaustive(t *testing.T) {
	yamlContent := `
retired-params: [13, 14]
params:
  - config-path: "type-bool"
    proto-tag: 1
    flag-name: "type-bool"
    type: "bool"
    usage: "test"
  - config-path: "type-direct-path-strategy"
    proto-tag: 2
    flag-name: "type-direct-path-strategy"
    type: "directPathStrategy"
    usage: "test"
  - config-path: "type-duration"
    proto-tag: 3
    flag-name: "type-duration"
    type: "duration"
    usage: "test"
  - config-path: "type-float"
    proto-tag: 4
    flag-name: "type-float"
    type: "float64"
    usage: "test"
  - config-path: "type-int"
    proto-tag: 5
    flag-name: "type-int"
    type: "int"
    usage: "test"
  - config-path: "type-log-severity"
    proto-tag: 6
    flag-name: "type-log-severity"
    type: "logSeverity"
    usage: "test"
  - config-path: "type-octal"
    proto-tag: 7
    flag-name: "type-octal"
    type: "octal"
    usage: "test"
  - config-path: "type-protocol"
    proto-tag: 8
    flag-name: "type-protocol"
    type: "protocol"
    usage: "test"
  - config-path: "type-resolved-path"
    proto-tag: 9
    flag-name: "type-resolved-path"
    type: "resolvedPath"
    usage: "test"
  - config-path: "type-slice-int"
    proto-tag: 10
    flag-name: "type-slice-int"
    type: "[]int"
    usage: "test"
  - config-path: "type-slice-string"
    proto-tag: 11
    flag-name: "type-slice-string"
    type: "[]string"
    usage: "test"
  - config-path: "type-string"
    proto-tag: 12
    flag-name: "type-string"
    type: "string"
    usage: "test"
`
	parsedYAML, err := parseParamsYAMLStr(yamlContent)
	require.NoError(t, err)

	expected := []struct {
		protoType      string
		protoFieldName string
	}{
		{"bool", "type_bool"},
		{"string", "type_direct_path_strategy"},
		{"int64", "type_duration"},
		{"double", "type_float"},
		{"int32", "type_int"},
		{"string", "type_log_severity"},
		{"int32", "type_octal"},
		{"string", "type_protocol"},
		{"bool", "is_type_resolved_path_set"},
		{"repeated int32", "type_slice_int"},
		{"bool", "is_type_slice_string_set"},
		{"bool", "is_type_string_set"},
	}

	require.Len(t, parsedYAML.Params, len(expected))
	for i, param := range parsedYAML.Params {
		assert.Equal(t, expected[i].protoType, param.ProtoType, "Failed protoType for %s", param.FlagName)
		assert.Equal(t, expected[i].protoFieldName, param.ProtoFieldName, "Failed protoFieldName for %s", param.FlagName)
	}

	assert.Equal(t, []int{13, 14}, parsedYAML.RetiredParams)
}
