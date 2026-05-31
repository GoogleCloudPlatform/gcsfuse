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
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvertMachineTypeGroups(t *testing.T) {
	testCases := []struct {
		name          string
		input         map[string][]string
		expected      map[string]string
		expectedError bool
	}{
		{
			name:     "EmptyMap",
			input:    map[string][]string{},
			expected: map[string]string{},
		},
		{
			name: "OneGroupOneMachine",
			input: map[string][]string{
				"group1": {"machine1"},
			},
			expected: map[string]string{
				"machine1": "group1",
			},
		},
		{
			name: "OneGroupMultipleMachines",
			input: map[string][]string{
				"group1": {"machine1", "machine2"},
			},
			expected: map[string]string{
				"machine1": "group1",
				"machine2": "group1",
			},
		},
		{
			name: "MultipleGroupsOneMachine",
			input: map[string][]string{
				"group1": {"machine1"},
				"group2": {"machine1"},
			},
			expectedError: true,
		},
		{
			name: "MultipleGroupsMultipleMachines",
			input: map[string][]string{
				"group1": {"machine1", "machine2"},
				"group2": {"machine2", "machine3"},
			},
			expectedError: true,
		},
		{
			name: "ComplexCase",
			input: map[string][]string{
				"high_cpu":    {"c2-standard-8", "c2-standard-16"},
				"high_memory": {"m1-megamem-96", "m2-megamem-416"},
				"general":     {"c2-standard-8", "m1-megamem-96"},
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tc.expectedError && r == nil {
					t.Errorf("Expected error, but got none.")
				} else if !tc.expectedError && r != nil {
					t.Errorf("Unexpectd error: %v", r)
				}
			}()
			machineTypeToGroupMap := invertMachineTypeGroups(tc.input)
			if !tc.expectedError {
				assert.Equalf(t, machineTypeToGroupMap, tc.expected, "invertMachineTypeGroups() = %v, want %v", machineTypeToGroupMap, tc.expected)
			}
		})
	}
}

func TestLoadProtoRegistry(t *testing.T) {
	tempDir := t.TempDir()
	protoPath := path.Join(tempDir, "config.proto")

	mockProto := `// Copyright 2026 Google LLC
syntax = "proto3";
package cfg;

message CloudProfilerConfig {
  bool allocated_heap = 343497;
  bool cpu = 2;
  repeated string label = 3;
}

message Config {
  string app_name = 1;
  CloudProfilerConfig cloud_profiler = 2;
}
`
	err := os.WriteFile(protoPath, []byte(mockProto), 0644)
	require.NoError(t, err)

	registry, err := loadProtoRegistry(tempDir)
	require.NoError(t, err)

	expected := RegistryMap{
		"CloudProfilerConfig": {
			"AllocatedHeap": FieldTagInfo{Tag: 343497, Type: "bool"},
			"Cpu":           FieldTagInfo{Tag: 2, Type: "bool"},
			"Label":         FieldTagInfo{Tag: 3, Type: "repeated string"},
		},
		"Config": {
			"AppName":       FieldTagInfo{Tag: 1, Type: "string"},
			"CloudProfiler": FieldTagInfo{Tag: 2, Type: "CloudProfilerConfig"},
		},
	}

	assert.Equal(t, expected, registry)
}

