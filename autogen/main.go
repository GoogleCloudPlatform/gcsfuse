// Copyright 2024 Google LLC
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
	"go/format"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// YAML Structures

type Metric struct {
	Name        string      `yaml:"metric-name"`
	Description string      `yaml:"description"`
	Unit        string      `yaml:"unit"`
	Attributes  []Attribute `yaml:"attributes"`
}

type Attribute struct {
	Name   string   `yaml:"attribute-name"`
	Type   string   `yaml:"attribute-type"`
	Values []string `yaml:"values,omitempty"`
}

// Template Data Structures

type TemplateData struct {
	PackageName string
	Metrics     []ProcessedMetric
	AttrMap     map[string]ProcessedAttribute
}

type ProcessedMetric struct {
	Name         string
	GoName       string
	Description  string
	Unit         string
	Attributes   []ProcessedAttribute
	Combinations []Combination
	SwitchTree   *SwitchNode
}

type ProcessedAttribute struct {
	Name   string
	GoName string
	GoType string
	Values []string
}

type Combination struct {
	Attributes     map[string]string
	AtomicVarName  string
	AttrSetVarName string
}

type SwitchNode struct {
	AttributeGoName string
	AttributeGoType string
	Children        map[string]*SwitchNode
	IsLeaf          bool
	LeafVarName     string
}

func main() {
	inputFile := flag.String("in", "metrics.yaml", "Input YAML file")
	outputFile := flag.String("out", "otel_metrics.go", "Output Go file")
	packageName := flag.String("pkg", "main", "Go package name for the generated file")
	flag.Parse()

	yamlFile, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}

	var metrics []Metric
	err = yaml.Unmarshal(yamlFile, &metrics)
	if err != nil {
		log.Fatalf("Error unmarshalling YAML: %v", err)
	}

	data := processMetrics(metrics, *packageName)

	tmpl, err := template.New(filepath.Base("otel_metrics.go.tmpl")).Funcs(template.FuncMap{
		"ToOtelType": toOtelType,
	}).ParseFiles("otel_metrics.go.tmpl")
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Fatalf("Error executing template: %v", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("Error formatting generated code: %v", err)
	}

	err = os.WriteFile(*outputFile, formatted, 0644)
	if err != nil {
		log.Fatalf("Error writing output file: %v", err)
	}

	fmt.Printf("Successfully generated %s\n", *outputFile)
}

func processMetrics(metrics []Metric, pkgName string) TemplateData {
	var processedMetrics []ProcessedMetric
	attrMap := make(map[string]ProcessedAttribute)

	for _, m := range metrics {
		pm := ProcessedMetric{
			Name:        m.Name,
			GoName:      toPascalCase(m.Name),
			Description: m.Description,
			Unit:        m.Unit,
		}

		for _, a := range m.Attributes {
			pa := ProcessedAttribute{
				Name:   a.Name,
				GoName: toCamelCase(a.Name),
				GoType: a.Type,
			}
			if a.Type == "bool" {
				pa.Values = []string{"true", "false"}
			} else {
				pa.Values = a.Values
			}
			pm.Attributes = append(pm.Attributes, pa)
			attrMap[pa.Name] = pa
		}

		// Sort attributes to ensure deterministic order for variable names
		sort.Slice(pm.Attributes, func(i, j int) bool {
			return pm.Attributes[i].Name < pm.Attributes[j].Name
		})

		pm.Combinations = generateCombinations(pm)
		pm.SwitchTree = buildSwitchTree(pm)
		processedMetrics = append(processedMetrics, pm)
	}

	return TemplateData{
		PackageName: pkgName,
		Metrics:     processedMetrics,
		AttrMap:     attrMap,
	}
}

func generateCombinations(pm ProcessedMetric) []Combination {
	var combinations []Combination

	var recurse func(int, map[string]string)
	recurse = func(attrIndex int, currentCombo map[string]string) {
		if attrIndex == len(pm.Attributes) {
			// Create a copy of the map
			finalCombo := make(map[string]string)
			var nameParts []string
			nameParts = append(nameParts, pm.GoName)

			// Sort keys for deterministic naming
			var sortedAttrNames []string
			for k := range currentCombo {
				sortedAttrNames = append(sortedAttrNames, k)
			}
			sort.Strings(sortedAttrNames)

			for _, attrName := range sortedAttrNames {
				val := currentCombo[attrName]
				finalCombo[attrName] = val
				nameParts = append(nameParts, toPascalCase(attrName), toPascalCase(val))
			}

			baseName := strings.Join(nameParts, "")
			combinations = append(combinations, Combination{
				Attributes:     finalCombo,
				AtomicVarName:  toCamelCase(baseName) + "Atomic",
				AttrSetVarName: toCamelCase(baseName) + "AttrSet",
			})
			return
		}

		attr := pm.Attributes[attrIndex]
		for _, val := range attr.Values {
			currentCombo[attr.Name] = val
			recurse(attrIndex+1, currentCombo)
		}
	}

	recurse(0, make(map[string]string))
	return combinations
}

func buildSwitchTree(pm ProcessedMetric) *SwitchNode {
	if len(pm.Attributes) == 0 {
		baseName := toPascalCase(pm.Name)
		return &SwitchNode{
			IsLeaf:      true,
			LeafVarName: toCamelCase(baseName) + "Atomic",
		}
	}

	var build func(int, map[string]string) *SwitchNode
	build = func(attrIndex int, path map[string]string) *SwitchNode {
		attr := pm.Attributes[attrIndex]
		node := &SwitchNode{
			AttributeGoName: attr.GoName,
			AttributeGoType: attr.GoType,
			Children:        make(map[string]*SwitchNode),
		}

		for _, val := range attr.Values {
			newPath := make(map[string]string)
			for k, v := range path {
				newPath[k] = v
			}
			newPath[attr.Name] = val

			if attrIndex+1 == len(pm.Attributes) {
				// Leaf node
				var nameParts []string
				nameParts = append(nameParts, pm.GoName)

				var sortedAttrNames []string
				for k := range newPath {
					sortedAttrNames = append(sortedAttrNames, k)
				}
				sort.Strings(sortedAttrNames)

				for _, attrName := range sortedAttrNames {
					v := newPath[attrName]
					nameParts = append(nameParts, toPascalCase(attrName), toPascalCase(v))
				}
				baseName := strings.Join(nameParts, "")

				node.Children[val] = &SwitchNode{
					IsLeaf:      true,
					LeafVarName: toCamelCase(baseName) + "Atomic",
				}
			} else {
				node.Children[val] = build(attrIndex+1, newPath)
			}
		}
		return node
	}

	return build(0, make(map[string]string))
}

// Template Helper Functions

func toPascalCase(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = cases.Title(language.English).String(s)
	return strings.ReplaceAll(s, " ", "")
}

func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if pascal == "" {
		return ""
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

func toOtelType(goType string) string {
	switch goType {
	case "string":
		return "String"
	case "bool":
		return "Bool"
	case "int64":
		return "Int64"
	default:
		return goType
	}
}
