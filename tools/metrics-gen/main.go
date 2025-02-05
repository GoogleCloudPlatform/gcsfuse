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
	"os"

	"gopkg.in/yaml.v3"
)

var (
	outDir      = flag.String("outDir", "", "Output directory where the auto-generated files are to be placed")
	paramsFile  = flag.String("paramsFile", "", "Params YAML file")
	templateDir = flag.String("templateDir", ".", "Directory containing the template files")
	varNum      = 0
)

type Attribute struct {
	Name   string
	Type   string
	Values []string
}

type Metric struct {
	Name        string
	Type        string
	Description string
	Attributes  []string
	Buckets     []int
}

type MetricsConfig struct {
	Metrics    []Metric
	Attributes []Attribute
}

type templateData struct {
}

func validateFlags() error {
	if *paramsFile == "" {
		return fmt.Errorf("params filename cannot be empty")
	}
	if *outDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}
	if *templateDir == "" {
		return fmt.Errorf("template directory cannot be empty")
	}
	return nil
}

func validateMetrics(_ MetricsConfig) error {
	return nil
}

func parseMetricsConfig() (MetricsConfig, error) {
	buf, err := os.ReadFile(*paramsFile)
	if err != nil {
		return MetricsConfig{}, err
	}
	var metricsConfig MetricsConfig
	dec := yaml.NewDecoder(bytes.NewReader(buf))
	dec.KnownFields(true)
	if err = dec.Decode(&metricsConfig); err != nil {
		return MetricsConfig{}, err
	}
	if err = validateMetrics(metricsConfig); err != nil {
		return MetricsConfig{}, err
	}
	return metricsConfig, nil
}

func main() {
	flag.Parse()
	err := validateFlags()

	if err != nil {
		panic(err)
	}

	metricsConfig, err := parseMetricsConfig()
	if err != nil {
		panic(err)
	}

	fmt.Println(metricsConfig)

	varName := "prefix"

	for i := 0; i < len(metricsConfig.Metrics); i++ {
		switch metricsConfig.Metrics[i].Type {
		case "int_counter":
			handleIntCounter(metricsConfig.Metrics[i], metricsConfig.Attributes)
		case "int_histogram":
			handleIntHistogram(metricsConfig.Metrics[i], metricsConfig.Attributes)
		}

	}
}

func handleIntCounter(m Metric, varName *string) {
	fmt.Println(m.Name)
}

func handleIntHistogram(m Metric, attributes map[string]Attribute) {
	for _, attr := range m.Attributes {
		if x, ok := attributes[attr]; ok {
			fmt.Println(x)
		} else {
			panic("Attribute doesn't exist")
		}
	}
	fmt.Println(m.Name)
}

func genVarName() string {
	varNum++
	return fmt.Sprintf("var%d", varNum)

}
