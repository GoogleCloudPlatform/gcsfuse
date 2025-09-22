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
	"math"
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
		{{getOptionVarName $metric.Name $combination}} = metric.WithAttributeSet(attribute.NewSet(
			{{- range $pair := $combination -}}
				attribute.{{if eq $pair.Type "string"}}String{{else}}Bool{{end}}("{{$pair.Name}}", {{if eq $pair.Type "string"}}"{{$pair.Value}}"{{else}}{{$pair.Value}}{{end}}),
			{{- end -}}
		))
	{{- end -}}
	{{- end -}}
	{{- end -}}
)

type histogramRecord struct {
	ctx        context.Context
	instrument metric.Int64Histogram
	value      int64
	attributes metric.RecordOption
}

type otelMetrics struct {
	ch chan histogramRecord
	wg *sync.WaitGroup
	{{- range $metric := .Metrics}}
		{{- if or (isCounter $metric) (isUpDownCounter $metric) (isGauge $metric)}}
			{{- range $combination := (index $.AttrCombinations $metric.Name)}}
				{{- if isFloat $metric}}
	{{getVarName $metric.Name $combination}} *float64
				{{- else}}
	{{getVarName $metric.Name $combination}} *atomic.Int64
				{{- end}}
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
	{{- else if isUpDownCounter . -}}
		inc {{if isFloat .}}float64{{else}}int64{{end}}
	{{- else if isGauge . -}}
		val {{if isFloat .}}float64{{else}}int64{{end}}
	{{- else }}
		ctx context.Context, latency time.Duration
	{{- end }}
	{{- if .Attributes}}, {{end}}
	{{- range $i, $attr := .Attributes -}}
		{{if $i}}, {{end}}{{toCamel $attr.Name}} {{getGoType $attr.Type}}
	{{- end }}) {
{{- if isCounter . }}
	if inc < 0 {
		logger.Errorf("Counter metric {{.Name}} received a negative increment: %d", inc)
		return
	}
	{{buildSwitches .}}
{{- else if or (isUpDownCounter .) (isGauge .) }}
	{{buildSwitches .}}
{{- else }}
	var record histogramRecord
	{{buildSwitches .}}
	select {
	case o.ch <- record: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
{{- end}}
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
		{{- if or (isCounter $metric) (isUpDownCounter $metric) (isGauge $metric)}}
			{{- range $combination := (index $.AttrCombinations $metric.Name)}}
				{{- if isFloat $metric}}
	var {{getVarName $metric.Name $combination}} float64
				{{- else}}
	var {{getVarName $metric.Name $combination}} atomic.Int64
				{{- end}}
			{{- end}}
		{{- end}}
	{{end}}

	{{- range $i, $metric := .Metrics}}
		{{- if isCounter $metric}}
	_, err{{$i}} := meter.Int64ObservableCounter("{{$metric.Name}}",
		metric.WithDescription("{{.Description}}"),
		metric.WithUnit("{{.Unit}}"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			{{- range $combination := (index $.AttrCombinations $metric.Name)}}
			observe(obsrv, &{{getVarName $metric.Name $combination}}{{if $metric.Attributes}}, {{getOptionVarName $metric.Name $combination}}{{end}})
			{{- end}}
			return nil
		}))
		{{- else if isUpDownCounter $metric}}
			{{- if isFloat $metric}}
	_, err{{$i}} := meter.Float64ObservableUpDownCounter("{{$metric.Name}}",
		metric.WithDescription("{{.Description}}"),
		metric.WithUnit("{{.Unit}}"),
		metric.WithFloat64Callback(func(_ context.Context, obsrv metric.Float64Observer) error {
			{{- range $combination := (index $.AttrCombinations $metric.Name)}}
			observeFloat(obsrv, &{{getVarName $metric.Name $combination}}{{if $metric.Attributes}}, {{getOptionVarName $metric.Name $combination}}{{end}})
			{{- end}}
			return nil
		}))
			{{- else}}
	_, err{{$i}} := meter.Int64ObservableUpDownCounter("{{$metric.Name}}",
		metric.WithDescription("{{.Description}}"),
		metric.WithUnit("{{.Unit}}"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			{{- range $combination := (index $.AttrCombinations $metric.Name)}}
			observe(obsrv, &{{getVarName $metric.Name $combination}}{{if $metric.Attributes}}, {{getOptionVarName $metric.Name $combination}}{{end}})
			{{- end}}
			return nil
		}))
			{{- end}}
		{{- else if isGauge $metric}}
			{{- if isFloat $metric}}
	_, err{{$i}} := meter.Float64ObservableGauge("{{$metric.Name}}",
		metric.WithDescription("{{.Description}}"),
		metric.WithUnit("{{.Unit}}"),
		metric.WithFloat64Callback(func(_ context.Context, obsrv metric.Float64Observer) error {
			{{- range $combination := (index $.AttrCombinations $metric.Name)}}
			observeFloat(obsrv, &{{getVarName $metric.Name $combination}}{{if $metric.Attributes}}, {{getOptionVarName $metric.Name $combination}}{{end}})
			{{- end}}
			return nil
		}))
			{{- else}}
	_, err{{$i}} := meter.Int64ObservableGauge("{{$metric.Name}}",
		metric.WithDescription("{{.Description}}"),
		metric.WithUnit("{{.Unit}}"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			{{- range $combination := (index $.AttrCombinations $metric.Name)}}
			observe(obsrv, &{{getVarName $metric.Name $combination}}{{if $metric.Attributes}}, {{getOptionVarName $metric.Name $combination}}{{end}})
			{{- end}}
			return nil
		}))
			{{- end}}
		{{- else}}
	{{toCamel $metric.Name}}, err{{$i}} := meter.Int64Histogram("{{$metric.Name}}",
		metric.WithDescription("{{.Description}}"),
		metric.WithUnit("{{.Unit}}"),
		{{- if .Boundaries}}
		metric.WithExplicitBucketBoundaries({{joinFloats .Boundaries}}))
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
		ch: ch,
		wg: &wg,
		{{- range $metric := .Metrics}}
			{{- if or (isCounter $metric) (isUpDownCounter $metric) (isGauge $metric)}}
				{{- range $combination := (index $.AttrCombinations $metric.Name)}}
					{{- if isFloat $metric}}
		{{getVarName $metric.Name $combination}}: &{{getVarName $metric.Name $combination}},
					{{- else}}
		{{getVarName $metric.Name $combination}}: &{{getVarName $metric.Name $combination}},
					{{- end}}
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

func observe(obsrv metric.Int64Observer, counter *atomic.Int64, obsrvOptions ...metric.ObserveOption) {
	obsrv.Observe(counter.Load(), obsrvOptions...)
}

func observeFloat(obsrv metric.Float64Observer, val *float64, obsrvOptions ...metric.ObserveOption) {
	obsrv.Observe(math.Float64frombits(atomic.LoadUint64((*uint64)(val))), obsrvOptions...)
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
