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
		// Test case 1: User has set the flag, no optimization should be applied.
		t.Run("user_set", func(t *testing.T) {
			c := &Config{}
			c.{{$flag.GoPath}} = {{$flag.DefaultValue}}
			isSet := &mockIsValueSet{setFlags: map[string]bool{"{{$flag.FlagName}}": true}}

			optimizedFlags := c.ApplyOptimizations(isSet)

			assert.Empty(t, optimizedFlags)
            {{- if eq $flag.GoType "int64" }}
			assert.Equal(t, int64({{$flag.DefaultValue}}), c.{{$flag.GoPath}})
            {{- else }}
			assert.Equal(t, {{$flag.DefaultValue}}, c.{{$flag.GoPath}})
            {{- end }}
		})

		// Test case 2: No profile or machine-based optimization match.
		t.Run("no_optimization", func(t *testing.T) {
			c := &Config{ Profile: "non_existent_profile" }
			c.{{$flag.GoPath}} = {{$flag.DefaultValue}}
			isSet := &mockIsValueSet{
				setFlags:  map[string]bool{"machine-type": true},
				stringFlags: map[string]string{"machine-type": "n1-standard-1"}, // A machine type not in any group
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
	})
{{- end }}
{{- end }}
}

