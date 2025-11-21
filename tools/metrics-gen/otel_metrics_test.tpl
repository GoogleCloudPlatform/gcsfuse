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

// metricValueMap maps attribute sets to metric values.
type metricValueMap map[string]int64

// metricHistogramMap maps attribute sets to histogram data points.
type metricHistogramMap map[string]metricdata.HistogramDataPoint[int64]

func waitForMetricsProcessing() {
	time.Sleep(5 * time.Millisecond)
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

// gatherHistogramMetrics collects all histogram metrics from the reader.
// It returns a map where the key is the metric name, and the value is another map.
// The inner map's key is a string representation of the attributes,
// and the value is the metricdata.HistogramDataPoint.
func gatherHistogramMetrics(ctx context.Context, t *testing.T, rd *metric.ManualReader) map[string]map[string]metricdata.HistogramDataPoint[int64] {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := rd.Collect(ctx, &rm)
	require.NoError(t, err)

	results := make(map[string]map[string]metricdata.HistogramDataPoint[int64])
	encoder := attribute.DefaultEncoder() // Using default encoder

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			// We are interested in Histogram[int64].
			hist, ok := m.Data.(metricdata.Histogram[int64])
			if !ok {
				continue
			}

			metricMap := make(metricHistogramMap)
			for _, dp := range hist.DataPoints {
				if dp.Count == 0 {
					continue
				}

				metricMap[dp.Attributes.Encoded(encoder)] = dp
			}

			if len(metricMap) > 0 {
				results[m.Name] = metricMap
			}
		}
	}

	return results
}

// gatherNonZeroCounterMetrics collects all non-zero counter metrics from the reader.
// It returns a map where the key is the metric name, and the value is another map.
// The inner map's key is a string representation of the attributes,
// and the value is the metric's value.
func gatherNonZeroCounterMetrics(ctx context.Context, t *testing.T, rd *metric.ManualReader) map[string]map[string]int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := rd.Collect(ctx, &rm)
	require.NoError(t, err)

	results := make(map[string]map[string]int64)
	encoder := attribute.DefaultEncoder()

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			// We are interested in Sum[int64] which corresponds to int_counter.
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}

			metricMap := make(metricValueMap)
			for _, dp := range sum.DataPoints {
				if dp.Value == 0 {
					continue
				}

				metricMap[dp.Attributes.Encoded(encoder)] = dp.Value
			}

			if len(metricMap) > 0 {
				results[m.Name] = metricMap
			}
		}
	}

	return results
}

{{range .Metrics}}
{{if or (isCounter .) (isUpDownCounter .)}}
func Test{{toPascal .Name}}(t *testing.T) {
	{{- if .Attributes}}
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{{- $metric := . -}}
		{{- range $combination := (index $.AttrCombinations $metric.Name)}}
		{
			name: "{{getTestName $combination}}",
			f: func(m *otelMetrics) {
				m.{{toPascal $metric.Name}}(5, {{getTestFuncArgs $combination}})
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet({{getExpectedAttrs $combination}}): 5,
			},
		},
		{{- end}}
		{{- $combinations := (index $.AttrCombinations $metric.Name) -}}
		{{- if and .Attributes (gt (len $combinations) 1) -}}
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				{{- $firstComb := (index $combinations 0) -}}
				{{- $secondComb := (index $combinations 1) -}}
				m.{{toPascal $metric.Name}}(5, {{getTestFuncArgs $firstComb}})
				m.{{toPascal $metric.Name}}(2, {{getTestFuncArgs $secondComb}})
				m.{{toPascal $metric.Name}}(3, {{getTestFuncArgs $firstComb}})
			},
			expected: map[attribute.Set]int64{
				{{- $firstComb := (index $combinations 0) -}}
				{{- $secondComb := (index $combinations 1) -}}
				attribute.NewSet({{getExpectedAttrs $firstComb}}): 8,
				attribute.NewSet({{getExpectedAttrs $secondComb}}): 2,
			},
		},
		{{- end}}
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				{{- $firstComb := (index $combinations 0) -}}
				m.{{toPascal $metric.Name}}(-5, {{getTestFuncArgs $firstComb}})
				m.{{toPascal $metric.Name}}(2, {{getTestFuncArgs $firstComb}})
			},
			expected: map[attribute.Set]int64{attribute.NewSet({{getExpectedAttrs (index $combinations 0)}}): {{if isUpDownCounter $metric}}-3{{else}}2{{end}}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["{{.Name}}"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "{{.Name}} metric should not be found")
				return
			}
			require.True(t, ok, "{{.Name}} metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
	{{- else}}
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)

	m.{{toPascal .Name}}(1024)
	m.{{toPascal .Name}}(2048)
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	metric, ok := metrics["{{.Name}}"]
	require.True(t, ok, "{{.Name}} metric not found")
	s := attribute.NewSet()
	assert.Equal(t, map[string]int64{s.Encoded(encoder): 3072}, metric, "Positive increments should be summed.")

	// Test negative increment
	m.{{toPascal .Name}}(-100)
	waitForMetricsProcessing()

	metrics = gatherNonZeroCounterMetrics(ctx, t, rd)
	metric, ok = metrics["{{.Name}}"]
	require.True(t, ok, "{{.Name}} metric not found after negative increment")
	assert.Equal(t, map[string]int64{s.Encoded(encoder): {{if isUpDownCounter .}}2972{{else}}3072{{end}}}, metric, "Negative increment should {{if isUpDownCounter .}}change{{else}}not change{{end}} the metric value.")
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
			{{toCamel $pair.Name}}: {{if eq $pair.Type "bool"}}{{$pair.Value}}{{else}}"{{$pair.Value}}"{{end}},
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

			metrics := gatherHistogramMetrics(ctx, t, rd)
			metric, ok := metrics["{{.Name}}"]
			require.True(t, ok, "{{.Name}} metric not found")

			attrs := []attribute.KeyValue{
				{{- range .Attributes}}
				attribute.{{if eq .Type "bool"}}Bool("{{.Name}}", tc.{{toCamel .Name}}){{else}}String("{{.Name}}", string(tc.{{toCamel .Name}})){{end}},
				{{- end}}
			}
			s := attribute.NewSet(attrs...)
			expectedKey := s.Encoded(encoder)
			dp, ok := metric[expectedKey]
			require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
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

	metrics := gatherHistogramMetrics(ctx, t, rd)
	metric, ok := metrics["{{.Name}}"]
	require.True(t, ok, "{{.Name}} metric not found")

	s := attribute.NewSet()
	expectedKey := s.Encoded(encoder)
	dp, ok := metric[expectedKey]
	require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
	assert.Equal(t, uint64(len(latencies)), dp.Count)
	assert.Equal(t, totalLatency.{{getLatencyMethod .Unit}}(), dp.Sum)
	{{- end}}
}
{{end}}
{{end}}
