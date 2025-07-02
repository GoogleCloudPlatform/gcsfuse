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

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Data structures to parse metrics.yaml
type Metric struct {
	Name        string      `yaml:"metric-name"`
	Description string      `yaml:"description"`
	Type        string      `yaml:"type"`
	Unit        string      `yaml:"unit"`
	Attributes  []Attribute `yaml:"attributes"`
	Boundaries  []int64     `yaml:"boundaries"`
}

type Attribute struct {
	Name   string   `yaml:"attribute-name"`
	Type   string   `yaml:"attribute-type"`
	Values []string `yaml:"values"`
}

// AttrValuePair is a helper struct for generating combinations.
type AttrValuePair struct {
	Name  string
	Type  string
	Value string // "true"/"false" for bools
}

// AttrCombination is a list of AttrValuePairs.
type AttrCombination []AttrValuePair

// Data structure to pass to the template.
type TemplateData struct {
	Metrics          []Metric
	AttrCombinations map[string][]AttrCombination
}

// Helper functions for the template.
var funcMap = template.FuncMap{
	"toPascal":      toPascal,
	"toCamel":       toCamel,
	"getVarName":    getVarName,
	"getAtomicName": getAtomicName,
	"getGoType":     getGoType,
	"getUnitMethod": getUnitMethod,
	"joinInts":      joinInts,
	"isCounter":     func(m Metric) bool { return m.Type == "int_counter" },
	"isHistogram":   func(m Metric) bool { return m.Type == "int_histogram" },
	"buildSwitches": buildSwitches,
}

func toPascal(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "_", "-")
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func toCamel(s string) string {
	pascal := toPascal(s)
	if len(pascal) > 0 {
		return strings.ToLower(pascal[:1]) + pascal[1:]
	}
	return ""
}

func getVarName(metricName string, combo AttrCombination) string {
	var parts []string
	parts = append(parts, toCamel(metricName))
	for _, pair := range combo {
		parts = append(parts, toPascal(pair.Name))
		parts = append(parts, toPascal(pair.Value))
	}
	parts = append(parts, "AttrSet")
	return strings.Join(parts, "")
}

func getAtomicName(metricName string, combo AttrCombination) string {
	var parts []string
	parts = append(parts, toCamel(metricName))
	for _, pair := range combo {
		parts = append(parts, toPascal(pair.Name))
		parts = append(parts, toPascal(pair.Value))
	}
	parts = append(parts, "Atomic")
	return strings.Join(parts, "")
}

func getGoType(t string) string {
	switch t {
	case "string":
		return "string"
	case "bool":
		return "bool"
	default:
		return "interface{}"
	}
}

func getUnitMethod(unit string) string {
	switch unit {
	case "us":
		return ".Microseconds()"
	case "ms":
		return ".Milliseconds()"
	case "s":
		return ".Seconds()"
	default:
		// Assumes the value is already in the correct unit if not time-based.
		return ""
	}
}

func joinInts(nums []int64) string {
	var s []string
	for _, n := range nums {
		s = append(s, strconv.FormatInt(n, 10))
	}
	return strings.Join(s, ", ")
}

// generateCombinations creates all possible combinations of attribute values.
func generateCombinations(attributes []Attribute) []AttrCombination {
	if len(attributes) == 0 {
		return []AttrCombination{{}}
	}

	firstAttr := attributes[0]
	remainingAttrs := attributes[1:]
	combsOfRest := generateCombinations(remainingAttrs)

	var firstAttrValues []AttrValuePair
	if firstAttr.Type == "string" {
		for _, v := range firstAttr.Values {
			firstAttrValues = append(firstAttrValues, AttrValuePair{Name: firstAttr.Name, Type: "string", Value: v})
		}
	} else if firstAttr.Type == "bool" {
		firstAttrValues = append(firstAttrValues, AttrValuePair{Name: firstAttr.Name, Type: "bool", Value: "true"})
		firstAttrValues = append(firstAttrValues, AttrValuePair{Name: firstAttr.Name, Type: "bool", Value: "false"})
	}

	var result []AttrCombination
	for _, val := range firstAttrValues {
		for _, comb := range combsOfRest {
			newComb := append(AttrCombination{val}, comb...)
			result = append(result, newComb)
		}
	}
	return result
}

