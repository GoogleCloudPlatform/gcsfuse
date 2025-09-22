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

// GENERATED CODE - DO NOT EDIT MANUALLY.

package cfg

import (
	"log"
	"time"
	"reflect"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg/shared"
)

// AllFlagOptimizationRules is the generated map from a flag's config-path to its specific rules.
var AllFlagOptimizationRules = map[string]shared.OptimizationRules{
{{- range .FlagTemplateData }}
	{{- if .Optimizations }}
	{{- $goType := .GoType -}}
	"{{ .ConfigPath }}": {
		{{- if .Optimizations.MachineBasedOptimization }}
		MachineBasedOptimization: []shared.MachineBasedOptimization{
			{{- range .Optimizations.MachineBasedOptimization }}
			{
				Group: "{{ .Group }}",
				Value: {{$goType}}({{ formatValue .Value }}),
			},
			{{- end }}
		},
		{{- end }}
		{{- if .Optimizations.Profiles }}
		Profiles: []shared.ProfileOptimization{
			{{- range .Optimizations.Profiles }}
			{
				Name: "{{ .Name }}",
				Environments: []shared.EnvironmentOptimization{
					{{- range .Environments }}
					{
						Name:  "{{ .Name }}",
						Value: {{$goType}}({{ formatValue .Value }}),
					},
					{{- end }}
				},
			},
			{{- end }}
		},
		{{- end }}
	},
	{{- end }}
{{- end }}
}

// machineTypeToGroupsMap is the generated map from machine type to the groups it belongs to.
var machineTypeToGroupsMap = map[string][]string{
{{- range $machineType, $groups := .MachineTypeToGroupsMap }}
	"{{ $machineType }}": { 
	{{- range $group := $groups }}
		"{{ $group }}",
	{{- end }}
	},
{{- end }}
}

// ApplyOptimizations modifies the config in-place with optimized values.
func (c *Config) ApplyOptimizations(isSet isValueSet) []string {
	var optimizedFlags []string
	// Skip all optimizations if autoconfig is disabled.
	if c.DisableAutoconfig {
		return nil
	}

	profileName := c.Profile
	envName := detectGKEEnvironment()
	machineType, err := getMachineType(isSet)
	if err != nil {
		// Non-fatal, just means machine-based optimizations won't apply.
		machineType = ""
	}
	c.MachineType = machineType

	// Apply optimizations for each flag that has rules defined.
{{- range .FlagTemplateData }}
{{- if .Optimizations }}
	if !isSet.IsSet("{{ .FlagName }}") {
		rules := AllFlagOptimizationRules["{{ .ConfigPath }}"]
		result := getOptimizedValue(&rules, c.{{ .GoPath }}, profileName, machineType, envName, machineTypeToGroupsMap)
		if result.Found {
			if val, ok := result.Value.({{ .GoType }}); ok {
				if !reflect.DeepEqual(c.{{ .GoPath }}, val) {
					log.Printf(
						"INFO: For flag '%s', value changed from %v to %v due to: %s",
						"{{ .ConfigPath }}",
						c.{{ .GoPath }},
						val,
						result.Reason,
					)
					c.{{ .GoPath }} = val
					optimizedFlags = append(optimizedFlags, "{{ .ConfigPath }}")
				}
			}
		}
	}
{{- end }}
{{- end }}
	return optimizedFlags
}

{{$bt := .Backticks}}
{{range .TypeTemplateData}}
type {{ .TypeName}} struct {
  {{- range $idx, $fld := .Fields}}
  {{ $fld.FieldName}} {{ $fld.DataType}} {{$bt}}yaml:"{{$fld.ConfigPath}}"{{$bt}}
{{end}}
}
{{end}}

func BuildFlagSet(flagSet *pflag.FlagSet) error {
  {{range .FlagTemplateData}}
  flagSet.{{ .Fn}}("{{ .FlagName}}", "{{ .Shorthand}}", {{ .DefaultValue}}, {{ .Usage}})
  {{if .IsDeprecated}}
  if err := flagSet.MarkDeprecated("{{ .FlagName}}", "{{ .DeprecationWarning}}"); err != nil {
    return err
  }
  {{end}}
  {{if .HideFlag}}
  if err := flagSet.MarkHidden("{{ .FlagName}}"); err != nil {
    return err
  }
  {{end}}
  {{if .HideShorthand}}flagSet.ShorthandLookup("{{ .Shorthand}}").Hidden = true{{end}}
  {{end}}
  return nil
}

func BindFlags(v *viper.Viper, flagSet *pflag.FlagSet) error {
  {{range .FlagTemplateData}}
  {{if ne .ConfigPath ""}}
  if err := v.BindPFlag("{{ .ConfigPath}}", flagSet.Lookup("{{ .FlagName}}")); err != nil {
    return err
  }
  {{end}}
  {{end}}
  return nil
}
