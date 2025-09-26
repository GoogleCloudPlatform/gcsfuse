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

// **** DO NOT EDIT - FILE IS AUTO-GENERATED ****
package metrics

import (
	"context"
	"time"
)

type noopMetrics struct {}
{{- range .Metrics}}
	func (*noopMetrics) {{toPascal .Name}}(
		{{- if or (isCounter .) (isUpDownCounter .) -}}
			inc int64
		{{- else -}}
			ctx context.Context, latency time.Duration
		{{- end }}
		{{- if .Attributes}}, {{end}}
		{{- range $i, $attr := .Attributes -}}
			{{if $i}}, {{end}}{{toCamel $attr.Name}} {{getGoType $attr.Type}}
		{{- end }}){}
{{end}}

func NewNoopMetrics() MetricHandle {
	var n noopMetrics
	return &n
}
