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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func waitForMetricsProcessing() {
	time.Sleep(time.Millisecond)
}

func setupOTel(ctx context.Context, t *testing.T) (*otelMetrics, *metric.ManualReader) {
	t.Helper()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	m, err := NewOTelMetrics(ctx, 10, 100)
	require.NoError(t, err)
	return m, reader
}

// gatherMetrics collects all metrics from the reader.
// It returns a map where the key is the metric name, and the value is the metric data.
func gatherMetrics(ctx context.Context, t *testing.T, rd *metric.ManualReader) map[string]metricdata.Metrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := rd.Collect(ctx, &rm)
	require.NoError(t, err)

	results := make(map[string]metricdata.Metrics)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			results[m.Name] = m
		}
	}
	return results
}

{{range .Metrics}}
{{if or (isCounter .) (isUpDownCounter .) (isGauge .)}}
func Test{{toPascal .Name}}(t *testing.T) {
	{{- if .Attributes}}
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]interface{}
	}{
		{{- $metric := . -}}
		{{- range $combination := (index $.AttrCombinations $metric.Name)}}
		{
			name: "{{getTestName $combination}}",
			f: func(m *otelMetrics) {
				m.{{toPascal $metric.Name}}({{getTestFuncArgs $metric $combination}})
			},
			expected: map[attribute.Set]interface{}{
				attribute.NewSet({{getExpectedAttrs $combination}}): {{if isFloat $metric}}123.456{{else}}123{{end}},
			},
		},
		{{- end}}
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherMetrics(ctx, t, rd)
			metric, ok := metrics["{{.Name}}"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "{{.Name}} metric should not be found")
				return
			}
			require.True(t, ok, "{{.Name}} metric not found")

			expectedMap := make(map[string]interface{})
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			{{if isFloat .}}
			sum, ok := metric.Data.(metricdata.Sum[float64])
			require.True(t, ok, "metric.Data should be of type Sum[float64]")
			actualMap := make(map[string]interface{})
			for _, dp := range sum.DataPoints {
				actualMap[dp.Attributes.Encoded(encoder)] = dp.Value
			}
			assert.Equal(t, expectedMap, actualMap)
			{{else}}
			sum, ok := metric.Data.(metricdata.Sum[int64])
			require.True(t, ok, "metric.Data should be of type Sum[int64]")
			actualMap := make(map[string]interface{})
			for _, dp := range sum.DataPoints {
				actualMap[dp.Attributes.Encoded(encoder)] = dp.Value
			}
			assert.Equal(t, expectedMap, actualMap)
			{{end}}
		})
	}
	{{- else}}
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)

	m.{{toPascal .Name}}({{if isFloat .}}123.456{{else}}123{{end}})
	m.{{toPascal .Name}}({{if isFloat .}}456.789{{else}}456{{end}})
	waitForMetricsProcessing()

	metrics := gatherMetrics(ctx, t, rd)
	metric, ok := metrics["{{.Name}}"]
	require.True(t, ok, "{{.Name}} metric not found")
	s := attribute.NewSet()
	{{if isFloat .}}
	sum, ok := metric.Data.(metricdata.Sum[float64])
	require.True(t, ok, "metric.Data should be of type Sum[float64]")
	actualMap := make(map[string]interface{})
	for _, dp := range sum.DataPoints {
		actualMap[dp.Attributes.Encoded(encoder)] = dp.Value
	}
	assert.Equal(t, map[string]interface{}{s.Encoded(encoder): {{if isGauge .}}456.789{{else}}580.245{{end}}}, actualMap)
	{{else}}
	sum, ok := metric.Data.(metricdata.Sum[int64])
	require.True(t, ok, "metric.Data should be of type Sum[int64]")
	actualMap := make(map[string]interface{})
	for _, dp := range sum.DataPoints {
		actualMap[dp.Attributes.Encoded(encoder)] = dp.Value
	}
	assert.Equal(t, map[string]interface{}{s.Encoded(encoder): int64({{if isGauge .}}456{{else}}579{{end}})}, actualMap)
	{{end}}
	{{- end}}
}
{{else if isHistogram .}}
func Test{{toPascal .Name}}(t *testing.T) {
	{{- if .Attributes}}
	tests := []struct {
		name      string
		latencies []time.Duration
		{{- range .Attributes}}
		{{toCamel .Name}} {{getGoType .Type}}
		{{- end}}
	}{
		{{- $metric := . -}}
		{{- range $combination := (index $.AttrCombinations $metric.Name)}}
		{
			name:      "{{getTestName $combination}}",
			latencies: []time.Duration{100 * time.{{getLatencyUnit $metric.Unit}}, 200 * time.{{getLatencyUnit $metric.Unit}}},
			{{- range $pair := $combination}}
			{{toCamel $pair.Name}}: {{if eq $pair.Type "string"}}"{{$pair.Value}}"{{else}}{{$pair.Value}}{{end}},
			{{- end}}
		},
		{{- end}}
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)
			var totalLatency time.Duration

			for _, latency := range tc.latencies {
				m.{{toPascal .Name}}(ctx, latency, {{getTestFuncArgsForHistogram "tc" .Attributes}})
				totalLatency += latency
			}
			waitForMetricsProcessing()

			metrics := gatherMetrics(ctx, t, rd)
			metric, ok := metrics["{{.Name}}"]
			require.True(t, ok, "{{.Name}} metric not found")

			hist, ok := metric.Data.(metricdata.Histogram[int64])
			require.True(t, ok, "metric.Data should be of type Histogram[int64]")

			attrs := []attribute.KeyValue{
				{{- range .Attributes}}
				attribute.{{if eq .Type "string"}}String{{else}}Bool{{end}}("{{.Name}}", tc.{{toCamel .Name}}),
				{{- end}}
			}
			s := attribute.NewSet(attrs...)
			expectedKey := s.Encoded(encoder)
			var dp metricdata.HistogramDataPoint[int64]
			for _, p := range hist.DataPoints {
				if p.Attributes.Encoded(encoder) == expectedKey {
					dp = p
					break
				}
			}
			require.NotNil(t, dp, "DataPoint not found for key: %s", expectedKey)
			assert.Equal(t, uint64(len(tc.latencies)), dp.Count)
			assert.Equal(t, totalLatency.{{getLatencyMethod .Unit}}(), dp.Sum)
		})
	}
	{{- else}}
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)
	var totalLatency time.Duration
	latencies := []time.Duration{100 * time.{{getLatencyUnit .Unit}}, 200 * time.{{getLatencyUnit .Unit}}}

	for _, latency := range latencies {
		m.{{toPascal .Name}}(ctx, latency)
		totalLatency += latency
	}
	waitForMetricsProcessing()

	metrics := gatherMetrics(ctx, t, rd)
	metric, ok := metrics["{{.Name}}"]
	require.True(t, ok, "{{.Name}} metric not found")

	hist, ok := metric.Data.(metricdata.Histogram[int64])
	require.True(t, ok, "metric.Data should be of type Histogram[int64]")

	s := attribute.NewSet()
	expectedKey := s.Encoded(encoder)
	var dp metricdata.HistogramDataPoint[int64]
	for _, p := range hist.DataPoints {
		if p.Attributes.Encoded(encoder) == expectedKey {
			dp = p
			break
		}
	}
	require.NotNil(t, dp, "DataPoint not found for key: %s", expectedKey)
	assert.Equal(t, uint64(len(latencies)), dp.Count)
	assert.Equal(t, totalLatency.{{getLatencyMethod .Unit}}(), dp.Sum)
	{{- end}}
}
{{end}}
{{end}}
