// Copyright 2024 Google Inc. All Rights Reserved.
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
	"time"
	"net/url"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

{{$bt := .Backticks}}
{{range .TypeTemplateData}}
type {{ .TypeName}} struct {
  {{- range $idx, $fld := .Fields}}
  {{ $fld.FieldName}} {{ $fld.DataType}} {{$bt}}yaml:"{{$fld.ConfigPath}}"{{$bt}}
{{end}}
}
{{end}}

func BindFlags(v *viper.Viper, flagSet *pflag.FlagSet) error {
  var err error
  {{range .FlagTemplateData}}
  flagSet.{{ .Fn}}("{{ .FlagName}}", "{{ .Shorthand}}", {{ .DefaultValue}}, {{ .Usage}})
  {{if .IsDeprecated}}
  err = flagSet.MarkDeprecated("{{ .FlagName}}", "{{ .DeprecationWarning}}")
  if err != nil {
    return err
  }
  {{end}}
  {{if .HideFlag}}
  err = flagSet.MarkHidden("{{ .FlagName}}")
  if err != nil {
    return err
  }
  {{end}}
  {{if .HideShorthand}}flagSet.ShorthandLookup("{{ .Shorthand}}").Hidden = true{{end}}
  {{if ne .ConfigPath ""}}
  err = v.BindPFlag("{{ .ConfigPath}}", flagSet.Lookup("{{ .FlagName}}"))
  if err != nil {
    return err
  }
  {{end}}
  {{end}}
  return nil
}
