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
	"testing"

	"github.com/stretchr/testify/assert"
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
