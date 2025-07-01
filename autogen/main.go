// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// A program that generates otel_metrics.go from metrics.yaml.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

const (
	inputFile  = "metrics.yaml"
	outputFile = "otel_metrics_2.go"
	// tmplFile is unused as the template is inlined below.
)

// Metric corresponds to a single metric definition in metrics.yaml.
type Metric struct {
	Name        string      `yaml:"metric-name"`
	Description string      `yaml:"description"`
	Unit        string      `yaml:"unit"`
	Attributes  []Attribute `yaml:"attributes"`
}

// Attribute corresponds to a metric attribute in metrics.yaml.
type Attribute struct {
	Name   string   `yaml:"attribute-name"`
	Type   string   `yaml:"attribute-type"`
	Values []string `yaml:"values,omitempty"`
}

// TemplateData is the data structure passed to the template.
type TemplateData struct {
	Metrics []TemplateMetric
}

// TemplateMetric is a processed metric ready for code generation.
type TemplateMetric struct {
	Metric
	CamelCaseName string
	Combinations  []AttributeCombination
}

// AttributeCombination represents a single combination of attribute values for a metric.
type AttributeCombination struct {
	AtomicVarName     string
	AttrSetVarName    string
	AttrSetDefinition string
}

// toPascalCase converts a string from snake-case or kebab-case to PascalCase.
func toPascalCase(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// toCamelCase converts a string to camelCase.
func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) == 0 {
		return ""
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

// generateCombinations creates all possible attribute combinations for a given metric.
func generateCombinations(metric Metric) []AttributeCombination {
	type attrDomain struct {
		Attribute
		domain []string
	}

	var domains []attrDomain
	for _, attr := range metric.Attributes {
		d := attrDomain{Attribute: attr}
		if attr.Type == "bool" {
			d.domain = []string{"true", "false"}
		} else {
			d.domain = attr.Values
		}
		domains = append(domains, d)
	}

	var combinations []AttributeCombination
	var rec func(int, []string)

	rec = func(domainIndex int, currentCombo []string) {
		if domainIndex == len(domains) {
			var baseNameParts []string
			baseNameParts = append(baseNameParts, toCamelCase(metric.Name))

			var attrSetParts []string
			for i, val := range currentCombo {
				attr := domains[i].Attribute
				baseNameParts = append(baseNameParts, toPascalCase(attr.Name))
				baseNameParts = append(baseNameParts, toPascalCase(val))
				if attr.Type == "bool" {
					attrSetParts = append(attrSetParts, fmt.Sprintf(`attribute.Bool("%s", %s)`, attr.Name, val))
				} else {
					attrSetParts = append(attrSetParts, fmt.Sprintf(`attribute.String("%s", "%s")`, attr.Name, val))
				}
			}

			baseName := strings.Join(baseNameParts, "")
			combo := AttributeCombination{
				AtomicVarName:     baseName + "Atomic",
				AttrSetVarName:    baseName + "AttrSet",
				AttrSetDefinition: fmt.Sprintf("metric.WithAttributeSet(attribute.NewSet(%s))", strings.Join(attrSetParts, ", ")),
			}
			combinations = append(combinations, combo)
			return
		}

		for _, val := range domains[domainIndex].domain {
			rec(domainIndex+1, append(currentCombo, val))
		}
	}

	rec(0, []string{})
	return combinations
}

func main() {
	log.SetFlags(0)
	yamlFile, err := os.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Error reading %s: %v", inputFile, err)
	}

	var metrics []Metric
	if err = yaml.Unmarshal(yamlFile, &metrics); err != nil {
		log.Fatalf("Error unmarshalling YAML from %s: %v", inputFile, err)
	}

	templateData := TemplateData{}
	for _, m := range metrics {
		tm := TemplateMetric{
			Metric:        m,
			CamelCaseName: toCamelCase(m.Name),
			Combinations:  generateCombinations(m),
		}
		templateData.Metrics = append(templateData.Metrics, tm)
	}

	tmpl, err := template.New("otel").Parse(otelTemplate)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, templateData); err != nil {
		log.Fatalf("Error executing template: %v", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("Error formatting generated code: %v\n---CODE---\n%s", err, buf.String())
	}

	outDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("Error creating output directory %s: %v", outDir, err)
	}

	if err := os.WriteFile(outputFile, formatted, 0644); err != nil {
		log.Fatalf("Error writing to %s: %v", outputFile, err)
	}

	fmt.Printf("Successfully generated %s from %s\n", outputFile, inputFile)
}

const otelTemplate = `// Copyright 2024 Google LLC
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

// Autogenerated by autogen/main.go, do not edit.

package common

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("gcsfuse")
)

type otelMetrics struct {
{{- range .Metrics}}
{{- range .Combinations}}
	{{.AtomicVarName}} *atomic.Int64
{{- end}}
{{- end}}
}

func NewOTelMetrics() (*otelMetrics, error) {
	var (
	{{- range .Metrics}}
	{{- range .Combinations}}
		{{.AtomicVarName}} atomic.Int64
	{{- end}}
	{{- end}}
	)

	{{- range .Metrics}}
	{{- range .Combinations}}
	{{.AttrSetVarName}} := {{.AttrSetDefinition}}
	{{- end}}
	{{- end}}

	{{- range .Metrics}}
	if _, err := meter.Int64ObservableCounter("{{.Name}}",
		metric.WithDescription("{{.Description}}"),
		metric.WithUnit("{{.Unit}}"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			{{- range .Combinations}}
			obsrv.Observe({{.AtomicVarName}}.Load(), {{.AttrSetVarName}})
			{{- end}}
			return nil
		})); err != nil {
		return nil, err
	}
	{{- end}}

	return &otelMetrics{
		{{- range .Metrics}}
		{{- range .Combinations}}
		{{.AtomicVarName}}: &{{.AtomicVarName}},
		{{- end}}
		{{- end}}
	}, nil
}
`
