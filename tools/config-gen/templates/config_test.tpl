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
		// Test case 1: User has set the flag to a non-default value; optimizations should be ignored FOR THAT FLAG.
		t.Run("user_set", func(t *testing.T) {
			{{- if and .Optimizations (or (eq $flag.GoType "int64") (eq $flag.GoType "bool")) }}
			{{- $profileName := "" -}}
			{{- if .Optimizations.Profiles -}}
			{{- $profile := index .Optimizations.Profiles 0 -}}
			{{- $profileName = $profile.Name -}}
			{{- end }}
			{{- $machineTypeForOptimisation := "a2-megagpu-16g" -}}
			{{- $machineTypeComment := "From the \"high-performance\" group." -}}
			{{- if .Optimizations.MachineBasedOptimization -}}
				{{- $mbo := index .Optimizations.MachineBasedOptimization 0 -}}
				{{- $foundMachineType := "" -}}
				{{- range $mt, $group := $.MachineTypeToGroupMap -}}
					{{- if and (not $foundMachineType) (eq $group $mbo.Group) -}}
						{{- $foundMachineType = $mt -}}
					{{- end -}}
				{{- end -}}
				{{- if $foundMachineType -}}
					{{- $machineTypeForOptimisation = $foundMachineType -}}
					{{- $machineTypeComment = printf "From the %q group." $mbo.Group -}}
				{{- end -}}
			{{- end }}
			{{- if eq $flag.GoType "int64" }}
			const nonDefaultValue = int64(98765)
			{{- else if eq $flag.GoType "bool" }}
			nonDefaultValue := !({{$flag.DefaultValue}})
			{{- end }}
			c := &Config{
				Profile: "{{$profileName}}", // A profile that would otherwise cause optimization.
			}
			c.{{$flag.GoPath}} = nonDefaultValue // Set a non-default value.
			isSet := &mockIsValueSet{
				setFlags: map[string]bool{
					"{{$flag.FlagName}}": true,
					"machine-type":       true, // A machine type that would otherwise cause optimization.
				},
				stringFlags: map[string]string{
					"machine-type": "{{$machineTypeForOptimisation}}", // {{ $machineTypeComment }}
				},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.NotContains(t, optimizedFlags, "{{$flag.ConfigPath}}")
			assert.Equal(t, nonDefaultValue, c.{{$flag.GoPath}})
			{{- end }}
		})

		// Test case 2: No profile or machine-based optimization match.
		t.Run("no_optimization", func(t *testing.T) {
			c := &Config{ Profile: "non_existent_profile" }
			c.{{$flag.GoPath}} = {{$flag.DefaultValue}}
			isSet := &mockIsValueSet{
				setFlags:  map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "low-end-machine"}, // A machine type not in any group
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
			{{- if eq $flag.GoType "int64" }}
			assert.Equal(t, int64({{$flag.DefaultValue}}), c.{{$flag.GoPath}})
			{{- else }}
			assert.Equal(t, {{$flag.DefaultValue}}, c.{{$flag.GoPath}})
			{{- end }}
		})

		// Test cases for profile-based optimizations
		{{- range .Optimizations.Profiles }}
		t.Run("profile_{{.Name}}", func(t *testing.T) {
			c := &Config{ Profile: "{{.Name}}" }
			c.{{$flag.GoPath}} = {{$flag.DefaultValue}}
			isSet := &mockIsValueSet{setFlags: map[string]bool{}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "{{$flag.ConfigPath}}")
			{{- if eq $flag.GoType "int64" }}
			assert.Equal(t, int64({{ formatValue .Value }}), c.{{$flag.GoPath}})
			{{- else }}
			assert.Equal(t, {{ formatValue .Value }}, c.{{$flag.GoPath}})
			{{- end }}
		})
		{{- end }}

		// Test cases for machine-based optimizations
		{{- range .Optimizations.MachineBasedOptimization }}
        {{- $mbo := . }}
		t.Run("machine_group_{{$mbo.Group}}", func(t *testing.T) {
			// Find a machine type from the group to use in the test
			{{ $machineType := "" -}}
			{{- range $mt, $group := $.MachineTypeToGroupMap -}}
			    {{- if and (not $machineType) (eq $group $mbo.Group) -}}
			        {{- $machineType = $mt -}}
			    {{- end -}}
			{{- end -}}
			c := &Config{ Profile: "" }
			c.{{$flag.GoPath}} = {{$flag.DefaultValue}}
			isSet := &mockIsValueSet{
				setFlags:  map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "{{$machineType}}"},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "{{$flag.ConfigPath}}")
			{{- if eq $flag.GoType "int64" }}
			assert.Equal(t, int64({{ formatValue $mbo.Value }}), c.{{$flag.GoPath}})
			{{- else }}
			assert.Equal(t, {{ formatValue $mbo.Value }}, c.{{$flag.GoPath}})
			{{- end }}
		})
		{{- end }}

		{{- if and .Optimizations.Profiles .Optimizations.MachineBasedOptimization }}
		// Test case: Profile optimization should override machine-based optimization.
		t.Run("profile_overrides_machine_type", func(t *testing.T) {
			{{- $profile := index .Optimizations.Profiles 0 -}}
			{{- $mbo := index .Optimizations.MachineBasedOptimization 0 -}}
			// Find a machine type from the group to use in the test
			{{ $machineType := "" -}}
			{{- range $mt, $group := $.MachineTypeToGroupMap -}}
				{{- if and (not $machineType) (eq $group $mbo.Group) -}}
					{{- $machineType = $mt -}}
				{{- end -}}
			{{- end -}}
			c := &Config{ Profile: "{{$profile.Name}}" }
			c.{{$flag.GoPath}} = {{$flag.DefaultValue}}
			isSet := &mockIsValueSet{
				setFlags:  map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "{{$machineType}}"},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "{{$flag.ConfigPath}}")
			// Assert that the profile value is used, not the machine-based one.
			{{- if eq $flag.GoType "int64" }}
			assert.Equal(t, int64({{ formatValue $profile.Value }}), c.{{$flag.GoPath}})
			{{- else }}
			assert.Equal(t, {{ formatValue $profile.Value }}, c.{{$flag.GoPath}})
			{{- end }}
		})
		{{- end }}

		{{- if .Optimizations.MachineBasedOptimization }}
		// Test case: Fallback to machine-based optimization when profile is non-existent.
		t.Run("fallback_to_machine_type_with_non_existent_profile", func(t *testing.T) {
			{{- $mbo := index .Optimizations.MachineBasedOptimization 0 -}}
			// Find a machine type from the group to use in the test
			{{ $machineType := "" -}}
			{{- range $mt, $group := $.MachineTypeToGroupMap -}}
				{{- if and (not $machineType) (eq $group $mbo.Group) -}}
					{{- $machineType = $mt -}}
				{{- end -}}
			{{- end -}}
			c := &Config{ Profile: "non_existent_profile" }
			c.{{$flag.GoPath}} = {{$flag.DefaultValue}}
			isSet := &mockIsValueSet{
				setFlags:  map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "{{$machineType}}"},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "{{$flag.ConfigPath}}")
			// Assert that the machine-based value is used.
			{{- if eq $flag.GoType "int64" }}
			assert.Equal(t, int64({{ formatValue $mbo.Value }}), c.{{$flag.GoPath}})
			{{- else }}
			assert.Equal(t, {{ formatValue $mbo.Value }}, c.{{$flag.GoPath}})
			{{- end }}
		})

		// Test case: Fallback to machine-based optimization when a profile is set, but has no rule for THIS flag.
		{{ $unrelatedProfile := "aiml-training" -}}
		{{- $hasRuleForUnrelatedProfile := false -}}
		{{- range .Optimizations.Profiles -}}
			{{- if eq .Name $unrelatedProfile -}}
				{{- $hasRuleForUnrelatedProfile = true -}}
			{{- end -}}
		{{- end -}}
		{{- if not $hasRuleForUnrelatedProfile -}}
		t.Run("fallback_to_machine_type_with_unrelated_profile", func(t *testing.T) {
			{{- $mbo := index .Optimizations.MachineBasedOptimization 0 -}}
			// Find a machine type from the group to use in the test
			{{ $machineType := "" -}}
			{{- range $mt, $group := $.MachineTypeToGroupMap -}}
				{{- if and (not $machineType) (eq $group $mbo.Group) -}}
					{{- $machineType = $mt -}}
				{{- end -}}
			{{- end -}}
			c := &Config{ Profile: "{{$unrelatedProfile}}" }
			c.{{$flag.GoPath}} = {{$flag.DefaultValue}}
			isSet := &mockIsValueSet{
				setFlags:  map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "{{$machineType}}"},
			}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Contains(t, optimizedFlags, "{{$flag.ConfigPath}}")
			// Assert that the machine-based value is used.
			{{- if eq $flag.GoType "int64" }}
			assert.Equal(t, int64({{ formatValue $mbo.Value }}), c.{{$flag.GoPath}})
			{{- else }}
			assert.Equal(t, {{ formatValue $mbo.Value }}, c.{{$flag.GoPath}})
			{{- end }}
		})
		{{- end -}}
		{{- end }}
	})
{{- end }}
{{- end }}
}

