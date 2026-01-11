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
{{- range .FlagTemplateData }}
{{- if .Optimizations }}
    {{- $flag := . }}
	// Tests for {{ $flag.ConfigPath }}
	t.Run("{{$flag.ConfigPath}}", func(t *testing.T) {
		testCases := []struct {
			name            string
			config          Config
			isSet           *mockIsValueSet
			input           *OptimizationInput
			expectOptimized bool
			expectedValue   any
		}{
			{
				name: "user_set",
				config: Config{
					{{- if .Optimizations.Profiles }}
					{{- $profile := index .Optimizations.Profiles 0 }}
					Profile: "{{$profile.Name}}",
					{{- end }}
				},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{
						"{{$flag.ConfigPath}}": true,
						"machine-type":       true,
					},
					{{- if .Optimizations.MachineBasedOptimization }}
					{{- $mbo := index .Optimizations.MachineBasedOptimization 0 }}
					{{- $machineType := index $.MachineTypeGroups $mbo.Group 0 }}
					stringFlags: map[string]string{
						"machine-type": "{{$machineType}}",
					},
					{{- end }}
				},
				{{- if .Optimizations.BucketTypeOptimization }}
				{{- $bto := index .Optimizations.BucketTypeOptimization 0 }}
				input:           &OptimizationInput{BucketType: BucketType{{ $bto.BucketType | title }}},
				{{- else }}
				input:           nil,
				{{- end }}
				expectOptimized: false,
				expectedValue:
				{{- if eq $flag.GoType "int64" }} int64(98765),
				{{- else if eq $flag.GoType "bool" }} !({{$flag.DefaultValue}}),
				{{- else if eq $flag.GoType "string" }} {{$flag.DefaultValue}} + "-non-default",
				{{- else if eq $flag.GoType "float64" }} {{$flag.DefaultValue}} + 1.23,
				{{- else }} // compilation error: unhandled type '{{$flag.GoType}}' in test generation for {{$flag.ConfigPath}}
				{{- end }}
			},
			{
				name:   "no_optimization",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "low-end-machine"},
				},
			input:           nil,
				expectOptimized: false,
				expectedValue:   {{$flag.DefaultValue}},
			},
		{{- range .Optimizations.Profiles }}
			{
				name:            "profile_{{.Name}}",
				config:          Config{Profile: "{{.Name}}"},
				isSet:           &mockIsValueSet{setFlags: map[string]bool{}},
				input:           nil,
				expectOptimized: true,
				expectedValue:   {{.Value}},
			},
		{{- end }}
		{{- range .Optimizations.MachineBasedOptimization }}
			{{- $mbo := . }}
			{{- $machineType := index $.MachineTypeGroups $mbo.Group 0 }}
			{
				name:   "machine_group_{{$mbo.Group}}",
				config: Config{Profile: ""},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "{{$machineType}}"},
				},
				input:           nil,
				expectOptimized: true,
				expectedValue:   {{$mbo.Value}},
			},
		{{- end }}
		{{- range .Optimizations.BucketTypeOptimization }}
			{{- $bto := . }}
			{
				name:   "bucket_type_{{$bto.BucketType}}",
				config: Config{Profile: ""},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{},
				},
				input:           &OptimizationInput{BucketType: BucketType{{ $bto.BucketType | title }}},
				expectOptimized: true,
				expectedValue:   {{$bto.Value}},
			},
		{{- end }}
		{{- if and .Optimizations.Profiles .Optimizations.MachineBasedOptimization }}
			{{- $profile := index .Optimizations.Profiles 0 -}}
			{{- $mbo := index .Optimizations.MachineBasedOptimization 0 -}}
			{{- $machineType := index $.MachineTypeGroups $mbo.Group 0 }}
			{
				name:   "profile_overrides_machine_type",
				config: Config{Profile: "{{$profile.Name}}"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "{{$machineType}}"},
				},
				input:           nil,
				expectOptimized: true,
				expectedValue:   {{$profile.Value}},
			},
		{{- end }}
		{{- if and .Optimizations.Profiles .Optimizations.BucketTypeOptimization }}
			{{- $profile := index .Optimizations.Profiles 0 -}}
			{{- $bto := index .Optimizations.BucketTypeOptimization 0 -}}
			{
				name:   "profile_overrides_bucket_type",
				config: Config{Profile: "{{$profile.Name}}"},
				isSet: &mockIsValueSet{
					setFlags: map[string]bool{},
				},
				input:           &OptimizationInput{BucketType: BucketType{{ $bto.BucketType | title }}},
				expectOptimized: true,
				expectedValue:   {{$profile.Value}},
			},
		{{- end }}
		{{- if and .Optimizations.MachineBasedOptimization .Optimizations.BucketTypeOptimization }}
			{{- $mbo := index .Optimizations.MachineBasedOptimization 0 -}}
			{{- $bto := index .Optimizations.BucketTypeOptimization 0 -}}
			{{- $machineType := index $.MachineTypeGroups $mbo.Group 0 }}
			{
				name:   "machine_type_overrides_bucket_type",
				config: Config{Profile: ""},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "{{$machineType}}"},
				},
				input:           &OptimizationInput{BucketType: BucketType{{ $bto.BucketType | title }}},
				expectOptimized: true,
				expectedValue:   {{$mbo.Value}},
			},
		{{- end }}
		{{- if .Optimizations.MachineBasedOptimization }}
			{{- $mbo := index .Optimizations.MachineBasedOptimization 0 -}}
			{{- $machineType := index $.MachineTypeGroups $mbo.Group 0 -}}
			{
				name:   "fallback_to_machine_type_with_non_existent_profile",
				config: Config{Profile: "non_existent_profile"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "{{$machineType}}"},
				},
				input:           nil,
				expectOptimized: true,
				expectedValue:   {{$mbo.Value}},
			},
			{{- $unrelatedProfile := "aiml-training" -}}
			{{- $hasRuleForUnrelatedProfile := false -}}
			{{- range .Optimizations.Profiles -}}
				{{- if eq .Name $unrelatedProfile -}}
					{{- $hasRuleForUnrelatedProfile = true -}}
				{{- end -}}
			{{- end -}}
			{{- if not $hasRuleForUnrelatedProfile -}}
			{
				name:   "fallback_to_machine_type_when_aiml-training_is_unrelated",
				config: Config{Profile: "{{$unrelatedProfile}}"},
				isSet: &mockIsValueSet{
					setFlags:    map[string]bool{"machine-type": true},
					stringFlags: map[string]string{"machine-type": "{{$machineType}}"},
				},
				input:           nil,
				expectOptimized: true,
				expectedValue:   {{$mbo.Value}},
			},
			{{- end }}
		{{- end }}
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// We need a copy of the config for each test case.
				c := tc.config
				// Set the default or non-default value on the config object.
				if tc.name == "user_set" {
					c.{{$flag.GoPath}} = tc.expectedValue.({{$flag.GoType}})
				} else {
					c.{{$flag.GoPath}} = {{$flag.DefaultValue}}
				}
				
				optimizedFlags := c.ApplyOptimizations(tc.isSet, tc.input)

				if tc.expectOptimized {
					assert.Contains(t, optimizedFlags, "{{$flag.ConfigPath}}")
				} else {
					assert.NotContains(t, optimizedFlags, "{{$flag.ConfigPath}}")
				}
				// Use EqualValues to handle the int vs int64 type mismatch for default values.
				assert.EqualValues(t, tc.expectedValue, c.{{$flag.GoPath}})
			})
		}
	})
{{- end }}
{{- end }}
}