// buildSwitches generates the nested switch statement code for a metric method.
func buildSwitches(metric Metric) string {
	var builder strings.Builder
	var recorder func(level int, combo AttrCombination)

	recorder = func(level int, combo AttrCombination) {
		if level == len(metric.Attributes) {
			// Base case: record the metric
			indent := strings.Repeat("\t", level+1)
			if metric.Type == "int_counter" {
				atomicName := getAtomicName(metric.Name, combo)
				builder.WriteString(fmt.Sprintf("%so.%s.Add(inc)\n", indent, atomicName))
			} else { // histogram
				varName := getVarName(metric.Name, combo)
				unitMethod := getUnitMethod(metric.Unit)
				builder.WriteString(fmt.Sprintf("%so.%s.Record(ctx, latency%s, %s)\n", indent, toCamel(metric.Name), unitMethod, varName))
			}
			return
		}

		attr := metric.Attributes[level]
		indent := strings.Repeat("\t", level+1)
		builder.WriteString(fmt.Sprintf("%sswitch %s {\n", indent, toCamel(attr.Name)))

		var values []string
		if attr.Type == "string" {
			values = attr.Values
		} else { // bool
			values = []string{"true", "false"}
		}

		for _, val := range values {
			caseVal := val
			if attr.Type == "string" {
				caseVal = `"` + val + `"`
			}
			builder.WriteString(fmt.Sprintf("%scase %s:\n", strings.Repeat("\t", level+2), caseVal))
			currentCombo := append(combo, AttrValuePair{Name: attr.Name, Type: attr.Type, Value: val})
			recorder(level+1, currentCombo)
		}
		builder.WriteString(fmt.Sprintf("%s}\n", indent))
	}

	if len(metric.Attributes) == 0 {
		if metric.Type == "int_histogram" {
			unitMethod := getUnitMethod(metric.Unit)
			builder.WriteString(fmt.Sprintf("\to.%s.Record(ctx, latency%s)\n", toCamel(metric.Name), unitMethod))
		} else if metric.Type == "int_counter" {
			atomicName := getAtomicName(metric.Name, AttrCombination{})
			builder.WriteString(fmt.Sprintf("\to.%s.Add(inc)\n", atomicName))
		}
	} else {
		recorder(0, AttrCombination{})
	}

	return builder.String()
}

const codeTemplate = `// Copyright 2025 Google LLC
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
	meter = otel.Meter("gcsfuse")
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
{{buildSwitches .}}
}
{{end}}

func NewOTelMetrics() (*otelMetrics, error) {
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
`

func main() {
	inputFile := flag.String("input", "metrics.yaml", "Input YAML file")
	outputFile := flag.String("output", "otel_metrics.go", "Output Go file")
	flag.Parse()

	yamlFile, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("error reading yaml file: %v", err)
	}

	var metrics []Metric
	err = yaml.Unmarshal(yamlFile, &metrics)
	if err != nil {
		log.Fatalf("error unmarshalling yaml: %v", err)
	}

	// Sort attributes and their string values for deterministic output
	for i := range metrics {
		sort.Slice(metrics[i].Attributes, func(k, j int) bool {
			return metrics[i].Attributes[k].Name < metrics[i].Attributes[j].Name
		})
		for j := range metrics[i].Attributes {
			if metrics[i].Attributes[j].Type == "string" {
				sort.Strings(metrics[i].Attributes[j].Values)
			}
		}
	}

	attrCombinations := make(map[string][]AttrCombination)
	for _, m := range metrics {
		attrCombinations[m.Name] = generateCombinations(m.Attributes)
	}

	data := TemplateData{
		Metrics:          metrics,
		AttrCombinations: attrCombinations,
	}

	tmpl, err := template.New("otel_metrics").Funcs(funcMap).Parse(codeTemplate)
	if err != nil {
		log.Fatalf("error parsing template: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Fatalf("error executing template: %v", err)
	}

	// Create the directory if it doesn't exist
	outputDir := filepath.Dir(*outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("error creating output directory: %v", err)
	}

	err = os.WriteFile(*outputFile, buf.Bytes(), 0644)
	if err != nil {
		log.Fatalf("error writing output file: %v", err)
	}

	fmt.Printf("Successfully generated %s\n", *outputFile)
}
