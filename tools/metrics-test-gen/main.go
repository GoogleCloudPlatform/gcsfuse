// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"strings"
	"text/template" // NOLINT

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
	"isCounter":     func(m Metric) bool { return m.Type == "int_counter" },
	"isHistogram":   func(m Metric) bool { return m.Type == "int_histogram" },
	"getAttrMapKey": getAttrMapKey,
	"getUnit":       getUnit,
}

func toPascal(s string) string {
	s = strings.ReplaceAll(s, "::", "-")
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

func getUnit(unit string) string {
	switch unit {
	case "us":
		return "Microsecond"
	case "ms":
		return "Millisecond"
	case "s":
		return "Second"
	default:
		return "" // For "By" or empty unit
	}
}

func getAttrMapKey(combo AttrCombination) string {
	var parts []string
	// Sort to ensure deterministic key
	c := make(AttrCombination, len(combo))
	copy(c, combo)
	sort.Slice(c, func(i, j int) bool {
		return c[i].Name < c[j].Name
	})
	for _, pair := range c {
		parts = append(parts, fmt.Sprintf("%s=%s", pair.Name, pair.Value))
	}
	return strings.Join(parts, ";")
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

	if len(firstAttrValues) == 0 {
		return combsOfRest
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

func main() {
	inputFile := flag.String("input", "metrics.yaml", "Input YAML file")
	outputFile := flag.String("output", "otel_metrics_test.go", "Output Go test file")
	templateFile := flag.String("template", "template_test.tpl", "Template file for the test")
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

	tmpl, err := template.New(filepath.Base(*templateFile)).Funcs(funcMap).ParseFiles(*templateFile)
	if err != nil {
		log.Fatalf("error parsing template: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Fatalf("error executing template: %v", err)
	}

	outputDir := filepath.Dir(*outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("error creating output directory: %v", err)
	}

	err = os.WriteFile(*outputFile, buf.Bytes(), 0644)
	if err != nil {
		log.Fatalf("error writing output file: %v", err)
	}
}
