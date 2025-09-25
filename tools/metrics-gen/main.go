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
	"sort"
	"strconv"
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

// AttrKey holds info about a distinct attribute key.
type AttrKey struct {
	// Original is the attribute name as defined in metrics.yaml.
	// e.g. "read_type"
	Original string
	// Constant is the Go constant name for the attribute key.
	// e.g. "ReadTypeAttrKey"
	Constant string
}

// StringAttrValue holds info about a distinct string attribute value.
type StringAttrValue struct {
	// Original is the attribute value as defined in metrics.yaml.
	// e.g. "Sequential"
	Original string
	// Constant is the Go constant name for the attribute value.
	// e.g. "SequentialAttrValue"
	Constant string
}

// AttrCombination is a list of AttrValuePairs.
type AttrCombination []AttrValuePair

// Data structure to pass to the template.
type TemplateData struct {
	Metrics           []Metric
	AttrCombinations  map[string][]AttrCombination
	AttrKeys          []AttrKey
	StringAttrValues  []StringAttrValue
	attrKeyToConst    map[string]string
	stringValueToConst map[string]string
}

// Helper functions for the template.
var funcMap template.FuncMap

func init() {
	funcMap = template.FuncMap{
		"toPascal":                    toPascal,
		"toCamel":                     toCamel,
		"getVarName":                  getVarName,
		"getAtomicName":               getAtomicName,
		"getGoType":                   getGoType,
		"getUnitMethod":               getUnitMethod,
		"joinInts":                    joinInts,
		"isCounter":                   func(m Metric) bool { return m.Type == "int_counter" },
		"isUpDownCounter":             func(m Metric) bool { return m.Type == "int_up_down_counter" },
		"isHistogram":                 func(m Metric) bool { return m.Type == "int_histogram" },
		"buildSwitches":               buildSwitches,
		"getTestName":                 getTestName,
		"getTestFuncArgs":             getTestFuncArgs,
		"getExpectedAttrs":            getExpectedAttrs,
		"getLatencyUnit":              getLatencyUnit,
		"getLatencyMethod":            getLatencyMethod,
		"getTestFuncArgsForHistogram": getTestFuncArgsForHistogram,
	}
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
		return "StringAttrValue"
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

// getTestName generates a test name from an attribute combination.
func getTestName(combo AttrCombination) string {
	if len(combo) == 0 {
		return "no_attributes"
	}
	var parts = make([]string, 0, len(combo)*2)
	for _, pair := range combo {
		parts = append(parts, pair.Name)
		parts = append(parts, pair.Value)
	}
	return strings.Join(parts, "_")
}

// getTestFuncArgs generates arguments for the metric function call in tests.
func getTestFuncArgs(combo AttrCombination) string {
	var parts []string
	for _, pair := range combo {
		if pair.Type == "string" {
			parts = append(parts, `StringAttrValue("`+pair.Value+`")`)
		} else {
			parts = append(parts, pair.Value)
		}
	}
	return strings.Join(parts, ", ")
}

// getExpectedAttrs generates attribute set for test expectations.
func getExpectedAttrs(combo AttrCombination, data TemplateData) string {
	var parts []string
	for _, pair := range combo {
		if pair.Type == "string" {
			keyConst := data.attrKeyToConst[pair.Name]
			parts = append(parts, fmt.Sprintf(`attribute.String(string(%s), "%s")`, keyConst, pair.Value))
		} else { // bool
			keyConst := data.attrKeyToConst[pair.Name]
			parts = append(parts, fmt.Sprintf(`attribute.Bool(string(%s), %s)`, keyConst, pair.Value))
		}
	}
	return strings.Join(parts, ", ")
}

func getLatencyUnit(unit string) string {
	switch unit {
	case "us":
		return "Microsecond"
	case "ms":
		return "Millisecond"
	case "s":
		return "Second"
	default:
		return ""
	}
}

func getLatencyMethod(unit string) string {
	return toPascal(getLatencyUnit(unit)) + "s"
}

func getTestFuncArgsForHistogram(prefix string, attrs []Attribute) string {
	var parts []string
	for _, attr := range attrs {
		parts = append(parts, prefix+"."+toCamel(attr.Name))
	}
	return strings.Join(parts, ", ")
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

func handleDefaultInSwitchCase(level int, attrName string, builder *strings.Builder) {
	builder.WriteString(fmt.Sprintf("%sdefault:\n", strings.Repeat("\t", level+2)))
	builder.WriteString(fmt.Sprintf("%supdateUnrecognizedAttribute(%s)\n", strings.Repeat("\t", level+3), toCamel(attrName)))
	builder.WriteString(fmt.Sprintf("%sreturn\n", strings.Repeat("\t", level+3)))
}

func validateMetric(m Metric) error {
	if m.Name == "" {
		return fmt.Errorf("metric-name is required")
	}
	if m.Description == "" {
		return fmt.Errorf("description is required for metric %q", m.Name)
	}
	if m.Type != "int_counter" && m.Type != "int_histogram" && m.Type != "int_up_down_counter" {
		return fmt.Errorf("type for metric %q must be 'int_counter', 'int_histogram', or 'int_up_down_counter', got %q", m.Name, m.Type)
	}

	if m.Type == "int_histogram" {
		if len(m.Boundaries) == 0 {
			return fmt.Errorf("boundaries are required for histogram metric %q", m.Name)
		}
	} else { // int_counter
		if len(m.Boundaries) > 0 {
			return fmt.Errorf("boundaries should not be present for counter metric %q", m.Name)
		}
	}

	for _, a := range m.Attributes {
		if a.Name == "" {
			return fmt.Errorf("attribute-name is required for an attribute in metric %q", m.Name)
		}
		if a.Type != "string" && a.Type != "bool" {
			return fmt.Errorf("attribute-type for attribute %q in metric %q must be 'string' or 'bool', got %q", a.Name, m.Name, a.Type)
		}

		if a.Type == "string" {
			if len(a.Values) == 0 {
				return fmt.Errorf("values are required for string attribute %q in metric %q", a.Name, m.Name)
			}
		}
		if a.Type == "bool" && len(a.Values) != 0 {
			return fmt.Errorf("values should not be present for bool attribute %q in metric %q", a.Name, m.Name)
		}
	}
	return nil
}

func validateForDuplicates(metrics []Metric) error {
	names := make(map[string]bool)
	for _, m := range metrics {
		if names[m.Name] {
			return fmt.Errorf("duplicate metric-name: %q", m.Name)
		}
		names[m.Name] = true
	}
	return nil
}

func validateSortOrder(metrics []Metric) error {
	for i := 1; i < len(metrics); i++ {
		if metrics[i-1].Name > metrics[i].Name {
			return fmt.Errorf("metrics are not sorted by name. %q should come before %q", metrics[i].Name, metrics[i-1].Name)
		}
	}
	return nil
}

func validateAttributeSortOrder(metrics []Metric) error {
	for _, m := range metrics {
		for i := 1; i < len(m.Attributes); i++ {
			if m.Attributes[i-1].Name > m.Attributes[i].Name {
				return fmt.Errorf("attributes for metric %q are not sorted by name. %q should come before %q", m.Name, m.Attributes[i].Name, m.Attributes[i-1].Name)
			}
		}
	}
	return nil
}

// validateMetrics checks for correctness of the metric definitions.
func validateMetrics(metrics []Metric) error {
	if err := validateForDuplicates(metrics); err != nil {
		return err
	}
	if err := validateSortOrder(metrics); err != nil {
		return err
	}
	if err := validateAttributeSortOrder(metrics); err != nil {
		return err
	}
	for _, m := range metrics {
		if err := validateMetric(m); err != nil {
			return err
		}
	}
	return nil
}

// buildSwitches generates the nested switch statement code for a metric method.
func buildSwitches(metric Metric, data TemplateData) string {
	var builder strings.Builder
	var recorder func(level int, combo AttrCombination)

	recorder = func(level int, combo AttrCombination) {
		if level == len(metric.Attributes) {
			// Base case: record the metric
			indent := strings.Repeat("\t", level+1)
			if metric.Type == "int_counter" || metric.Type == "int_up_down_counter" {
				atomicName := getAtomicName(metric.Name, combo)
				builder.WriteString(fmt.Sprintf("%so.%s.Add(inc)\n", indent, atomicName))
			} else { // histogram
				varName := getVarName(metric.Name, combo)
				unitMethod := getUnitMethod(metric.Unit)
				builder.WriteString(fmt.Sprintf("%srecord = histogramRecord{ctx: ctx,instrument: o.%s, value: latency%s, attributes: %s}\n", indent, toCamel(metric.Name), unitMethod, varName))
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
				caseVal = data.stringValueToConst[val]
			}
			builder.WriteString(fmt.Sprintf("%scase %s:\n", strings.Repeat("\t", level+2), caseVal))
			currentCombo := append(combo, AttrValuePair{Name: attr.Name, Type: attr.Type, Value: val})
			recorder(level+1, currentCombo)
		}
		if attr.Type == "string" {
			handleDefaultInSwitchCase(level, attr.Name, &builder)
		}
		builder.WriteString(fmt.Sprintf("%s}\n", indent))
	}

	if len(metric.Attributes) == 0 {
		if metric.Type == "int_histogram" {
			unitMethod := getUnitMethod(metric.Unit)
			builder.WriteString(fmt.Sprintf("\trecord = histogramRecord{ctx: ctx, instrument: o.%s, value: latency%s}\n", toCamel(metric.Name), unitMethod))
		} else if metric.Type == "int_counter" || metric.Type == "int_up_down_counter" {
			atomicName := getAtomicName(metric.Name, AttrCombination{})
			builder.WriteString(fmt.Sprintf("\to.%s.Add(inc)\n", atomicName))
		}
	} else {
		recorder(0, AttrCombination{})
	}

	return builder.String()
}

func validateAttributeConstants(keys []AttrKey, values []StringAttrValue) error {
	constants := make(map[string]string)

	for _, k := range keys {
		if original, ok := constants[k.Constant]; ok {
			return fmt.Errorf("attribute key constant collision: %q and %q both generate constant %q", original, k.Original, k.Constant)
		}
		constants[k.Constant] = k.Original
	}

	for _, v := range values {
		if original, ok := constants[v.Constant]; ok {
			return fmt.Errorf("string attribute value constant collision: %q and %q both generate constant %q", original, v.Original, v.Constant)
		}
		constants[v.Constant] = v.Original
	}

	return nil
}

func main() {
	inputFile := flag.String("input", "metrics.yaml", "Input YAML file")
	outputDir := flag.String("outDir", ".", "Output directory to dump artifacts.")
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

	// Validate metrics
	if err := validateMetrics(metrics); err != nil {
		log.Fatalf("invalid metrics.yaml: %v", err)
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

	// Extract unique attribute keys and string values.
	attrKeys := make(map[string]bool)
	stringAttrValues := make(map[string]bool)
	for _, m := range metrics {
		for _, a := range m.Attributes {
			attrKeys[a.Name] = true
			if a.Type == "string" {
				for _, v := range a.Values {
					stringAttrValues[v] = true
				}
			}
		}
	}

	// Convert to slices of structs for the template.
	var attrKeySlice []AttrKey
	attrKeyToConst := make(map[string]string)
	for k := range attrKeys {
		constant := toPascal(k) + "AttrKey"
		attrKeySlice = append(attrKeySlice, AttrKey{
			Original: k,
			Constant: constant,
		})
		attrKeyToConst[k] = constant
	}
	sort.Slice(attrKeySlice, func(i, j int) bool {
		return attrKeySlice[i].Original < attrKeySlice[j].Original
	})

	var stringAttrValueSlice []StringAttrValue
	stringValueToConst := make(map[string]string)
	for v := range stringAttrValues {
		constant := toPascal(v) + "AttrValue"
		stringAttrValueSlice = append(stringAttrValueSlice, StringAttrValue{
			Original: v,
			Constant: constant,
		})
		stringValueToConst[v] = constant
	}
	sort.Slice(stringAttrValueSlice, func(i, j int) bool {
		return stringAttrValueSlice[i].Original < stringAttrValueSlice[j].Original
	})

	if err := validateAttributeConstants(attrKeySlice, stringAttrValueSlice); err != nil {
		log.Fatalf("invalid metrics.yaml: %v", err)
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("error creating output directory: %v", err)
	}
	data := TemplateData{
		Metrics:           metrics,
		AttrCombinations:  attrCombinations,
		AttrKeys:          attrKeySlice,
		StringAttrValues:  stringAttrValueSlice,
		attrKeyToConst:    attrKeyToConst,
		stringValueToConst: stringValueToConst,
	}
	createFile(&data, fmt.Sprintf("%s/metric_handle.go", *outputDir), "metric_handle.tpl")
	createFile(&data, fmt.Sprintf("%s/noop_metrics.go", *outputDir), "noop_metrics.tpl")
	createFile(&data, fmt.Sprintf("%s/otel_metrics.go", *outputDir), "otel_metrics.tpl")
	createFile(&data, fmt.Sprintf("%s/otel_metrics_test.go", *outputDir), "otel_metrics_test.tpl")

}

func createFile(data *TemplateData, fName string, templateName string) {
	// We need to modify the funcMap to pass the TemplateData to some functions.
	// A bit of a hack, but it's the simplest way to give template functions access
	// to the top-level data.
	funcMap["getExpectedAttrs"] = func(combo AttrCombination) string {
		return getExpectedAttrs(combo, *data)
	}
	funcMap["buildSwitches"] = func(metric Metric) string {
		return buildSwitches(metric, *data)
	}

	tmpl, err := template.New(templateName).Funcs(funcMap).ParseFiles(templateName)
	if err != nil {
		log.Fatalf("error parsing template: %v", err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Fatalf("error executing template: %v", err)
	}

	if err := os.WriteFile(fName, buf.Bytes(), 0644); err != nil {
		log.Fatalf("error writing output file: %v", err)
	}
}