func TestConstructProtoTypeTemplateData(t *testing.T) {
	// Mock registry loaded from old config.proto containing big tags (which triggers migration).
	registry := RegistryMap{
		"CloudProfilerConfig": {
			"AllocatedHeap": FieldTagInfo{Tag: 343497, Type: "bool"},
			"Cpu":           FieldTagInfo{Tag: 382386, Type: "bool"},
			"Enabled":       FieldTagInfo{Tag: 393223, Type: "bool"},
			"DeletedField":  FieldTagInfo{Tag: 410647, Type: "string"}, // Deleted field from previous config
		},
	}

	// Active params configuration containing Enabled, Cpu, and a new field NewActiveField.
	params := []Param{
		{
			ConfigPath: "cloud-profiler.enabled",
			Type:       "bool",
			FlagName:   "enabled",
			Sensitive:  new(bool),
		},
		{
			ConfigPath: "cloud-profiler.cpu",
			Type:       "bool",
			FlagName:   "cpu",
			Sensitive:  new(bool),
		},
		{
			ConfigPath: "cloud-profiler.new-active-field",
			Type:       "string",
			FlagName:   "new-active-field",
			Sensitive:  new(bool),
		},
	}

	protoTd, err := constructProtoTypeTemplateData(params, registry)
	require.NoError(t, err)

	// Assert that for "CloudProfilerConfig":
	// 1. Old tags > 1000 got successfully migrated/sequentialized starting from 1 alphabetically!
	// The alphabetical list of all fields (active + deleted):
	//   - AllocatedHeap (active: no, but parsed from old registry) -> gets 1
	//   - Cpu (active: yes) -> gets 2
	//   - DeletedField (active: no, parsed from old registry) -> gets 3
	//   - Enabled (active: yes) -> gets 4
	//   - NewActiveField (active: yes) -> gets 5

	require.Len(t, protoTd, 2)
	assert.Equal(t, "CloudProfilerConfig", protoTd[0].TypeName)
	assert.Equal(t, "Config", protoTd[1].TypeName)

	fields := protoTd[0].Fields
	require.Len(t, fields, 5)

	// Assert name and tags match sequential ordering
	assert.Equal(t, "AllocatedHeap", fields[0].FieldName)
	assert.Equal(t, 1, registry["CloudProfilerConfig"]["AllocatedHeap"].Tag)

	assert.Equal(t, "Cpu", fields[1].FieldName)
	assert.Equal(t, 2, registry["CloudProfilerConfig"]["Cpu"].Tag)

	assert.Equal(t, "DeletedField", fields[2].FieldName)
	assert.Equal(t, 3, registry["CloudProfilerConfig"]["DeletedField"].Tag)

	assert.Equal(t, "Enabled", fields[3].FieldName)
	assert.Equal(t, 4, registry["CloudProfilerConfig"]["Enabled"].Tag)

	assert.Equal(t, "NewActiveField", fields[4].FieldName)
	assert.Equal(t, 5, registry["CloudProfilerConfig"]["NewActiveField"].Tag)
}

func TestNewAndDeletedConfigFieldLifecycle(t *testing.T) {
	// 1. Setup a mock registry of an already sequentialized message.
	// Message has fields A (tag 1), B (tag 2), C (tag 3).
	registry := RegistryMap{
		"CloudProfilerConfig": {
			"A": FieldTagInfo{Tag: 1, Type: "bool"},
			"B": FieldTagInfo{Tag: 2, Type: "string"},
			"C": FieldTagInfo{Tag: 3, Type: "bool"},
		},
	}

	// 2. Scenario A: We add a new active field "D", and delete active field "B".
	// Active fields are now: "A", "C", and "D".
	params := []Param{
		{
			ConfigPath: "cloud-profiler.a",
			Type:       "bool",
			FlagName:   "a",
			Sensitive:  new(bool),
		},
		{
			ConfigPath: "cloud-profiler.c",
			Type:       "bool",
			FlagName:   "c",
			Sensitive:  new(bool),
		},
		{
			ConfigPath: "cloud-profiler.d",
			Type:       "string",
			FlagName:   "d",
			Sensitive:  new(bool),
		},
	}

	protoTd, err := constructProtoTypeTemplateData(params, registry)
	require.NoError(t, err)

	// We expect 2 types (Config, CloudProfilerConfig). CloudProfilerConfig is at index 0.
	require.Len(t, protoTd, 2)
	assert.Equal(t, "CloudProfilerConfig", protoTd[0].TypeName)

	fields := protoTd[0].Fields
	// Fields list should have 4 items: A, B (retained/deleted), C, D (new).
	require.Len(t, fields, 4)

	// Assert stable tags and new field allocation:
	// A is active, kept at tag 1
	assert.Equal(t, "A", fields[0].FieldName)
	assert.Equal(t, 1, registry["CloudProfilerConfig"]["A"].Tag)

	// B is deleted, retained at tag 2
	assert.Equal(t, "B", fields[1].FieldName)
	assert.Equal(t, 2, registry["CloudProfilerConfig"]["B"].Tag)
	assert.Equal(t, "", fields[1].ConfigPath) // Verifies B is correctly marked as not active/deleted

	// C is active, kept at tag 3
	assert.Equal(t, "C", fields[2].FieldName)
	assert.Equal(t, 3, registry["CloudProfilerConfig"]["C"].Tag)

	// D is new, gets maxTag + 1 = 4
	assert.Equal(t, "D", fields[3].FieldName)
	assert.Equal(t, 4, registry["CloudProfilerConfig"]["D"].Tag)
}

