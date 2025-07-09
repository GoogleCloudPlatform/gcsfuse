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

package common

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
{{- range $metric := .Metrics -}}
{{- if .Attributes}}
{{- range $combination := (index $.AttrCombinations $metric.Name)}}
	{{getVarName $metric.Name $combination}} = metric.WithAttributeSet(attribute.NewSet(
		{{- range $pair := $combination -}}
			attribute.{{if eq $pair.Type "string"}}String{{else}}Bool{{end}}("{{$pair.Name}}", {{if eq $pair.Type "string"}}"{{$pair.Value}}"{{else}}{{$pair.Value}}{{end}}),
		{{- end -}}
	))
{{- end -}}
{{- end -}}
{{- end -}}
)

type MetricHandle interface {
{{- range .Metrics}}
	{{toPascal .Name}}(
		{{- if isCounter . }}
			inc int64
		{{- else }}
			ctx context.Context, duration time.Duration
		{{- end }}
		{{- if .Attributes}}, {{end}}
		{{- range $i, $attr := .Attributes -}}
			{{if $i}}, {{end}}{{toCamel $attr.Name}} {{getGoType $attr.Type}}
		{{- end }},
	)
{{- end}}
}

type otelMetrics struct {
    ch chan func()
	{{- range $metric := .Metrics}}
		{{- if isCounter $metric}}
			{{- range $combination := (index $.AttrCombinations $metric.Name)}}
	{{getAtomicName $metric.Name $combination}} *atomic.Int64
			{{- end}}
		{{- end}}
	{{- end}}
	{{- range $metric := .Metrics}}
		{{- if isHistogram $metric}}
	{{toCamel $metric.Name}} metric.Int64Histogram
		{{- end}}
	{{- end}}
}

{{range .Metrics}}
func (o *otelMetrics) {{toPascal .Name}}(
	{{- if isCounter . }}
		inc int64
	{{- else }}
		ctx context.Context, latency time.Duration
	{{- end }}
	{{- if .Attributes}}, {{end}}
	{{- range $i, $attr := .Attributes -}}
		{{if $i}}, {{end}}{{toCamel $attr.Name}} {{getGoType $attr.Type}}
	{{- end }},
) {
	select {
	  case o.ch <- func() {
        {{buildSwitches .}}
      }: // Do nothing
      default: // Unblock writes to channel if it's full.
    }
}
{{end}}

func NewOTelMetrics(ctx context.Context, workers int, bufferSize int) (*otelMetrics, error) {
  ch := make(chan func(), bufferSize)
  for range workers {
    go func() {
	  for {
	    f, ok := <-ch
		if !ok {
		  return
		}
		f()
	  }
	}()
  }
  meter := otel.Meter("gcsfuse")
{{- range $metric := .Metrics}}
	{{- if isCounter $metric}}
	var {{range $i, $combination := (index $.AttrCombinations $metric.Name)}}{{if $i}}, {{end}}{{getAtomicName $metric.Name $combination}}{{end}} atomic.Int64
	{{- end}}
{{- end}}

{{- range $i, $metric := .Metrics}}
	{{- if isCounter $metric}}
	_, err{{$i}} := meter.Int64ObservableCounter("{{$metric.Name}}",
		metric.WithDescription("{{.Description}}"),
		metric.WithUnit("{{.Unit}}"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			{{- range $combination := (index $.AttrCombinations $metric.Name)}}
			obsrv.Observe({{getAtomicName $metric.Name $combination}}.Load(){{if $metric.Attributes}}, {{getVarName $metric.Name $combination}}{{end}})
			{{- end}}
			return nil
		}))
	{{- else}}
	{{toCamel $metric.Name}}, err{{$i}} := meter.Int64Histogram("{{$metric.Name}}",
		metric.WithDescription("{{.Description}}"),
		metric.WithUnit("{{.Unit}}"),
		{{- if .Boundaries}}
		metric.WithExplicitBucketBoundaries({{joinInts .Boundaries}}))
		{{- else}}
		)
		{{- end}}
	{{- end}}
{{end}}

	errs := []error{
		{{- range $i, $metric := .Metrics -}}
			{{if $i}}, {{end}}err{{$i}}
		{{- end -}}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	return &otelMetrics{
	ch : ch,
	{{- range $metric := .Metrics}}
		{{- if isCounter $metric}}
			{{- range $combination := (index $.AttrCombinations $metric.Name)}}
		{{getAtomicName $metric.Name $combination}}: &{{getAtomicName $metric.Name $combination}},
			{{- end}}
		{{- else}}
		{{toCamel $metric.Name}}: {{toCamel $metric.Name}},
		{{- end}}
	{{- end}}
	}, nil
}

func (o *otelMetrics) Flush(ctx context.Context) error {
	// Flush the internal channel of metric updates.
	// This ensures all pending updates are passed to the OTel SDK.
	done := make(chan struct{})
	select {
	case o.ch <- func() {
		close(done)
	}:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Force flush the global meter provider to ensure metrics are exported.
	type flusher interface {
		ForceFlush(context.Context) error
	}
	if p, ok := otel.GetMeterProvider().(flusher); ok {
		return p.ForceFlush(ctx)
	}
	return nil
}