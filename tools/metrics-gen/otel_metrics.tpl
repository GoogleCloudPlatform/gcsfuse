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
	"errors"
	"sync"
	"sync/atomic"
	"time"

    "github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const logInterval = 5 * time.Minute

var (
	unrecognizedAttr atomic.Value
{{- range $metric := .Metrics -}}
{{- if .Attributes}}
{{- range $combination := (index $.AttrCombinations $metric.Name)}}
	{{getVarName $metric.Name $combination}} = metric.WithAttributeSet(attribute.NewSet(
		{{- range $pair := $combination -}}
			attribute.{{if eq $pair.Type "bool"}}Bool{{else}}String{{end}}("{{$pair.Name}}", {{if eq $pair.Type "bool"}}{{$pair.Value}}{{else}}"{{$pair.Value}}"{{end}}),
		{{- end -}}
	))
{{- end -}}
{{- end -}}
{{- end -}}
)

type histogramRecord struct {
	ctx context.Context
	instrument metric.Int64Histogram
	value      int64
	attributes metric.RecordOption
}

type otelMetrics struct {
    ch chan histogramRecord
	wg *sync.WaitGroup
	{{- range $metric := .Metrics}}
		{{- if or (isCounter $metric) (isUpDownCounter $metric)}}
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
	{{- if or (isCounter .) (isUpDownCounter .)}}
		inc int64
	{{- else }}
		ctx context.Context, latency time.Duration
	{{- end }}
	{{- if .Attributes}}, {{end}}
	{{- range $i, $attr := .Attributes -}}
		{{if $i}}, {{end}}{{toCamel $attr.Name}} {{getGoType $attr.Type}}
	{{- end }}) {
{{- if or (isCounter .) (isUpDownCounter .)}}
	{{- if isCounter . }}
	if inc < 0 {
		logger.Errorf("Counter metric {{.Name}} received a negative increment: %d", inc)
		return
	}
	{{- end}}
	{{buildSwitches .}}
{{- else }}
	var record histogramRecord
	{{buildSwitches .}}
	select {
	  case o.ch <- record: // Do nothing
	  default: // Unblock writes to channel if it's full.
	}
	{{- end -}}
}
{{end}}

func NewOTelMetrics(ctx context.Context, workers int, bufferSize int) (*otelMetrics, error) {
  ch := make(chan histogramRecord, bufferSize)
  var wg sync.WaitGroup
  startSampledLogging(ctx)
  for range workers {
	wg.Add(1)
    go func() {
	  defer wg.Done()
	  for record := range ch {
		if record.attributes != nil {
            record.instrument.Record(record.ctx, record.value, record.attributes)
        } else {
            record.instrument.Record(record.ctx, record.value)
        }
	  }
	}()
  }
  meter := otel.Meter("gcsfuse")
{{- range $metric := .Metrics}}
	{{- if or (isCounter $metric) (isUpDownCounter $metric) }}
	var {{range $i, $combination := (index $.AttrCombinations $metric.Name)}}{{if $i}},
	{{end}}{{getAtomicName $metric.Name $combination}}{{end}} atomic.Int64
	{{- end}}

{{end}}

{{- range $i, $metric := .Metrics}}
	{{- if or (isCounter $metric) (isUpDownCounter $metric)}}
		{{- $instrumentCreationFunc := "" -}}
		{{- $observationFunc := "" -}}
		{{- if isCounter $metric -}}
			{{- $instrumentCreationFunc = "meter.Int64ObservableCounter" -}}
			{{- $observationFunc = "conditionallyObserve" -}}
		{{- else -}}
			{{- $instrumentCreationFunc = "meter.Int64ObservableUpDownCounter" -}}
			{{- $observationFunc = "observeUpDownCounter" -}}
		{{- end}}

	_, err{{$i}} := {{$instrumentCreationFunc}}("{{$metric.Name}}",
		metric.WithDescription("{{.Description}}"),
		metric.WithUnit("{{.Unit}}"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			{{- range $combination := (index $.AttrCombinations $metric.Name)}}
			{{$observationFunc}}(obsrv, &{{getAtomicName $metric.Name $combination}}{{if $metric.Attributes}}, {{getVarName $metric.Name $combination}}{{end}})
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
		wg: &wg,
		{{- range $metric := .Metrics}}
			{{- if or (isCounter $metric) (isUpDownCounter $metric)}}
				{{- range $combination := (index $.AttrCombinations $metric.Name)}}
			{{getAtomicName $metric.Name $combination}}: &{{getAtomicName $metric.Name $combination}},
				{{- end}}
			{{- else}}
			{{toCamel $metric.Name}}: {{toCamel $metric.Name}},
			{{- end}}
		{{- end}}
	}, nil
}

func (o *otelMetrics) Close() {
	close(o.ch)
	o.wg.Wait()
}

func conditionallyObserve(obsrv metric.Int64Observer, counter *atomic.Int64, obsrvOptions ...metric.ObserveOption) {
	if val := counter.Load(); val > 0 {
		obsrv.Observe(val, obsrvOptions...)
	}
}

func observeUpDownCounter(obsrv metric.Int64Observer, counter *atomic.Int64, obsrvOptions ...metric.ObserveOption) {
	obsrv.Observe(counter.Load(), obsrvOptions...)
}

func updateUnrecognizedAttribute(newValue string) {
	unrecognizedAttr.CompareAndSwap("", newValue)
}

// StartSampledLogging starts a goroutine that logs unrecognized attributes periodically.
func startSampledLogging(ctx context.Context) {
	// Init the atomic.Value
	unrecognizedAttr.Store("")

	go func() {
		ticker := time.NewTicker(logInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logUnrecognizedAttribute()
			}
		}
	}()
}

// logUnrecognizedAttribute retrieves and logs any unrecognized attributes.
func logUnrecognizedAttribute() {
	// Atomically load and reset the attribute name, then generate a log
	// if an unrecognized attribute was encountered.
	if currentAttr := unrecognizedAttr.Swap("").(string); currentAttr != "" {
		logger.Tracef("Attribute %s is not declared", currentAttr)
	}
}
