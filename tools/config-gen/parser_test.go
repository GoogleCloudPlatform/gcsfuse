/*
 * Copyright 2025 Google LLC
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
	"os"
	"path/filepath"
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

func TestParseParamsYAML_Success(t *testing.T) {
	// ARRANGE
	yamlContent := `
machine-type-groups:
  high-performance:
    - "a2-megagpu-16g"
    - "a3-highgpu-8g"
  low-latency:
    - "c2-standard-4"
params:
  - config-path: "app-name"
    flag-name: "app-name"
    type: "string"
    default: "gcsfuse"
    "usage": "Application name"
  - config-path: "implicit-dirs"
    flag-name: "implicit-dirs"
    type: "bool"
    default: false
    "usage": "Whether or not to enable implicit directories"
    optimizations:
      machine-based-optimization:
        - group: high-performance
          value: true
  - config-path: "metadata-cache.ttl-secs"
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
          environments:
            - name: default
              value: -1
`
	// Create a temporary directory and the params.yaml file.
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "params.yaml")
	err := os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	require.NoError(t, err)
	// Point the global paramsFile variable to our temporary file.
	originalParamsFile := *paramsFile
	*paramsFile = tmpFile
	defer func() { *paramsFile = originalParamsFile }()

	// ACT
	parsedYAML, err := parseParamsYAML()

	// ASSERT
	require.NoError(t, err)

	t.Run("TestMachineTypeGroupsParsing", func(t *testing.T) {
		expectedGroups := map[string][]string{
			"high-performance": {"a2-megagpu-16g", "a3-highgpu-8g"},
			"low-latency":      {"c2-standard-4"},
		}
		assert.Equal(t, expectedGroups, parsedYAML.MachineTypeGroups)
	})

	t.Run("TestParamWithOnlyMachineBasedOptimizations", func(t *testing.T) {
		param := parsedYAML.Params[1]
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
		param := parsedYAML.Params[2]
		require.NotNil(t, param.Optimizations)
		expected := &shared.OptimizationRules{
			MachineBasedOptimization: []shared.MachineBasedOptimization{
				{Group: "high-performance", Value: -1},
			},
			Profiles: []shared.ProfileOptimization{
				{
					Name: "aiml-training",
					Environments: []shared.EnvironmentOptimization{
						{Name: "default", Value: -1},
					},
				},
			},
		}
		assert.Equal(t, "metadata-cache.ttl-secs", param.ConfigPath)
		assert.Equal(t, expected, param.Optimizations)
	})

	t.Run("TestParamWithNoOptimizations", func(t *testing.T) {
		param := parsedYAML.Params[0]
		assert.Equal(t, "app-name", param.ConfigPath)
		assert.Nil(t, param.Optimizations)
	})
}

func TestParseParamsYAML_Negative(t *testing.T) {
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
  - flag-name: "my-flag"
    config-path: "b"
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "params.yaml")
			err := os.WriteFile(tmpFile, []byte(tc.yamlContent), 0644)
			require.NoError(t, err)

			originalParamsFile := *paramsFile
			*paramsFile = tmpFile
			defer func() { *paramsFile = originalParamsFile }()

			// ACT
			_, err = parseParamsYAML()

			// ASSERT
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), tc.expectedErrorSubstring), "Expected error to contain %q, but got: %q", tc.expectedErrorSubstring, err.Error())
		})
	}
}
