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
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Helper to wait for async metric processing.
// This is a hack. A better solution would be to have a Flush/Close method on otelMetrics.
func waitForMetrics() {
	time.Sleep(100 * time.Millisecond)
}

{{range .Metrics}}
func Test{{toPascal .Name}}(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	{{$combinations := (index $.AttrCombinations .Name)}}
	{{$firstCombo := index $combinations 0}}
	{{if .Attributes -}}
		{{- if $firstCombo -}}
	m.{{toPascal .Name}}(
		{{if isCounter .}}5{{else}}ctx, 123*time.{{getUnit .Unit}}{{end}},
		{{- range $i, $pair := $firstCombo -}}
			{{if $i}}, {{end}}{{if eq $pair.Type "string"}}"{{$pair.Value}}"{{else}}{{$pair.Value}}{{end}}
		{{- end -}}
	)
	
		{{- end -}}
		{{- if gt (len $combinations) 1 -}}
			{{- $secondCombo := index $combinations 1 -}}

	m.{{toPascal .Name}}(
		{{if isCounter .}}2{{else}}ctx, 456*time.{{getUnit .Unit}}{{end}},
		{{- range $i, $pair := $secondCombo -}}
			{{if $i}}, {{end}}{{if eq $pair.Type "string"}}"{{$pair.Value}}"{{else}}{{$pair.Value}}{{end}}
		{{- end -}}
	)
		{{- end -}}

		{{- if $firstCombo -}}
	m.{{toPascal .Name}}(
		{{if isCounter .}}3{{else}}ctx, 789*time.{{getUnit .Unit}}{{end}},
		{{- range $i, $pair := $firstCombo -}}
			{{if $i}}, {{end}}{{if eq $pair.Type "string"}}"{{$pair.Value}}"{{else}}{{$pair.Value}}{{end}}
		{{- end -}}
	)
		{{- end -}}

	{{- else -}}
	m.{{toPascal .Name}}({{if isCounter .}}5{{else}}ctx, 123{{if getUnit .Unit}}*time.{{getUnit .Unit}}{{end}}{{end}})

	m.{{toPascal .Name}}({{if isCounter .}}3{{else}}ctx, 456{{if getUnit .Unit}}*time.{{getUnit .Unit}}{{end}}{{end}})
	{{- end}}

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "{{.Name}}" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric {{.Name}} not found")
	assert.Equal(t, "{{.Description}}", foundMetric.Description)

	{{if isCounter . -}}
	sum, ok := foundMetric.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "metric {{.Name}} is not a Sum")
	dataPoints := sum.DataPoints
	{{- else -}}
	hist, ok := foundMetric.Data.(metricdata.Histogram[int64])
	assert.True(t, ok, "metric {{.Name}} is not a Histogram")
	dataPoints := hist.DataPoints
	{{- end}}

	{{if .Attributes -}}
		{{- if $firstCombo -}}
			{{- if gt (len $combinations) 1 -}}
				{{- $secondCombo := index $combinations 1 -}}
	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"{{getAttrMapKey $firstCombo}}": {val: {{if isCounter .}}8{{else}}912{{end}}, count: 2},
		"{{getAttrMapKey $secondCombo}}": {val: {{if isCounter .}}2{{else}}456{{end}}, count: 1},
	}
			{{- else -}}
	assert.Len(t, dataPoints, 1)
	expected := map[string]struct{ val int64; count uint64 }{
		"{{getAttrMapKey $firstCombo}}": {val: {{if isCounter .}}8{{else}}912{{end}}, count: 2},
	}
			{{- end -}}
	for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		{{if isCounter . -}}
		assert.Equal(t, e.val, dp.Value, "value mismatch for %s", key)
		{{- else -}}
		assert.Equal(t, e.count, dp.Count, "count mismatch for %s", key)
		assert.Equal(t, e.val, dp.Sum, "sum mismatch for %s", key)
		{{- end}}
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
		{{- else -}}
	assert.Len(t, dataPoints, 0)
		{{- end -}}
	{{- else -}}
	assert.Len(t, dataPoints, 1)
	dp := dataPoints[0]
	{{if isCounter . -}}
	assert.Equal(t, int64(8), dp.Value)
	{{- else -}}
	assert.Equal(t, uint64(2), dp.Count)
	assert.Equal(t, int64(579), dp.Sum) // 123 + 456
	{{- end}}
	{{- end}}
}
{{end}}
