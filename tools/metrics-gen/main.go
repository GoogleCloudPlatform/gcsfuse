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
)

type Metric struct {
	Name        string `yaml:"flag-name"`
	Type        string
	Description string
	Attributes  []string
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

func validateMetrics(_ []Metric) error {
	return nil
}

func parseMetricsConfig() ([]Metric, error) {
	buf, err := os.ReadFile(*paramsFile)
	if err != nil {
		return nil, err
	}
	var metricsConfig []Metric
	dec := yaml.NewDecoder(bytes.NewReader(buf))
	dec.KnownFields(true)
	if err = dec.Decode(&metricsConfig); err != nil {
		return nil, err
	}
	if err = validateMetrics(metricsConfig); err != nil {
		return nil, err
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
}
